package repository

import (
	"context"

	"Team8-App/internal/domain/model"
)

type GridCellsRepository interface {
	GetByID(ctx context.Context, id int) (*model.GridCell, error)
	GetContainingPoint(ctx context.Context, lat, lng float64) (*model.GridCell, error)
	GetByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.GridCell, error)
	Create(ctx context.Context, gridCell *model.GridCell) error
	Update(ctx context.Context, gridCell *model.GridCell) error
	Delete(ctx context.Context, id int) error
	GetAll(ctx context.Context) ([]model.GridCell, error)
}
