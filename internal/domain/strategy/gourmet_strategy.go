package strategy

import (
    "Team8-App/internal/domain/helper"
    "Team8-App/internal/domain/model"
    "Team8-App/internal/domain/repository"
    "context"
    "errors"
    "fmt"
    "strings"
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

// 段階的検索の共通設定
type SearchConfig struct {
    Categories []string
    Range      int
    Limit      int
}

var (
    // 各シナリオ用の検索設定
    cafeSearchConfigs = []SearchConfig{
        {[]string{"カフェ"}, 1500, 10},
        {[]string{"店舗"}, 3000, 15},
        {[]string{"観光名所"}, 5000, 20},
    }
    
    bakerySearchConfigs = []SearchConfig{
        {[]string{"ベーカリー"}, 1500, 10},
        {[]string{"店舗"}, 3000, 15},
        {[]string{"観光名所"}, 5000, 20},
    }
    
    shopSearchConfigs = []SearchConfig{
        {[]string{"雑貨店"}, 800, 10},
        {[]string{"店舗"}, 1500, 15},
        {[]string{"観光名所"}, 2500, 20},
    }
    
    bookStoreSearchConfigs = []SearchConfig{
        {[]string{"書店", "雑貨店"}, 1500, 10},
        {[]string{"店舗"}, 2500, 15},
        {[]string{"観光名所"}, 4000, 20},
    }
)

// 段階的検索の共通化
func (s *GourmetStrategy) findPOIWithFallback(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig) ([]*model.POI, error) {
    for _, config := range searchConfigs {
        pois, err := s.poiRepo.FindNearbyByCategories(ctx, location, config.Categories, config.Range, config.Limit)
        if err == nil && len(pois) > 0 {
            return s.filterGourmetPOIs(pois), nil
        }
    }
    return nil, nil
}

// findBestPOI は指定された検索設定で最適なPOIを1つ見つける
func (s *GourmetStrategy) findBestPOI(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig) *model.POI {
    pois, err := s.findPOIWithFallback(ctx, location, searchConfigs)
    if err != nil || len(pois) == 0 {
        return nil
    }
    return helper.FindHighestRated(pois)
}

// 目的地ありメソッド用の共通ヘルパー
func (s *GourmetStrategy) findDestinationPOI(ctx context.Context, destination model.LatLng, categories []string) (*model.POI, error) {
    destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, categories)
    if err != nil {
        return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
    }
    return destinationPOI, nil
}

