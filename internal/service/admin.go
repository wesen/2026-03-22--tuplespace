package service

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/manuel/wesen/tuplespace/internal/admin"
	"github.com/manuel/wesen/tuplespace/internal/notify"
	"github.com/manuel/wesen/tuplespace/internal/validation"
)

var requiredSchemaTables = []string{"tuples", "tuple_fields"}
var requiredSchemaIndexes = []string{
	"tuples_space_arity_id_idx",
	"tuple_fields_text_idx",
	"tuple_fields_int_idx",
	"tuple_fields_bool_idx",
}

func (s *Service) Spaces(ctx context.Context) ([]admin.SpaceSummary, error) {
	return s.store.ListSpaces(ctx, s.db)
}

func (s *Service) Dump(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	if err := validateTupleFilter(filter); err != nil {
		return nil, err
	}
	return s.store.ListTuples(ctx, s.db, filter)
}

func (s *Service) Peek(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	return s.Dump(ctx, filter)
}

func (s *Service) Export(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	return s.Dump(ctx, filter)
}

func (s *Service) Stats(ctx context.Context) (admin.StatsSnapshot, error) {
	spaceSummaries, err := s.store.ListSpaces(ctx, s.db)
	if err != nil {
		return admin.StatsSnapshot{}, err
	}
	tupleCount, err := s.store.CountTuples(ctx, s.db, admin.TupleFilter{})
	if err != nil {
		return admin.StatsSnapshot{}, err
	}
	notifierSnapshot, err := s.notifier.Snapshot()
	if err != nil {
		return admin.StatsSnapshot{}, err
	}

	waiters := s.waitersSnapshot()
	return admin.StatsSnapshot{
		StartedAt:           s.startedAt,
		UptimeMS:            time.Since(s.startedAt).Milliseconds(),
		SpaceCount:          len(spaceSummaries),
		TupleCount:          tupleCount,
		WaiterCount:         len(waiters),
		NotifierChannels:    notifierSnapshot.ChannelCount,
		NotifierSubscribers: notifierSnapshot.SubscriberCount,
		NotifierByChannel:   notifierSnapshot.Channels,
		CandidateLimit:      s.candidateLimit,
	}, nil
}

func (s *Service) Config(ctx context.Context) (admin.ConfigSnapshot, error) {
	_ = ctx
	return s.configSnapshot, nil
}

func (s *Service) Schema(ctx context.Context) (admin.SchemaSnapshot, error) {
	tables, err := s.listSchemaObjects(ctx, `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = current_schema()
	`)
	if err != nil {
		return admin.SchemaSnapshot{}, err
	}
	indexes, err := s.listSchemaObjects(ctx, `
		SELECT indexname
		FROM pg_indexes
		WHERE schemaname = current_schema()
	`)
	if err != nil {
		return admin.SchemaSnapshot{}, err
	}

	sort.Strings(tables)
	sort.Strings(indexes)

	return admin.SchemaSnapshot{
		MigrationFiles: append([]string(nil), s.migrationFiles...),
		Tables:         tables,
		Indexes:        indexes,
		MissingTables:  missingStrings(requiredSchemaTables, tables),
		MissingIndexes: missingStrings(requiredSchemaIndexes, indexes),
	}, nil
}

func (s *Service) GetTuple(ctx context.Context, tupleID int64) (admin.TupleRecord, bool, error) {
	if tupleID <= 0 {
		return admin.TupleRecord{}, false, fmt.Errorf("tuple id must be > 0")
	}
	return s.store.GetTupleByID(ctx, s.db, tupleID)
}

