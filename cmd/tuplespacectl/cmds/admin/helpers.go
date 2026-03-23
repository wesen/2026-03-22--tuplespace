package admin

import (
	"context"
	"time"

	"github.com/go-go-golems/glazed/pkg/middlewares"
	glazedtypes "github.com/go-go-golems/glazed/pkg/types"

	adminapi "github.com/manuel/wesen/tuplespace/internal/admin"
)

func buildTupleFilter(space string, limit int, offset int, createdBefore string, createdAfter string) (adminapi.TupleFilter, error) {
	filter := adminapi.TupleFilter{
		Space:  space,
		Limit:  limit,
		Offset: offset,
	}
	if createdBefore != "" {
		t, err := time.Parse(time.RFC3339, createdBefore)
		if err != nil {
			return adminapi.TupleFilter{}, err
		}
		filter.CreatedBefore = &t
	}
	if createdAfter != "" {
		t, err := time.Parse(time.RFC3339, createdAfter)
		if err != nil {
			return adminapi.TupleFilter{}, err
		}
		filter.CreatedAfter = &t
	}
	return filter, nil
}

func addTupleRows(ctx context.Context, gp middlewares.Processor, tuples []adminapi.TupleRecord) error {
	for _, tuple := range tuples {
		if err := gp.AddRow(ctx, glazedtypes.NewRow(
			glazedtypes.MRP("id", tuple.ID),
			glazedtypes.MRP("space", tuple.Space),
			glazedtypes.MRP("arity", tuple.Arity),
			glazedtypes.MRP("created_at", tuple.CreatedAt),
			glazedtypes.MRP("tuple", tuple.Tuple),
		)); err != nil {
			return err
		}
	}
	return nil
}
