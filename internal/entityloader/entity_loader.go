package entityloader

import (
	"context"
	"fmt"
	"time"

	"github.com/rpattn/engql/internal/domain"
	"github.com/rpattn/engql/internal/repository"

	"github.com/google/uuid"
	"github.com/graph-gophers/dataloader"
)

type EntityLoader struct {
	Loader *dataloader.Loader
}

func NewEntityLoader(repo repository.EntityRepository) *EntityLoader {
	batchFn := func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		// Convert keys to []uuid.UUID
		ids := make([]uuid.UUID, len(keys))
		for i, k := range keys {
			id, err := uuid.Parse(k.String())
			if err != nil {
				return []*dataloader.Result{{Error: fmt.Errorf("invalid UUID: %w", err)}}
			}
			ids[i] = id
		}

		// Fetch entities in batch
		entities, err := repo.GetByIDs(ctx, ids)
		if err != nil {
			results := make([]*dataloader.Result, len(keys))
			for i := range results {
				results[i] = &dataloader.Result{Error: err}
			}
			return results
		}

		// Map UUID -> entity for ordering
		entityMap := make(map[uuid.UUID]domain.Entity)
		for _, e := range entities {
			entityMap[e.ID] = e
		}

		// Build results in the same order as keys
		results := make([]*dataloader.Result, len(keys))
		for i, id := range ids {
			if e, ok := entityMap[id]; ok {
				results[i] = &dataloader.Result{Data: e}
			} else {
				results[i] = &dataloader.Result{Data: nil}
			}
		}

		return results
	}

	loader := dataloader.NewBatchedLoader(batchFn, dataloader.WithWait(5*time.Millisecond))

	return &EntityLoader{Loader: loader}
}
