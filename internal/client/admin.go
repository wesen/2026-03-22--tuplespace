package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/manuel/wesen/tuplespace/internal/admin"
)

type tupleListEnvelope struct {
	OK     bool                `json:"ok"`
	Tuples []admin.TupleRecord `json:"tuples"`
}

type spacesEnvelope struct {
	OK     bool                 `json:"ok"`
	Spaces []admin.SpaceSummary `json:"spaces"`
}

type statsEnvelope struct {
	OK    bool                `json:"ok"`
	Stats admin.StatsSnapshot `json:"stats"`
}

type configEnvelope struct {
	OK     bool                 `json:"ok"`
	Config admin.ConfigSnapshot `json:"config"`
}

type schemaEnvelope struct {
	OK     bool                 `json:"ok"`
	Schema admin.SchemaSnapshot `json:"schema"`
}

type waitersEnvelope struct {
	OK      bool               `json:"ok"`
	Waiters []admin.WaiterInfo `json:"waiters"`
}

type tupleEnvelope struct {
	OK    bool              `json:"ok"`
	Tuple admin.TupleRecord `json:"tuple"`
}

type deleteEnvelope struct {
	OK     bool               `json:"ok"`
	Result admin.DeleteResult `json:"result"`
}

type purgeEnvelope struct {
	OK     bool              `json:"ok"`
	Result admin.PurgeResult `json:"result"`
}

type notifyTestEnvelope struct {
	OK     bool                   `json:"ok"`
	Result admin.NotifyTestResult `json:"result"`
}

func (c *Client) Spaces(ctx context.Context) ([]admin.SpaceSummary, error) {
	var response spacesEnvelope
	if err := c.get(ctx, "/v1/admin/spaces", &response); err != nil {
		return nil, err
	}
	return response.Spaces, nil
}

func (c *Client) Dump(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	return c.tupleListPost(ctx, "/v1/admin/dump", filter)
}

func (c *Client) Peek(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	return c.tupleListPost(ctx, "/v1/admin/peek", filter)
}

func (c *Client) Export(ctx context.Context, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	return c.tupleListPost(ctx, "/v1/admin/export", filter)
}

func (c *Client) Stats(ctx context.Context) (admin.StatsSnapshot, error) {
	var response statsEnvelope
	if err := c.get(ctx, "/v1/admin/stats", &response); err != nil {
		return admin.StatsSnapshot{}, err
	}
	return response.Stats, nil
}

func (c *Client) Config(ctx context.Context) (admin.ConfigSnapshot, error) {
	var response configEnvelope
	if err := c.get(ctx, "/v1/admin/config", &response); err != nil {
		return admin.ConfigSnapshot{}, err
	}
	return response.Config, nil
}

func (c *Client) Schema(ctx context.Context) (admin.SchemaSnapshot, error) {
	var response schemaEnvelope
	if err := c.get(ctx, "/v1/admin/schema", &response); err != nil {
		return admin.SchemaSnapshot{}, err
	}
	return response.Schema, nil
}

func (c *Client) Waiters(ctx context.Context) ([]admin.WaiterInfo, error) {
	var response waitersEnvelope
	if err := c.get(ctx, "/v1/admin/waiters", &response); err != nil {
		return nil, err
	}
	return response.Waiters, nil
}

func (c *Client) GetTuple(ctx context.Context, tupleID int64) (admin.TupleRecord, error) {
	var response tupleEnvelope
	if err := c.get(ctx, "/v1/admin/tuples/"+strconv.FormatInt(tupleID, 10), &response); err != nil {
		return admin.TupleRecord{}, err
	}
	return response.Tuple, nil
}

func (c *Client) DeleteTuple(ctx context.Context, tupleID int64) (admin.DeleteResult, error) {
	var response deleteEnvelope
	if err := c.delete(ctx, "/v1/admin/tuples/"+strconv.FormatInt(tupleID, 10), &response); err != nil {
		return admin.DeleteResult{}, err
	}
	return response.Result, nil
}

func (c *Client) Purge(ctx context.Context, filter admin.TupleFilter, confirm bool) (admin.PurgeResult, error) {
	var response purgeEnvelope
	if err := c.post(ctx, "/v1/admin/purge", map[string]any{"filter": filter, "confirm": confirm}, &response); err != nil {
		return admin.PurgeResult{}, err
	}
	return response.Result, nil
}

func (c *Client) NotifyTest(ctx context.Context, space string) (admin.NotifyTestResult, error) {
	var response notifyTestEnvelope
	if err := c.post(ctx, "/v1/admin/notify-test", map[string]any{"space": space}, &response); err != nil {
		return admin.NotifyTestResult{}, err
	}
	return response.Result, nil
}

func (c *Client) tupleListPost(ctx context.Context, path string, filter admin.TupleFilter) ([]admin.TupleRecord, error) {
	var response tupleListEnvelope
	if err := c.post(ctx, path, map[string]any{"filter": filter}, &response); err != nil {
		return nil, err
	}
	return response.Tuples, nil
}

func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer res.Body.Close()

	return decodeResponse(res, dst)
}

func (c *Client) delete(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer res.Body.Close()

	return decodeResponse(res, dst)
}
