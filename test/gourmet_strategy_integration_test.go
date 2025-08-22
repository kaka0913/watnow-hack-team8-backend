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
	// ユーザーが自由に変更可能な目的地座標
	// 以下の座標は例として京都の著名スポットを設定していますが、
	// 任意の緯度経度に変更することができます
	return map[string]model.LatLng{
		model.ScenarioCafeHopping: {
			Lat: 35.0030,  // 例: 祇園周辺 - カフェ文化エリア
			Lng: 135.7728, // ユーザーが任意の座標に変更可能
		},
		model.ScenarioBakeryTour: {
			Lat: 35.0116,  // 例: 京都駅周辺 - 交通の要所
			Lng: 135.7681, // ユーザーが任意の座標に変更可能
		},
		model.ScenarioLocalGourmet: {
			Lat: 35.0073,  // 例: 先斗町周辺 - 地元グルメエリア
			Lng: 135.7704, // ユーザーが任意の座標に変更可能
		},
		model.ScenarioSweetJourney: {
			Lat: 35.0052,  // 例: 木屋町周辺 - スイーツ店多い
			Lng: 135.7692, // ユーザーが任意の座標に変更可能
		},
	}
}

// testScenarioWithDestination 指定したシナリオと目的地でテストを実行する汎用関数
func testScenarioWithDestination(t *testing.T, gourmetStrategy strategy.StrategyInterface, ctx context.Context,
	scenario string, testLocation model.LatLng, destination model.LatLng, scenarioName string, areaName string) {

	fmt.Printf("\n🎯 %sのテスト（%s）\n", scenarioName, areaName)
	fmt.Printf("📍 目的地座標: (%.4f, %.4f)\n", destination.Lat, destination.Lng)

	start := time.Now()
	combinations, err := gourmetStrategy.FindCombinationsWithDestination(ctx, scenario, testLocation, destination)
	duration := time.Since(start)

	if err != nil {
		t.Logf("⚠️  %sでエラー: %v", scenarioName, err)
		return
	}

	fmt.Printf("✅ %s組み合わせ数: %d (実行時間: %v)\n", scenarioName, len(combinations), duration)

	if len(combinations) > 0 {
		combination := combinations[0]
		fmt.Printf("📍 推奨ルート:\n")
		for i, poi := range combination {
			fmt.Printf("  %d. %s (%s) - 評価: %.1f\n", i+1, poi.Name, getPrimaryCategory(poi), poi.Rate)
		}

		if len(combination) < 2 {
			t.Errorf("%sには最低2つのスポットが必要です。実際: %d", scenarioName, len(combination))
		}
	} else {
		t.Logf("⚠️  %sの組み合わせが見つかりませんでした", scenarioName)
	}
}