func (s *GourmetStrategy) buildDestinationCombination(pois []*model.POI, destinationPOI *model.POI) ([][]*model.POI, error) {
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

// filterGourmetPOIs はグルメシナリオで除外したいPOIをフィルタリングする
func (s *GourmetStrategy) filterGourmetPOIs(pois []*model.POI) []*model.POI {
    var filtered []*model.POI
    for _, poi := range pois {
        if poi != nil && !s.shouldExcludeFromGourmet(poi.Name) {
            filtered = append(filtered, poi)
        }
    }
    return filtered
}

// shouldExcludeFromGourmet はグルメシナリオで除外すべきPOIかどうかを判定する
func (s *GourmetStrategy) shouldExcludeFromGourmet(poiName string) bool {
    excludePatterns := []string{
        "サモエドカフェ",
        "マクドナルド",
        "マック",
        "McDonald's",
    }

    // 名前に除外パターンが含まれているかをチェック
    for _, pattern := range excludePatterns {
        if strings.Contains(poiName, pattern) {
            return true
        }
    }
    return false
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

//  カフェ巡りシナリオの短縮版
func (s *GourmetStrategy) findCafeHoppingCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 書店/雑貨店を選択
    bookStore := s.findBestPOI(ctx, userLocation, bookStoreSearchConfigs)
    
    // Step 2: メインのカフェを選択
    searchLocation := userLocation
    if bookStore != nil {
        searchLocation = bookStore.ToLatLng()
    }
    cafe := s.findBestPOI(ctx, searchLocation, cafeSearchConfigs)
    if cafe == nil {
        return nil, errors.New("カフェが見つかりませんでした")
    }

    // Step 3: 公園/ベーカリーを選択
    finaleSpot := s.findFinaleSpot(ctx, cafe.ToLatLng(), cafe, bookStore)

    return s.buildCafeHoppingCombination(bookStore, cafe, finaleSpot), nil
}

func (s *GourmetStrategy) findFinaleSpot(ctx context.Context, location model.LatLng, excludePOIs ...*model.POI) *model.POI {
    finaleConfigs := []SearchConfig{
        {[]string{"公園", "ベーカリー"}, 800, 10},
        {[]string{"観光名所", "店舗"}, 1500, 15},
        {[]string{"観光名所"}, 2500, 20},
    }

    spots, err := s.findPOIWithFallback(ctx, location, finaleConfigs)
    if err != nil || len(spots) == 0 {
        return nil
    }
    
    // 除外POIを削除
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

func (s *GourmetStrategy) buildCafeHoppingCombination(bookStore, cafe, finaleSpot *model.POI) [][]*model.POI {
    var combinations [][]*model.POI
    
    if bookStore != nil && finaleSpot != nil {
        combinations = append(combinations, []*model.POI{bookStore, cafe, finaleSpot})
    } else if finaleSpot != nil {
        combinations = append(combinations, []*model.POI{cafe, finaleSpot})
    } else if bookStore != nil {
        combinations = append(combinations, []*model.POI{bookStore, cafe})
    } else {
        combinations = append(combinations, []*model.POI{cafe})
    }
    
    return combinations
}

// ベーカリー巡りシナリオの短縮版
func (s *GourmetStrategy) findBakeryTourCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 評価の高いベーカリーを選択
    bakeryA := s.findBestPOI(ctx, userLocation, bakerySearchConfigs)
    if bakeryA == nil {
        return nil, errors.New("ベーカリーが見つかりませんでした")
    }

    // Step 2: 2つ目のベーカリーを選択
    bakeryB := s.findSecondaryBakery(ctx, bakeryA.ToLatLng(), bakeryA)

    // Step 3: 中間地点の公園を選択
    park := s.findParkBetween(ctx, bakeryA, bakeryB)

    return s.buildBakeryTourCombination(bakeryA, bakeryB, park), nil
}

func (s *GourmetStrategy) findSecondaryBakery(ctx context.Context, location model.LatLng, excludeBakery *model.POI) *model.POI {
    bakeries, err := s.findPOIWithFallback(ctx, location, bakerySearchConfigs)
    if err != nil || len(bakeries) == 0 {
        return nil
    }
    
    filteredBakeries := helper.RemovePOI(bakeries, excludeBakery)
    if len(filteredBakeries) == 0 {
        return nil
    }
    
    return helper.FindHighestRated(filteredBakeries)
}

func (s *GourmetStrategy) findParkBetween(ctx context.Context, bakeryA, bakeryB *model.POI) *model.POI {
    var midLocation model.LatLng
    if bakeryB != nil {
        bakeryBLocation := bakeryB.ToLatLng()
        bakeryALocation := bakeryA.ToLatLng()
        midLat := (bakeryALocation.Lat + bakeryBLocation.Lat) / 2
        midLng := (bakeryALocation.Lng + bakeryBLocation.Lng) / 2
        midLocation = model.LatLng{Lat: midLat, Lng: midLng}
    } else {
        midLocation = bakeryA.ToLatLng()
    }

    parkConfigs := []SearchConfig{
        {[]string{"公園"}, 1000, 10},
        {[]string{"観光名所", "店舗"}, 1500, 15},
        {[]string{"観光名所"}, 2500, 20},
    }

    parks, err := s.findPOIWithFallback(ctx, midLocation, parkConfigs)
    if err != nil || len(parks) == 0 {
        return nil
    }

    // 既存のベーカリーを除外
    filteredParks := helper.RemovePOI(parks, bakeryA)
    if bakeryB != nil {
        filteredParks = helper.RemovePOI(filteredParks, bakeryB)
    }

    if len(filteredParks) == 0 {
        return nil
    }

    helper.SortByDistanceFromLocation(midLocation, filteredParks)
    return filteredParks[0]
}

func (s *GourmetStrategy) buildBakeryTourCombination(bakeryA, bakeryB, park *model.POI) [][]*model.POI {
    var combinations [][]*model.POI

    if bakeryB != nil && park != nil {
        combinations = append(combinations, []*model.POI{bakeryA, park, bakeryB})
    } else if bakeryB != nil {
        combinations = append(combinations, []*model.POI{bakeryA, bakeryB})
    } else if park != nil {
        combinations = append(combinations, []*model.POI{bakeryA, park})
    } else {
        combinations = append(combinations, []*model.POI{bakeryA})
    }

    return combinations
}

// ℹ 地元グルメ巡りシナリオの短縮版
func (s *GourmetStrategy) findLocalGourmetCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 食前のお茶ができるカフェを選択
    cafe := s.findBestPOI(ctx, userLocation, cafeSearchConfigs)

    // Step 2: メインとなる地元の名店を選択
    searchLocation := userLocation
    if cafe != nil {
        searchLocation = cafe.ToLatLng()
    }
    
    restaurantConfigs := []SearchConfig{
        {[]string{"店舗"}, 1000, 10},
        {[]string{"カフェ"}, 1800, 15},
        {[]string{"観光名所"}, 2500, 20},
    }
    
    restaurant := s.findBestPOI(ctx, searchLocation, restaurantConfigs)
    if restaurant == nil {
        return nil, errors.New("地元の食事処が見つかりませんでした")
    }

    // Step 3: 食後の散歩スポットを選択
    afterSpot := s.findAfterDiningSpot(ctx, restaurant.ToLatLng(), cafe, restaurant)

    return s.buildLocalGourmetCombination(cafe, restaurant, afterSpot), nil
}

