package admin

import (
	"time"

	"github.com/manuel/wesen/tuplespace/internal/types"
)

type TupleFilter struct {
	Space         string     `json:"space,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	Limit         int        `json:"limit,omitempty"`
	Offset        int        `json:"offset,omitempty"`
}

type TupleRecord struct {
	ID        int64       `json:"id"`
	Space     string      `json:"space"`
	Arity     int         `json:"arity"`
	CreatedAt time.Time   `json:"created_at"`
	Tuple     types.Tuple `json:"tuple"`
}

type SpaceSummary struct {
	Space         string     `json:"space"`
	TupleCount    int64      `json:"tuple_count"`
	OldestTupleAt *time.Time `json:"oldest_tuple_at,omitempty"`
	NewestTupleAt *time.Time `json:"newest_tuple_at,omitempty"`
}

type WaiterInfo struct {
	ID        uint64         `json:"id"`
	Space     string         `json:"space"`
	Operation string         `json:"operation"`
	WaitMS    int64          `json:"wait_ms"`
	StartedAt time.Time      `json:"started_at"`
	Template  types.Template `json:"template"`
}

type StatsSnapshot struct {
	StartedAt           time.Time      `json:"started_at"`
	UptimeMS            int64          `json:"uptime_ms"`
	SpaceCount          int            `json:"space_count"`
	TupleCount          int64          `json:"tuple_count"`
	WaiterCount         int            `json:"waiter_count"`
	NotifierChannels    int            `json:"notifier_channels"`
	NotifierSubscribers int            `json:"notifier_subscribers"`
	NotifierByChannel   map[string]int `json:"notifier_by_channel,omitempty"`
	CandidateLimit      int            `json:"candidate_limit"`
}

type ConfigSnapshot struct {
	HTTPListenAddr string `json:"http_listen_addr"`
	DatabaseURL    string `json:"database_url"`
	DatabaseHost   string `json:"database_host,omitempty"`
	DatabaseName   string `json:"database_name,omitempty"`
	CandidateLimit int    `json:"candidate_limit"`
	ShutdownGrace  string `json:"shutdown_grace"`
}

type SchemaSnapshot struct {
	MigrationFiles []string `json:"migration_files"`
	Tables         []string `json:"tables"`
	Indexes        []string `json:"indexes"`
	MissingTables  []string `json:"missing_tables,omitempty"`
	MissingIndexes []string `json:"missing_indexes,omitempty"`
}

type DeleteResult struct {
	TupleID int64 `json:"tuple_id"`
	Deleted bool  `json:"deleted"`
}

type PurgeResult struct {
	DeletedCount int64 `json:"deleted_count"`
}

type NotifyTestResult struct {
	Space                  string         `json:"space"`
	Channel                string         `json:"channel"`
	SubscriberCount        int            `json:"subscriber_count"`
	ChannelSubscriberCount int            `json:"channel_subscriber_count"`
	NotifierChannels       int            `json:"notifier_channels"`
	NotifierByChannel      map[string]int `json:"notifier_by_channel,omitempty"`
}
