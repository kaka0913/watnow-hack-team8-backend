package strategy

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"fmt"
)

// HistoryAndCultureStrategy は歴史・文化を巡るルートを提案する
type HistoryAndCultureStrategy struct {
	poiRepo repository.POIsRepository
}

func NewHistoryAndCultureStrategy(repo repository.POIsRepository) StrategyInterface {
	return &HistoryAndCultureStrategy{
		poiRepo: repo,
	}
}

// GetAvailableScenarios はHistoryAndCultureテーマで利用可能なシナリオ一覧を取得する
func (s *HistoryAndCultureStrategy) GetAvailableScenarios() []string {
	return model.GetHistoryAndCultureScenarios()
}

// FindCombinations は指定されたシナリオでPOI組み合わせを見つける
func (s *HistoryAndCultureStrategy) FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error) {
	// NOTE: 一時的な実装 - 将来的にはシナリオごとの詳細ロジックを実装
	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, model.GetHistoryAndCultureCategories(), 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("POI検索に失敗: %w", err)
	}

	if len(candidates) < 3 {
		return nil, fmt.Errorf("十分なPOIが見つかりませんでした")
	}

	// シンプルな組み合わせを生成
	combination := []*model.POI{candidates[0], candidates[1], candidates[2]}
	return [][]*model.POI{combination}, nil
}

// FindCombinationsWithDestination は目的地を含むルート組み合わせを見つける
func (s *HistoryAndCultureStrategy) FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// NOTE: 一時的な実装 - 将来的にはシナリオごとの詳細ロジックを実装
	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, model.GetHistoryAndCultureCategories(), 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("POI検索に失敗: %w", err)
	}

	if len(candidates) < 2 {
		return nil, fmt.Errorf("十分なPOIが見つかりませんでした")
	}

	// 目的地周辺のPOIを取得
	destinationPOIs, err := s.poiRepo.FindNearbyByCategories(ctx, destination, []string{"tourist_attraction", "store"}, 500, 1)
	if err != nil || len(destinationPOIs) == 0 {
		// 目的地POIが見つからない場合は座標から生成
		destinationPOI := &model.POI{
			ID:   "destination",
			Name: "目的地",
			Location: &model.Geometry{
				Type:        "Point",
				Coordinates: []float64{destination.Lng, destination.Lat},
			},
		}
		combination := []*model.POI{candidates[0], candidates[1], destinationPOI}
		return [][]*model.POI{combination}, nil
	}

	combination := []*model.POI{candidates[0], candidates[1], destinationPOIs[0]}
	return [][]*model.POI{combination}, nil
}

// ExploreNewSpots は歴史・文化テーマで新しいスポットを探索します
func (s *HistoryAndCultureStrategy) ExploreNewSpots(ctx context.Context, location model.LatLng) ([]*model.POI, error) {
	// 歴史・文化テーマのカテゴリで周辺のPOIを検索
	pois, err := s.poiRepo.FindNearbyByCategories(ctx, location, model.GetHistoryAndCultureCategories(), 1500, 8)
	if err != nil {
		return nil, fmt.Errorf("歴史・文化テーマのPOI検索に失敗: %w", err)
	}

	// 最大3つまでの高評価POIを選択
	var result []*model.POI
	for i, poi := range pois {
		if i >= 3 {
			break
		}
		if poi.Rate >= 3.0 {
			result = append(result, poi)
		}
	}

	return result, nil
}
