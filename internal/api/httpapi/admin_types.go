package httpapi

import "github.com/manuel/wesen/tuplespace/internal/admin"

type FilterRequest struct {
	Filter admin.TupleFilter `json:"filter"`
}

type PurgeRequest struct {
	Filter  admin.TupleFilter `json:"filter"`
	Confirm bool              `json:"confirm"`
}

type NotifyTestRequest struct {
	Space string `json:"space"`
}

type SpacesResponse struct {
	OK     bool                 `json:"ok"`
	Spaces []admin.SpaceSummary `json:"spaces"`
}

type TupleListResponse struct {
	OK     bool                `json:"ok"`
	Tuples []admin.TupleRecord `json:"tuples"`
}

type StatsResponse struct {
	OK    bool                `json:"ok"`
	Stats admin.StatsSnapshot `json:"stats"`
}

type ConfigResponse struct {
	OK     bool                 `json:"ok"`
	Config admin.ConfigSnapshot `json:"config"`
}

type SchemaResponse struct {
	OK     bool                 `json:"ok"`
	Schema admin.SchemaSnapshot `json:"schema"`
}

type WaitersResponse struct {
	OK      bool               `json:"ok"`
	Waiters []admin.WaiterInfo `json:"waiters"`
}

type TupleGetResponse struct {
	OK    bool              `json:"ok"`
	Tuple admin.TupleRecord `json:"tuple"`
}

type TupleDeleteResponse struct {
	OK     bool               `json:"ok"`
	Result admin.DeleteResult `json:"result"`
}

type PurgeResponse struct {
	OK     bool              `json:"ok"`
	Result admin.PurgeResult `json:"result"`
}

type NotifyTestResponse struct {
	OK     bool                   `json:"ok"`
	Result admin.NotifyTestResult `json:"result"`
}