func TestGourmetStrategyIntegration(t *testing.T) {
	fmt.Println("🍰 グルメストラテジー統合テスト開始")
	fmt.Println("============================================================")

	// テスト用POIリポジトリのセットアップ
	poiRepo, cleanup, err := setupTestPOIRepositoryWithWarmup()
	if err != nil {
		t.Fatalf("❌ テストリポジトリのセットアップに失敗: %v", err)
	}
	defer cleanup()

	// グルメストラテジーの初期化
	gourmetStrategy := strategy.NewGourmetStrategy(poiRepo)

	// テスト用の座標（京都河原町周辺）
	testLocation := model.LatLng{
		Lat: 35.0041,
		Lng: 135.7681,
	}

	// ユーザー定義の目的地を取得
	userDestinations := createUserDefinedDestinations()

	ctx := context.Background()

	t.Run("利用可能シナリオ一覧の取得", func(t *testing.T) {
		scenarios := gourmetStrategy.GetAvailableScenarios()
		fmt.Printf("✅ 利用可能なシナリオ数: %d\n", len(scenarios))

		expectedScenarios := []string{
			model.ScenarioCafeHopping,
			model.ScenarioBakeryTour,
			model.ScenarioLocalGourmet,
			model.ScenarioSweetJourney,
		}

		if len(scenarios) != len(expectedScenarios) {
			t.Errorf("期待されるシナリオ数: %d, 実際: %d", len(expectedScenarios), len(scenarios))
		}

		for _, expected := range expectedScenarios {
			found := false
			for _, actual := range scenarios {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("期待されるシナリオが見つかりません: %s", expected)
			}
		}
	})

	t.Run("カフェ巡りシナリオ", func(t *testing.T) {
		fmt.Println("\n☕ カフェ巡りシナリオのテスト")

		start := time.Now()
		combinations, err := gourmetStrategy.FindCombinations(ctx, model.ScenarioCafeHopping, testLocation)
		duration := time.Since(start)

		if err != nil {
			t.Logf("⚠️  カフェ巡りシナリオでエラー: %v", err)
			return
		}

		fmt.Printf("✅ カフェ巡り組み合わせ数: %d (実行時間: %v)\n", len(combinations), duration)

		if len(combinations) > 0 {
			combination := combinations[0]
			fmt.Printf("📍 推奨ルート:\n")
			for i, poi := range combination {
				categories := "未分類"
				if len(poi.Categories) > 0 {
					categories = poi.Categories[0]
				}
				fmt.Printf("  %d. %s (%s) - 評価: %.1f\n", i+1, poi.Name, categories, poi.Rate)
			}

			// 組み合わせの妥当性をチェック
			if len(combination) < 2 {
				t.Errorf("カフェ巡りには最低2つのスポットが必要です。実際: %d", len(combination))
			}
		} else {
			t.Logf("⚠️  カフェ巡りの組み合わせが見つかりませんでした")
		}
	})

	t.Run("ベーカリー巡りシナリオ", func(t *testing.T) {
		fmt.Println("\n🥖 ベーカリー巡りシナリオのテスト")

		start := time.Now()
		combinations, err := gourmetStrategy.FindCombinations(ctx, model.ScenarioBakeryTour, testLocation)
		duration := time.Since(start)

		if err != nil {
			t.Logf("⚠️  ベーカリー巡りシナリオでエラー: %v", err)
			return
		}

		fmt.Printf("✅ ベーカリー巡り組み合わせ数: %d (実行時間: %v)\n", len(combinations), duration)

		if len(combinations) > 0 {
			combination := combinations[0]
			fmt.Printf("📍 推奨ルート:\n")
			for i, poi := range combination {
				fmt.Printf("  %d. %s (%s) - 評価: %.1f\n", i+1, poi.Name, getPrimaryCategory(poi), poi.Rate)
			}

			// ベーカリーが含まれているかチェック
			hasBakery := false
			for _, poi := range combination {
				if hasCategory(poi, "ベーカリー") {
					hasBakery = true
					break
				}
			}
			if !hasBakery {
				t.Errorf("ベーカリー巡りにはベーカリーが含まれている必要があります")
			}
		} else {
			t.Logf("⚠️  ベーカリー巡りの組み合わせが見つかりませんでした")
		}
	})

	t.Run("地元グルメシナリオ", func(t *testing.T) {
		fmt.Println("\n🍜 地元グルメシナリオのテスト")

		start := time.Now()
		combinations, err := gourmetStrategy.FindCombinations(ctx, model.ScenarioLocalGourmet, testLocation)
		duration := time.Since(start)

		if err != nil {
			t.Logf("⚠️  地元グルメシナリオでエラー: %v", err)
			return
		}

		fmt.Printf("✅ 地元グルメ組み合わせ数: %d (実行時間: %v)\n", len(combinations), duration)

		if len(combinations) > 0 {
			combination := combinations[0]
			fmt.Printf("📍 推奨ルート:\n")
			for i, poi := range combination {
				fmt.Printf("  %d. %s (%s) - 評価: %.1f\n", i+1, poi.Name, getPrimaryCategory(poi), poi.Rate)
			}

			// レストランまたは食品店が含まれているかチェック
			hasRestaurant := false
			for _, poi := range combination {
				if hasCategoryInList(poi, []string{"店舗"}) {
					hasRestaurant = true
					break
				}
			}
			if !hasRestaurant {
				t.Errorf("地元グルメには店舗が含まれている必要があります")
			}
		} else {
			t.Logf("⚠️  地元グルメの組み合わせが見つかりませんでした")
		}
	})

	t.Run("スイーツ巡りシナリオ", func(t *testing.T) {
		fmt.Println("\n🍰 スイーツ巡りシナリオのテスト")

		start := time.Now()
		combinations, err := gourmetStrategy.FindCombinations(ctx, model.ScenarioSweetJourney, testLocation)
		duration := time.Since(start)

		if err != nil {
			t.Logf("⚠️  スイーツ巡りシナリオでエラー: %v", err)
			return
		}

		fmt.Printf("✅ スイーツ巡り組み合わせ数: %d (実行時間: %v)\n", len(combinations), duration)

		if len(combinations) > 0 {
			combination := combinations[0]
			fmt.Printf("📍 推奨ルート:\n")
			for i, poi := range combination {
				fmt.Printf("  %d. %s (%s) - 評価: %.1f\n", i+1, poi.Name, getPrimaryCategory(poi), poi.Rate)
			}

			// カフェまたは商店が含まれているかチェック
			hasSweet := false
			for _, poi := range combination {
				if hasCategoryInList(poi, []string{"カフェ", "店舗"}) {
					hasSweet = true
					break
				}
			}
			if !hasSweet {
				t.Errorf("スイーツ巡りにはカフェまたは店舗が含まれている必要があります")
			}
		} else {
			t.Logf("⚠️  スイーツ巡りの組み合わせが見つかりませんでした")
		}
	})

	t.Run("目的地指定カフェ巡り", func(t *testing.T) {
		testScenarioWithDestination(t, gourmetStrategy, ctx,
			model.ScenarioCafeHopping, testLocation,
			userDestinations[model.ScenarioCafeHopping],
			"目的地指定カフェ巡り", "ユーザー指定エリア")
	})

	t.Run("目的地指定ベーカリー巡り", func(t *testing.T) {
		testScenarioWithDestination(t, gourmetStrategy, ctx,
			model.ScenarioBakeryTour, testLocation,
			userDestinations[model.ScenarioBakeryTour],
			"目的地指定ベーカリー巡り", "ユーザー指定エリア")
	})

	t.Run("目的地指定地元グルメ", func(t *testing.T) {
		testScenarioWithDestination(t, gourmetStrategy, ctx,
			model.ScenarioLocalGourmet, testLocation,
			userDestinations[model.ScenarioLocalGourmet],
			"目的地指定地元グルメ", "ユーザー指定エリア")
	})

	t.Run("目的地指定スイーツ巡り", func(t *testing.T) {
		testScenarioWithDestination(t, gourmetStrategy, ctx,
			model.ScenarioSweetJourney, testLocation,
			userDestinations[model.ScenarioSweetJourney],
			"目的地指定スイーツ巡り", "ユーザー指定エリア")
	})

	t.Run("無効なシナリオのエラーハンドリング", func(t *testing.T) {
		fmt.Println("\n❌ 無効なシナリオのテスト")

		_, err := gourmetStrategy.FindCombinations(ctx, "invalid_scenario", testLocation)
		if err == nil {
			t.Error("無効なシナリオに対してエラーが返されませんでした")
		} else {
			fmt.Printf("✅ 無効なシナリオエラー: %v\n", err)
		}
	})

	t.Run("データベース接続パフォーマンス", func(t *testing.T) {
		fmt.Println("\n⚡ パフォーマンステスト")

		totalStart := time.Now()
		successCount := 0

		scenarios := []string{
			model.ScenarioCafeHopping,
			model.ScenarioBakeryTour,
			model.ScenarioLocalGourmet,
			model.ScenarioSweetJourney,
		}

		for _, scenario := range scenarios {
			start := time.Now()
			_, err := gourmetStrategy.FindCombinations(ctx, scenario, testLocation)
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("⚠️  %s: エラー (%v) - %v\n", scenario, duration, err)
			} else {
				fmt.Printf("✅ %s: 成功 (%v)\n", scenario, duration)
				successCount++
			}
		}

		totalDuration := time.Since(totalStart)
		fmt.Printf("\n📊 パフォーマンス結果:\n")
		fmt.Printf("  - 成功率: %d/%d (%.1f%%)\n", successCount, len(scenarios), float64(successCount)/float64(len(scenarios))*100)
		fmt.Printf("  - 総実行時間: %v\n", totalDuration)
		fmt.Printf("  - 平均実行時間: %v\n", totalDuration/time.Duration(len(scenarios)))

		if totalDuration > 10*time.Second {
			t.Logf("⚠️  総実行時間が長すぎます: %v", totalDuration)
		}
	})

	t.Run("目的地ありパフォーマンステスト", func(t *testing.T) {
		fmt.Println("\n🎯 目的地ありパフォーマンステスト")

		totalStart := time.Now()
		successCount := 0

		scenarios := []string{
			model.ScenarioCafeHopping,
			model.ScenarioBakeryTour,
			model.ScenarioLocalGourmet,
			model.ScenarioSweetJourney,
		}

		for _, scenario := range scenarios {
			destination := userDestinations[scenario]
			start := time.Now()
			_, err := gourmetStrategy.FindCombinationsWithDestination(ctx, scenario, testLocation, destination)
			duration := time.Since(start)

			if err != nil {
				fmt.Printf("⚠️  %s (目的地あり): エラー (%v) - %v\n", scenario, duration, err)
			} else {
				fmt.Printf("✅ %s (目的地あり): 成功 (%v)\n", scenario, duration)
				successCount++
			}
		}

		totalDuration := time.Since(totalStart)
		fmt.Printf("\n📊 目的地ありパフォーマンス結果:\n")
		fmt.Printf("  - 成功率: %d/%d (%.1f%%)\n", successCount, len(scenarios), float64(successCount)/float64(len(scenarios))*100)
		fmt.Printf("  - 総実行時間: %v\n", totalDuration)
		fmt.Printf("  - 平均実行時間: %v\n", totalDuration/time.Duration(len(scenarios)))

		if totalDuration > 10*time.Second {
			t.Logf("⚠️  総実行時間が長すぎます: %v", totalDuration)
		}
	})

	fmt.Println("============================================================")
	fmt.Printf("🎉 グルメストラテジー統合テスト完了\n")
}
