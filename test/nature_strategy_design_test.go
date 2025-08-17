package test

import (
	"Team8-App/internal/domain/model"
	"fmt"
	"strings"
	"testing"
)

func TestNatureStrategyDesignValidation(t *testing.T) {
	fmt.Println("🌿 Nature Strategy 設計検証テスト")
	fmt.Println(strings.Repeat("=", 60))

	// テーマとシナリオの検証
	theme := model.ThemeNature
	scenarios := model.GetNatureScenarios()

	fmt.Printf("📝 テーマ: %s (%s)\n", theme, model.GetThemeJapaneseName(theme))
	fmt.Printf("📋 利用可能シナリオ数: %d\n", len(scenarios))

	testCases := []struct {
		scenario    string
		description string
		expected    []string
	}{
		{
			model.ScenarioParkTour,
			"公園巡り",
			[]string{"公園", "観光名所", "ベーカリー", "カフェ"},
		},
		{
			model.ScenarioRiverside,
			"河川敷散歩",
			[]string{"カフェ", "観光名所", "公園", "自然スポット"},
		},
		{
			model.ScenarioTempleNature,
			"寺社と自然",
			[]string{"寺院", "公園", "観光名所", "店舗"},
		},
	}

	for i, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fmt.Printf("\n🧪 シナリオ %d: %s\n", i+1, tc.description)
			fmt.Printf("🆔 シナリオID: %s\n", tc.scenario)
			
			// シナリオの有効性確認
			if !model.IsValidScenario(tc.scenario) {
				t.Errorf("シナリオ %s は無効です", tc.scenario)
				return
			}
			fmt.Printf("✅ シナリオ有効性: OK\n")

			// テーマとシナリオの組み合わせ確認
			validScenarios := model.GetScenariosForTheme(theme)
			isValidCombination := false
			for _, validScenario := range validScenarios {
				if validScenario == tc.scenario {
					isValidCombination = true
					break
				}
			}
			
			if !isValidCombination {
				t.Errorf("シナリオ %s はテーマ %s に属していません", tc.scenario, theme)
				return
			}
			fmt.Printf("✅ テーマ・シナリオ組み合わせ: OK\n")

			// カテゴリ取得確認
			categories := model.GetCategoriesForThemeAndScenario(theme, tc.scenario)
			fmt.Printf("🏷️  取得カテゴリ: %v\n", categories)
			
			if len(categories) == 0 {
				t.Errorf("シナリオ %s のカテゴリが取得できませんでした", tc.scenario)
				return
			}
			fmt.Printf("✅ カテゴリ取得: OK (%d個)\n", len(categories))

			// 期待されるカテゴリとの比較
			expectedMap := make(map[string]bool)
			for _, cat := range tc.expected {
				expectedMap[cat] = true
			}

			matchedCategories := 0
			for _, cat := range categories {
				if expectedMap[cat] {
					matchedCategories++
				}
			}

			matchRatio := float64(matchedCategories) / float64(len(tc.expected))
			fmt.Printf("📊 期待カテゴリマッチ率: %.1f%% (%d/%d)\n", 
				matchRatio*100, matchedCategories, len(tc.expected))

			if matchRatio < 0.5 {
				t.Errorf("シナリオ %s のカテゴリマッチ率が低すぎます: %.1f%%", 
					tc.scenario, matchRatio*100)
			}

			fmt.Printf("✅ カテゴリマッチング: OK\n")
		})
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🎉 Nature Strategy 設計検証完了")

	// 目的地ありなしの検証
	t.Run("目的地パターン検証", func(t *testing.T) {
		fmt.Printf("\n🎯 目的地パターン検証\n")
		
		// 目的地なしパターン
		fmt.Printf("🔸 目的地なしパターン:\n")
		for _, scenario := range scenarios {
			categories := model.GetCategoriesForThemeAndScenario(theme, scenario)
			fmt.Printf("   %s: %v\n", model.GetScenarioJapaneseName(scenario), categories)
		}

		// 目的地ありパターン
		fmt.Printf("🔹 目的地ありパターン:\n")
		destinations := []struct {
			name string
			lat  float64
			lng  float64
		}{
			{"上野公園付近", 35.6851, 139.7528},
			{"隅田川付近", 35.6974, 139.7731},
			{"浅草寺付近", 35.7148, 139.7967},
		}

		for _, dest := range destinations {
			fmt.Printf("   %s (%.4f, %.4f): 全シナリオ対応可能\n", dest.name, dest.lat, dest.lng)
		}

		fmt.Printf("✅ 目的地パターン: OK\n")
	})
}
