package strategy

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/helper"
)

// NatureStrategy は自然や観光地を巡るルートを提案する
type NatureStrategy struct{}

func NewNatureStrategy() StrategyInterface {
	return &NatureStrategy{}
}

// GetAvailableScenarios はNatureテーマで利用可能なシナリオ一覧を取得する
func (s *NatureStrategy) GetAvailableScenarios() []string {
	return model.GetNatureScenarios()
}

// GetTargetCategories は指定されたシナリオで検索対象となるPOIカテゴリを取得する
func (s *NatureStrategy) GetTargetCategories(scenario string) []string {
	switch scenario {
	case model.ScenarioParkTour:
		return []string{"park", "tourist_attraction", "establishment"}
	case model.ScenarioRiverside:
		return []string{"park", "natural_feature", "point_of_interest"}
	case model.ScenarioTempleNature:
		return []string{"place_of_worship", "park", "tourist_attraction"}
	default:
		return []string{"park", "tourist_attraction"}
	}
}

// FindCombinations は指定されたシナリオで候補POIから組み合わせを見つける
func (s *NatureStrategy) FindCombinations(scenario string, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)
	
	if len(filteredSpots) < 3 {
		return nil
	}

	// TODO: この中でシナリオごとに異なる組み合わせロジックを実装
	switch scenario {
	case model.ScenarioParkTour:
		return s.findParkTourCombinations(filteredSpots)
	case model.ScenarioRiverside:
		return s.findRiversideCombinations(filteredSpots)
	case model.ScenarioTempleNature:
		return s.findTempleNatureCombinations(filteredSpots)
	default:
		return s.findDefaultCombinations(filteredSpots)
	}
}

// findParkTourCombinations は公園中心巡りの組み合わせを見つける
func (s *NatureStrategy) findParkTourCombinations(spots []*model.POI) [][]*model.POI {
	// 最も評価の高い公園を起点とする
	mainPark := helper.FindHighestRated(spots)
	others := helper.RemovePOI(spots, mainPark)
	helper.SortByDistance(mainPark, others)

	// 公園 → 近くの観光地 → 別の公園 の順序
	if len(others) >= 2 {
		combination := []*model.POI{mainPark, others[0], others[1]}
		return [][]*model.POI{combination}
	}
	return nil
}

// findRiversideCombinations は河川敷散歩の組み合わせを見つける
func (s *NatureStrategy) findRiversideCombinations(spots []*model.POI) [][]*model.POI {
	// 河川敷エリアは一直線に配置されることが多いため、距離ベースでソート
	if len(spots) < 3 {
		return nil
	}
	
	startPoint := spots[0]
	others := spots[1:]
	helper.SortByDistance(startPoint, others)
	
	combination := []*model.POI{startPoint, others[0], others[1]}
	return [][]*model.POI{combination}
}

// findTempleNatureCombinations は寺社仏閣と自然の組み合わせを見つける
func (s *NatureStrategy) findTempleNatureCombinations(spots []*model.POI) [][]*model.POI {
	// 寺社 → 自然スポット → 寺社 の順序を優先
	temples := helper.FilterByCategory(spots, []string{"place_of_worship"})
	nature := helper.FilterByCategory(spots, []string{"park", "tourist_attraction"})
	
	if len(temples) >= 2 && len(nature) >= 1 {
		combination := []*model.POI{temples[0], nature[0], temples[1]}
		return [][]*model.POI{combination}
	}
	
	return s.findDefaultCombinations(spots)
}

// FindCombinationsWithDestination は目的地を含むルート組み合わせを見つける
func (s *NatureStrategy) FindCombinationsWithDestination(scenario string, destinationPOI *model.POI, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	// TODO:ここフィルターされてるやつ受け取ってるからいらんのちゃうんかな？
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)
	
	if len(filteredSpots) < 2 {
		return nil
	}

	// TODO:現状では距離順にソートしてるだけなのでそれを変更
	switch scenario {
	case model.ScenarioParkTour:
		return s.findParkTourCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioRiverside:
		return s.findRiversideCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioTempleNature:
		return s.findTempleNatureCombinationsWithDestination(destinationPOI, filteredSpots)
	default:
		return s.findDefaultCombinationsWithDestination(destinationPOI, filteredSpots)
	}
}

// findParkTourCombinationsWithDestination は公園中心巡りで目的地を含む組み合わせを見つける
func (s *NatureStrategy) findParkTourCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 目的地に近い順にソートして、途中に立ち寄る2つのスポットを選択
	helper.SortByDistance(destination, spots)
	
	if len(spots) >= 2 {
		// POI1 → POI2 → 目的地 の順序
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}

// findRiversideCombinationsWithDestination は河川敷散歩で目的地を含む組み合わせを見つける
func (s *NatureStrategy) findRiversideCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 河川敷は一直線なので、目的地への道筋を考慮
	helper.SortByDistance(destination, spots)
	
	if len(spots) >= 2 {
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}

// findTempleNatureCombinationsWithDestination は寺社仏閣と自然で目的地を含む組み合わせを見つける
func (s *NatureStrategy) findTempleNatureCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 目的地のタイプに応じて適切な順序を決定
	temples := helper.FilterByCategory(spots, []string{"place_of_worship"})
	nature := helper.FilterByCategory(spots, []string{"park", "tourist_attraction"})
	
	// 目的地が寺社の場合: 自然スポット → 自然スポット → 寺社（目的地）
	if helper.HasCategory(destination, []string{"place_of_worship"}) && len(nature) >= 2 {
		combination := []*model.POI{nature[0], nature[1], destination}
		return [][]*model.POI{combination}
	}
	
	// 目的地が自然スポットの場合: 寺社 → 寺社 → 自然スポット（目的地）
	if helper.HasCategory(destination, []string{"park", "tourist_attraction"}) && len(temples) >= 2 {
		combination := []*model.POI{temples[0], temples[1], destination}
		return [][]*model.POI{combination}
	}
	
	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findDefaultCombinationsWithDestination はデフォルトの目的地を含む組み合わせを見つける
func (s *NatureStrategy) findDefaultCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 評価の高い順にソートして2つ選択
	helper.SortByRating(spots)
	
	if len(spots) >= 2 {
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}

// findDefaultCombinations はデフォルトの組み合わせを見つける
func (s *NatureStrategy) findDefaultCombinations(spots []*model.POI) [][]*model.POI {
	spotA := helper.FindHighestRated(spots)
	others := helper.RemovePOI(spots, spotA)
	helper.SortByDistance(spotA, others)

	if len(others) >= 2 {
		combination := []*model.POI{spotA, others[0], others[1]}
		return [][]*model.POI{combination}
	}
	return nil
}
