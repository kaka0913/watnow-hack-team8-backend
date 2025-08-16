package strategy

import (
	"Team8-App/internal/domain/helper"
	"Team8-App/internal/domain/model"
)

// HistoryAndCultureStrategy は歴史・文化探訪スポットを巡るルートを提案する
type HistoryAndCultureStrategy struct{}

func NewHistoryAndCultureStrategy() StrategyInterface {
	return &HistoryAndCultureStrategy{}
}

// GetAvailableScenarios は歴史・文化探訪テーマで利用可能なシナリオ一覧を取得する
func (s *HistoryAndCultureStrategy) GetAvailableScenarios() []string {
	return model.GetHistoryAndCultureScenarios()
}

// GetTargetCategories は指定されたシナリオで検索対象となるPOIカテゴリを取得する
func (s *HistoryAndCultureStrategy) GetTargetCategories(scenario string) []string {
	switch scenario {
	case model.ScenarioTempleShrine:
		return []string{"place_of_worship", "tourist_attraction"}
	case model.ScenarioMuseumTour:
		return []string{"museum", "art_gallery", "tourist_attraction"}
	case model.ScenarioOldTown:
		return []string{"tourist_attraction", "establishment", "book_store"}
	case model.ScenarioCulturalWalk:
		return []string{"tourist_attraction", "museum", "book_store", "art_gallery"}
	default:
		return []string{"tourist_attraction", "museum", "book_store"}
	}
}

// FindCombinations は指定されたシナリオで候補POIから組み合わせを見つける
func (s *HistoryAndCultureStrategy) FindCombinations(scenario string, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)

	if len(filteredSpots) < 3 {
		return nil
	}

	switch scenario {
	case model.ScenarioTempleShrine:
		return s.findTempleShrineComnations(filteredSpots)
	case model.ScenarioMuseumTour:
		return s.findMuseumTourCombinations(filteredSpots)
	case model.ScenarioOldTown:
		return s.findOldTownCombinations(filteredSpots)
	case model.ScenarioCulturalWalk:
		return s.findCulturalWalkCombinations(filteredSpots)
	default:
		return s.findDefaultCombinations(filteredSpots)
	}
}

// findTempleShrineComnations は寺社仏閣巡りの組み合わせを見つける
func (s *HistoryAndCultureStrategy) findTempleShrineComnations(spots []*model.POI) [][]*model.POI {
	// 神社仏閣を優先的に選択
	temples := helper.FilterByCategory(spots, []string{"place_of_worship"})
	if len(temples) >= 2 {
		mainTemple := helper.FindHighestRated(temples)
		others := helper.RemovePOI(spots, mainTemple)
		helper.SortByDistance(mainTemple, others)

		combination := []*model.POI{mainTemple, others[0], others[1]}
		return [][]*model.POI{combination}
	}

	return s.findDefaultCombinations(spots)
}

