package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/manuel/wesen/tuplespace/internal/admin"
	"github.com/manuel/wesen/tuplespace/internal/match"
	"github.com/manuel/wesen/tuplespace/internal/notify"
	"github.com/manuel/wesen/tuplespace/internal/store"
	"github.com/manuel/wesen/tuplespace/internal/types"
	"github.com/manuel/wesen/tuplespace/internal/validation"
)

type TupleSpace interface {
	Out(ctx context.Context, space string, tuple types.Tuple) error
	Rd(ctx context.Context, space string, template types.Template, wait time.Duration) (types.Tuple, types.Bindings, error)
	In(ctx context.Context, space string, template types.Template, wait time.Duration) (types.Tuple, types.Bindings, error)
	Rdp(ctx context.Context, space string, template types.Template) (types.Tuple, types.Bindings, bool, error)
	Inp(ctx context.Context, space string, template types.Template) (types.Tuple, types.Bindings, bool, error)
	Spaces(ctx context.Context) ([]admin.SpaceSummary, error)
	Dump(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error)
	Peek(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error)
	Export(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error)
	Stats(ctx context.Context) (admin.StatsSnapshot, error)
	Config(ctx context.Context) (admin.ConfigSnapshot, error)
	Schema(ctx context.Context) (admin.SchemaSnapshot, error)
	GetTuple(ctx context.Context, tupleID int64) (admin.TupleRecord, bool, error)
	DeleteTuple(ctx context.Context, tupleID int64) (admin.DeleteResult, error)
	Purge(ctx context.Context, filter admin.TupleFilter) (admin.PurgeResult, error)
	Waiters(ctx context.Context) ([]admin.WaiterInfo, error)
	NotifyTest(ctx context.Context, space string) (admin.NotifyTestResult, error)
}

type Service struct {
	db             *pgxpool.Pool
	store          *store.TupleStore
	notifier       *notify.Notifier
	candidateLimit int
	startedAt      time.Time
	configSnapshot admin.ConfigSnapshot
	migrationFiles []string

	waitersMu     sync.Mutex
	nextWaiterID  uint64
	activeWaiters map[uint64]admin.WaiterInfo
}

type Options struct {
	CandidateLimit int
	StartedAt      time.Time
	ConfigSnapshot admin.ConfigSnapshot
	MigrationFiles []string
}

func New(db *pgxpool.Pool, tupleStore *store.TupleStore, notifier *notify.Notifier, options Options) *Service {
	if options.CandidateLimit <= 0 {
		options.CandidateLimit = 64
	}
	if options.StartedAt.IsZero() {
		options.StartedAt = time.Now().UTC()
	}
	return &Service{
		db:             db,
		store:          tupleStore,
		notifier:       notifier,
		candidateLimit: options.CandidateLimit,
		startedAt:      options.StartedAt,
		configSnapshot: options.ConfigSnapshot,
		migrationFiles: append([]string(nil), options.MigrationFiles...),
		activeWaiters:  map[uint64]admin.WaiterInfo{},
	}
}