func (s *GourmetStrategy) findAfterDiningSpot(ctx context.Context, location model.LatLng, excludePOIs ...*model.POI) *model.POI {
    afterSpotConfigs := []SearchConfig{
        {[]string{"公園", "観光名所"}, 800, 10},
        {[]string{"店舗", "雑貨店"}, 1500, 15},
        {[]string{"観光名所"}, 2500, 20},
    }

    spots, err := s.findPOIWithFallback(ctx, location, afterSpotConfigs)
    if err != nil || len(spots) == 0 {
        return nil
    }

    // 除外POIを削除
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

func (s *GourmetStrategy) buildLocalGourmetCombination(cafe, restaurant, afterSpot *model.POI) [][]*model.POI {
    var combinations [][]*model.POI

    if cafe != nil && afterSpot != nil {
        combinations = append(combinations, []*model.POI{cafe, restaurant, afterSpot})
    } else if afterSpot != nil {
        combinations = append(combinations, []*model.POI{restaurant, afterSpot})
    } else if cafe != nil {
        combinations = append(combinations, []*model.POI{cafe, restaurant})
    } else {
        combinations = append(combinations, []*model.POI{restaurant})
    }

    return combinations
}

// スイーツ巡りシナリオの短縮版
func (s *GourmetStrategy) findSweetJourneyCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: ケーキやパフェが評判のカフェを選択
    sweetSpot1 := s.findBestPOI(ctx, userLocation, cafeSearchConfigs)
    if sweetSpot1 == nil {
        return nil, errors.New("スイーツカフェが見つかりませんでした")
    }

    // Step 2: 気分転換の雑貨店を選択
    shop := s.findBestPOI(ctx, sweetSpot1.ToLatLng(), shopSearchConfigs)

    // Step 3: 別のスイーツスポットを選択
    searchLocation := sweetSpot1.ToLatLng()
    if shop != nil {
        searchLocation = shop.ToLatLng()
    }
    
    sweetSpot2 := s.findSecondarySweetSpot(ctx, searchLocation, sweetSpot1, shop)

    return s.buildSweetJourneyCombination(sweetSpot1, shop, sweetSpot2), nil
}

func (s *GourmetStrategy) findSecondarySweetSpot(ctx context.Context, location model.LatLng, excludePOIs ...*model.POI) *model.POI {
    sweetSpotConfigs := []SearchConfig{
        {[]string{"カフェ", "店舗"}, 1000, 10},
        {[]string{"観光名所"}, 1800, 15},
        {[]string{"観光名所"}, 3000, 20},
    }

    spots, err := s.findPOIWithFallback(ctx, location, sweetSpotConfigs)
    if err != nil || len(spots) == 0 {
        return nil
    }

    // 除外POIを削除
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

func (s *GourmetStrategy) buildSweetJourneyCombination(sweetSpot1, shop, sweetSpot2 *model.POI) [][]*model.POI {
    var combinations [][]*model.POI

    if shop != nil && sweetSpot2 != nil {
        combinations = append(combinations, []*model.POI{sweetSpot1, shop, sweetSpot2})
    } else if sweetSpot2 != nil {
        combinations = append(combinations, []*model.POI{sweetSpot1, sweetSpot2})
    } else if shop != nil {
        combinations = append(combinations, []*model.POI{sweetSpot1, shop})
    } else {
        combinations = append(combinations, []*model.POI{sweetSpot1})
    }

    return combinations
}

//  目的地を含むルート組み合わせを見つける
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

// カフェ巡り目的地ありの短縮版
func (s *GourmetStrategy) findCafeHoppingWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    // 目的地POIを特定
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"カフェ", "公園", "観光名所"})
    if err != nil {
        return nil, err
    }

    // カフェ2つを段階的検索で取得
    cafe1 := s.findBestPOI(ctx, userLocation, cafeSearchConfigs)
    if cafe1 == nil {
        return nil, errors.New("前半のカフェが見つかりませんでした")
    }

    cafe2 := s.findSecondaryCafe(ctx, cafe1.ToLatLng(), cafe1)
    
    // 組み合わせ生成
    var pois []*model.POI
    if cafe2 != nil {
        pois = []*model.POI{cafe1, cafe2}
    } else {
        pois = []*model.POI{cafe1}
    }

    return s.buildDestinationCombination(pois, destinationPOI)
}

