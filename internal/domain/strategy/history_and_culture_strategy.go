package strategy

import (
    "Team8-App/internal/domain/helper"
    "Team8-App/internal/domain/model"
    "Team8-App/internal/domain/repository"
    "context"
    "errors"
    "fmt"
)

// HistoryAndCultureStrategy は歴史・文化を巡るルートを提案する
type HistoryAndCultureStrategy struct {
    poiRepo         repository.POIsRepository
    poiSearchHelper *helper.POISearchHelper
}

func NewHistoryAndCultureStrategy(repo repository.POIsRepository) StrategyInterface {
    return &HistoryAndCultureStrategy{
        poiRepo:         repo,
        poiSearchHelper: helper.NewPOISearchHelper(repo),
    }
}

// 🚨 [must] 🚨 SearchConfig構造体の定義を削除（gourmet_strategy.goで定義済み）

var (
    // 🚨 [must] 🚨 各シナリオ用の段階的検索設定
    templeSearchConfigs = []SearchConfig{
        {[]string{"寺院", "神社"}, 1500, 10},
        {[]string{"観光名所"}, 3000, 15},
        {[]string{"店舗"}, 5000, 20},
    }
    
    museumSearchConfigs = []SearchConfig{
        {[]string{"博物館", "美術館・ギャラリー"}, 1500, 10},
        {[]string{"観光名所"}, 3000, 15},
        {[]string{"店舗"}, 5000, 20},
    }
    
    historicBuildingSearchConfigs = []SearchConfig{
        {[]string{"観光名所"}, 1000, 10},
        {[]string{"店舗"}, 2500, 15},
        {[]string{"寺院", "神社"}, 4000, 20},
    }
    
    bookstoreSearchConfigs = []SearchConfig{
        {[]string{"書店"}, 800, 10},
        {[]string{"店舗"}, 1500, 15},
        {[]string{"観光名所"}, 2500, 20},
    }
    
    // ✨ [nits] ✨ セカンダリ検索用の段階的設定
    historyCafeSearchConfigs = []SearchConfig{
        {[]string{"カフェ"}, 1200, 10},
        {[]string{"店舗"}, 2000, 15},
        {[]string{"観光名所"}, 3000, 20},
    }
    
    historyShopSearchConfigs = []SearchConfig{
        {[]string{"店舗", "観光名所"}, 1000, 10},
        {[]string{"店舗"}, 1800, 15},
        {[]string{"観光名所"}, 2500, 20},
    }
    
    parkSearchConfigs = []SearchConfig{
        {[]string{"公園"}, 1000, 10},
        {[]string{"観光名所"}, 1800, 15},
        {[]string{"店舗"}, 2500, 20},
    }
)

// 🚨 [must] 🚨 段階的検索の共通化メソッド
func (s *HistoryAndCultureStrategy) findPOIWithFallback(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig) ([]*model.POI, error) {
    for _, config := range searchConfigs {
        pois, err := s.poiRepo.FindNearbyByCategories(ctx, location, config.Categories, config.Range, config.Limit)
        if err == nil && len(pois) > 0 {
            return pois, nil
        }
    }
    return nil, nil
}

// findBestPOI は指定された検索設定で最適なPOIを1つ見つける
func (s *HistoryAndCultureStrategy) findBestPOI(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig) *model.POI {
    pois, err := s.findPOIWithFallback(ctx, location, searchConfigs)
    if err != nil || len(pois) == 0 {
        return nil
    }
    return helper.FindHighestRated(pois)
}

// ℹ️ [fyi] ℹ️ 目的地なしメソッド用の共通ヘルパー（gourmet_strategyと同じパターン）
func (s *HistoryAndCultureStrategy) buildCombination(spots ...*model.POI) [][]*model.POI {
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

// 💡 [imo] 💡 距離優先検索の統一メソッド
func (s *HistoryAndCultureStrategy) findNearestPOI(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig, excludePOIs ...*model.POI) *model.POI {
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

// 評価優先検索の統一メソッド
func (s *HistoryAndCultureStrategy) findRatedPOI(ctx context.Context, location model.LatLng, searchConfigs []SearchConfig, excludePOIs ...*model.POI) *model.POI {
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

    return helper.FindHighestRated(spots)
}

// 🚨 [must] 🚨 目的地ありメソッド用の共通ヘルパー
func (s *HistoryAndCultureStrategy) findDestinationPOI(ctx context.Context, destination model.LatLng, categories []string) (*model.POI, error) {
    destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, categories)
    if err != nil {
        return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
    }
    return destinationPOI, nil
}

func (s *HistoryAndCultureStrategy) buildDestinationCombination(pois []*model.POI, destinationPOI *model.POI) ([][]*model.POI, error) {
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

// GetAvailableScenarios はHistoryAndCultureテーマで利用可能なシナリオ一覧を取得する
func (s *HistoryAndCultureStrategy) GetAvailableScenarios() []string {
    return model.GetHistoryAndCultureScenarios()
}

// 💡 [imo] 💡 目的地なしの統一ハンドラー（段階的検索で3つのスポットを巡る）
func (s *HistoryAndCultureStrategy) FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error) {
    switch scenario {
    case model.ScenarioTempleShrine:
        return s.findTempleShrineCombinations(ctx, userLocation)
    case model.ScenarioMuseumTour:
        return s.findMuseumTourCombinations(ctx, userLocation)
    case model.ScenarioOldTown:
        return s.findOldTownCombinations(ctx, userLocation)
    case model.ScenarioCulturalWalk:
        return s.findCulturalWalkCombinations(ctx, userLocation)
    default:
        return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
    }
}

// 🚨 [must] 🚨 寺社仏閣巡りシナリオ（段階的検索で3スポット確保）
func (s *HistoryAndCultureStrategy) findTempleShrineCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: メインの寺社（段階的検索: 1500m→3000m→5000m）
    mainTemple := s.findBestPOI(ctx, userLocation, templeSearchConfigs)
    if mainTemple == nil {
        return nil, errors.New("メインの寺社が見つかりませんでした")
    }

    // Step 2: 参道の食事処/カフェ（段階的検索: 1200m→2000m→3000m）
    restaurant := s.findNearestPOI(ctx, mainTemple.ToLatLng(), historyCafeSearchConfigs, mainTemple)

    // Step 3: 小規模な寺社（段階的検索: 1500m→3000m→5000m）
    searchLocation := mainTemple.ToLatLng()
    if restaurant != nil {
        searchLocation = restaurant.ToLatLng()
    }
    smallTemple := s.findRatedPOI(ctx, searchLocation, templeSearchConfigs, mainTemple, restaurant)

    return s.buildCombination(mainTemple, restaurant, smallTemple), nil
}

