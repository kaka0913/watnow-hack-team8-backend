package strategy

import (
	"Team8-App/internal/domain/helper"
	"Team8-App/internal/domain/model"
)

// GourmetStrategy はカフェやベーカリーを巡るルートを提案する
type GourmetStrategy struct{}

func NewGourmetStrategy() StrategyInterface {
	return &GourmetStrategy{}
}

// GetAvailableScenarios はGourmetテーマで利用可能なシナリオ一覧を取得する
func (s *GourmetStrategy) GetAvailableScenarios() []string {
	return model.GetGourmetScenarios()
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

// FindCombinations は指定されたシナリオで候補POIから組み合わせを見つける
func (s *GourmetStrategy) FindCombinations(scenario string, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)

	if len(filteredSpots) < 3 {
		return nil
	}

	switch scenario {
	case model.ScenarioCafeHopping:
		return s.findCafeHoppingCombinations(filteredSpots)
	case model.ScenarioBakeryTour:
		return s.findBakeryTourCombinations(filteredSpots)
	case model.ScenarioLocalGourmet:
		return s.findLocalGourmetCombinations(filteredSpots)
	case model.ScenarioSweetJourney:
		return s.findSweetJourneyCombinations(filteredSpots)
	default:
		return s.findDefaultCombinations(filteredSpots)
	}
}

// findCafeHoppingCombinations はカフェ巡りの組み合わせを見つける
func (s *GourmetStrategy) findCafeHoppingCombinations(spots []*model.POI) [][]*model.POI {
	// 評価の高いカフェを起点とし、近隣のカフェを巡る
	mainCafe := helper.FindHighestRated(spots)
	others := helper.RemovePOI(spots, mainCafe)
	helper.SortByDistance(mainCafe, others)

	combination := []*model.POI{mainCafe, others[0], others[1]}
	return [][]*model.POI{combination}
}

// findBakeryTourCombinations はベーカリー巡りの組み合わせを見つける
func (s *GourmetStrategy) findBakeryTourCombinations(spots []*model.POI) [][]*model.POI {
	// ベーカリーを優先的に選択
	bakeries := helper.FilterByCategory(spots, []string{"bakery"})
	if len(bakeries) >= 2 {
		mainBakery := helper.FindHighestRated(bakeries)
		others := helper.RemovePOI(spots, mainBakery)
		helper.SortByDistance(mainBakery, others)

		combination := []*model.POI{mainBakery, others[0], others[1]}
		return [][]*model.POI{combination}
	}

	return s.findDefaultCombinations(spots)
}

// findLocalGourmetCombinations は地元グルメの組み合わせを見つける
func (s *GourmetStrategy) findLocalGourmetCombinations(spots []*model.POI) [][]*model.POI {
	// レストランや食事処を中心とした組み合わせ
	restaurants := helper.FilterByCategory(spots, []string{"restaurant", "food"})
	if len(restaurants) >= 1 {
		mainRestaurant := helper.FindHighestRated(restaurants)
		others := helper.RemovePOI(spots, mainRestaurant)
		helper.SortByDistance(mainRestaurant, others)

		if len(others) >= 2 {
			combination := []*model.POI{mainRestaurant, others[0], others[1]}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinations(spots)
}

// findSweetJourneyCombinations はスイーツ巡りの組み合わせを見つける
func (s *GourmetStrategy) findSweetJourneyCombinations(spots []*model.POI) [][]*model.POI {
	// ベーカリーとカフェを組み合わせたスイーツ巡り
	sweets := helper.FilterByCategory(spots, []string{"bakery", "cafe"})
	if len(sweets) >= 3 {
		mainSweet := helper.FindHighestRated(sweets)
		others := helper.RemovePOI(sweets, mainSweet)
		helper.SortByDistance(mainSweet, others)

		combination := []*model.POI{mainSweet, others[0], others[1]}
		return [][]*model.POI{combination}
	}

	return s.findDefaultCombinations(spots)
}

// findDefaultCombinations はデフォルトの組み合わせを見つける
func (s *GourmetStrategy) findDefaultCombinations(spots []*model.POI) [][]*model.POI {
	spotA := helper.FindHighestRated(spots)
	others := helper.RemovePOI(spots, spotA)
	helper.SortByDistance(spotA, others)

	if len(others) >= 2 {
		combination := []*model.POI{spotA, others[0], others[1]}
		return [][]*model.POI{combination}
	}
	return nil
}

// FindCombinationsWithDestination は目的地を含むルート組み合わせを見つける
func (s *GourmetStrategy) FindCombinationsWithDestination(scenario string, destinationPOI *model.POI, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)

	if len(filteredSpots) < 2 {
		return nil
	}

	switch scenario {
	case model.ScenarioCafeHopping:
		return s.findCafeHoppingCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioBakeryTour:
		return s.findBakeryTourCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioLocalGourmet:
		return s.findLocalGourmetCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioSweetJourney:
		return s.findSweetJourneyCombinationsWithDestination(destinationPOI, filteredSpots)
	default:
		return s.findDefaultCombinationsWithDestination(destinationPOI, filteredSpots)
	}
}

// findCafeHoppingCombinationsWithDestination はカフェ巡りで目的地を含む組み合わせを見つける
func (s *GourmetStrategy) findCafeHoppingCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 評価の高いカフェから順に選択し、最後に目的地
	helper.SortByRating(spots)

	if len(spots) >= 2 {
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}

// findBakeryTourCombinationsWithDestination はベーカリー巡りで目的地を含む組み合わせを見つける
func (s *GourmetStrategy) findBakeryTourCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// ベーカリーを優先して選択
	bakeries := helper.FilterByCategory(spots, []string{"bakery"})

	if len(bakeries) >= 2 {
		helper.SortByRating(bakeries)
		combination := []*model.POI{bakeries[0], bakeries[1], destination}
		return [][]*model.POI{combination}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findLocalGourmetCombinationsWithDestination は地元グルメで目的地を含む組み合わせを見つける
func (s *GourmetStrategy) findLocalGourmetCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// レストランを優先的に選択
	restaurants := helper.FilterByCategory(spots, []string{"restaurant", "food"})

	if len(restaurants) >= 1 && len(spots) >= 2 {
		helper.SortByRating(restaurants)
		others := helper.RemovePOI(spots, restaurants[0])
		if len(others) >= 1 {
			combination := []*model.POI{restaurants[0], others[0], destination}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findSweetJourneyCombinationsWithDestination はスイーツ巡りで目的地を含む組み合わせを見つける
func (s *GourmetStrategy) findSweetJourneyCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// ベーカリーとカフェを優先
	sweets := helper.FilterByCategory(spots, []string{"bakery", "cafe"})

	if len(sweets) >= 2 {
		helper.SortByRating(sweets)
		combination := []*model.POI{sweets[0], sweets[1], destination}
		return [][]*model.POI{combination}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findDefaultCombinationsWithDestination はデフォルトの目的地を含む組み合わせを見つける
func (s *GourmetStrategy) findDefaultCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 評価の高い順にソートして2つ選択
	helper.SortByRating(spots)

	if len(spots) >= 2 {
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}
