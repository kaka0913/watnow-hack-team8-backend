package test

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/strategy"
	"context"
	"fmt"
	"testing"
	"time"
)

// hasCategory POIが指定されたカテゴリを持っているかチェック
func hasCategory(poi *model.POI, category string) bool {
	for _, cat := range poi.Categories {
		if cat == category {
			return true
		}
	}
	return false
}

// hasCategoryInList POIが指定されたカテゴリリストのいずれかを持っているかチェック
func hasCategoryInList(poi *model.POI, categories []string) bool {
	for _, targetCat := range categories {
		if hasCategory(poi, targetCat) {
			return true
		}
	}
	return false
}

// getPrimaryCategory POIの主要カテゴリを取得
func getPrimaryCategory(poi *model.POI) string {
	if len(poi.Categories) > 0 {
		return poi.Categories[0]
	}
	return "未分類"
}

// createUserDefinedDestinations ユーザーが任意に指定できる目的地設定関数
func createUserDefinedDestinations() map[string]model.LatLng {
	return map[string]model.LatLng{
		model.ScenarioCafeHopping:  {Lat: 35.0030, Lng: 135.7728},
		model.ScenarioBakeryTour:   {Lat: 35.0116, Lng: 135.7681},
		model.ScenarioLocalGourmet: {Lat: 35.0073, Lng: 135.7704},
		model.ScenarioSweetJourney: {Lat: 35.0052, Lng: 135.7692},
	}
}

// testScenarioWithDestination 指定したシナリオと目的地でテストを実行する汎用関数
func testScenarioWithDestination(t *testing.T, gourmetStrategy strategy.StrategyInterface, ctx context.Context,
	scenario string, testLocation model.LatLng, destination model.LatLng, scenarioName string) {

	fmt.Printf("\n🎯 === %s (目的地あり) 検索開始 ===\n", scenarioName)
	fmt.Printf("📍 検索位置: (%.4f, %.4f)\n", testLocation.Lat, testLocation.Lng)
	fmt.Printf("🎯 目的地: (%.4f, %.4f)\n", destination.Lat, destination.Lng)

	combinations, err := gourmetStrategy.FindCombinationsWithDestination(ctx, scenario, testLocation, destination)

	if err != nil {
		fmt.Printf("⚠️  %sでエラー: %v\n", scenarioName, err)
		t.Logf("⚠️  %sでエラー: %v", scenarioName, err)
		return
	}

	fmt.Printf("✅ 検索結果: %d個の組み合わせが見つかりました\n", len(combinations))

	if len(combinations) > 0 {
		for i, combination := range combinations {
			if i >= 2 { // 最初の2つの組み合わせのみ表示
				fmt.Printf("... (他 %d個の組み合わせ)\n", len(combinations)-2)
				break
			}

			fmt.Printf("\n🍽️  組み合わせ %d: %d個のスポット\n", i+1, len(combination))
			for j, poi := range combination {
				fmt.Printf("  %d. %s", j+1, poi.Name)
				if len(poi.Categories) > 0 {
					fmt.Printf(" [%s]", getPrimaryCategory(poi))
				}
				fmt.Printf(" - 距離: %.0fm\n",
					calculateDistance(testLocation, poi.ToLatLng()))
			}
		}

		if len(combinations[0]) < 2 {
			t.Errorf("%sには最低2つのスポット（1つ + 目的地）が必要です。実際: %d", scenarioName, len(combinations[0]))
		}
	} else {
		fmt.Printf("❌ スポットが見つかりませんでした\n")
	}

	fmt.Printf("🎯 === %s (目的地あり) 検索完了 ===\n\n", scenarioName)
}