// findMuseumTourCombinations は博物館・美術館巡りの組み合わせを見つける
func (s *HistoryAndCultureStrategy) findMuseumTourCombinations(spots []*model.POI) [][]*model.POI {
	// 博物館・美術館を中心とした組み合わせ
	museums := helper.FilterByCategory(spots, []string{"museum", "art_gallery"})
	if len(museums) >= 1 {
		mainMuseum := helper.FindHighestRated(museums)
		others := helper.RemovePOI(spots, mainMuseum)
		helper.SortByDistance(mainMuseum, others)

		if len(others) >= 2 {
			combination := []*model.POI{mainMuseum, others[0], others[1]}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinations(spots)
}

// findOldTownCombinations は古い街並み散策の組み合わせを見つける
func (s *HistoryAndCultureStrategy) findOldTownCombinations(spots []*model.POI) [][]*model.POI {
	// 観光地を中心として、本屋などの文化的スポットを組み合わせ
	attractions := helper.FilterByCategory(spots, []string{"tourist_attraction"})
	if len(attractions) >= 1 {
		mainAttraction := helper.FindHighestRated(attractions)
		others := helper.RemovePOI(spots, mainAttraction)
		helper.SortByDistance(mainAttraction, others)

		if len(others) >= 2 {
			combination := []*model.POI{mainAttraction, others[0], others[1]}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinations(spots)
}

// findCulturalWalkCombinations は文化的散歩の組み合わせを見つける
func (s *HistoryAndCultureStrategy) findCulturalWalkCombinations(spots []*model.POI) [][]*model.POI {
	// バランス良く文化的スポットを組み合わせる
	spotA := helper.FindHighestRated(spots)
	others := helper.RemovePOI(spots, spotA)
	helper.SortByDistance(spotA, others)

	if len(others) >= 2 {
		combination := []*model.POI{spotA, others[0], others[1]}
		return [][]*model.POI{combination}
	}
	return nil
}

// findDefaultCombinations はデフォルトの組み合わせを見つける
func (s *HistoryAndCultureStrategy) findDefaultCombinations(spots []*model.POI) [][]*model.POI {
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
func (s *HistoryAndCultureStrategy) FindCombinationsWithDestination(scenario string, destinationPOI *model.POI, candidates []*model.POI) [][]*model.POI {
	targetCategories := s.GetTargetCategories(scenario)
	filteredSpots := helper.FilterByCategory(candidates, targetCategories)

	if len(filteredSpots) < 2 {
		return nil
	}

	switch scenario {
	case model.ScenarioTempleShrine:
		return s.findTempleShrineComnationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioMuseumTour:
		return s.findMuseumTourCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioOldTown:
		return s.findOldTownCombinationsWithDestination(destinationPOI, filteredSpots)
	case model.ScenarioCulturalWalk:
		return s.findCulturalWalkCombinationsWithDestination(destinationPOI, filteredSpots)
	default:
		return s.findDefaultCombinationsWithDestination(destinationPOI, filteredSpots)
	}
}

// findTempleShrineComnationsWithDestination は寺社仏閣巡りで目的地を含む組み合わせを見つける
func (s *HistoryAndCultureStrategy) findTempleShrineComnationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 神社仏閣を優先的に選択
	temples := helper.FilterByCategory(spots, []string{"place_of_worship"})

	if len(temples) >= 2 {
		helper.SortByRating(temples)
		combination := []*model.POI{temples[0], temples[1], destination}
		return [][]*model.POI{combination}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findMuseumTourCombinationsWithDestination は博物館・美術館巡りで目的地を含む組み合わせを見つける
func (s *HistoryAndCultureStrategy) findMuseumTourCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 博物館・美術館を優先
	museums := helper.FilterByCategory(spots, []string{"museum", "art_gallery"})

	if len(museums) >= 1 && len(spots) >= 2 {
		helper.SortByRating(museums)
		others := helper.RemovePOI(spots, museums[0])
		if len(others) >= 1 {
			combination := []*model.POI{museums[0], others[0], destination}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findOldTownCombinationsWithDestination は古い街並み散策で目的地を含む組み合わせを見つける
func (s *HistoryAndCultureStrategy) findOldTownCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 観光地を中心とした選択
	attractions := helper.FilterByCategory(spots, []string{"tourist_attraction"})

	if len(attractions) >= 1 && len(spots) >= 2 {
		helper.SortByRating(attractions)
		others := helper.RemovePOI(spots, attractions[0])
		if len(others) >= 1 {
			combination := []*model.POI{attractions[0], others[0], destination}
			return [][]*model.POI{combination}
		}
	}

	return s.findDefaultCombinationsWithDestination(destination, spots)
}

// findCulturalWalkCombinationsWithDestination は文化的散歩で目的地を含む組み合わせを見つける
func (s *HistoryAndCultureStrategy) findCulturalWalkCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// バランス良く文化的スポットを選択
	helper.SortByRating(spots)

	if len(spots) >= 2 {
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}

// findDefaultCombinationsWithDestination はデフォルトの目的地を含む組み合わせを見つける
func (s *HistoryAndCultureStrategy) findDefaultCombinationsWithDestination(destination *model.POI, spots []*model.POI) [][]*model.POI {
	// 評価の高い順にソートして2つ選択
	helper.SortByRating(spots)

	if len(spots) >= 2 {
		combination := []*model.POI{spots[0], spots[1], destination}
		return [][]*model.POI{combination}
	}
	return nil
}
