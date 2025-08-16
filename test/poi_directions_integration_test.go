package test

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/infrastructure/maps"
	"Team8-App/internal/repository"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// TestNatureStrategyWithExistingPOIs は既存のPOIカテゴリを使用して動作確認する
func TestNatureStrategyWithExistingPOIs(t *testing.T) {
	// 環境変数の読み込み
	if err := godotenv.Load("../.env"); err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")

	if supabaseURL == "" || supabaseAnonKey == "" || googleMapsAPIKey == "" {
		t.Skip("環境変数が設定されていません。統合テストをスキップします。")
	}

	// Supabaseクライアントの初期化
	supabaseClient, err := database.NewSupabaseClient()
	if err != nil {
		t.Fatalf("Supabaseクライアント初期化失敗: %v", err)
	}

	// ヘルスチェック
	if err := supabaseClient.HealthCheck(); err != nil {
		t.Fatalf("Supabaseヘルスチェック失敗: %v", err)
	}

	// POIリポジトリとGoogleDirectionsProviderの初期化
	poiRepo := repository.NewSupabasePOIsRepository(supabaseClient)
	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)

	// テスト用の位置設定（大阪梅田エリア - モックデータがある場所）
	userLocation := model.LatLng{
		Lat: 34.7024,  // JR大阪駅付近
		Lng: 135.4959, 
	}

	ctx := context.Background()

	// 1. まず、存在するPOIカテゴリを確認
	t.Logf("=== 現在のPOIデータベースのカテゴリ確認 ===")
	testCategories := []string{"shopping", "restaurant", "landmark", "architecture", "shrine"}
	
	for _, category := range testCategories {
		pois, err := poiRepo.FindNearbyByCategories(ctx, userLocation, []string{category}, 2000, 5)
		if err != nil {
			t.Logf("❌ カテゴリ '%s' の検索エラー: %v", category, err)
		} else {
			t.Logf("✅ カテゴリ '%s': %d個のPOI", category, len(pois))
			for i, poi := range pois {
				if i >= 3 { // 最初の3つのみ表示
					break
				}
				categories := "なし"
				if len(poi.Categories) > 0 {
					categories = poi.Categories[0]
				}
				t.Logf("  - %s (%s, 評価: %.1f)", poi.Name, categories, poi.Rate)
			}
		}
	}

	// 2. NatureStrategyを既存データに合わせてテスト
	natureStrategy := strategy.NewNatureStrategy(poiRepo)
	
	t.Logf("\n=== 実際のPOI組み合わせとDirections APIテスト ===")
	
	// モックデータに存在するカテゴリでテスト組み合わせを作成
	testPOIs, err := poiRepo.FindNearbyByCategories(ctx, userLocation, []string{"shopping", "landmark", "shrine"}, 2000, 10)
	if err != nil {
		t.Fatalf("テスト用POI取得失敗: %v", err)
	}

	if len(testPOIs) >= 3 {
		// 3つのPOIで組み合わせを作成
		testCombination := []*model.POI{testPOIs[0], testPOIs[1], testPOIs[2]}
		
		t.Logf("🎯 テスト用ルート組み合わせ:")
		for i, poi := range testCombination {
			categories := "なし"
			if len(poi.Categories) > 0 {
				categories = poi.Categories[0]
			}
			t.Logf("  %d. %s (%s, 評価: %.1f)", i+1, poi.Name, categories, poi.Rate)
		}

		// Directions APIでルートを取得
		t.Logf("\n🗺️  Directions APIでルートを取得中...")
		directions, err := directionsProvider.GetWalkingRouteFromPOIs(ctx, testCombination[0], testCombination[1:]...)
		if err != nil {
			t.Logf("❌ Directions API エラー: %v", err)
		} else {
			t.Logf("✅ ルート取得成功")
			t.Logf("⏱️  総時間: %v", directions.TotalDuration)
			if directions.Polyline != "" {
				polylineLength := len(directions.Polyline)
				if polylineLength > 100 {
					t.Logf("🗺️  ポリライン: %s... (%d文字)", directions.Polyline[:100], polylineLength)
				} else {
					t.Logf("🗺️  ポリライン: %s", directions.Polyline)
				}
			}
		}
	} else {
		t.Logf("⚠️  POIが3つ未満のため、Directions APIテストをスキップ")
	}

	// 3. NatureStrategyの各シナリオ確認（期待通りエラーが出ることを確認）
	scenarios := natureStrategy.GetAvailableScenarios()
	t.Logf("\n=== NatureStrategyシナリオ確認 ===")
	
	for _, scenario := range scenarios {
		t.Logf("📝 シナリオ: %s", scenario)
		_, err := natureStrategy.FindCombinations(ctx, scenario, userLocation)
		if err != nil {
			t.Logf("  ❌ 予想通りのエラー: %v", err)
		} else {
			t.Logf("  ✅ 組み合わせが見つかりました（予想外）")
		}
	}
}
