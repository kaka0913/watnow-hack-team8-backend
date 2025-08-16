package strategy

import (
	"Team8-App/internal/domain/helper"
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"errors"
	"fmt"
)

// NatureStrategy は自然や観光地を巡るルートを提案する
// route-proposal.mdの詳細なロジック仕様に基づいた体験設計を提供
type NatureStrategy struct {
	poiRepo         repository.POIsRepository
	poiSearchHelper *helper.POISearchHelper
}

func NewNatureStrategy(repo repository.POIsRepository) StrategyInterface {
	return &NatureStrategy{
		poiRepo:         repo,
		poiSearchHelper: helper.NewPOISearchHelper(repo),
	}
}

// GetAvailableScenarios はNatureテーマで利用可能なシナリオ一覧を取得する
func (s *NatureStrategy) GetAvailableScenarios() []string {
	return model.GetNatureScenarios()
}

// FindCombinations は自然テーマの詳細なシナリオロジックに基づいて組み合わせを生成する
func (s *NatureStrategy) FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error) {
	switch scenario {
	case model.ScenarioParkTour:
		return s.findParkTourCombinations(ctx, userLocation)
	case model.ScenarioRiverside:
		return s.findRiversideCombinations(ctx, userLocation)
	case model.ScenarioTempleNature:
		return s.findTempleNatureCombinations(ctx, userLocation)
	default:
		return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
	}
}

