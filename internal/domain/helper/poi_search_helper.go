package helper

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"errors"
	"time"
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

// ValidateCombination は組み合わせが有効かどうかをチェックする
// 1. 同一POIの重複チェック（2個以上の重複は無効）
// 2. 所要時間制限チェック（健康テーマのロングコース以外は1時間30分以内）
func (h *POISearchHelper) ValidateCombination(combination []*model.POI, estimatedDuration time.Duration, isHealthLongDistance bool) bool {
	// 1. 同一POI重複チェック
	if hasDuplicatePOIs(combination) {
		return false
	}

	// 2. 所要時間制限チェック
	maxDuration := 90 * time.Minute // 1時間30分
	if isHealthLongDistance {
		// 健康テーマのロングコースの場合は制限なし
		return true
	}

	return estimatedDuration <= maxDuration
}

// hasDuplicatePOIs は組み合わせに2個以上の同一POIが含まれているかチェック
func hasDuplicatePOIs(combination []*model.POI) bool {
	poiCount := make(map[string]int)
	
	for _, poi := range combination {
		if poi != nil {
			poiCount[poi.ID]++
			if poiCount[poi.ID] >= 2 {
				return true
			}
		}
	}
	
	return false
}

// FilterValidCombinations は有効な組み合わせのみを返す
func (h *POISearchHelper) FilterValidCombinations(combinations [][]*model.POI, estimatedDurations []time.Duration, isHealthLongDistance bool) [][]*model.POI {
	var validCombinations [][]*model.POI
	
	for i, combination := range combinations {
		var duration time.Duration
		if i < len(estimatedDurations) {
			duration = estimatedDurations[i]
		}
		
		if h.ValidateCombination(combination, duration, isHealthLongDistance) {
			validCombinations = append(validCombinations, combination)
		}
	}
	
	return validCombinations
}
