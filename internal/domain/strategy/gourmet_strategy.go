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

// findCafeHoppingCombinations はカフェ巡りシナリオの詳細ロジックを実装
// ロジック: [① 書店/雑貨店] → [② メインのカフェ] → [③ 公園/ベーカリー]
func (s *GourmetStrategy) findCafeHoppingCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 書店/雑貨店を選択（段階的に探索範囲を拡大）
    bookStores, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"書店", "雑貨店"}, 1500, 10)
    if err != nil {
        return nil, fmt.Errorf("書店/雑貨店検索に失敗: %w", err)
    }
    var bookStore *model.POI
    if len(bookStores) > 0 {
        bookStore = helper.FindHighestRated(bookStores)
    } else {
        // 書店/雑貨店が見つからない場合はより広い範囲で店舗カテゴリも含める
        bookStores, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"店舗"}, 2500, 15)
        if err == nil && len(bookStores) > 0 {
            bookStore = bookStores[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            bookStores, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"観光名所"}, 4000, 20)
            if err == nil && len(bookStores) > 0 {
                bookStore = bookStores[0]
            }
        }
    }

    // Step 2: メインのカフェを選択（段階的に探索範囲を拡大）
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

    // グルメシナリオで除外したいPOIをフィルタリング
    cafes = s.filterGourmetPOIs(cafes)

    var mainCafe *model.POI
    if len(cafes) > 0 {
        if bookStore != nil {
            helper.SortByDistance(bookStore, cafes)
            mainCafe = cafes[0]
        } else {
            mainCafe = helper.FindHighestRated(cafes)
        }
    } else {
        // カフェが見つからない場合はより広い範囲で検索、店舗カテゴリも含める
        cafes, err = s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"店舗"}, 2000, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張カフェ検索に失敗: %w", err)
        }
        // グルメシナリオで除外したいPOIをフィルタリング
        cafes = s.filterGourmetPOIs(cafes)
        
        if len(cafes) > 0 {
            mainCafe = cafes[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            cafes, err = s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"観光名所"}, 3500, 20)
            if err != nil {
                return nil, fmt.Errorf("最終カフェ検索に失敗: %w", err)
            }
            if len(cafes) > 0 {
                mainCafe = cafes[0]
            } else {
                return nil, errors.New("カフェが見つかりませんでした")
            }
        }
    }

    // Step 3: 公園/ベーカリーを選択（段階的に探索範囲を拡大）
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
    } else {
        // 公園/ベーカリーが見つからない場合はより広い範囲で観光名所も含める
        finaleSpots, err = s.poiRepo.FindNearbyByCategories(ctx, cafeLocation, []string{"観光名所", "店舗"}, 1500, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張終点スポット検索に失敗: %w", err)
        }
        
        // カフェを除外
        filteredFinaleSpots = helper.RemovePOI(finaleSpots, mainCafe)
        if bookStore != nil {
            filteredFinaleSpots = helper.RemovePOI(filteredFinaleSpots, bookStore)
        }
        
        if len(filteredFinaleSpots) > 0 {
            helper.SortByDistance(mainCafe, filteredFinaleSpots)
            finaleSpot = filteredFinaleSpots[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            finaleSpots, err = s.poiRepo.FindNearbyByCategories(ctx, cafeLocation, []string{"観光名所"}, 2500, 20)
            if err == nil && len(finaleSpots) > 0 {
                // カフェを除外
                filteredFinaleSpots = helper.RemovePOI(finaleSpots, mainCafe)
                if bookStore != nil {
                    filteredFinaleSpots = helper.RemovePOI(filteredFinaleSpots, bookStore)
                }
                if len(filteredFinaleSpots) > 0 {
                    finaleSpot = filteredFinaleSpots[0]
                }
            }
        }
    }

    // 組み合わせを生成（条件を緩和して見つけやすくする）
    var combinations [][]*model.POI

    // まず理想的な組み合わせを試行
    if bookStore != nil && finaleSpot != nil {
        combination := []*model.POI{bookStore, mainCafe, finaleSpot}
        combinations = append(combinations, combination)
    } else if finaleSpot != nil {
        // 書店が見つからない場合はカフェと公園/ベーカリーのみ
        combination := []*model.POI{mainCafe, finaleSpot}
        combinations = append(combinations, combination)
    } else if bookStore != nil {
        // フィナーレスポットが見つからない場合は書店とカフェのみ
        combination := []*model.POI{bookStore, mainCafe}
        combinations = append(combinations, combination)
    }

    // 組み合わせが見つからない場合は、カフェのみでも提案
    if len(combinations) == 0 {
        combination := []*model.POI{mainCafe}
        combinations = append(combinations, combination)
    }

    // nature_strategy.goと同様のエラーハンドリングを追加
    if len(combinations) == 0 {
        return nil, errors.New("カフェ巡りの組み合わせが見つかりませんでした")
    }

    return combinations, nil
}

