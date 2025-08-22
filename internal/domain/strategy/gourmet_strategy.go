package strategy

import (
	"Team8-App/internal/domain/helper"
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"errors"
	"fmt"
)

// GourmetStrategy はカフェやベーカリーを巡るルートを提案する
// route-proposal.mdの詳細なロジック仕様に基づいた体験設計を提供
type GourmetStrategy struct {
	poiRepo         repository.POIsRepository
	poiSearchHelper *helper.POISearchHelper
}

func NewGourmetStrategy(repo repository.POIsRepository) StrategyInterface {
	return &GourmetStrategy{
		poiRepo:         repo,
		poiSearchHelper: helper.NewPOISearchHelper(repo),
	}
}

// GetAvailableScenarios はGourmetテーマで利用可能なシナリオ一覧を取得する
func (s *GourmetStrategy) GetAvailableScenarios() []string {
	return model.GetGourmetScenarios()
}

// FindCombinations はグルメテーマの詳細なシナリオロジックに基づいて組み合わせを生成する
func (s *GourmetStrategy) FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error) {
	switch scenario {
	case model.ScenarioCafeHopping:
		return s.findCafeHoppingCombinations(ctx, userLocation)
	case model.ScenarioBakeryTour:
		return s.findBakeryTourCombinations(ctx, userLocation)
	case model.ScenarioLocalGourmet:
		return s.findLocalGourmetCombinations(ctx, userLocation)
	case model.ScenarioSweetJourney:
		return s.findSweetJourneyCombinations(ctx, userLocation)
	default:
		return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
	}
}

