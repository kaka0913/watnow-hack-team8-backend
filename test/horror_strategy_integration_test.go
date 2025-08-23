package test

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/strategy"
	"context"
	"fmt"
	"testing"
	"time"
)

// hasHorrorCategory POIが指定されたホラーカテゴリを持っているかチェック
func hasHorrorCategory(poi *model.POI, category string) bool {
	for _, cat := range poi.Categories {
		if cat == category {
			return true
		}
	}
	return false
}

// hasHorrorCategoryInList POIが指定されたホラーカテゴリリストのいずれかを持っているかチェック
func hasHorrorCategoryInList(poi *model.POI, categories []string) bool {
	for _, targetCat := range categories {
		if hasHorrorCategory(poi, targetCat) {
			return true
		}
	}
	return false
}

// getHorrorPrimaryCategory POIの主要カテゴリを取得
func getHorrorPrimaryCategory(poi *model.POI) string {
	if len(poi.Categories) > 0 {
		return poi.Categories[0]
	}
	return "未分類"
}

// createHorrorDestinations ホラーテーマの目的地設定関数
func createHorrorDestinations() map[string]model.LatLng {
	return map[string]model.LatLng{
		model.ScenarioGhostTour:    {Lat: 35.0120, Lng: 135.7690}, // 心霊スポット巡り
		model.ScenarioHauntedRuins: {Lat: 35.0140, Lng: 135.7710}, // 廃墟探索
		model.ScenarioCursedNature: {Lat: 35.0100, Lng: 135.7730}, // 呪いの自然
		model.ScenarioCemeteryWalk: {Lat: 35.0080, Lng: 135.7750}, // 墓地・慰霊散歩
	}
}

// testHorrorScenarioWithDestination ホラーシナリオと目的地でテストを実行する汎用関数
func testHorrorScenarioWithDestination(t *testing.T, horrorStrategy strategy.StrategyInterface, ctx context.Context,
	scenario string, testLocation model.LatLng, destination model.LatLng, scenarioName string) {

	combinations, err := horrorStrategy.FindCombinationsWithDestination(ctx, scenario, testLocation, destination)

	if err != nil {
		t.Logf("⚠️  %sでエラー: %v", scenarioName, err)
		return
	}

	if len(combinations) > 0 && len(combinations[0]) < 2 {
		t.Errorf("%sには最低2つのスポット（1つ + 目的地）が必要です。実際: %d", scenarioName, len(combinations[0]))
	}
}