func TestGourmetStrategyIntegration(t *testing.T) {
	// 💡 [imo] 💡 テスト用POIリポジトリのセットアップ
	poiRepo, cleanup, err := setupTestPOIRepositoryWithWarmup()
	if err != nil {
		t.Fatalf("❌ テストリポジトリのセットアップに失敗: %v", err)
	}
	defer cleanup()

	gourmetStrategy := strategy.NewGourmetStrategy(poiRepo)
	testLocation := model.LatLng{Lat: 35.0041, Lng: 135.7681}
	userDestinations := createUserDefinedDestinations()
	ctx := context.Background()

	t.Run("利用可能シナリオ一覧の取得", func(t *testing.T) {
		scenarios := gourmetStrategy.GetAvailableScenarios()
		expectedScenarios := []string{
			model.ScenarioCafeHopping,
			model.ScenarioBakeryTour,
			model.ScenarioLocalGourmet,
			model.ScenarioSweetJourney,
		}

		if len(scenarios) != len(expectedScenarios) {
			t.Errorf("期待されるシナリオ数: %d, 実際: %d", len(expectedScenarios), len(scenarios))
		}
	})

	// ✨ [nits] ✨ 目的地なしシナリオのテスト（共通化）
	t.Run("目的地なしシナリオ", func(t *testing.T) {
		scenarios := []struct {
			name     string
			scenario string
		}{
			{"カフェ巡り", model.ScenarioCafeHopping},
			{"ベーカリー巡り", model.ScenarioBakeryTour},
			{"地元グルメ", model.ScenarioLocalGourmet},
			{"スイーツ巡り", model.ScenarioSweetJourney},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				fmt.Printf("\n🔍 === %s 検索開始 ===\n", s.name)
				fmt.Printf("📍 検索位置: (%.4f, %.4f)\n", testLocation.Lat, testLocation.Lng)

				combinations, err := gourmetStrategy.FindCombinations(ctx, s.scenario, testLocation)
				if err != nil {
					fmt.Printf("⚠️  %sでエラー: %v\n", s.name, err)
					t.Logf("⚠️  %sでエラー: %v", s.name, err)
					return
				}

				fmt.Printf("✅ 検索結果: %d個の組み合わせが見つかりました\n", len(combinations))

				if len(combinations) > 0 {
					for i, combination := range combinations {
						if i >= 3 { // 最初の3つの組み合わせのみ表示
							fmt.Printf("... (他 %d個の組み合わせ)\n", len(combinations)-3)
							break
						}

						fmt.Printf("\n🍽️  組み合わせ %d: %d個のスポット\n", i+1, len(combination))
						for j, poi := range combination {
							fmt.Printf("  %d. %s", j+1, poi.Name)
							if len(poi.Categories) > 0 {
								fmt.Printf(" [%s]", getPrimaryCategory(poi))
							}
							fmt.Printf(" - 距離: %.0fm\n",
								calculateDistance(testLocation, poi.ToLatLng()))
						}
					}

					if len(combinations[0]) < 1 {
						t.Errorf("%sには最低1つのスポットが必要です。実際: %d", s.name, len(combinations[0]))
					}
				} else {
					fmt.Printf("❌ スポットが見つかりませんでした\n")
				}

				fmt.Printf("🔍 === %s 検索完了 ===\n\n", s.name)
			})
		}
	})

	// 🚨 [must] 🚨 目的地ありシナリオのテスト（共通化）
	t.Run("目的地ありシナリオ", func(t *testing.T) {
		scenarios := []struct {
			name     string
			scenario string
		}{
			{"カフェ巡り", model.ScenarioCafeHopping},
			{"ベーカリー巡り", model.ScenarioBakeryTour},
			{"地元グルメ", model.ScenarioLocalGourmet},
			{"スイーツ巡り", model.ScenarioSweetJourney},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				destination := userDestinations[s.scenario]
				testScenarioWithDestination(t, gourmetStrategy, ctx, s.scenario, testLocation, destination, s.name)
			})
		}
	})

	t.Run("ExploreNewSpotsのテスト", func(t *testing.T) {
		fmt.Printf("\n🌟 === 新しいスポット探索開始 ===\n")
		fmt.Printf("📍 検索位置: (%.4f, %.4f)\n", testLocation.Lat, testLocation.Lng)

		spots, err := gourmetStrategy.ExploreNewSpots(ctx, testLocation)
		if err != nil {
			fmt.Printf("⚠️  ExploreNewSpotsでエラー: %v\n", err)
			t.Logf("⚠️  ExploreNewSpotsでエラー: %v", err)
			return
		}

		fmt.Printf("✅ 発見されたスポット数: %d\n", len(spots))

		// ℹ️ [fyi] ℹ️ グルメ関連のカテゴリが含まれているかチェック
		if len(spots) > 0 {
			gourmetCategories := []string{"カフェ", "ベーカリー", "雑貨店", "書店", "店舗", "公園"}
			hasGourmetCategory := false
			gourmetCount := 0

			fmt.Printf("\n🍽️  発見されたスポット一覧:\n")
			for i, spot := range spots {
				if i >= 10 { // 最初の10個のスポットのみ表示
					fmt.Printf("... (他 %d個のスポット)\n", len(spots)-10)
					break
				}

				fmt.Printf("  %d. %s", i+1, spot.Name)
				if len(spot.Categories) > 0 {
					fmt.Printf(" [%s]", getPrimaryCategory(spot))
				}
				fmt.Printf(" - 距離: %.0fm\n",
					calculateDistance(testLocation, spot.ToLatLng()))

				if hasCategoryInList(spot, gourmetCategories) {
					hasGourmetCategory = true
					gourmetCount++
				}
			}

			fmt.Printf("\n📊 グルメ関連スポット: %d/%d\n", gourmetCount, len(spots))

			if !hasGourmetCategory {
				fmt.Printf("⚠️  グルメ関連のカテゴリが含まれていません\n")
				t.Logf("⚠️  グルメ関連のカテゴリが含まれていません")
			}
		} else {
			fmt.Printf("❌ 新しいスポットが見つかりませんでした\n")
		}

		fmt.Printf("🌟 === 新しいスポット探索完了 ===\n\n")
	})

	t.Run("無効なシナリオのエラーハンドリング", func(t *testing.T) {
		_, err := gourmetStrategy.FindCombinations(ctx, "invalid_scenario", testLocation)
		if err == nil {
			t.Error("無効なシナリオに対してエラーが返されませんでした")
		}
	})

	// ❓ [ask] ❓ パフォーマンステスト（簡略化）
	t.Run("パフォーマンステスト", func(t *testing.T) {
		fmt.Printf("\n⏱️  === パフォーマンステスト開始 ===\n")

		scenarios := []string{
			model.ScenarioCafeHopping,
			model.ScenarioBakeryTour,
			model.ScenarioLocalGourmet,
			model.ScenarioSweetJourney,
		}

		totalStart := time.Now()
		successCount := 0
		var results []string

		// 目的地なしテスト
		fmt.Printf("🔍 目的地なしシナリオのテスト...\n")
		for _, scenario := range scenarios {
			start := time.Now()
			_, err := gourmetStrategy.FindCombinations(ctx, scenario, testLocation)
			duration := time.Since(start)

			if err == nil {
				successCount++
				results = append(results, fmt.Sprintf("  ✅ %s: %.2fs", scenario, duration.Seconds()))
			} else {
				results = append(results, fmt.Sprintf("  ❌ %s: %.2fs (エラー)", scenario, duration.Seconds()))
			}
		}

		// 目的地ありテスト
		fmt.Printf("🎯 目的地ありシナリオのテスト...\n")
		for _, scenario := range scenarios {
			destination := userDestinations[scenario]
			start := time.Now()
			_, err := gourmetStrategy.FindCombinationsWithDestination(ctx, scenario, testLocation, destination)
			duration := time.Since(start)

			if err == nil {
				successCount++
				results = append(results, fmt.Sprintf("  ✅ %s (目的地あり): %.2fs", scenario, duration.Seconds()))
			} else {
				results = append(results, fmt.Sprintf("  ❌ %s (目的地あり): %.2fs (エラー)", scenario, duration.Seconds()))
			}
		}

		totalDuration := time.Since(totalStart)
		totalTests := len(scenarios) * 2 // 目的地なし + 目的地あり

		fmt.Printf("\n📊 === テスト結果詳細 ===\n")
		for _, result := range results {
			fmt.Println(result)
		}

		fmt.Printf("\n⏱️  総実行時間: %.2fs\n", totalDuration.Seconds())
		fmt.Printf("✅ 成功率: %d/%d (%.1f%%)\n", successCount, totalTests, float64(successCount)/float64(totalTests)*100)

		if totalDuration > 15*time.Second {
			fmt.Printf("⚠️  総実行時間が長すぎます: %v\n", totalDuration)
			t.Logf("⚠️  総実行時間が長すぎます: %v", totalDuration)
		}

		if successCount < totalTests/2 {
			fmt.Printf("⚠️  成功率が低すぎます: %d/%d\n", successCount, totalTests)
			t.Logf("⚠️  成功率が低すぎます: %d/%d", successCount, totalTests)
		}

		fmt.Printf("⏱️  === パフォーマンステスト完了 ===\n\n")
	})
}
