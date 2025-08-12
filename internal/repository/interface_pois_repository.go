package repository

import (
	"context"

	"Team8-App/model"
)

type POIsRepository interface {
	GetByID(ctx context.Context, id string) (*model.POI, error)
	GetByGridCellID(ctx context.Context, gridCellID int) ([]model.POI, error)
	GetNearbyPOIs(ctx context.Context, lat, lng float64, radiusMeters int) ([]model.POI, error)
	GetByCategories(ctx context.Context, categories []string, lat, lng float64, radiusMeters int) ([]model.POI, error)
	GetByCategory(ctx context.Context, category string, lat, lng float64, radiusMeters int) ([]model.POI, error)
	GetByRatingRange(ctx context.Context, minRating float64, lat, lng float64, radiusMeters int) ([]model.POI, error)
	Create(ctx context.Context, poi *model.POI) error
	Update(ctx context.Context, poi *model.POI) error
	Delete(ctx context.Context, id string) error
	BulkCreate(ctx context.Context, pois []model.POI) error
}