// findParkTourCombinations は公園巡りシナリオの詳細ロジックを実装
// ロジック: [① メインの公園] → [② ベーカリー/カフェ] → [③ 小さな公園/河川敷]
func (s *NatureStrategy) findParkTourCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: メインとなる大きな公園を選択（検索範囲を徒歩圏内に縮小）
	mainParks, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"公園", "観光名所"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("メインの公園検索に失敗: %w", err)
	}
	if len(mainParks) == 0 {
		return nil, errors.New("ルートの起点となる公園が見つかりませんでした")
	}
	mainPark := helper.FindHighestRated(mainParks)

	// Step 2: 公園周辺で休憩ができるベーカリー/カフェを選択（検索範囲を縮小）
	mainParkLocation := mainPark.ToLatLng()
	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, mainParkLocation, []string{"ベーカリー", "カフェ"}, 800, 5)
	if err != nil {
		return nil, fmt.Errorf("カフェ/ベーカリー検索に失敗: %w", err)
	}
	var cafe *model.POI
	if len(cafes) > 0 {
		helper.SortByDistance(mainPark, cafes)
		cafe = cafes[0]
	}

	// Step 3: 帰り道にある別の公園や河川敷を選択
	var searchLocation model.LatLng
	if cafe != nil {
		searchLocation = cafe.ToLatLng()
	} else {
		searchLocation = mainParkLocation
	}

	otherNature, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"公園", "自然スポット"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("終点の自然スポット検索に失敗: %w", err)
	}

	// メイン公園を除外
	filteredNature := helper.RemovePOI(otherNature, mainPark)
	var finalSpot *model.POI
	if len(filteredNature) > 0 {
		helper.SortByDistanceFromLocation(searchLocation, filteredNature)
		finalSpot = filteredNature[0]
	}

	// 組み合わせを生成
	var combinations [][]*model.POI
	if cafe != nil && finalSpot != nil {
		combinations = append(combinations, []*model.POI{mainPark, cafe, finalSpot})
	} else if finalSpot != nil {
		// カフェが見つからない場合は公園のみでルート生成
		if len(filteredNature) >= 2 {
			combinations = append(combinations, []*model.POI{mainPark, filteredNature[0], filteredNature[1]})
		}
	}

	if len(combinations) == 0 {
		return nil, errors.New("公園巡りの組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// findRiversideCombinations は河川敷散歩シナリオの詳細ロジックを実装
// ロジック: [① カフェ] → [② 河川敷] → [③ 公園]
func (s *NatureStrategy) findRiversideCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: テイクアウト可能なカフェで飲み物を準備
	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1000, 5)
	if err != nil {
		return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
	}
	var cafe *model.POI
	if len(cafes) > 0 {
		cafe = helper.FindHighestRated(cafes)
	}

	// Step 2: メインとなる河川敷を選択
	var searchLocation model.LatLng
	if cafe != nil {
		searchLocation = cafe.ToLatLng()
	} else {
		searchLocation = userLocation
	}

	rivers, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"観光名所"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("河川敷検索に失敗: %w", err)
	}
	if len(rivers) == 0 {
		return nil, errors.New("散歩できる河川敷が見つかりませんでした")
	}

	var river *model.POI
	if cafe != nil {
		helper.SortByDistance(cafe, rivers)
		river = rivers[0]
	} else {
		river = helper.FindHighestRated(rivers)
	}

	// Step 3: 河川敷の終点近くの公園で休憩
	riverLocation := river.ToLatLng()
	parks, err := s.poiRepo.FindNearbyByCategories(ctx, riverLocation, []string{"公園"}, 800, 5)
	if err != nil {
		return nil, fmt.Errorf("終点の公園検索に失敗: %w", err)
	}
	var park *model.POI
	if len(parks) > 0 {
		helper.SortByDistance(river, parks)
		park = parks[0]
	}

	// 組み合わせを生成
	var combinations [][]*model.POI
	if cafe != nil && river != nil && park != nil {
		combinations = append(combinations, []*model.POI{cafe, river, park})
	} else if river != nil && park != nil {
		// カフェが見つからない場合は河川敷と公園のみ
		combinations = append(combinations, []*model.POI{river, park, river}) // 河川敷を往復
	}

	if len(combinations) == 0 {
		return nil, errors.New("河川敷散歩の組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// findTempleNatureCombinations は寺社と自然シナリオの詳細ロジックを実装
// ロジック: [① 庭園のある寺社] → [② 開けた公園] → [③ 参道の店]
func (s *NatureStrategy) findTempleNatureCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 庭園のある寺社（寺院 + 公園 の両カテゴリ）を選択
	temples, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"寺院"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("寺社検索に失敗: %w", err)
	}
	if len(temples) == 0 {
		return nil, errors.New("訪れることができる寺社が見つかりませんでした")
	}

	// 庭園を持つ寺社を優先的に選択（実際にはカテゴリの組み合わせで判定）
	var templeGarden *model.POI
	for _, temple := range temples {
		if helper.HasCategory(temple, []string{"公園"}) {
			templeGarden = temple
			break
		}
	}
	if templeGarden == nil {
		// 庭園のある寺社が見つからない場合は評価の高い寺社を選択
		templeGarden = helper.FindHighestRated(temples)
	}

	// Step 2: 視界が開ける大きな公園を選択
	templeLocation := templeGarden.ToLatLng()
	parks, err := s.poiRepo.FindNearbyByCategories(ctx, templeLocation, []string{"公園", "観光名所"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("公園検索に失敗: %w", err)
	}

	// 寺社を除外して、開けた公園を選択
	filteredParks := helper.RemovePOI(parks, templeGarden)
	var openPark *model.POI
	if len(filteredParks) > 0 {
		openPark = helper.FindHighestRated(filteredParks)
	}

	// Step 3: 参道の店舗を選択
	var searchLocation model.LatLng
	if openPark != nil {
		searchLocation = openPark.ToLatLng()
	} else {
		searchLocation = templeLocation
	}

	stores, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"店舗", "観光名所"}, 800, 5)
	if err != nil {
		return nil, fmt.Errorf("参道の店舗検索に失敗: %w", err)
	}

	// 寺社と公園を除外
	filteredStores := stores
	if openPark != nil {
		filteredStores = helper.RemovePOI(filteredStores, openPark)
	}
	filteredStores = helper.RemovePOI(filteredStores, templeGarden)

	var store *model.POI
	if len(filteredStores) > 0 {
		helper.SortByDistanceFromLocation(templeLocation, filteredStores) // 寺社から近い順
		store = filteredStores[0]
	}

	// 組み合わせを生成
	var combinations [][]*model.POI
	if openPark != nil && store != nil {
		combinations = append(combinations, []*model.POI{templeGarden, openPark, store})
	} else if openPark != nil {
		// 店舗が見つからない場合は寺社と公園のみ
		combinations = append(combinations, []*model.POI{templeGarden, openPark})
	} else if store != nil {
		// 公園が見つからない場合は寺社と店舗のみ
		combinations = append(combinations, []*model.POI{templeGarden, store})
	}

	if len(combinations) == 0 {
		return nil, errors.New("寺社と自然の組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// FindCombinationsWithDestination は目的地を含むルート組み合わせを見つける
func (s *NatureStrategy) FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	switch scenario {
	case model.ScenarioParkTour:
		return s.findParkTourWithDestination(ctx, userLocation, destination)
	case model.ScenarioRiverside:
		return s.findRiversideWithDestination(ctx, userLocation, destination)
	case model.ScenarioTempleNature:
		return s.findTempleNatureWithDestination(ctx, userLocation, destination)
	default:
		return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
	}
}

// findParkTourWithDestination は公園巡りで目的地を含む組み合わせを見つける
// ロジック: [① 公園A] → [② 公園B] (目的地へのルート上)
func (s *NatureStrategy) findParkTourWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// 目的地周辺のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination)
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// ルート経路上の公園を2つ選択
	parks, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"公園"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("公園検索に失敗: %w", err)
	}

	if len(parks) < 2 {
		return nil, errors.New("ルート上に十分な公園が見つかりませんでした")
	}

	// 距離に基づいて2つの公園を選択
	park1 := helper.FindHighestRated(parks)
	filteredParks := helper.RemovePOI(parks, park1)
	var park2 *model.POI
	if len(filteredParks) > 0 {
		park2 = filteredParks[0]
	}

	var combinations [][]*model.POI
	if park1 != nil && park2 != nil {
		combinations = append(combinations, []*model.POI{park1, park2, destinationPOI})
	}

	return combinations, nil
}

