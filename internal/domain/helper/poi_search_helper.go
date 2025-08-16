package helper

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"errors"
)

// POISearchHelper はPOI検索に関するヘルパー関数を提供する
type POISearchHelper struct {
	poiRepo repository.POIsRepository
}

// NewPOISearchHelper は新しいPOISearchHelperインスタンスを作成する
func NewPOISearchHelper(repo repository.POIsRepository) *POISearchHelper {
	return &POISearchHelper{
		poiRepo: repo,
	}
}

// FindNearestPOI は目的地に該当するPOIがないかを確認するために、指定座標に最も近いPOIを見つける
func (h *POISearchHelper) FindNearestPOI(ctx context.Context, location model.LatLng) (*model.POI, error) {
	// 目的地周辺のPOIを検索（半径500m、最大10件）
	nearbyPOIs, err := h.poiRepo.FindNearbyByCategories(ctx, location, []string{}, 500, 10)
	if err != nil {
		return nil, err
	}
	if len(nearbyPOIs) == 0 {
		return nil, errors.New("目的地周辺にPOIが見つかりません")
	}

	// 最も近いPOIを返す
	return nearbyPOIs[0], nil
}