// findBakeryTourCombinations はベーカリー巡りシナリオの詳細ロジックを実装
// ロジック: [① ベーカリー A] → [② 公園] → [③ ベーカリー B]
func (s *GourmetStrategy) findBakeryTourCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 評価の高いベーカリーを選択（段階的に探索範囲を拡大）
    bakeries, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"ベーカリー"}, 1500, 10)
    if err != nil {
        return nil, fmt.Errorf("ベーカリー検索に失敗: %w", err)
    }
    var bakeryA *model.POI
    if len(bakeries) > 0 {
        bakeryA = helper.FindHighestRated(bakeries)
    } else {
        // ベーカリーが見つからない場合はより広い範囲で店舗カテゴリも含める
        bakeries, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"店舗"}, 2500, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張ベーカリー検索に失敗: %w", err)
        }
        if len(bakeries) > 0 {
            bakeryA = bakeries[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            bakeries, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"観光名所"}, 4000, 20)
            if err != nil {
                return nil, fmt.Errorf("最終ベーカリー検索に失敗: %w", err)
            }
            if len(bakeries) > 0 {
                bakeryA = bakeries[0]
            } else {
                return nil, errors.New("ベーカリーが見つかりませんでした")
            }
        }
    }

    // Step 2: ベーカリーB候補を検索（段階的に探索範囲を拡大）
    bakeryALocation := bakeryA.ToLatLng()
    otherBakeries, err := s.poiRepo.FindNearbyByCategories(ctx, bakeryALocation, []string{"ベーカリー"}, 1200, 10)
    if err != nil {
        return nil, fmt.Errorf("2つ目のベーカリー検索に失敗: %w", err)
    }

    // ベーカリーAを除外
    filteredBakeries := helper.RemovePOI(otherBakeries, bakeryA)
    var bakeryB *model.POI
    if len(filteredBakeries) > 0 {
        bakeryB = helper.FindHighestRated(filteredBakeries)
    } else {
        // 2つ目のベーカリーが見つからない場合はより広い範囲で店舗カテゴリも含める
        otherBakeries, err = s.poiRepo.FindNearbyByCategories(ctx, bakeryALocation, []string{"店舗"}, 2000, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張2つ目ベーカリー検索に失敗: %w", err)
        }
        // ベーカリーAを除外
        filteredBakeries = helper.RemovePOI(otherBakeries, bakeryA)
        if len(filteredBakeries) > 0 {
            bakeryB = filteredBakeries[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            otherBakeries, err = s.poiRepo.FindNearbyByCategories(ctx, bakeryALocation, []string{"観光名所"}, 3500, 20)
            if err == nil && len(otherBakeries) > 0 {
                // ベーカリーAを除外
                filteredBakeries = helper.RemovePOI(otherBakeries, bakeryA)
                if len(filteredBakeries) > 0 {
                    bakeryB = filteredBakeries[0]
                }
            }
        }
    }

    // Step 3: 2つのベーカリーの中間にある公園を選択（段階的に探索範囲を拡大）
    var midLocation model.LatLng
    if bakeryB != nil {
        bakeryBLocation := bakeryB.ToLatLng()
        // 中間地点を計算
        midLat := (bakeryALocation.Lat + bakeryBLocation.Lat) / 2
        midLng := (bakeryALocation.Lng + bakeryBLocation.Lng) / 2
        midLocation = model.LatLng{Lat: midLat, Lng: midLng}
    } else {
        midLocation = bakeryALocation
    }

    parks, err := s.poiRepo.FindNearbyByCategories(ctx, midLocation, []string{"公園"}, 1000, 10)
    if err != nil {
        return nil, fmt.Errorf("公園検索に失敗: %w", err)
    }

    // ベーカリーAとBを除外
    filteredParks := helper.RemovePOI(parks, bakeryA)
    if bakeryB != nil {
        filteredParks = helper.RemovePOI(filteredParks, bakeryB)
    }

    var park *model.POI
    if len(filteredParks) > 0 {
        helper.SortByDistanceFromLocation(midLocation, filteredParks)
        park = filteredParks[0]
    } else {
        // 公園が見つからない場合はより広い範囲で観光名所も含める
        parks, err = s.poiRepo.FindNearbyByCategories(ctx, midLocation, []string{"観光名所", "店舗"}, 1500, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張公園検索に失敗: %w", err)
        }
        
        // ベーカリーAとBを除外
        filteredParks = helper.RemovePOI(parks, bakeryA)
        if bakeryB != nil {
            filteredParks = helper.RemovePOI(filteredParks, bakeryB)
        }
        
        if len(filteredParks) > 0 {
            helper.SortByDistanceFromLocation(midLocation, filteredParks)
            park = filteredParks[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            parks, err = s.poiRepo.FindNearbyByCategories(ctx, midLocation, []string{"観光名所"}, 2500, 20)
            if err == nil && len(parks) > 0 {
                // ベーカリーAとBを除外
                filteredParks = helper.RemovePOI(parks, bakeryA)
                if bakeryB != nil {
                    filteredParks = helper.RemovePOI(filteredParks, bakeryB)
                }
                if len(filteredParks) > 0 {
                    park = filteredParks[0]
                }
            }
        }
    }

    // 組み合わせを生成（条件を緩和して見つけやすくする）
    var combinations [][]*model.POI

    // まず理想的な組み合わせを試行
    if bakeryB != nil && park != nil {
        combination := []*model.POI{bakeryA, park, bakeryB}
        combinations = append(combinations, combination)
    } else if bakeryB != nil {
        // 公園が見つからない場合はベーカリー2つのみ
        combination := []*model.POI{bakeryA, bakeryB}
        combinations = append(combinations, combination)
    } else if park != nil {
        // ベーカリーBが見つからない場合はベーカリーAと公園のみ
        combination := []*model.POI{bakeryA, park}
        combinations = append(combinations, combination)
    }

    // 組み合わせが見つからない場合は、ベーカリーAのみでも提案
    if len(combinations) == 0 {
        combination := []*model.POI{bakeryA}
        combinations = append(combinations, combination)
    }

    // nature_strategy.goと同様のエラーハンドリングを追加
    if len(combinations) == 0 {
        return nil, errors.New("ベーカリー巡りの組み合わせが見つかりませんでした")
    }

    return combinations, nil
}

// findLocalGourmetCombinations は地元グルメシナリオの詳細ロジックを実装
// ロジック: [① カフェ] → [② メインの食事処] → [③ 公園/店舗街]
func (s *GourmetStrategy) findLocalGourmetCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: 食前のお茶ができるカフェを選択（段階的に探索範囲を拡大）
    cafes, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1200, 10)
    if err != nil {
        return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
    }

    // グルメシナリオで除外したいPOIをフィルタリング
    cafes = s.filterGourmetPOIs(cafes)

    var cafe *model.POI
    if len(cafes) > 0 {
        cafe = helper.FindHighestRated(cafes)
    } else {
        // カフェが見つからない場合はより広い範囲で店舗カテゴリも含める
        cafes, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"店舗"}, 2000, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張カフェ検索に失敗: %w", err)
        }
        // グルメシナリオで除外したいPOIをフィルタリング
        cafes = s.filterGourmetPOIs(cafes)
        
        if len(cafes) > 0 {
            cafe = cafes[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            cafes, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"観光名所"}, 3000, 20)
            if err == nil && len(cafes) > 0 {
                cafe = cafes[0]
            }
        }
    }

    // Step 2: メインとなる地元の名店（店舗カテゴリ）を選択（段階的に探索範囲を拡大）
    var searchLocation model.LatLng
    if cafe != nil {
        searchLocation = cafe.ToLatLng()
    } else {
        searchLocation = userLocation
    }

    restaurants, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"店舗"}, 1000, 10)
    if err != nil {
        return nil, fmt.Errorf("店舗検索に失敗: %w", err)
    }
    var mainRestaurant *model.POI
    if len(restaurants) > 0 {
        // 評価の高い店舗を選択
        mainRestaurant = helper.FindHighestRated(restaurants)
    } else {
        // 店舗が見つからない場合はより広い範囲でカフェも含める
        restaurants, err = s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"カフェ"}, 1800, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張店舗検索に失敗: %w", err)
        }
        if len(restaurants) > 0 {
            mainRestaurant = restaurants[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            restaurants, err = s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"観光名所"}, 2500, 20)
            if err != nil {
                return nil, fmt.Errorf("最終店舗検索に失敗: %w", err)
            }
            if len(restaurants) > 0 {
                mainRestaurant = restaurants[0]
            } else {
                return nil, errors.New("地元の食事処が見つかりませんでした")
            }
        }
    }

    // Step 3: 食後の散歩に最適な公園や店舗街を選択（段階的に探索範囲を拡大）
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
    } else {
        // 食後スポットが見つからない場合はより広い範囲で店舗カテゴリも含める
        afterSpots, err = s.poiRepo.FindNearbyByCategories(ctx, restaurantLocation, []string{"店舗", "雑貨店"}, 1500, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張食後スポット検索に失敗: %w", err)
        }
        
        // レストランとカフェを除外
        filteredAfterSpots = helper.RemovePOI(afterSpots, mainRestaurant)
        if cafe != nil {
            filteredAfterSpots = helper.RemovePOI(filteredAfterSpots, cafe)
        }
        
        if len(filteredAfterSpots) > 0 {
            helper.SortByDistance(mainRestaurant, filteredAfterSpots)
            afterSpot = filteredAfterSpots[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            afterSpots, err = s.poiRepo.FindNearbyByCategories(ctx, restaurantLocation, []string{"観光名所"}, 2500, 20)
            if err == nil && len(afterSpots) > 0 {
                // レストランとカフェを除外
                filteredAfterSpots = helper.RemovePOI(afterSpots, mainRestaurant)
                if cafe != nil {
                    filteredAfterSpots = helper.RemovePOI(filteredAfterSpots, cafe)
                }
                if len(filteredAfterSpots) > 0 {
                    afterSpot = filteredAfterSpots[0]
                }
            }
        }
    }

    // 組み合わせを生成（条件を緩和して見つけやすくする）
    var combinations [][]*model.POI

    // まず理想的な組み合わせを試行
    if cafe != nil && afterSpot != nil {
        combination := []*model.POI{cafe, mainRestaurant, afterSpot}
        combinations = append(combinations, combination)
    } else if afterSpot != nil {
        // カフェが見つからない場合はレストランと食後スポットのみ
        combination := []*model.POI{mainRestaurant, afterSpot}
        combinations = append(combinations, combination)
    } else if cafe != nil {
        // 食後スポットが見つからない場合はカフェとレストランのみ
        combination := []*model.POI{cafe, mainRestaurant}
        combinations = append(combinations, combination)
    }

    // 組み合わせが見つからない場合は、レストランのみでも提案
    if len(combinations) == 0 {
        combination := []*model.POI{mainRestaurant}
        combinations = append(combinations, combination)
    }

    // nature_strategy.goと同様のエラーハンドリングを追加
    if len(combinations) == 0 {
        return nil, errors.New("地元グルメの組み合わせが見つかりませんでした")
    }

    return combinations, nil
}

