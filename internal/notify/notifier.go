package notify

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

type Subscription interface {
	C() <-chan struct{}
	Close() error
}

type Notifier struct {
	conn      *pgx.Conn
	controlCh chan controlRequest
	doneCh    chan struct{}
}

type waitResult struct {
	notification *pgconn.Notification
	err          error
}

type controlRequest struct {
	apply func(*state) error
	resp  chan error
}

type state struct {
	refCounts    map[string]int
	subscribers  map[string]map[*subscription]struct{}
	closed       bool
	deliveryDone chan struct{}
}

type subscription struct {
	ch       chan struct{}
	notifier *Notifier
	channel  string
	closed   bool
}

func New(ctx context.Context, databaseURL string) (*Notifier, error) {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect notifier: %w", err)
	}

	notifier := &Notifier{
		conn:      conn,
		controlCh: make(chan controlRequest),
		doneCh:    make(chan struct{}),
	}
	log.Debug().Msg("initialized notifier connection")
	go notifier.loop()
	return notifier, nil
}

func (n *Notifier) Subscribe(space string) (Subscription, error) {
	channel := ChannelName(space)
	sub := &subscription{
		ch:       make(chan struct{}, 1),
		notifier: n,
		channel:  channel,
	}

	if err := n.execute(func(st *state) error {
		if st.subscribers[channel] == nil {
			st.subscribers[channel] = map[*subscription]struct{}{}
		}
		if st.refCounts[channel] == 0 {
			if _, err := n.conn.Exec(context.Background(), "LISTEN "+channel); err != nil {
				return fmt.Errorf("listen %s: %w", channel, err)
			}
		}
		st.refCounts[channel]++
		st.subscribers[channel][sub] = struct{}{}
		log.Debug().
			Str("space", space).
			Str("channel", channel).
			Int("subscriber_count", st.refCounts[channel]).
			Msg("subscribed to tuplespace notifications")
		return nil
	}); err != nil {
		return nil, err
	}

	return sub, nil
}

func (n *Notifier) Close() error {
	err := n.execute(func(st *state) error {
		if st.closed {
			return nil
		}
		st.closed = true
		return nil
	})
	<-n.doneCh
	return err
}

func (n *Notifier) loop() {
	st := &state{
		refCounts:    map[string]int{},
		subscribers:  map[string]map[*subscription]struct{}{},
		deliveryDone: make(chan struct{}),
	}
	defer close(n.doneCh)
	defer n.conn.Close(context.Background())

	for {
		if st.closed {
			return
		}

		if len(st.refCounts) == 0 {
			req := <-n.controlCh
			req.resp <- req.apply(st)
			continue
		}

		waitCtx, cancel := context.WithCancel(context.Background())
		waitCh := make(chan waitResult, 1)
		go func() {
			notification, err := n.conn.WaitForNotification(waitCtx)
			waitCh <- waitResult{notification: notification, err: err}
		}()

		select {
		case req := <-n.controlCh:
			cancel()
			result := <-waitCh
			if result.notification != nil {
				deliverNotification(st, result.notification)
			}
			if result.err != nil && !errors.Is(result.err, context.Canceled) {
				log.Warn().Err(result.err).Msg("interrupted notification wait returned unexpected error")
			}
			req.resp <- req.apply(st)
		case result := <-waitCh:
			cancel()
			if result.err != nil {
				if errors.Is(result.err, context.Canceled) {
					continue
				}
				log.Error().Err(result.err).Msg("wait for postgres notification failed")
				time.Sleep(100 * time.Millisecond)
				continue
			}
			if result.notification != nil {
				deliverNotification(st, result.notification)
			}
		}
	}
}

func (n *Notifier) execute(apply func(*state) error) error {
	resp := make(chan error, 1)
	n.controlCh <- controlRequest{apply: apply, resp: resp}
	return <-resp
}

func (n *Notifier) unsubscribe(sub *subscription) error {
	return n.execute(func(st *state) error {
		if st.closed || sub.closed {
			return nil
		}
		sub.closed = true

		subs := st.subscribers[sub.channel]
		delete(subs, sub)
		if len(subs) == 0 {
			delete(st.subscribers, sub.channel)
		}
		if st.refCounts[sub.channel] > 0 {
			st.refCounts[sub.channel]--
		}
		if st.refCounts[sub.channel] == 0 {
			delete(st.refCounts, sub.channel)
			if _, err := n.conn.Exec(context.Background(), "UNLISTEN "+sub.channel); err != nil {
				return fmt.Errorf("unlisten %s: %w", sub.channel, err)
			}
		}
		close(sub.ch)
		log.Debug().
			Str("channel", sub.channel).
			Int("remaining_subscribers", st.refCounts[sub.channel]).
			Msg("unsubscribed from tuplespace notifications")
		return nil
	})
}

func (s *subscription) C() <-chan struct{} {
	return s.ch
}

func (s *subscription) Close() error {
	return s.notifier.unsubscribe(s)
}

func ChannelName(space string) string {
	sum := sha1.Sum([]byte(space))
	return "tuplespace_" + hex.EncodeToString(sum[:8])
}

func deliverNotification(st *state, notification *pgconn.Notification) {
	subscribers := st.subscribers[notification.Channel]
	if len(subscribers) == 0 {
		log.Debug().
			Str("channel", notification.Channel).
			Msg("received notification without active subscribers")
		return
	}

	delivered := 0
	for sub := range subscribers {
		select {
		case sub.ch <- struct{}{}:
			delivered++
		default:
		}
	}

	log.Debug().
		Str("channel", notification.Channel).
		Int("subscriber_count", len(subscribers)).
		Int("delivered_count", delivered).
		Msg("delivered postgres notification")
}
