package strategy

import (
	"Team8-App/internal/domain/helper"
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"errors"
	"fmt"
)

// HorrorStrategy はホラーやスリルを楽しむルートを提案する
type HorrorStrategy struct {
	poiRepo         repository.POIsRepository
	poiSearchHelper *helper.POISearchHelper
}

func NewHorrorStrategy(repo repository.POIsRepository) StrategyInterface {
	return &HorrorStrategy{
		poiRepo:         repo,
		poiSearchHelper: helper.NewPOISearchHelper(repo),
	}
}

var (
	// 車移動前提で最大30kmまで拡大した段階的検索設定
	horrorSpotSearchConfigs = []SearchConfig{
		{[]string{"horror_spot"}, 8000, 15},                                         // 8km圏内（近距離車移動）
		{[]string{"horror_spot", "tourist_attraction"}, 15000, 20},                  // 15km圏内（中距離車移動）
		{[]string{"horror_spot", "tourist_attraction", "establishment"}, 30000, 25}, // 30km圏内（遠距離車移動）
	}

	worshipPlaceSearchConfigs = []SearchConfig{
		{[]string{"place_of_worship"}, 7000, 15},                                         // 7km圏内
		{[]string{"place_of_worship", "tourist_attraction"}, 12000, 20},                  // 12km圏内
		{[]string{"place_of_worship", "tourist_attraction", "establishment"}, 25000, 25}, // 25km圏内
	}

	naturalFeatureSearchConfigs = []SearchConfig{
		{[]string{"natural_feature"}, 6000, 15},                                // 6km圏内
		{[]string{"natural_feature", "park"}, 10000, 20},                       // 10km圏内
		{[]string{"natural_feature", "park", "tourist_attraction"}, 20000, 25}, // 20km圏内
	}

	establishmentSearchConfigs = []SearchConfig{
		{[]string{"establishment"}, 5000, 15},                                 // 5km圏内
		{[]string{"establishment", "store"}, 8000, 20},                        // 8km圏内
		{[]string{"establishment", "store", "tourist_attraction"}, 15000, 25}, // 15km圏内
	}

	// セカンダリ検索用の段階的設定（車移動対応の大幅拡大）
	horrorStoreSearchConfigs = []SearchConfig{
		{[]string{"store"}, 6000, 15},                                         // 6km圏内
		{[]string{"store", "establishment"}, 10000, 20},                       // 10km圏内
		{[]string{"store", "establishment", "tourist_attraction"}, 18000, 25}, // 18km圏内
	}

	horrorCafeSearchConfigs = []SearchConfig{
		{[]string{"cafe"}, 7000, 15},                            // 7km圏内
		{[]string{"cafe", "store"}, 12000, 20},                  // 12km圏内
		{[]string{"cafe", "store", "establishment"}, 20000, 25}, // 20km圏内
	}

	horrorParkSearchConfigs = []SearchConfig{
		{[]string{"park"}, 6000, 15},                                           // 6km圏内
		{[]string{"park", "natural_feature"}, 10000, 20},                       // 10km圏内
		{[]string{"park", "natural_feature", "tourist_attraction"}, 18000, 25}, // 18km圏内
	}
)

// 段階的検索の共通化メソッド
func (s *HorrorStrategy) findPOIWithFallback(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig) ([]*model.POI, error) {
	for _, config := range searchConfigs {
		pois, err := s.poiRepo.FindNearbyByCategories(ctx, location, config.Categories, config.Range, config.Limit)
		if err == nil && len(pois) > 0 {
			return pois, nil
		}
	}
	return nil, nil
}

// findBestPOI は指定された検索設定で最初のPOIを1つ見つける（距離順）
func (s *HorrorStrategy) findBestPOI(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig) *model.POI {
	pois, err := s.findPOIWithFallback(ctx, location, searchConfigs)
	if err != nil || len(pois) == 0 {
		return nil
	}
	helper.SortByDistanceFromLocation(location, pois)
	return pois[0]
}

// 目的地なしメソッド用の共通ヘルパー
func (s *HorrorStrategy) buildCombination(spots ...*model.POI) [][]*model.POI {
	var validSpots []*model.POI
	for _, spot := range spots {
		if spot != nil {
			validSpots = append(validSpots, spot)
		}
	}

	if len(validSpots) == 0 {
		return nil
	}

	return [][]*model.POI{validSpots}
}