// ✨ [nits] ✨ 博物館巡りシナリオ（段階的検索で3スポット確保）
func (s *HistoryAndCultureStrategy) findMuseumTourCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: メインの博物館/美術館（段階的検索: 1500m→3000m→5000m）
    mainMuseum := s.findBestPOI(ctx, userLocation, museumSearchConfigs)
    if mainMuseum == nil {
        return nil, errors.New("博物館/美術館が見つかりませんでした")
    }

    // Step 2: カフェ（段階的検索: 1200m→2000m→3000m）
    cafe := s.findNearestPOI(ctx, mainMuseum.ToLatLng(), historyCafeSearchConfigs, mainMuseum)

    // Step 3: 歴史的建造物（段階的検索: 1000m→2500m→4000m）
    searchLocation := mainMuseum.ToLatLng()
    if cafe != nil {
        searchLocation = cafe.ToLatLng()
    }
    historicBuilding := s.findRatedPOI(ctx, searchLocation, historicBuildingSearchConfigs, mainMuseum, cafe)

    return s.buildCombination(mainMuseum, cafe, historicBuilding), nil
}

// ℹ️ [fyi] ℹ️ 古い街並み散策シナリオ（段階的検索で3スポット確保）
func (s *HistoryAndCultureStrategy) findOldTownCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 歴史的建造物A（段階的検索: 1000m→2500m→4000m）
    buildingA := s.findBestPOI(ctx, userLocation, historicBuildingSearchConfigs)
    if buildingA == nil {
        return nil, errors.New("歴史的建造物が見つかりませんでした")
    }

    // Step 2: 歴史的な商店（段階的検索: 1000m→1800m→2500m）
    historicShop := s.findRatedPOI(ctx, buildingA.ToLatLng(), historyShopSearchConfigs, buildingA)

    // Step 3: 別の歴史的建造物B（段階的検索: 1000m→2500m→4000m）
    searchLocation := buildingA.ToLatLng()
    if historicShop != nil {
        searchLocation = historicShop.ToLatLng()
    }
    buildingB := s.findRatedPOI(ctx, searchLocation, historicBuildingSearchConfigs, buildingA, historicShop)

    return s.buildCombination(buildingA, historicShop, buildingB), nil
}

// ❓ [ask] ❓ 文化的散歩シナリオ（段階的検索で3スポット確保）
func (s *HistoryAndCultureStrategy) findCulturalWalkCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 博物館/美術館（段階的検索: 1500m→3000m→5000m）
    museum := s.findBestPOI(ctx, userLocation, museumSearchConfigs)
    if museum == nil {
        return nil, errors.New("博物館/美術館が見つかりませんでした")
    }

    // Step 2: 公園（段階的検索: 1000m→1800m→2500m）
    park := s.findNearestPOI(ctx, museum.ToLatLng(), parkSearchConfigs, museum)

    // Step 3: 書店/図書館（段階的検索: 800m→1500m→2500m）
    searchLocation := museum.ToLatLng()
    if park != nil {
        searchLocation = park.ToLatLng()
    }
    bookstore := s.findRatedPOI(ctx, searchLocation, bookstoreSearchConfigs, museum, park)

    return s.buildCombination(museum, park, bookstore), nil
}