// findSweetJourneyCombinations はスイーツ巡りシナリオの詳細ロジックを実装
// ロジック: [① カフェ(ケーキ等)] → [② 雑貨店] → [③ カフェ(ジェラート等)]
func (s *GourmetStrategy) findSweetJourneyCombinations(ctx context.Context, userLocation model.LatLng) ([][]*model.POI, error) {
    // Step 1: ケーキやパフェが評判のカフェを選択（段階的に探索範囲を拡大）
    cafes, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1500, 10)
    if err != nil {
        return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
    }

    // グルメシナリオで除外したいPOIをフィルタリング
    cafes = s.filterGourmetPOIs(cafes)

    var cafeA *model.POI
    if len(cafes) > 0 {
        cafeA = helper.FindHighestRated(cafes)
    } else {
        // カフェが見つからない場合はより広い範囲で店舗カテゴリも含める
        cafes, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"店舗"}, 2500, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張カフェ検索に失敗: %w", err)
        }
        // グルメシナリオで除外したいPOIをフィルタリング
        cafes = s.filterGourmetPOIs(cafes)
        
        if len(cafes) > 0 {
            cafeA = cafes[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            cafes, err = s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"観光名所"}, 4000, 20)
            if err != nil {
                return nil, fmt.Errorf("最終カフェ検索に失敗: %w", err)
            }
            if len(cafes) > 0 {
                cafeA = cafes[0]
            } else {
                return nil, errors.New("スイーツカフェが見つかりませんでした")
            }
        }
    }

    // Step 2: 気分転換に立ち寄れる可愛い雑貨店を選択（段階的に探索範囲を拡大）
    cafeALocation := cafeA.ToLatLng()
    shops, err := s.poiRepo.FindNearbyByCategories(ctx, cafeALocation, []string{"雑貨店"}, 800, 10)
    if err != nil {
        return nil, fmt.Errorf("雑貨店検索に失敗: %w", err)
    }

    var shop *model.POI
    if len(shops) > 0 {
        helper.SortByDistance(cafeA, shops)
        shop = shops[0]
    } else {
        // 雑貨店が見つからない場合はより広い範囲で店舗カテゴリも含める
        shops, err = s.poiRepo.FindNearbyByCategories(ctx, cafeALocation, []string{"店舗"}, 1500, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張雑貨店検索に失敗: %w", err)
        }
        // カフェAを除外
        filteredShops := helper.RemovePOI(shops, cafeA)
        if len(filteredShops) > 0 {
            helper.SortByDistance(cafeA, filteredShops)
            shop = filteredShops[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            shops, err = s.poiRepo.FindNearbyByCategories(ctx, cafeALocation, []string{"観光名所"}, 2500, 20)
            if err == nil && len(shops) > 0 {
                // カフェAを除外
                filteredShops = helper.RemovePOI(shops, cafeA)
                if len(filteredShops) > 0 {
                    shop = filteredShops[0]
                }
            }
        }
    }

    // Step 3: ジェラート等が楽しめる別のカフェや店舗を選択（段階的に探索範囲を拡大）
    var searchLocation model.LatLng
    if shop != nil {
        searchLocation = shop.ToLatLng()
    } else {
        searchLocation = cafeALocation
    }

    sweetSpots, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"カフェ", "店舗"}, 1000, 10)
    if err != nil {
        return nil, fmt.Errorf("スイーツスポット検索に失敗: %w", err)
    }

    // グルメシナリオで除外したいPOIをフィルタリング
    sweetSpots = s.filterGourmetPOIs(sweetSpots)

    // 1つ目のカフェを除外
    filteredSweetSpots := helper.RemovePOI(sweetSpots, cafeA)
    if shop != nil {
        filteredSweetSpots = helper.RemovePOI(filteredSweetSpots, shop)
    }

    var sweetSpot *model.POI
    if len(filteredSweetSpots) > 0 {
        helper.SortByDistanceFromLocation(searchLocation, filteredSweetSpots)
        sweetSpot = filteredSweetSpots[0]
    } else {
        // スイーツスポットが見つからない場合はより広い範囲で観光名所も含める
        sweetSpots, err = s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"観光名所"}, 1800, 15)
        if err != nil {
            return nil, fmt.Errorf("拡張スイーツスポット検索に失敗: %w", err)
        }
        
        // 1つ目のカフェと雑貨店を除外
        filteredSweetSpots = helper.RemovePOI(sweetSpots, cafeA)
        if shop != nil {
            filteredSweetSpots = helper.RemovePOI(filteredSweetSpots, shop)
        }
        
        if len(filteredSweetSpots) > 0 {
            helper.SortByDistanceFromLocation(searchLocation, filteredSweetSpots)
            sweetSpot = filteredSweetSpots[0]
        } else {
            // 最後の手段：観光名所カテゴリで検索
            sweetSpots, err = s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"観光名所"}, 3000, 20)
            if err == nil && len(sweetSpots) > 0 {
                // 1つ目のカフェと雑貨店を除外
                filteredSweetSpots = helper.RemovePOI(sweetSpots, cafeA)
                if shop != nil {
                    filteredSweetSpots = helper.RemovePOI(filteredSweetSpots, shop)
                }
                if len(filteredSweetSpots) > 0 {
                    sweetSpot = filteredSweetSpots[0]
                }
            }
        }
    }

    // 組み合わせを生成（条件を緩和して見つけやすくする）
    var combinations [][]*model.POI

    // まず理想的な組み合わせを試行
    if shop != nil && sweetSpot != nil {
        combination := []*model.POI{cafeA, shop, sweetSpot}
        combinations = append(combinations, combination)
    } else if sweetSpot != nil {
        // 雑貨店が見つからない場合はカフェ2つのみ
        combination := []*model.POI{cafeA, sweetSpot}
        combinations = append(combinations, combination)
    } else if shop != nil {
        // スイーツスポットが見つからない場合はカフェと雑貨店のみ
        combination := []*model.POI{cafeA, shop}
        combinations = append(combinations, combination)
    }

    // 組み合わせが見つからない場合は、カフェのみでも提案
    if len(combinations) == 0 {
        combination := []*model.POI{cafeA}
        combinations = append(combinations, combination)
    }

    // nature_strategy.goと同様のエラーハンドリングを追加
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
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"カフェ", "公園", "観光名所"})
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// ルート前半にある評価の高いカフェを選択
	cafes1, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("前半のカフェ検索に失敗: %w", err)
	}

	// グルメシナリオで除外したいPOIをフィルタリング
	cafes1 = s.filterGourmetPOIs(cafes1)

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

	// グルメシナリオで除外したいPOIをフィルタリング
	cafes2 = s.filterGourmetPOIs(cafes2)

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
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"店舗", "カフェ", "ベーカリー"})
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
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"カフェ", "店舗", "観光名所"})
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// ルート前半にあるカフェで一息つく
	cafes, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("カフェ検索に失敗: %w", err)
	}

	// グルメシナリオで除外したいPOIをフィルタリング
	cafes = s.filterGourmetPOIs(cafes)

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

	restaurants, err := s.poiRepo.FindNearbyByCategories(ctx, searchLocation, []string{"店舗"}, 1000, 10)
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
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination, []string{"カフェ", "店舗", "雑貨店"})
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// ルート前半にある、評価の高いカフェや店舗（スイーツ系）を1つ目に選択
	sweetSpots1, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"カフェ", "店舗"}, 1500, 10)
	if err != nil {
		return nil, fmt.Errorf("前半のスイーツスポット検索に失敗: %w", err)
	}

	// グルメシナリオで除外したいPOIをフィルタリング
	sweetSpots1 = s.filterGourmetPOIs(sweetSpots1)
	if len(sweetSpots1) == 0 {
		return nil, errors.New("前半のスイーツスポットが見つかりませんでした")
	}
	sweetSpot1 := helper.FindHighestRated(sweetSpots1)

	// ルート後半にある、1軒目とは種類の違うスイーツが楽しめるカフェや店舗を2つ目に選択
	sweetSpot1Location := sweetSpot1.ToLatLng()
	sweetSpots2, err := s.poiRepo.FindNearbyByCategories(ctx, sweetSpot1Location, []string{"カフェ", "店舗"}, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("後半のスイーツスポット検索に失敗: %w", err)
	}

	// グルメシナリオで除外したいPOIをフィルタリング
	sweetSpots2 = s.filterGourmetPOIs(sweetSpots2)

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

// ExploreNewSpots はルート再計算用の新しいスポット探索を行う
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