// findRiversideWithDestination は河川敷散歩で目的地を含む組み合わせを見つける
// ロジック: [① 河川敷の入口] → [② 河川敷沿いの公園] (目的地へのルート上)
func (s *NatureStrategy) findRiversideWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// 目的地周辺のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination)
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// 河川敷の入口を選択（観光名所として登録されている水辺）
	rivers, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"観光名所"}, 1500, 5)
	if err != nil {
		return nil, fmt.Errorf("河川敷検索に失敗: %w", err)
	}
	if len(rivers) == 0 {
		return nil, errors.New("河川敷が見つかりませんでした")
	}
	river := rivers[0]

	// 河川敷沿いの公園を選択
	riverLocation := river.ToLatLng()
	parks, err := s.poiRepo.FindNearbyByCategories(ctx, riverLocation, []string{"公園"}, 1000, 5)
	if err != nil {
		return nil, fmt.Errorf("河川敷沿いの公園検索に失敗: %w", err)
	}
	var park *model.POI
	if len(parks) > 0 {
		park = parks[0]
	}

	var combinations [][]*model.POI
	if river != nil && park != nil {
		combinations = append(combinations, []*model.POI{river, park, destinationPOI})
	}

	return combinations, nil
}

// findTempleNatureWithDestination は寺社と自然で目的地を含む組み合わせを見つける
// ロジック: [① 庭園のある寺社] → [② 開けた公園] (目的地へのルート上)
func (s *NatureStrategy) findTempleNatureWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// 目的地周辺のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination)
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// 庭園のある寺社を選択
	temples, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"寺院"}, 1500, 5)
	if err != nil {
		return nil, fmt.Errorf("寺社検索に失敗: %w", err)
	}
	if len(temples) == 0 {
		return nil, errors.New("寺社が見つかりませんでした")
	}
	temple := helper.FindHighestRated(temples)

	// 開けた公園を選択
	templeLocation := temple.ToLatLng()
	parks, err := s.poiRepo.FindNearbyByCategories(ctx, templeLocation, []string{"公園"}, 1000, 5)
	if err != nil {
		return nil, fmt.Errorf("公園検索に失敗: %w", err)
	}
	var park *model.POI
	if len(parks) > 0 {
		park = parks[0]
	}

	var combinations [][]*model.POI
	if temple != nil && park != nil {
		combinations = append(combinations, []*model.POI{temple, park, destinationPOI})
	}

	return combinations, nil
}