func (s *GourmetStrategy) findSecondaryCafe(ctx context.Context, location model.LatLng, excludeCafe *model.POI) *model.POI {
    cafes, err := s.findPOIWithFallback(ctx, location, cafeSearchConfigs)
    if err != nil || len(cafes) == 0 {
        return nil
    }
    
    filteredCafes := helper.RemovePOI(cafes, excludeCafe)
    if len(filteredCafes) == 0 {
        return nil
    }
    
    return helper.FindHighestRated(filteredCafes)
}

// ベーカリー巡り目的地ありの短縮版
func (s *GourmetStrategy) findBakeryTourWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    // 目的地POIを特定
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"ベーカリー", "カフェ", "店舗"})
    if err != nil {
        return nil, err
    }

    // ベーカリーとカフェを段階的検索で取得
    bakery := s.findBestPOI(ctx, userLocation, bakerySearchConfigs)
    if bakery == nil {
        return nil, errors.New("ベーカリーが見つかりませんでした")
    }

    cafe := s.findBestPOI(ctx, bakery.ToLatLng(), cafeSearchConfigs)
    
    // 組み合わせ生成
    var pois []*model.POI
    if cafe != nil {
        pois = []*model.POI{bakery, cafe}
    } else {
        pois = []*model.POI{bakery}
    }

    return s.buildDestinationCombination(pois, destinationPOI)
}

// 地元グルメ目的地ありの短縮版
func (s *GourmetStrategy) findLocalGourmetWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    // 目的地POIを特定
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"店舗", "カフェ"})
    if err != nil {
        return nil, err
    }

    // カフェと食事処を段階的検索で取得
    cafe := s.findBestPOI(ctx, userLocation, cafeSearchConfigs)
    
    searchLocation := userLocation
    if cafe != nil {
        searchLocation = cafe.ToLatLng()
    }
    
    restaurantConfigs := []SearchConfig{
        {[]string{"店舗"}, 1000, 10},
        {[]string{"カフェ"}, 1800, 15},
        {[]string{"観光名所"}, 2500, 20},
    }
    
    restaurant := s.findBestPOI(ctx, searchLocation, restaurantConfigs)
    if restaurant == nil {
        return nil, errors.New("地元の食事処が見つかりませんでした")
    }

    // 組み合わせ生成
    var pois []*model.POI
    if cafe != nil {
        pois = []*model.POI{cafe, restaurant}
    } else {
        pois = []*model.POI{restaurant}
    }

    return s.buildDestinationCombination(pois, destinationPOI)
}

// スイーツ巡り目的地ありの短縮版
func (s *GourmetStrategy) findSweetJourneyWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    // 目的地POIを特定
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"カフェ", "店舗"})
    if err != nil {
        return nil, err
    }

    // スイーツスポット2つを段階的検索で取得
    sweetSpot1 := s.findBestPOI(ctx, userLocation, cafeSearchConfigs)
    if sweetSpot1 == nil {
        return nil, errors.New("スイーツスポットが見つかりませんでした")
    }

    sweetSpot2 := s.findSecondarySweetSpot(ctx, sweetSpot1.ToLatLng(), sweetSpot1)
    
    // 組み合わせ生成
    var pois []*model.POI
    if sweetSpot2 != nil {
        pois = []*model.POI{sweetSpot1, sweetSpot2}
    } else {
        pois = []*model.POI{sweetSpot1}
    }

    return s.buildDestinationCombination(pois, destinationPOI)
}

//  ExploreNewSpots はルート再計算用の新しいスポット探索を行う
func (s *GourmetStrategy) ExploreNewSpots(ctx context.Context, searchLocation model.LatLng) ([]*model.POI, error) {
    // グルメテーマに関連するカテゴリで段階的に検索
    gourmetCategories := []string{"カフェ", "ベーカリー", "雑貨店", "書店", "店舗", "公園"}

    // 半径を段階的に拡張して検索
    radiuses := []int{500, 1000, 1500}

    var allSpots []*model.POI
    for _, radius := range radiuses {
        spots, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, gourmetCategories, radius, 20)
        if err != nil {
            continue // エラーがあっても次の半径で試行
        }

        // 重複除去
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

        // 十分な数が見つかったら終了
        if len(allSpots) >= 15 {
            break
        }
    }

    return allSpots, nil
}