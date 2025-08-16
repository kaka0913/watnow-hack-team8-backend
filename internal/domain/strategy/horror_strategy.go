package strategy

import (
	"Team8-App/internal/domain/helper"
	"Team8-App/internal/domain/model"
)

// HorrorStrategy はホラースポットを巡るルートを提案する
type HorrorStrategy struct{}

func NewHorrorStrategy() StrategyInterface {
	return &HorrorStrategy{}
}

// GetAvailableScenarios はホラーテーマで利用可能なシナリオ一覧を取得する
func (s *HorrorStrategy) GetAvailableScenarios() []string {
	return model.GetHorrorScenarios()
}

// GetTargetCategories は指定されたシナリオで検索対象となるPOIカテゴリを取得する
func (s *HorrorStrategy) GetTargetCategories(scenario string) []string {
	switch scenario {
	case model.ScenarioGhostTour:
		return []string{"horror_spot", "establishment", "natural_feature", "tourist_attraction"}
	case model.ScenarioHauntedRuins:
		return []string{"horror_spot", "establishment", "tourist_attraction"}
	case model.ScenarioCursedNature:
		return []string{"horror_spot", "natural_feature", "park", "place_of_worship"}
	case model.ScenarioCemeteryWalk:
		return []string{"horror_spot", "place_of_worship", "tourist_attraction"}
	default:
		return []string{"horror_spot", "establishment", "tourist_attraction"}
	}
}

// FindCombinations は指定されたシナリオで候補POIから組み合わせを見つける
func (s *HorrorStrategy) FindCombinations(scenario string, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)

	if len(filteredSpots) < 3 {
		return nil
	}

	switch scenario {
	case model.ScenarioGhostTour:
		return s.findGhostTourCombinations(filteredSpots)
	case model.ScenarioHauntedRuins:
		return s.findHauntedRuinsCombinations(filteredSpots)
	case model.ScenarioCursedNature:
		return s.findCursedNatureCombinations(filteredSpots)
	case model.ScenarioCemeteryWalk:
		return s.findCemeteryWalkCombinations(filteredSpots)
	default:
		return s.findDefaultCombinations(filteredSpots)
	}
}

// findGhostTourCombinations は心霊スポット巡りの組み合わせを見つける
func (s *HorrorStrategy) findGhostTourCombinations(spots []*model.POI) [][]*model.POI {
	// horror_spotを優先的に選択
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})
	if len(horrorSpots) >= 2 {
		mainHorror := helper.FindHighestRated(horrorSpots)
		others := helper.RemovePOI(spots, mainHorror)
		helper.SortByDistance(mainHorror, others)

		if len(others) >= 2 {
			combination := []*model.POI{mainHorror, others[0], others[1]}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinations(spots)
}

// findHauntedRuinsCombinations は廃墟探索の組み合わせを見つける
func (s *HorrorStrategy) findHauntedRuinsCombinations(spots []*model.POI) [][]*model.POI {
	// 廃墟系のホラースポットと建物を中心とした組み合わせ
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})

	if len(horrorSpots) >= 1 {
		mainHorror := helper.FindHighestRated(horrorSpots)
		others := helper.RemovePOI(spots, mainHorror)
		helper.SortByDistance(mainHorror, others)

		if len(others) >= 2 {
			combination := []*model.POI{mainHorror, others[0], others[1]}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinations(spots)
}

// findCursedNatureCombinations は呪いの自然の組み合わせを見つける
func (s *HorrorStrategy) findCursedNatureCombinations(spots []*model.POI) [][]*model.POI {
	// 自然系のホラースポットを中心とした組み合わせ
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})

	if len(horrorSpots) >= 1 {
		mainHorror := helper.FindHighestRated(horrorSpots)
		others := helper.RemovePOI(spots, mainHorror)
		helper.SortByDistance(mainHorror, others)

		if len(others) >= 2 {
			combination := []*model.POI{mainHorror, others[0], others[1]}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinations(spots)
}

// findCemeteryWalkCombinations は墓地・慰霊散歩の組み合わせを見つける
func (s *HorrorStrategy) findCemeteryWalkCombinations(spots []*model.POI) [][]*model.POI {
	// 宗教施設・慰霊関連のホラースポットを中心とした組み合わせ
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})

	if len(horrorSpots) >= 1 {
		mainHorror := helper.FindHighestRated(horrorSpots)
		others := helper.RemovePOI(spots, mainHorror)
		helper.SortByDistance(mainHorror, others)

		if len(others) >= 2 {
			combination := []*model.POI{mainHorror, others[0], others[1]}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinations(spots)
}

// findDefaultCombinations はデフォルトの組み合わせを見つける
func (s *HorrorStrategy) findDefaultCombinations(spots []*model.POI) [][]*model.POI {
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
func (s *HorrorStrategy) FindCombinationsWithDestination(scenario string, destinationPOI *model.POI, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)

	if len(filteredSpots) < 2 {
		return nil
	}

	switch scenario {
	case model.ScenarioGhostTour:
		return s.findGhostTourCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioHauntedRuins:
		return s.findHauntedRuinsCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioCursedNature:
		return s.findCursedNatureCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioCemeteryWalk:
		return s.findCemeteryWalkCombinationsWithDestination(destinationPOI, filteredSpots)
	default:
		return s.findDefaultCombinationsWithDestination(destinationPOI, filteredSpots)
	}
}

// findGhostTourCombinationsWithDestination は心霊スポット巡りで目的地を含む組み合わせを見つける
func (s *HorrorStrategy) findGhostTourCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// horror_spotを優先的に選択
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})

	if len(horrorSpots) >= 2 {
		helper.SortByRating(horrorSpots)
		combination := []*model.POI{horrorSpots[0], horrorSpots[1], destination}
		return [][]*model.POI{combination}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findHauntedRuinsCombinationsWithDestination は廃墟探索で目的地を含む組み合わせを見つける
func (s *HorrorStrategy) findHauntedRuinsCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// ホラースポットを優先
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})

	if len(horrorSpots) >= 1 && len(spots) >= 2 {
		helper.SortByRating(horrorSpots)
		others := helper.RemovePOI(spots, horrorSpots[0])
		if len(others) >= 1 {
			combination := []*model.POI{horrorSpots[0], others[0], destination}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findCursedNatureCombinationsWithDestination は呪いの自然で目的地を含む組み合わせを見つける
func (s *HorrorStrategy) findCursedNatureCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 自然系ホラースポットを優先
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})

	if len(horrorSpots) >= 1 && len(spots) >= 2 {
		helper.SortByRating(horrorSpots)
		others := helper.RemovePOI(spots, horrorSpots[0])
		if len(others) >= 1 {
			combination := []*model.POI{horrorSpots[0], others[0], destination}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findCemeteryWalkCombinationsWithDestination は墓地・慰霊散歩で目的地を含む組み合わせを見つける
func (s *HorrorStrategy) findCemeteryWalkCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 宗教施設系ホラースポットを優先
	horrorSpots := helper.FilterByCategory(spots, []string{"horror_spot"})

	if len(horrorSpots) >= 1 && len(spots) >= 2 {
		helper.SortByRating(horrorSpots)
		others := helper.RemovePOI(spots, horrorSpots[0])
		if len(others) >= 1 {
			combination := []*model.POI{horrorSpots[0], others[0], destination}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findDefaultCombinationsWithDestination はデフォルトの目的地を含む組み合わせを見つける
func (s *HorrorStrategy) findDefaultCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 評価の高い順にソートして2つ選択
	helper.SortByRating(spots)

	if len(spots) >= 2 {
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}
