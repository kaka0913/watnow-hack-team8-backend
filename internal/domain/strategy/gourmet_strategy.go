package strategy

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"fmt"
)

// GourmetStrategy はカフェやベーカリーを巡るルートを提案する
type GourmetStrategy struct {
	poiRepo repository.POIsRepository
}

func NewGourmetStrategy(repo repository.POIsRepository) StrategyInterface {
	return &GourmetStrategy{
		poiRepo: repo,
	}
}

// GetAvailableScenarios はGourmetテーマで利用可能なシナリオ一覧を取得する
func (s *GourmetStrategy) GetAvailableScenarios() []string {
	return model.GetGourmetScenarios()
}

// FindCombinations は指定されたシナリオでPOI組み合わせを見つける
func (s *GourmetStrategy) FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error) {
	// NOTE: 一時的な実装 - 将来的にはシナリオごとの詳細ロジックを実装
	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, s.GetTargetCategories(scenario), 1500, 10)
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
func (s *GourmetStrategy) FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// NOTE: 一時的な実装 - 将来的にはシナリオごとの詳細ロジックを実装
	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, s.GetTargetCategories(scenario), 1500, 10)
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

// GetTargetCategories は指定されたシナリオで検索対象となるPOIカテゴリを取得する
func (s *GourmetStrategy) GetTargetCategories(scenario string) []string {
	switch scenario {
	case model.ScenarioCafeHopping:
		return []string{"cafe", "restaurant"}
	case model.ScenarioBakeryTour:
		return []string{"bakery", "cafe", "store"}
	case model.ScenarioLocalGourmet:
		return []string{"restaurant", "food", "meal_takeaway"}
	case model.ScenarioSweetJourney:
		return []string{"bakery", "cafe", "store"}
	default:
		return []string{"cafe", "bakery"}
	}
}
