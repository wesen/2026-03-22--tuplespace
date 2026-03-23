package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/manuel/wesen/tuplespace/internal/admin"
)

type RowQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func (s *TupleStore) ListSpaces(ctx context.Context, q Queryer) ([]admin.SpaceSummary, error) {
	rows, err := q.Query(ctx, `
		SELECT space, COUNT(*), MIN(created_at), MAX(created_at)
		FROM tuples
		GROUP BY space
		ORDER BY space
	`)
	if err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}
	defer rows.Close()

	ret := make([]admin.SpaceSummary, 0)
	for rows.Next() {
		var summary admin.SpaceSummary
		if err := rows.Scan(&summary.Space, &summary.TupleCount, &summary.OldestTupleAt, &summary.NewestTupleAt); err != nil {
			return nil, fmt.Errorf("scan space summary: %w", err)
		}
		ret = append(ret, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate space summaries: %w", err)
	}
	return ret, nil
}

func (s *TupleStore) ListTuples(ctx context.Context, q Queryer, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	query, args := buildTupleFilterQuery(`
		SELECT id, space, arity, created_at, fields_json
		FROM tuples
	`, filter, true)

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tuples: %w", err)
	}
	defer rows.Close()

	ret := make([]admin.TupleRecord, 0)
	for rows.Next() {
		record, err := scanAdminTupleRecord(rows)
		if err != nil {
			return nil, err
		}
		ret = append(ret, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tuples: %w", err)
	}
	return ret, nil
}

func (s *TupleStore) CountTuples(ctx context.Context, q RowQueryer, filter admin.TupleFilter) (int64, error) {
	query, args := buildTupleFilterQuery(`
		SELECT COUNT(*)
		FROM tuples
	`, filter, false)

	var count int64
	if err := q.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count tuples: %w", err)
	}
	return count, nil
}

func (s *TupleStore) GetTupleByID(ctx context.Context, q RowQueryer, tupleID int64) (admin.TupleRecord, bool, error) {
	row := q.QueryRow(ctx, `
		SELECT id, space, arity, created_at, fields_json
		FROM tuples
		WHERE id = $1
	`, tupleID)

	record, err := scanAdminTupleRecord(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return admin.TupleRecord{}, false, nil
		}
		return admin.TupleRecord{}, false, err
	}
	return record, true, nil
}

func (s *TupleStore) DeleteTupleByID(ctx context.Context, q Execer, tupleID int64) (bool, error) {
	tag, err := q.Exec(ctx, `DELETE FROM tuples WHERE id = $1`, tupleID)
	if err != nil {
		return false, fmt.Errorf("delete tuple by id %d: %w", tupleID, err)
	}
	return tag.RowsAffected() > 0, nil
}

func (s *TupleStore) DeleteTuples(ctx context.Context, q Execer, filter admin.TupleFilter) (int64, error) {
	query, args := buildTupleDeleteQuery(filter)
	tag, err := q.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("delete tuples: %w", err)
	}
	return tag.RowsAffected(), nil
}

type adminTupleScanner interface {
	Scan(dest ...any) error
}

func scanAdminTupleRecord(scanner adminTupleScanner) (admin.TupleRecord, error) {
	var (
		record  admin.TupleRecord
		payload []byte
	)
	if err := scanner.Scan(&record.ID, &record.Space, &record.Arity, &record.CreatedAt, &payload); err != nil {
		return admin.TupleRecord{}, err
	}
	tuple, err := decodeTuple(payload)
	if err != nil {
		return admin.TupleRecord{}, err
	}
	record.Tuple = tuple
	return record, nil
}

func buildTupleFilterQuery(base string, filter admin.TupleFilter, includePage bool) (string, []any) {
	args := make([]any, 0, 4)
	arg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	var builder strings.Builder
	builder.WriteString(strings.TrimSpace(base))
	builder.WriteString(" WHERE 1=1")
	appendTupleFilter(&builder, arg, filter)

	if includePage {
		builder.WriteString(" ORDER BY space, id")
		if filter.Limit > 0 {
			builder.WriteString(" LIMIT ")
			builder.WriteString(arg(filter.Limit))
		}
		if filter.Offset > 0 {
			builder.WriteString(" OFFSET ")
			builder.WriteString(arg(filter.Offset))
		}
	}

	return builder.String(), args
}

func buildTupleDeleteQuery(filter admin.TupleFilter) (string, []any) {
	args := make([]any, 0, 4)
	arg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	var builder strings.Builder
	builder.WriteString("DELETE FROM tuples WHERE 1=1")
	appendTupleFilter(&builder, arg, filter)
	return builder.String(), args
}

func appendTupleFilter(builder *strings.Builder, arg func(any) string, filter admin.TupleFilter) {
	if filter.Space != "" {
		builder.WriteString(" AND space = ")
		builder.WriteString(arg(filter.Space))
	}
	if filter.CreatedBefore != nil {
		builder.WriteString(" AND created_at < ")
		builder.WriteString(arg(*filter.CreatedBefore))
	}
	if filter.CreatedAfter != nil {
		builder.WriteString(" AND created_at > ")
		builder.WriteString(arg(*filter.CreatedAfter))
	}
}
