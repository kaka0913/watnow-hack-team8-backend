package repository

import (
	"context"

	"Team8-App/internal/domain/model"
)

type POIsRepository interface {
	GetByID(ctx context.Context, id string) (*model.POI, error)
	GetByGridCellID(ctx context.Context, gridCellID int) ([]model.POI, error)
	GetNearbyPOIs(ctx context.Context, lat, lng float64, radiusMeters int) ([]model.POI, error)
	GetByCategories(ctx context.Context, categories []string, lat, lng float64, radiusMeters int) ([]model.POI, error)
	GetByCategory(ctx context.Context, category string, lat, lng float64, radiusMeters int) ([]model.POI, error)
	GetByRatingRange(ctx context.Context, minRating float64, lat, lng float64, radiusMeters int) ([]model.POI, error)
	FindNearbyByCategories(ctx context.Context, location model.LatLng, categories []string, radiusMeters int, limit int) ([]*model.POI, error)
	// ホラースポットを含めてPOIをカテゴリと位置に基づいて検索
	FindNearbyByCategoriesIncludingHorror(ctx context.Context, location model.LatLng, categories []string, radiusMeters int, limit int) ([]*model.POI, error)
}