// 🚨 [must] 🚨 目的地を含むルート組み合わせを見つける（段階的検索で2つのスポット確保）
func (s *HistoryAndCultureStrategy) FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    switch scenario {
    case model.ScenarioTempleShrine:
        return s.findTempleShrineCombinationsWithDestination(ctx, userLocation, destination)
    case model.ScenarioMuseumTour:
        return s.findMuseumTourCombinationsWithDestination(ctx, userLocation, destination)
    case model.ScenarioOldTown:
        return s.findOldTownCombinationsWithDestination(ctx, userLocation, destination)
    case model.ScenarioCulturalWalk:
        return s.findCulturalWalkCombinationsWithDestination(ctx, userLocation, destination)
    default:
        return nil, fmt.Errorf("不明なシナリオです: %s", scenario)
    }
}

// 🚨 [must] 🚨 寺社仏閣巡り目的地あり（段階的検索で2つのスポット確保）
func (s *HistoryAndCultureStrategy) findTempleShrineCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"寺院", "神社", "観光名所"})
    if err != nil {
        return nil, err
    }

    // Step 1: 前半の神社（段階的検索: 1500m→3000m→5000m）
    shrine := s.findBestPOI(ctx, userLocation, templeSearchConfigs)
    if shrine == nil {
        return nil, errors.New("前半の神社が見つかりませんでした")
    }

    // Step 2: 後半の寺院（段階的検索: 1500m→3000m→5000m）
    temple := s.findRatedPOI(ctx, shrine.ToLatLng(), templeSearchConfigs, shrine)
    if temple == nil {
        return nil, errors.New("後半の寺院が見つかりませんでした")
    }

    pois := []*model.POI{shrine, temple}
    return s.buildDestinationCombination(pois, destinationPOI)
}

// ✨ [nits] ✨ 博物館巡り目的地あり（段階的検索で2つのスポット確保）
func (s *HistoryAndCultureStrategy) findMuseumTourCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"博物館", "美術館・ギャラリー", "観光名所"})
    if err != nil {
        return nil, err
    }

    // Step 1: 主要な博物館（段階的検索: 1500m→3000m→5000m）
    museum := s.findBestPOI(ctx, userLocation, museumSearchConfigs)
    if museum == nil {
        return nil, errors.New("博物館/美術館が見つかりませんでした")
    }

    // Step 2: 関連スポット（書店等）（段階的検索: 800m→1500m→2500m）
    relatedSpot := s.findRatedPOI(ctx, museum.ToLatLng(), bookstoreSearchConfigs, museum)
    if relatedSpot == nil {
        return nil, errors.New("関連スポットが見つかりませんでした")
    }

    pois := []*model.POI{museum, relatedSpot}
    return s.buildDestinationCombination(pois, destinationPOI)
}

// ℹ️ [fyi] ℹ️ 古い街並み散策目的地あり（段階的検索で2つのスポット確保）
func (s *HistoryAndCultureStrategy) findOldTownCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"観光名所", "店舗"})
    if err != nil {
        return nil, err
    }

    // Step 1: 街並みの入口（段階的検索: 1000m→2500m→4000m）
    entrance := s.findBestPOI(ctx, userLocation, historicBuildingSearchConfigs)
    if entrance == nil {
        return nil, errors.New("街並みの入口が見つかりませんでした")
    }

    // Step 2: 街並みの出口（段階的検索: 1000m→2500m→4000m）
    exit := s.findRatedPOI(ctx, entrance.ToLatLng(), historicBuildingSearchConfigs, entrance)
    if exit == nil {
        return nil, errors.New("街並みの出口が見つかりませんでした")
    }

    pois := []*model.POI{entrance, exit}
    return s.buildDestinationCombination(pois, destinationPOI)
}

// ❓ [ask] ❓ 文化的散歩目的地あり（段階的検索で2つのスポット確保）
func (s *HistoryAndCultureStrategy) findCulturalWalkCombinationsWithDestination(ctx context.Context, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error) {
    destinationPOI, err := s.findDestinationPOI(ctx, destination, []string{"書店", "博物館", "美術館・ギャラリー"})
    if err != nil {
        return nil, err
    }

    // Step 1: 博物館（段階的検索: 1500m→3000m→5000m）
    museum := s.findBestPOI(ctx, userLocation, museumSearchConfigs)
    if museum == nil {
        return nil, errors.New("博物館/美術館が見つかりませんでした")
    }

    // Step 2: 書店（段階的検索: 800m→1500m→2500m）
    bookstore := s.findRatedPOI(ctx, museum.ToLatLng(), bookstoreSearchConfigs, museum)
    if bookstore == nil {
        return nil, errors.New("書店が見つかりませんでした")
    }

    pois := []*model.POI{museum, bookstore}
    return s.buildDestinationCombination(pois, destinationPOI)
}

// ℹ️ [fyi] ℹ️ ExploreNewSpots はルート再計算用の新しいスポット探索を行う
func (s *HistoryAndCultureStrategy) ExploreNewSpots(ctx context.Context, searchLocation model.LatLng) ([]*model.POI, error) {
    historyCultureCategories := []string{"寺院", "神社", "博物館", "美術館・ギャラリー", "書店", "観光名所", "公園"}

    radiuses := []int{500, 1000, 1500}

    var allSpots []*model.POI
    for _, radius := range radiuses {
        spots, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, historyCultureCategories, radius, 20)
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