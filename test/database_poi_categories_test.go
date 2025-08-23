package test

import (
	"Team8-App/internal/domain/model"
	"context"
	"fmt"
	"testing"
)

func TestDatabasePOICategories(t *testing.T) {
	fmt.Println("🔍 データベース内POIカテゴリ調査")
	fmt.Println("============================================================")

	// テスト用POIリポジトリのセットアップ
	poiRepo, cleanup, err := setupTestPOIRepositoryWithWarmup()
	if err != nil {
		t.Fatalf("❌ テストリポジトリのセットアップに失敗: %v", err)
	}
	defer cleanup()

	// テスト用の座標（京都河原町周辺）
	testLocation := model.LatLng{
		Lat: 35.0041,
		Lng: 135.7681,
	}

	ctx := context.Background()

	t.Run("周辺POIの総数とカテゴリ分布", func(t *testing.T) {
		// 半径1500m以内のPOIをすべて取得
		allPOIs, err := poiRepo.FindNearbyByCategories(ctx, testLocation, []string{}, 1500, 100)
		if err != nil {
			t.Fatalf("❌ POI検索エラー: %v", err)
		}

		fmt.Printf("✅ 総POI数: %d\n", len(allPOIs))

		// カテゴリごとの集計
		categoryCount := make(map[string]int)
		for _, poi := range allPOIs {
			for _, category := range poi.Categories {
				categoryCount[category]++
			}
		}

		fmt.Println("\n📊 カテゴリ別POI数:")
		for category, count := range categoryCount {
			fmt.Printf("  - %s: %d件\n", category, count)
		}
	})

	t.Run("グルメ関連カテゴリの詳細調査", func(t *testing.T) {
		gourmetCategories := []string{"カフェ", "ベーカリー", "レストラン", "食品店", "商店", "雑貨店"}

		for _, category := range gourmetCategories {
			pois, err := poiRepo.FindNearbyByCategories(ctx, testLocation, []string{category}, 1500, 20)
			if err != nil {
				t.Logf("⚠️  %sの検索でエラー: %v", category, err)
				continue
			}

			fmt.Printf("\n🍽️ %s (件数: %d)\n", category, len(pois))
			if len(pois) > 0 {
				for i, poi := range pois {
					if i >= 5 { // 最初の5件のみ表示
						fmt.Printf("  ... 他%d件\n", len(pois)-5)
						break
					}
					fmt.Printf("  %d. %s (評価: %.1f) - カテゴリ: %v\n", i+1, poi.Name, poi.Rate, poi.Categories)
				}
			} else {
				fmt.Printf("  該当するPOIが見つかりませんでした\n")
			}
		}
	})

	t.Run("公園カテゴリの調査", func(t *testing.T) {
		parks, err := poiRepo.FindNearbyByCategories(ctx, testLocation, []string{"公園"}, 1500, 20)
		if err != nil {
			t.Fatalf("❌ 公園検索エラー: %v", err)
		}

		fmt.Printf("\n🌳 公園 (件数: %d)\n", len(parks))
		for i, poi := range parks {
			if i >= 10 { // 最初の10件のみ表示
				break
			}
			fmt.Printf("  %d. %s (評価: %.1f)\n", i+1, poi.Name, poi.Rate)
		}
	})

	fmt.Println("\n============================================================")
	fmt.Printf("🎉 データベース内POIカテゴリ調査完了\n")
}
