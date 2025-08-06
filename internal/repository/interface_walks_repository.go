package repository

import (
	"context"

	"Team8-App/model"
)

type WalksRepository interface {
	Create(ctx context.Context, walk *model.Walk) error
	GetByID(ctx context.Context, id string) (*model.Walk, error)
	GetWalksByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.WalkSummary, error)
	GetWalkDetail(ctx context.Context, id string) (*model.WalkDetail, error)
	Delete(ctx context.Context, id string) error
	GetAll(ctx context.Context) ([]model.Walk, error)
}