// 距離優先検索の統一メソッド
func (s *HorrorStrategy) findNearestPOI(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig, excludePOIs ...*model.POI) *model.POI {
	spots, err := s.findPOIWithFallback(ctx, location, searchConfigs)
	if err != nil || len(spots) == 0 {
		return nil
	}

	for _, excludePOI := range excludePOIs {
		if excludePOI != nil {
			spots = helper.RemovePOI(spots, excludePOI)
		}
	}

	if len(spots) == 0 {
		return nil
	}

	helper.SortByDistanceFromLocation(location, spots)
	return spots[0]
}

// 距離優先検索の統一メソッド
func (s *HorrorStrategy) findRatedPOI(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig, excludePOIs ...*model.POI) *model.POI {
	spots, err := s.findPOIWithFallback(ctx, location, searchConfigs)
	if err != nil || len(spots) == 0 {
		return nil
	}

	for _, excludePOI := range excludePOIs {
		if excludePOI != nil {
			spots = helper.RemovePOI(spots, excludePOI)
		}
	}

	if len(spots) == 0 {
		return nil
	}

	helper.SortByDistanceFromLocation(location, spots)
	return spots[0]
}

// 目的地ありメソッド用の共通ヘルパー
func (s *HorrorStrategy) findDestinationPOI(ctx context.Context, destination model.LatLng, categories []string) (*model.POI, error) {
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, categories)
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}
	return destinationPOI, nil
}

func (s *HorrorStrategy) buildDestinationCombination(pois []*model.POI, destinationPOI *model.POI) ([][]*model.POI, error) {
	if len(pois) == 0 {
		return nil, errors.New("組み合わせが見つかりませんでした")
	}

	var combinations [][]*model.POI
	allPOIs := append(pois, destinationPOI)

	if s.poiSearchHelper.ValidateCombination(allPOIs, 0, false) {
		combinations = append(combinations, allPOIs)
	}

	if len(combinations) == 0 {
		return nil, errors.New("有効な組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// GetAvailableScenarios はHorrorテーマで利用可能なシナリオ一覧を取得する
func (s *HorrorStrategy) GetAvailableScenarios() []string {
	return model.GetHorrorScenarios()
}

// 目的地なしの統一ハンドラー（段階的検索で3つのスポットを巡る）
func (s *HorrorStrategy) FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error) {
	switch scenario {
	case model.ScenarioGhostTour:
		return s.findGhostTourCombinations(ctx, userLocation)
	case model.ScenarioHauntedRuins:
		return s.findHauntedRuinsCombinations(ctx, userLocation)
	case model.ScenarioCursedNature:
		return s.findCursedNatureCombinations(ctx, userLocation)
	case model.ScenarioCemeteryWalk:
		return s.findCemeteryWalkCombinations(ctx, userLocation)
	default:
		return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
	}
}

// 心霊スポット巡りシナリオ（距離ベースで3スポット確保）
func (s *HorrorStrategy) findGhostTourCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 曰く付きの寺社（段階的検索: place_of_worship → +tourist_attraction → +establishment）
	cursedTemple := s.findBestPOI(ctx, userLocation, worshipPlaceSearchConfigs)
	if cursedTemple == nil {
		return nil, errors.New("曰く付きの寺社が見つかりませんでした")
	}

	// Step 2: メインの心霊スポット（段階的検索: horror_spot → +tourist_attraction → +establishment）
	mainHorrorSpot := s.findRatedPOI(ctx, cursedTemple.ToLatLng(), horrorSpotSearchConfigs, cursedTemple)

	// Step 3: コンビニ/明るい大通り（段階的検索: store → +establishment → +tourist_attraction）
	searchLocation := cursedTemple.ToLatLng()
	if mainHorrorSpot != nil {
		searchLocation = mainHorrorSpot.ToLatLng()
	}
	safeStore := s.findNearestPOI(ctx, searchLocation, horrorStoreSearchConfigs, cursedTemple, mainHorrorSpot)

	return s.buildCombination(cursedTemple, mainHorrorSpot, safeStore), nil
}

// 廃墟探索シナリオ（距離ベースで3スポット確保）
func (s *HorrorStrategy) findHauntedRuinsCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 関連施設（段階的検索: establishment → +store → +tourist_attraction）
	relatedFacility := s.findBestPOI(ctx, userLocation, establishmentSearchConfigs)
	if relatedFacility == nil {
		return nil, errors.New("関連施設が見つかりませんでした")
	}

	// Step 2: 廃墟スポット（段階的検索: horror_spot → +tourist_attraction → +establishment）
	ruinSpot := s.findRatedPOI(ctx, relatedFacility.ToLatLng(), horrorSpotSearchConfigs, relatedFacility)

	// Step 3: カフェ（段階的検索: cafe → +store → +establishment）
	searchLocation := relatedFacility.ToLatLng()
	if ruinSpot != nil {
		searchLocation = ruinSpot.ToLatLng()
	}
	cafe := s.findNearestPOI(ctx, searchLocation, horrorCafeSearchConfigs, relatedFacility, ruinSpot)

	return s.buildCombination(relatedFacility, ruinSpot, cafe), nil
}