func TestHorrorStrategyIntegration(t *testing.T) {
	// 💡 [imo] 💡 テスト用POIリポジトリのセットアップ（API使用しない軽量版）
	poiRepo, cleanup, err := setupTestPOIRepositoryWithWarmup()
	if err != nil {
		t.Skipf("⚠️  テストリポジトリのセットアップに失敗: %v (API料金回避のためスキップ)", err)
		return
	}
	defer cleanup()

	horrorStrategy := strategy.NewHorrorStrategy(poiRepo)
	testLocation := model.LatLng{Lat: 35.0041, Lng: 135.7681}
	userDestinations := createHorrorDestinations()
	ctx := context.Background()

	t.Run("利用可能シナリオ一覧の取得", func(t *testing.T) {
		scenarios := horrorStrategy.GetAvailableScenarios()
		expectedScenarios := []string{
			model.ScenarioGhostTour,
			model.ScenarioHauntedRuins,
			model.ScenarioCursedNature,
			model.ScenarioCemeteryWalk,
		}

		if len(scenarios) != len(expectedScenarios) {
			t.Errorf("期待されるシナリオ数: %d, 実際: %d", len(expectedScenarios), len(scenarios))
		}

		fmt.Printf("✅ 利用可能なホラーシナリオ: %v\n", scenarios)
	})

	// ✨ [nits] ✨ 目的地なしシナリオのテスト（ホラーテーマ、軽量版）
	t.Run("目的地なしシナリオ", func(t *testing.T) {
		scenarios := []struct {
			name     string
			scenario string
		}{
			{"心霊スポット巡り", model.ScenarioGhostTour},
			{"廃墟探索", model.ScenarioHauntedRuins},
			{"呪いの自然", model.ScenarioCursedNature},
			{"墓地・慰霊散歩", model.ScenarioCemeteryWalk},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				fmt.Printf("\n🔍 === %s 検索開始 ===\n", s.name)
				fmt.Printf("📍 検索位置: (%.4f, %.4f)\n", testLocation.Lat, testLocation.Lng)

				combinations, err := horrorStrategy.FindCombinations(ctx, s.scenario, testLocation)
				if err != nil {
					t.Logf("⚠️  %sでエラー: %v", s.name, err)
					return
				}

				fmt.Printf("✅ 検索結果: %d個の組み合わせが見つかりました\n\n", len(combinations))

				for i, combination := range combinations {
					if i >= 3 { // 最初の3個の組み合わせのみ表示
						fmt.Printf("... 他 %d個の組み合わせ\n", len(combinations)-3)
						break
					}
					fmt.Printf("👻 組み合わせ %d: %d個のスポット\n", i+1, len(combination))
					for j, poi := range combination {
						poiLocation := poi.ToLatLng()
						distance := calculateDistance(testLocation, poiLocation)
						category := getHorrorPrimaryCategory(poi)
						fmt.Printf("  %d. %s [%s] - 距離: %.0fm\n", j+1, poi.Name, category, distance*1000)
					}
					fmt.Printf("\n")
				}
				fmt.Printf("🔍 === %s 検索完了 ===\n\n", s.name)

				if len(combinations) > 0 && len(combinations[0]) < 1 {
					t.Errorf("%sには最低1つのスポットが必要です。実際: %d", s.name, len(combinations[0]))
				}
			})
		}
	})

	// 🚨 [must] 🚨 目的地ありシナリオのテスト（ホラーテーマ、軽量版）
	t.Run("目的地ありシナリオ", func(t *testing.T) {
		scenarios := []struct {
			name     string
			scenario string
		}{
			{"心霊スポット巡り", model.ScenarioGhostTour},
			{"廃墟探索", model.ScenarioHauntedRuins},
			{"呪いの自然", model.ScenarioCursedNature},
			{"墓地・慰霊散歩", model.ScenarioCemeteryWalk},
		}

		for _, s := range scenarios {
			t.Run(s.name, func(t *testing.T) {
				destination := userDestinations[s.scenario]
				fmt.Printf("\n🎯 === %s (目的地あり) 検索開始 ===\n", s.name)
				fmt.Printf("📍 検索位置: (%.4f, %.4f)\n", testLocation.Lat, testLocation.Lng)
				fmt.Printf("🎯 目的地: (%.4f, %.4f)\n", destination.Lat, destination.Lng)

				combinations, err := horrorStrategy.FindCombinationsWithDestination(ctx, s.scenario, testLocation, destination)
				if err != nil {
					t.Logf("⚠️  %sでエラー: %v", s.name, err)
					return
				}

				fmt.Printf("✅ 検索結果: %d個の組み合わせが見つかりました\n\n", len(combinations))

				for i, combination := range combinations {
					if i >= 2 { // 最初の2個の組み合わせのみ表示
						fmt.Printf("... 他 %d個の組み合わせ\n", len(combinations)-2)
						break
					}
					fmt.Printf("👻 組み合わせ %d: %d個のスポット\n", i+1, len(combination))
					for j, poi := range combination {
						poiLocation := poi.ToLatLng()
						distance := calculateDistance(testLocation, poiLocation)
						category := getHorrorPrimaryCategory(poi)
						fmt.Printf("  %d. %s [%s] - 距離: %.0fm\n", j+1, poi.Name, category, distance*1000)
					}
					fmt.Printf("\n")
				}
				fmt.Printf("🎯 === %s (目的地あり) 検索完了 ===\n\n", s.name)

				testHorrorScenarioWithDestination(t, horrorStrategy, ctx, s.scenario, testLocation, destination, s.name)
			})
		}
	})

	t.Run("ExploreNewSpotsのテスト", func(t *testing.T) {
		fmt.Printf("\n🌟 === 新しいホラースポット探索開始 ===\n")
		fmt.Printf("📍 検索位置: (%.4f, %.4f)\n", testLocation.Lat, testLocation.Lng)

		spots, err := horrorStrategy.ExploreNewSpots(ctx, testLocation)
		if err != nil {
			t.Logf("⚠️  ExploreNewSpotsでエラー: %v", err)
			return
		}

		fmt.Printf("✅ 発見されたスポット数: %d\n", len(spots))

		if len(spots) > 0 {
			fmt.Printf("\n📍 発見されたスポット一覧:\n")
			for i, spot := range spots {
				if i >= 10 { // 最初の10個のみ表示
					fmt.Printf("  ... 他 %d 個のスポット\n", len(spots)-10)
					break
				}
				spotLocation := spot.ToLatLng()
				distance := calculateDistance(testLocation, spotLocation)
				category := getHorrorPrimaryCategory(spot)
				fmt.Printf("  %d. %s [%s] - 距離: %.0fm\n", i+1, spot.Name, category, distance*1000)
			}
		} else {
			fmt.Printf("❌ 新しいスポットが見つかりませんでした\n")
		}
		fmt.Printf("🌟 === 新しいホラースポット探索完了 ===\n\n")

		// ℹ️ [fyi] ℹ️ ホラー関連のカテゴリが含まれているかチェック
		if len(spots) > 0 {
			horrorCategories := []string{"horror_spot", "place_of_worship", "natural_feature", "establishment", "tourist_attraction"}
			hasHorrorCategory := false
			for _, spot := range spots {
				if hasHorrorCategoryInList(spot, horrorCategories) {
					hasHorrorCategory = true
					break
				}
			}
			if !hasHorrorCategory {
				t.Logf("⚠️  ホラー関連のカテゴリが含まれていません")
			}
		}
	})

	t.Run("無効なシナリオのエラーハンドリング", func(t *testing.T) {
		_, err := horrorStrategy.FindCombinations(ctx, "invalid_horror_scenario", testLocation)
		if err == nil {
			t.Error("無効なシナリオに対してエラーが返されませんでした")
		}
	})

	// ❓ [ask] ❓ パフォーマンステスト（ホラーテーマ、軽量版）
	t.Run("パフォーマンステスト", func(t *testing.T) {
		fmt.Printf("\n⏱️  === ホラーテーマパフォーマンステスト開始 ===\n")
		scenarios := []string{
			model.ScenarioGhostTour,
			model.ScenarioHauntedRuins,
			model.ScenarioCursedNature,
			model.ScenarioCemeteryWalk,
		}

		totalStart := time.Now()
		successCount := 0
		var testResults []string

		fmt.Printf("🔍 目的地なしシナリオのテスト...\n")
		// 目的地なしテスト
		for _, scenario := range scenarios {
			start := time.Now()
			_, err := horrorStrategy.FindCombinations(ctx, scenario, testLocation)
			duration := time.Since(start)
			if err == nil {
				successCount++
				testResults = append(testResults, fmt.Sprintf("  ✅ %s: %.2fs", scenario, duration.Seconds()))
			} else {
				testResults = append(testResults, fmt.Sprintf("  ❌ %s: %.2fs (エラー)", scenario, duration.Seconds()))
			}
		}

		fmt.Printf("🎯 目的地ありシナリオのテスト...\n")
		// 目的地ありテスト
		for _, scenario := range scenarios {
			destination := userDestinations[scenario]
			start := time.Now()
			_, err := horrorStrategy.FindCombinationsWithDestination(ctx, scenario, testLocation, destination)
			duration := time.Since(start)
			if err == nil {
				successCount++
				testResults = append(testResults, fmt.Sprintf("  ✅ %s (目的地あり): %.2fs", scenario, duration.Seconds()))
			} else {
				testResults = append(testResults, fmt.Sprintf("  ❌ %s (目的地あり): %.2fs (エラー)", scenario, duration.Seconds()))
			}
		}

		totalDuration := time.Since(totalStart)
		totalTests := len(scenarios) * 2 // 目的地なし + 目的地あり

		fmt.Printf("\n📊 === テスト結果詳細 ===\n")
		for _, result := range testResults {
			fmt.Println(result)
		}
		fmt.Printf("\n⏱️  総実行時間: %.2fs\n", totalDuration.Seconds())
		fmt.Printf("✅ 成功率: %d/%d (%.1f%%)\n", successCount, totalTests, float64(successCount)/float64(totalTests)*100)
		fmt.Printf("⏱️  === ホラーテーマパフォーマンステスト完了 ===\n\n")

		if totalDuration > 15*time.Second {
			t.Logf("⚠️  総実行時間が長すぎます: %v", totalDuration)
		}

		if successCount < totalTests/2 {
			t.Logf("⚠️  成功率が低すぎます: %d/%d", successCount, totalTests)
		}
	})
}