func (s *Service) Out(ctx context.Context, space string, tuple types.Tuple) error {
	if err := validation.ValidateSpace(space); err != nil {
		return err
	}
	normalizedTuple, err := validation.ValidateTuple(tuple)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin out transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := s.store.InsertTuple(ctx, tx, space, normalizedTuple); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_notify($1, '')`, notify.ChannelName(space)); err != nil {
		return fmt.Errorf("notify listeners: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit out transaction: %w", err)
	}

	log.Debug().
		Str("space", space).
		Int("arity", len(normalizedTuple.Fields)).
		Msg("stored tuple")
	return nil
}

func (s *Service) Rd(ctx context.Context, space string, template types.Template, wait time.Duration) (types.Tuple, types.Bindings, error) {
	return s.read(ctx, space, template, wait, false)
}

func (s *Service) In(ctx context.Context, space string, template types.Template, wait time.Duration) (types.Tuple, types.Bindings, error) {
	return s.read(ctx, space, template, wait, true)
}

func (s *Service) Rdp(ctx context.Context, space string, template types.Template) (types.Tuple, types.Bindings, bool, error) {
	tuple, bindings, err := s.Rd(ctx, space, template, 0)
	if err == ErrNotFound {
		return types.Tuple{}, nil, false, nil
	}
	return tuple, bindings, err == nil, err
}

func (s *Service) Inp(ctx context.Context, space string, template types.Template) (types.Tuple, types.Bindings, bool, error) {
	tuple, bindings, err := s.In(ctx, space, template, 0)
	if err == ErrNotFound {
		return types.Tuple{}, nil, false, nil
	}
	return tuple, bindings, err == nil, err
}

func (s *Service) read(ctx context.Context, space string, template types.Template, wait time.Duration, destructive bool) (types.Tuple, types.Bindings, error) {
	if err := validation.ValidateSpace(space); err != nil {
		return types.Tuple{}, nil, err
	}
	normalizedTemplate, err := validation.ValidateTemplate(template)
	if err != nil {
		return types.Tuple{}, nil, err
	}
	if wait < 0 {
		return types.Tuple{}, nil, fmt.Errorf("wait duration must be >= 0")
	}

	log.Debug().
		Str("space", space).
		Bool("destructive", destructive).
		Dur("wait", wait).
		Int("arity", len(normalizedTemplate.Fields)).
		Msg("starting tuple read")

	opCtx, cancel := withOptionalTimeout(ctx, wait)
	defer cancel()

	var sub notify.Subscription
	var waiterID uint64
	waiterRegistered := false
	if wait > 0 {
		sub, err = s.notifier.Subscribe(space)
		if err != nil {
			return types.Tuple{}, nil, err
		}
		defer sub.Close()
		defer func() {
			if waiterRegistered {
				s.unregisterWaiter(waiterID)
			}
		}()
	}

	for {
		if destructive {
			tuple, bindings, found, err := s.consumeOnce(opCtx, space, normalizedTemplate)
			if err != nil {
				return types.Tuple{}, nil, err
			}
			if found {
				log.Debug().
					Str("space", space).
					Bool("destructive", destructive).
					Msg("matched tuple")
				return tuple, bindings, nil
			}
		} else {
			tuple, bindings, found, err := s.readOnce(opCtx, space, normalizedTemplate)
			if err != nil {
				return types.Tuple{}, nil, err
			}
			if found {
				log.Debug().
					Str("space", space).
					Bool("destructive", destructive).
					Msg("matched tuple")
				return tuple, bindings, nil
			}
		}

		if wait == 0 {
			log.Debug().
				Str("space", space).
				Bool("destructive", destructive).
				Msg("no matching tuple found")
			return types.Tuple{}, nil, ErrNotFound
		}

		if !waiterRegistered {
			waiterID = s.registerWaiter(space, normalizedTemplate, wait, destructive)
			waiterRegistered = true
		}

		log.Debug().
			Str("space", space).
			Bool("destructive", destructive).
			Msg("waiting for tuple notification")
		select {
		case <-opCtx.Done():
			if opCtx.Err() == context.DeadlineExceeded {
				log.Debug().
					Str("space", space).
					Bool("destructive", destructive).
					Dur("wait", wait).
					Msg("tuple read timed out")
				return types.Tuple{}, nil, ErrTimeout
			}
			return types.Tuple{}, nil, opCtx.Err()
		case <-sub.C():
			log.Debug().
				Str("space", space).
				Bool("destructive", destructive).
				Msg("retrying tuple read after notification")
		}
	}
}

func (s *Service) readOnce(ctx context.Context, space string, template types.Template) (types.Tuple, types.Bindings, bool, error) {
	candidates, err := s.store.FindCandidates(ctx, s.db, space, template, s.candidateLimit)
	if err != nil {
		return types.Tuple{}, nil, false, err
	}
	for _, candidate := range candidates {
		bindings, ok := match.Match(template, candidate.Tuple)
		if ok {
			return candidate.Tuple, bindings, true, nil
		}
	}
	return types.Tuple{}, nil, false, nil
}

func (s *Service) consumeOnce(ctx context.Context, space string, template types.Template) (types.Tuple, types.Bindings, bool, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return types.Tuple{}, nil, false, fmt.Errorf("begin consume transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	candidates, err := s.store.LockCandidatesForConsume(ctx, tx, space, template, s.candidateLimit)
	if err != nil {
		return types.Tuple{}, nil, false, err
	}
	for _, candidate := range candidates {
		bindings, ok := match.Match(template, candidate.Tuple)
		if !ok {
			continue
		}
		if err := s.store.DeleteTuple(ctx, tx, candidate.ID); err != nil {
			return types.Tuple{}, nil, false, err
		}
		if err := tx.Commit(ctx); err != nil {
			return types.Tuple{}, nil, false, fmt.Errorf("commit consume transaction: %w", err)
		}
		return candidate.Tuple, bindings, true, nil
	}
	return types.Tuple{}, nil, false, nil
}

func withOptionalTimeout(ctx context.Context, wait time.Duration) (context.Context, context.CancelFunc) {
	if wait <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, wait)
}

func (s *Service) registerWaiter(space string, template types.Template, wait time.Duration, destructive bool) uint64 {
	s.waitersMu.Lock()
	defer s.waitersMu.Unlock()

	s.nextWaiterID++
	id := s.nextWaiterID
	operation := "rd"
	if destructive {
		operation = "in"
	}
	s.activeWaiters[id] = admin.WaiterInfo{
		ID:        id,
		Space:     space,
		Operation: operation,
		WaitMS:    wait.Milliseconds(),
		StartedAt: time.Now().UTC(),
		Template:  template,
	}
	return id
}

func (s *Service) unregisterWaiter(id uint64) {
	s.waitersMu.Lock()
	defer s.waitersMu.Unlock()
	delete(s.activeWaiters, id)
}

func (s *Service) waitersSnapshot() []admin.WaiterInfo {
	s.waitersMu.Lock()
	defer s.waitersMu.Unlock()

	ret := make([]admin.WaiterInfo, 0, len(s.activeWaiters))
	for _, waiter := range s.activeWaiters {
		ret = append(ret, waiter)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].StartedAt.Equal(ret[j].StartedAt) {
			return ret[i].ID < ret[j].ID
		}
		return ret[i].StartedAt.Before(ret[j].StartedAt)
	})
	return ret
}