func (s *Service) DeleteTuple(ctx context.Context, tupleID int64) (admin.DeleteResult, error) {
	if tupleID <= 0 {
		return admin.DeleteResult{}, fmt.Errorf("tuple id must be > 0")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return admin.DeleteResult{}, fmt.Errorf("begin delete transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	deleted, err := s.store.DeleteTupleByID(ctx, tx, tupleID)
	if err != nil {
		return admin.DeleteResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return admin.DeleteResult{}, fmt.Errorf("commit delete transaction: %w", err)
	}
	return admin.DeleteResult{TupleID: tupleID, Deleted: deleted}, nil
}

func (s *Service) Purge(ctx context.Context, filter admin.TupleFilter) (admin.PurgeResult, error) {
	if err := validateTupleFilter(filter); err != nil {
		return admin.PurgeResult{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return admin.PurgeResult{}, fmt.Errorf("begin purge transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	deletedCount, err := s.store.DeleteTuples(ctx, tx, filter)
	if err != nil {
		return admin.PurgeResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return admin.PurgeResult{}, fmt.Errorf("commit purge transaction: %w", err)
	}
	return admin.PurgeResult{DeletedCount: deletedCount}, nil
}

func (s *Service) Waiters(ctx context.Context) ([]admin.WaiterInfo, error) {
	_ = ctx
	return s.waitersSnapshot(), nil
}

func (s *Service) NotifyTest(ctx context.Context, space string) (admin.NotifyTestResult, error) {
	if err := validation.ValidateSpace(space); err != nil {
		return admin.NotifyTestResult{}, err
	}
	if err := s.notifier.Notify(ctx, space); err != nil {
		return admin.NotifyTestResult{}, err
	}
	snapshot, err := s.notifier.Snapshot()
	if err != nil {
		return admin.NotifyTestResult{}, err
	}
	channel := notify.ChannelName(space)
	return admin.NotifyTestResult{
		Space:                  space,
		Channel:                channel,
		SubscriberCount:        snapshot.SubscriberCount,
		ChannelSubscriberCount: snapshot.Channels[channel],
		NotifierChannels:       snapshot.ChannelCount,
		NotifierByChannel:      snapshot.Channels,
	}, nil
}

func validateTupleFilter(filter admin.TupleFilter) error {
	if filter.Space != "" {
		if err := validation.ValidateSpace(filter.Space); err != nil {
			return err
		}
	}
	if filter.Limit < 0 {
		return fmt.Errorf("limit must be >= 0")
	}
	if filter.Offset < 0 {
		return fmt.Errorf("offset must be >= 0")
	}
	if filter.CreatedBefore != nil && filter.CreatedAfter != nil && !filter.CreatedAfter.Before(*filter.CreatedBefore) {
		return fmt.Errorf("created_after must be before created_before")
	}
	return nil
}

func (s *Service) listSchemaObjects(ctx context.Context, query string) ([]string, error) {
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query schema objects: %w", err)
	}
	defer rows.Close()

	ret := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema object: %w", err)
		}
		ret = append(ret, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema objects: %w", err)
	}
	return ret, nil
}

func missingStrings(required []string, present []string) []string {
	presentSet := map[string]struct{}{}
	for _, value := range present {
		presentSet[value] = struct{}{}
	}

	ret := make([]string, 0)
	for _, value := range required {
		if _, ok := presentSet[value]; !ok {
			ret = append(ret, value)
		}
	}
	return ret
}

func RedactedConfigSnapshot(listenAddr string, databaseURL string, candidateLimit int, shutdownGrace time.Duration) admin.ConfigSnapshot {
	snapshot := admin.ConfigSnapshot{
		HTTPListenAddr: listenAddr,
		DatabaseURL:    "<redacted>",
		CandidateLimit: candidateLimit,
		ShutdownGrace:  shutdownGrace.String(),
	}

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return snapshot
	}
	snapshot.DatabaseHost = parsed.Hostname()
	snapshot.DatabaseName = strings.TrimPrefix(parsed.Path, "/")

	redacted := *parsed
	if redacted.User != nil {
		username := redacted.User.Username()
		redacted.User = url.UserPassword(username, "redacted")
	}
	snapshot.DatabaseURL = redacted.String()
	return snapshot
}
