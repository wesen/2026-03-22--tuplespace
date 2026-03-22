package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

type Queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type TupleStore struct{}

func New() *TupleStore {
	return &TupleStore{}
}

func (s *TupleStore) InsertTuple(ctx context.Context, tx pgx.Tx, space string, tuple types.Tuple) (int64, error) {
	payload, err := json.Marshal(tuple)
	if err != nil {
		return 0, fmt.Errorf("marshal tuple: %w", err)
	}

	var tupleID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO tuples(space, arity, fields_json) VALUES ($1, $2, $3) RETURNING id`,
		space,
		len(tuple.Fields),
		payload,
	).Scan(&tupleID)
	if err != nil {
		return 0, fmt.Errorf("insert tuple: %w", err)
	}

	for pos, field := range tuple.Fields {
		var textVal *string
		var intVal *int64
		var boolVal *bool

		switch field.Type {
		case types.TypeString:
			value := field.Value.(string)
			textVal = &value
		case types.TypeInt:
			value := field.Value.(int64)
			intVal = &value
		case types.TypeBool:
			value := field.Value.(bool)
			boolVal = &value
		default:
			return 0, fmt.Errorf("unsupported field type %q", field.Type)
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO tuple_fields(tuple_id, pos, type, text_val, int_val, bool_val) VALUES ($1, $2, $3, $4, $5, $6)`,
			tupleID,
			pos,
			string(field.Type),
			textVal,
			intVal,
			boolVal,
		); err != nil {
			return 0, fmt.Errorf("insert tuple field %d: %w", pos, err)
		}
	}

	return tupleID, nil
}

func (s *TupleStore) FindCandidates(ctx context.Context, q Queryer, space string, template types.Template, limit int) ([]StoredTuple, error) {
	query, args, err := BuildCandidateQuery(space, template, limit, false)
	if err != nil {
		return nil, err
	}
	return s.queryCandidates(ctx, q, query, args...)
}

func (s *TupleStore) LockCandidatesForConsume(ctx context.Context, tx pgx.Tx, space string, template types.Template, limit int) ([]StoredTuple, error) {
	query, args, err := BuildCandidateQuery(space, template, limit, true)
	if err != nil {
		return nil, err
	}
	return s.queryCandidates(ctx, tx, query, args...)
}

func (s *TupleStore) DeleteTuple(ctx context.Context, tx pgx.Tx, tupleID int64) error {
	if _, err := tx.Exec(ctx, `DELETE FROM tuples WHERE id = $1`, tupleID); err != nil {
		return fmt.Errorf("delete tuple %d: %w", tupleID, err)
	}
	return nil
}

func (s *TupleStore) queryCandidates(ctx context.Context, q Queryer, query string, args ...any) ([]StoredTuple, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query candidates: %w", err)
	}
	defer rows.Close()

	stored := make([]StoredTuple, 0)
	for rows.Next() {
		var (
			id      int64
			space   string
			payload []byte
		)
		if err := rows.Scan(&id, &space, &payload); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		tuple, err := decodeTuple(payload)
		if err != nil {
			return nil, err
		}
		stored = append(stored, StoredTuple{
			ID:    id,
			Space: space,
			Tuple: tuple,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate candidates: %w", err)
	}

	return stored, nil
}

func decodeTuple(payload []byte) (types.Tuple, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	var tuple types.Tuple
	if err := decoder.Decode(&tuple); err != nil {
		return types.Tuple{}, fmt.Errorf("decode tuple payload: %w", err)
	}
	normalized, err := types.NormalizeTuple(tuple)
	if err != nil {
		return types.Tuple{}, fmt.Errorf("normalize stored tuple: %w", err)
	}
	return normalized, nil
}

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	return pgxpool.NewWithConfig(ctx, config)
}