// 呪いの自然シナリオ（距離ベースで3スポット確保）
func (s *HorrorStrategy) findCursedNatureCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 静かな公園（段階的検索: park → +natural_feature → +tourist_attraction）
	quietPark := s.findBestPOI(ctx, userLocation, horrorParkSearchConfigs)
	if quietPark == nil {
		return nil, errors.New("静かな公園が見つかりませんでした")
	}

	// Step 2: 呪いの自然スポット（段階的検索: natural_feature → +park → +tourist_attraction）
	cursedNature := s.findRatedPOI(ctx, quietPark.ToLatLng(), naturalFeatureSearchConfigs, quietPark)

	// Step 3: 賑やかな場所（段階的検索: store → +establishment → +tourist_attraction）
	searchLocation := quietPark.ToLatLng()
	if cursedNature != nil {
		searchLocation = cursedNature.ToLatLng()
	}
	bustlingPlace := s.findNearestPOI(ctx, searchLocation, horrorStoreSearchConfigs, quietPark, cursedNature)

	return s.buildCombination(quietPark, cursedNature, bustlingPlace), nil
}

// 墓地・慰霊散歩シナリオ（距離ベースで3スポット確保）
func (s *HorrorStrategy) findCemeteryWalkCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 歴史的建造物（段階的検索: place_of_worship → +tourist_attraction → +establishment）
	historicBuilding := s.findBestPOI(ctx, userLocation, worshipPlaceSearchConfigs)
	if historicBuilding == nil {
		return nil, errors.New("歴史的建造物が見つかりませんでした")
	}

	// Step 2: 墓地/慰霊碑（段階的検索: horror_spot → +tourist_attraction → +establishment）
	memorial := s.findRatedPOI(ctx, historicBuilding.ToLatLng(), horrorSpotSearchConfigs, historicBuilding)

	// Step 3: カフェ（段階的検索: cafe → +store → +establishment）
	searchLocation := historicBuilding.ToLatLng()
	if memorial != nil {
		searchLocation = memorial.ToLatLng()
	}
	cafe := s.findNearestPOI(ctx, searchLocation, horrorCafeSearchConfigs, historicBuilding, memorial)

	return s.buildCombination(historicBuilding, memorial, cafe), nil
}

// 目的地を含むルート組み合わせを見つける（距離ベースで2つのスポット確保）
func (s *HorrorStrategy) FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	switch scenario {
	case model.ScenarioGhostTour:
		return s.findGhostTourCombinationsWithDestination(ctx, userLocation, destination)
	case model.ScenarioHauntedRuins:
		return s.findHauntedRuinsCombinationsWithDestination(ctx, userLocation, destination)
	case model.ScenarioCursedNature:
		return s.findCursedNatureCombinationsWithDestination(ctx, userLocation, destination)
	case model.ScenarioCemeteryWalk:
		return s.findCemeteryWalkCombinationsWithDestination(ctx, userLocation, destination)
	default:
		return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
	}
}

// 心霊スポット巡り目的地あり（距離ベースで2つのスポット確保）
func (s *HorrorStrategy) findGhostTourCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"horror_spot", "place_of_worship", "tourist_attraction"})
	if err != nil {
		return nil, err
	}

	// Step 1: 曰く付きの寺社（複数カテゴリ組み合わせ段階的検索）
	cursedTemple := s.findBestPOI(ctx, userLocation, worshipPlaceSearchConfigs)
	if cursedTemple == nil {
		return nil, errors.New("曰く付きの寺社が見つかりませんでした")
	}

	// Step 2: メインの心霊スポット（複数カテゴリ組み合わせ段階的検索）
	mainHorrorSpot := s.findRatedPOI(ctx, cursedTemple.ToLatLng(), horrorSpotSearchConfigs, cursedTemple)
	if mainHorrorSpot == nil {
		return nil, errors.New("メインの心霊スポットが見つかりませんでした")
	}

	pois := []*model.POI{cursedTemple, mainHorrorSpot}
	return s.buildDestinationCombination(pois, destinationPOI)
}