// findCafeHoppingCombinations はカフェ巡りシナリオの詳細ロジックを実装
// ロジック: [① 書店/雑貨店] → [② メインのカフェ] → [③ 公園/ベーカリー]
func (s *GourmetStrategy) findCafeHoppingCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 書店/雑貨店を選択
	bookStores, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"書店", "雑貨店"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("書店/雑貨店検索に失敗: %w", err)
	}
	var bookStore *model.POI
	if len(bookStores) > 0 {
		bookStore = helper.FindHighestRated(bookStores)
	}

	// Step 2: メインのカフェを選択
	var searchLocation model.LatLng
	if bookStore != nil {
		searchLocation = bookStore.ToLatLng()
	} else {
		searchLocation = userLocation
	}

	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"カフェ"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
	}
	if len(cafes) == 0 {
		return nil, errors.New("カフェが見つかりませんでした")
	}

	var mainCafe *model.POI
	if bookStore != nil {
		helper.SortByDistance(bookStore, cafes)
		mainCafe = cafes[0]
	} else {
		mainCafe = helper.FindHighestRated(cafes)
	}

	// Step 3: 公園/ベーカリーを選択
	cafeLocation := mainCafe.ToLatLng()
	finaleSpots, err := s.poiRepo.FindNearbyByCategories(ctx, cafeLocation, []string{"公園", "ベーカリー"}, 800, 10)
	if err != nil {
		return nil, fmt.Errorf("公園/ベーカリー検索に失敗: %w", err)
	}

	// カフェを除外
	filteredFinaleSpots := helper.RemovePOI(finaleSpots, mainCafe)
	var finaleSpot *model.POI
	if len(filteredFinaleSpots) > 0 {
		helper.SortByDistance(mainCafe, filteredFinaleSpots)
		finaleSpot = filteredFinaleSpots[0]
	}

	// 組み合わせを生成
	var combinations [][]*model.POI
	if bookStore != nil && finaleSpot != nil {
		combination := []*model.POI{bookStore, mainCafe, finaleSpot}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	} else if finaleSpot != nil {
		// 書店が見つからない場合はカフェと公園/ベーカリーのみ
		combination := []*model.POI{mainCafe, finaleSpot}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	if len(combinations) == 0 {
		return nil, errors.New("カフェ巡りの組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// findBakeryTourCombinations はベーカリー巡りシナリオの詳細ロジックを実装
// ロジック: [① ベーカリー A] → [② 公園] → [③ ベーカリー B]
func (s *GourmetStrategy) findBakeryTourCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 評価の高いベーカリーを選択
	bakeries, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"ベーカリー"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("ベーカリー検索に失敗: %w", err)
	}
	if len(bakeries) < 2 {
		return nil, errors.New("ルートに必要なベーカリーが見つかりませんでした")
	}

	bakeryA := helper.FindHighestRated(bakeries)
	filteredBakeries := helper.RemovePOI(bakeries, bakeryA)
	bakeryB := helper.FindHighestRated(filteredBakeries)

	// Step 2: 2つのベーカリーの中間にある公園を選択
	bakeryALocation := bakeryA.ToLatLng()
	bakeryBLocation := bakeryB.ToLatLng()

	// 中間地点を計算
	midLat := (bakeryALocation.Lat + bakeryBLocation.Lat) / 2
	midLng := (bakeryALocation.Lng + bakeryBLocation.Lng) / 2
	midLocation := model.LatLng{Lat: midLat, Lng: midLng}

	parks, err := s.poiRepo.FindNearbyByCategories(ctx, midLocation, []string{"公園"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("公園検索に失敗: %w", err)
	}

	var park *model.POI
	if len(parks) > 0 {
		helper.SortByDistanceFromLocation(midLocation, parks)
		park = parks[0]
	}

	// 組み合わせを生成
	var combinations [][]*model.POI
	if park != nil {
		combination := []*model.POI{bakeryA, park, bakeryB}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	} else {
		// 公園が見つからない場合はベーカリー2つのみ
		combination := []*model.POI{bakeryA, bakeryB}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	if len(combinations) == 0 {
		return nil, errors.New("ベーカリー巡りの組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// findLocalGourmetCombinations は地元グルメシナリオの詳細ロジックを実装
// ロジック: [① カフェ] → [② メインの食事処] → [③ 公園/商店街]
func (s *GourmetStrategy) findLocalGourmetCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: 食前のお茶ができるカフェを選択
	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1200, 10)
	if err != nil {
		return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
	}
	var cafe *model.POI
	if len(cafes) > 0 {
		cafe = helper.FindHighestRated(cafes)
	}

	// Step 2: メインとなる地元の名店（レストラン+商店カテゴリ）を選択
	var searchLocation model.LatLng
	if cafe != nil {
		searchLocation = cafe.ToLatLng()
	} else {
		searchLocation = userLocation
	}

	restaurants, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"レストラン", "食品店"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("レストラン検索に失敗: %w", err)
	}
	if len(restaurants) == 0 {
		return nil, errors.New("地元の食事処が見つかりませんでした")
	}

	// レストラン+商店の両カテゴリを持つスポットを優先
	var mainRestaurant *model.POI
	for _, restaurant := range restaurants {
		if helper.HasCategory(restaurant, []string{"商店"}) {
			mainRestaurant = restaurant
			break
		}
	}
	if mainRestaurant == nil {
		mainRestaurant = helper.FindHighestRated(restaurants)
	}

	// Step 3: 食後の散歩に最適な公園や商店街を選択
	restaurantLocation := mainRestaurant.ToLatLng()
	afterSpots, err := s.poiRepo.FindNearbyByCategories(ctx, restaurantLocation, []string{"公園", "観光名所"}, 800, 10)
	if err != nil {
		return nil, fmt.Errorf("食後スポット検索に失敗: %w", err)
	}

	// レストランとカフェを除外
	filteredAfterSpots := helper.RemovePOI(afterSpots, mainRestaurant)
	if cafe != nil {
		filteredAfterSpots = helper.RemovePOI(filteredAfterSpots, cafe)
	}

	var afterSpot *model.POI
	if len(filteredAfterSpots) > 0 {
		helper.SortByDistance(mainRestaurant, filteredAfterSpots)
		afterSpot = filteredAfterSpots[0]
	}

	// 組み合わせを生成
	var combinations [][]*model.POI
	if cafe != nil && afterSpot != nil {
		combination := []*model.POI{cafe, mainRestaurant, afterSpot}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	} else if afterSpot != nil {
		// カフェが見つからない場合はレストランと食後スポットのみ
		combination := []*model.POI{mainRestaurant, afterSpot}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	if len(combinations) == 0 {
		return nil, errors.New("地元グルメの組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// findSweetJourneyCombinations はスイーツ巡りシナリオの詳細ロジックを実装
// ロジック: [① カフェ(ケーキ等)] → [② 雑貨店] → [③ カフェ(ジェラート等)]
func (s *GourmetStrategy) findSweetJourneyCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
	// Step 1: ケーキやパフェが評判のカフェを選択
	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
	}
	if len(cafes) == 0 {
		return nil, errors.New("スイーツカフェが見つかりませんでした")
	}
	cafeA := helper.FindHighestRated(cafes)

	// Step 2: 気分転換に立ち寄れる可愛い雑貨店を選択
	cafeALocation := cafeA.ToLatLng()
	shops, err := s.poiRepo.FindNearbyByCategories(ctx, cafeALocation, []string{"雑貨店", "商店"}, 800, 10)
	if err != nil {
		return nil, fmt.Errorf("雑貨店検索に失敗: %w", err)
	}

	var shop *model.POI
	if len(shops) > 0 {
		helper.SortByDistance(cafeA, shops)
		shop = shops[0]
	}

	// Step 3: ジェラート等が楽しめる別のカフェや商店を選択
	var searchLocation model.LatLng
	if shop != nil {
		searchLocation = shop.ToLatLng()
	} else {
		searchLocation = cafeALocation
	}

	sweetSpots, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"カフェ", "商店"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("スイーツスポット検索に失敗: %w", err)
	}

	// 1つ目のカフェを除外
	filteredSweetSpots := helper.RemovePOI(sweetSpots, cafeA)
	if shop != nil {
		filteredSweetSpots = helper.RemovePOI(filteredSweetSpots, shop)
	}

	var sweetSpot *model.POI
	if len(filteredSweetSpots) > 0 {
		helper.SortByDistanceFromLocation(searchLocation, filteredSweetSpots)
		sweetSpot = filteredSweetSpots[0]
	}

	// 組み合わせを生成
	var combinations [][]*model.POI
	if shop != nil && sweetSpot != nil {
		combination := []*model.POI{cafeA, shop, sweetSpot}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	} else if sweetSpot != nil {
		// 雑貨店が見つからない場合はカフェ2つのみ
		combination := []*model.POI{cafeA, sweetSpot}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	if len(combinations) == 0 {
		return nil, errors.New("スイーツ巡りの組み合わせが見つかりませんでした")
	}

	return combinations, nil
}

// FindCombinationsWithDestination は目的地を含むルート組み合わせを見つける
func (s *GourmetStrategy) FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	switch scenario {
	case model.ScenarioCafeHopping:
		return s.findCafeHoppingWithDestination(ctx, userLocation, destination)
	case model.ScenarioBakeryTour:
		return s.findBakeryTourWithDestination(ctx, userLocation, destination)
	case model.ScenarioLocalGourmet:
		return s.findLocalGourmetWithDestination(ctx, userLocation, destination)
	case model.ScenarioSweetJourney:
		return s.findSweetJourneyWithDestination(ctx, userLocation, destination)
	default:
		return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
	}
}

// findCafeHoppingWithDestination はカフェ巡りで目的地を含む組み合わせを見つける
// ロジック: [① 前半のカフェ] → [② 後半のカフェ]
func (s *GourmetStrategy) findCafeHoppingWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// 目的地周辺のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"カフェ"})
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// ルート前半にある評価の高いカフェを選択
	cafes1, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("前半のカフェ検索に失敗: %w", err)
	}
	if len(cafes1) == 0 {
		return nil, errors.New("前半のカフェが見つかりませんでした")
	}
	cafe1 := helper.FindHighestRated(cafes1)

	// ルート後半にある雰囲気の違うカフェを選択
	cafe1Location := cafe1.ToLatLng()
	cafes2, err := s.poiRepo.FindNearbyByCategories(ctx, cafe1Location, []string{"カフェ"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("後半のカフェ検索に失敗: %w", err)
	}

	// 1つ目のカフェを除外
	filteredCafes2 := helper.RemovePOI(cafes2, cafe1)
	var cafe2 *model.POI
	if len(filteredCafes2) > 0 {
		cafe2 = helper.FindHighestRated(filteredCafes2)
	}

	var combinations [][]*model.POI
	if cafe2 != nil {
		combination := []*model.POI{cafe1, cafe2, destinationPOI}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	return combinations, nil
}

// findBakeryTourWithDestination はベーカリー巡りで目的地を含む組み合わせを見つける
// ロジック: [① 出発地のベーカリー] → [② イートイン可能なカフェ]
func (s *GourmetStrategy) findBakeryTourWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// 目的地周辺のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"レストラン", "食品店"})
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// 出発地近くにある評価の高いベーカリーを選択
	bakeries, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"ベーカリー"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("ベーカリー検索に失敗: %w", err)
	}
	if len(bakeries) == 0 {
		return nil, errors.New("ベーカリーが見つかりませんでした")
	}
	bakery := helper.FindHighestRated(bakeries)

	// 目的地のすぐ手前にある、カフェカテゴリも持つスポット（イートイン可能なベーカリーカフェなど）を選択
	bakeryLocation := bakery.ToLatLng()
	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, bakeryLocation, []string{"カフェ", "ベーカリー"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("イートイン可能なカフェ検索に失敗: %w", err)
	}

	// ベーカリーを除外
	filteredCafes := helper.RemovePOI(cafes, bakery)
	var cafe *model.POI
	if len(filteredCafes) > 0 {
		// カフェとベーカリーの両カテゴリを持つスポットを優先
		for _, c := range filteredCafes {
			if helper.HasCategory(c, []string{"ベーカリー"}) {
				cafe = c
				break
			}
		}
		if cafe == nil {
			cafe = helper.FindHighestRated(filteredCafes)
		}
	}

	var combinations [][]*model.POI
	if cafe != nil {
		combination := []*model.POI{bakery, cafe, destinationPOI}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	return combinations, nil
}

// findLocalGourmetWithDestination は地元グルメで目的地を含む組み合わせを見つける
// ロジック: [① カフェ] → [② 食事処]
func (s *GourmetStrategy) findLocalGourmetWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// 目的地周辺のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"カフェ", "商店"})
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// ルート前半にあるカフェで一息つく
	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
	}
	var cafe *model.POI
	if len(cafes) > 0 {
		cafe = helper.FindHighestRated(cafes)
	}

	// 目的地の近くにある評価の高いレストランを選択
	var searchLocation model.LatLng
	if cafe != nil {
		searchLocation = cafe.ToLatLng()
	} else {
		searchLocation = userLocation
	}

	restaurants, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"レストラン", "食品店"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("レストラン検索に失敗: %w", err)
	}
	var restaurant *model.POI
	if len(restaurants) > 0 {
		restaurant = helper.FindHighestRated(restaurants)
	}

	var combinations [][]*model.POI
	if cafe != nil && restaurant != nil {
		combination := []*model.POI{cafe, restaurant, destinationPOI}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	} else if restaurant != nil {
		// カフェが見つからない場合はレストランのみ
		combination := []*model.POI{restaurant, destinationPOI}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	return combinations, nil
}

// findSweetJourneyWithDestination はスイーツ巡りで目的地を含む組み合わせを見つける
// ロジック: [① スイーツ店 A] → [② スイーツ店 B]
func (s *GourmetStrategy) findSweetJourneyWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
	// 目的地周辺のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"カフェ", "商店"})
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// ルート前半にある、評価の高いカフェや商店（スイーツ系）を1つ目に選択
	sweetSpots1, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ", "商店"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("前半のスイーツスポット検索に失敗: %w", err)
	}
	if len(sweetSpots1) == 0 {
		return nil, errors.New("前半のスイーツスポットが見つかりませんでした")
	}
	sweetSpot1 := helper.FindHighestRated(sweetSpots1)

	// ルート後半にある、1軒目とは種類の違うスイーツが楽しめるカフェや商店を2つ目に選択
	sweetSpot1Location := sweetSpot1.ToLatLng()
	sweetSpots2, err := s.poiRepo.FindNearbyByCategories(ctx, sweetSpot1Location, []string{"カフェ", "商店"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("後半のスイーツスポット検索に失敗: %w", err)
	}

	// 1つ目のスイーツスポットを除外
	filteredSweetSpots2 := helper.RemovePOI(sweetSpots2, sweetSpot1)
	var sweetSpot2 *model.POI
	if len(filteredSweetSpots2) > 0 {
		sweetSpot2 = helper.FindHighestRated(filteredSweetSpots2)
	}

	var combinations [][]*model.POI
	if sweetSpot2 != nil {
		combination := []*model.POI{sweetSpot1, sweetSpot2, destinationPOI}
		if s.poiSearchHelper.ValidateCombination(combination, 0, false) {
			combinations = append(combinations, combination)
		}
	}

	return combinations, nil
}

// ExploreNewSpots はグルメテーマで新しいスポットを探索します
func (s *GourmetStrategy) ExploreNewSpots(ctx context.Context, location model.LatLng) ([]*model.POI, error) {
	// グルメテーマのカテゴリで周辺のPOIを検索
	pois, err := s.poiRepo.FindNearbyByCategories(ctx, location, model.GetGourmetCategories(), 1500, 8)
	if err != nil {
		return nil, fmt.Errorf("グルメテーマのPOI検索に失敗: %w", err)
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