// 廃墟探索目的地あり（距離ベースで2つのスポット確保）
func (s *HorrorStrategy) findHauntedRuinsCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"horror_spot", "establishment", "cafe"})
	if err != nil {
		return nil, err
	}

	// Step 1: 関連施設（複数カテゴリ組み合わせ段階的検索）
	relatedFacility := s.findBestPOI(ctx, userLocation, establishmentSearchConfigs)
	if relatedFacility == nil {
		return nil, errors.New("関連施設が見つかりませんでした")
	}

	// Step 2: 廃墟スポット（複数カテゴリ組み合わせ段階的検索）
	ruinSpot := s.findRatedPOI(ctx, relatedFacility.ToLatLng(), horrorSpotSearchConfigs, relatedFacility)
	if ruinSpot == nil {
		return nil, errors.New("廃墟スポットが見つかりませんでした")
	}

	pois := []*model.POI{relatedFacility, ruinSpot}
	return s.buildDestinationCombination(pois, destinationPOI)
}

// 呪いの自然目的地あり（距離ベースで2つのスポット確保）
func (s *HorrorStrategy) findCursedNatureCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"natural_feature", "horror_spot", "tourist_attraction"})
	if err != nil {
		return nil, err
	}

	// Step 1: 静かな公園（複数カテゴリ組み合わせ段階的検索）
	quietPark := s.findBestPOI(ctx, userLocation, horrorParkSearchConfigs)
	if quietPark == nil {
		return nil, errors.New("静かな公園が見つかりませんでした")
	}

	// Step 2: 呪いの自然スポット（複数カテゴリ組み合わせ段階的検索）
	cursedNature := s.findRatedPOI(ctx, quietPark.ToLatLng(), naturalFeatureSearchConfigs, quietPark)
	if cursedNature == nil {
		return nil, errors.New("呪いの自然スポットが見つかりませんでした")
	}

	pois := []*model.POI{quietPark, cursedNature}
	return s.buildDestinationCombination(pois, destinationPOI)
}

// 墓地・慰霊散歩目的地あり（距離ベースで2つのスポット確保）
func (s *HorrorStrategy) findCemeteryWalkCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"horror_spot", "place_of_worship", "cafe"})
	if err != nil {
		return nil, err
	}

	// Step 1: 歴史的建造物（複数カテゴリ組み合わせ段階的検索）
	historicBuilding := s.findBestPOI(ctx, userLocation, worshipPlaceSearchConfigs)
	if historicBuilding == nil {
		return nil, errors.New("歴史的建造物が見つかりませんでした")
	}

	// Step 2: 墓地/慰霊碑（複数カテゴリ組み合わせ段階的検索）
	memorial := s.findRatedPOI(ctx, historicBuilding.ToLatLng(), horrorSpotSearchConfigs, historicBuilding)
	if memorial == nil {
		return nil, errors.New("墓地/慰霊碑が見つかりませんでした")
	}

	pois := []*model.POI{historicBuilding, memorial}
	return s.buildDestinationCombination(pois, destinationPOI)
}

// ExploreNewSpots はルート再計算用の新しいスポット探索を行う（車移動対応の大幅範囲拡大）
func (s *HorrorStrategy) ExploreNewSpots(ctx context.Context, searchLocation model.LatLng) ([]*model.POI, error) {
	horrorCategories := []string{"horror_spot", "place_of_worship", "natural_feature", "establishment", "tourist_attraction"}

	radiuses := []int{3000, 8000, 15000} // 車移動対応の大幅拡大（3km, 8km, 15km）

	var allSpots []*model.POI
	for _, radius := range radiuses {
		spots, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, horrorCategories, radius, 20)
		if err != nil {
			continue
		}

		for _, spot := range spots {
			isDuplicate := false
			for _, existing := range allSpots {
				if existing.ID == spot.ID {
					isDuplicate = true
					break
				}
			}
			if !isDuplicate {
				allSpots = append(allSpots, spot)
			}
		}

		if len(allSpots) >= 15 {
			break
		}
	}

	return allSpots, nil
}
