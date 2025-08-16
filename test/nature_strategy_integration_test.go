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
	"time"

	"github.com/joho/godotenv"
)

// TestNatureStrategyIntegration は自然テーマの全シナリオでPOI選択とDirections APIの動作を確認する
func TestNatureStrategyIntegration(t *testing.T) {
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

	// POIリポジトリとNatureStrategyの初期化
	poiRepo := repository.NewSupabasePOIsRepository(supabaseClient)
	natureStrategy := strategy.NewNatureStrategy(poiRepo)

	// GoogleDirectionsProviderの初期化
	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)

	// テスト用の位置設定
	// 現在地: 京都河原町の寺町あたり（京都市中京区寺町通四条上る中之町）
	userLocation := model.LatLng{
		Lat: 35.0041,  // 河原町寺町付近
		Lng: 135.7699, 
	}

	// 目的地: 京都駅
	destination := model.LatLng{
		Lat: 34.9858,  // 京都駅
		Lng: 135.7581,
	}

	ctx := context.Background()

	// 利用可能なシナリオを取得
	scenarios := natureStrategy.GetAvailableScenarios()
	t.Logf("=== 自然テーマの利用可能シナリオ ===")
	for i, scenario := range scenarios {
		t.Logf("%d. %s", i+1, scenario)
	}

	// 各シナリオをテスト
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("Scenario_%s", scenario), func(t *testing.T) {
			testScenario(t, natureStrategy, directionsProvider, ctx, scenario, userLocation, destination)
		})
	}
}

// testScenario は個別のシナリオをテストする
func testScenario(t *testing.T, strategy strategy.StrategyInterface, directionsProvider *maps.GoogleDirectionsProvider, ctx context.Context, scenario string, userLocation, destination model.LatLng) {
	t.Logf("\n🌿 === %s シナリオのテスト ===", scenario)

	// 1. 目的地なしのルート組み合わせをテスト
	t.Logf("\n📍 目的地なしのルート組み合わせ:")
	combinations, err := strategy.FindCombinations(ctx, scenario, userLocation)
	if err != nil {
		t.Logf("❌ エラー: %v", err)
	} else {
		t.Logf("✅ %d個の組み合わせが見つかりました", len(combinations))
		for i, combination := range combinations {
			t.Logf("  組み合わせ %d:", i+1)
			for j, poi := range combination {
				categories := "なし"
				if len(poi.Categories) > 0 {
					categories = poi.Categories[0]
				}
				t.Logf("    %d. %s (%s) - 評価: %.1f", j+1, poi.Name, categories, poi.Rate)
			}

			// Directions APIでルートを取得
			if len(combination) >= 2 {
				t.Logf("  🗺️  Directions APIでルートを取得中...")
				testDirectionsAPI(t, directionsProvider, ctx, combination, "目的地なし")
			}
		}
	}

	// 2. 目的地ありのルート組み合わせをテスト
	t.Logf("\n🎯 目的地ありのルート組み合わせ (目的地: 京都駅):")
	combinationsWithDest, err := strategy.FindCombinationsWithDestination(ctx, scenario, userLocation, destination)
	if err != nil {
		t.Logf("❌ エラー: %v", err)
	} else {
		t.Logf("✅ %d個の組み合わせが見つかりました", len(combinationsWithDest))
		for i, combination := range combinationsWithDest {
			t.Logf("  組み合わせ %d:", i+1)
			for j, poi := range combination {
				categories := "なし"
				if len(poi.Categories) > 0 {
					categories = poi.Categories[0]
				}
				t.Logf("    %d. %s (%s) - 評価: %.1f", j+1, poi.Name, categories, poi.Rate)
			}

			// Directions APIでルートを取得
			if len(combination) >= 2 {
				t.Logf("  🗺️  Directions APIでルートを取得中...")
				testDirectionsAPI(t, directionsProvider, ctx, combination, "目的地あり")
			}
		}
	}
}

// testDirectionsAPI はDirections APIを使ってルートを取得し、結果を表示する
func testDirectionsAPI(t *testing.T, provider *maps.GoogleDirectionsProvider, ctx context.Context, pois []*model.POI, routeType string) {
	if len(pois) < 2 {
		t.Logf("    ⚠️  POIが2つ未満のためルート取得をスキップ")
		return
	}

	// GetWalkingRouteFromPOIsメソッドを使ってルートを取得
	directions, err := provider.GetWalkingRouteFromPOIs(ctx, pois[0], pois[1:]...)
	if err != nil {
		t.Logf("    ❌ Directions API エラー: %v", err)
		return
	}

	// 結果を表示
	t.Logf("    ✅ ルート取得成功 (%s)", routeType)
	t.Logf("    ⏱️  総時間: %v", directions.TotalDuration)

	// エンコードされたポリライン情報
	if directions.Polyline != "" {
		polylineLength := len(directions.Polyline)
		if polylineLength > 50 {
			t.Logf("    🗺️  ポリライン: %s... (%d文字)", directions.Polyline[:50], polylineLength)
		} else {
			t.Logf("    🗺️  ポリライン: %s", directions.Polyline)
		}
	}
}

// TestNatureStrategyBenchmark は各シナリオのパフォーマンスを測定する
func TestNatureStrategyBenchmark(t *testing.T) {
	// 環境変数チェック
	if err := godotenv.Load("../.env"); err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	if os.Getenv("SUPABASE_URL") == "" || os.Getenv("GOOGLE_MAPS_API_KEY") == "" {
		t.Skip("環境変数が設定されていません。ベンチマークテストをスキップします。")
	}

	// 初期化処理は上記と同じ
	supabaseClient, err := database.NewSupabaseClient()
	if err != nil {
		t.Fatalf("Supabaseクライアント初期化失敗: %v", err)
	}

	poiRepo := repository.NewSupabasePOIsRepository(supabaseClient)
	natureStrategy := strategy.NewNatureStrategy(poiRepo)

	userLocation := model.LatLng{Lat: 35.0041, Lng: 135.7699}
	ctx := context.Background()

	scenarios := natureStrategy.GetAvailableScenarios()

	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("Benchmark_%s", scenario), func(t *testing.T) {
			start := time.Now()
			_, err := natureStrategy.FindCombinations(ctx, scenario, userLocation)
			elapsed := time.Since(start)
			if err != nil {
				t.Logf("シナリオ %s でエラー: %v", scenario, err)
			} else {
				t.Logf("シナリオ %s の処理完了 (時間: %v)", scenario, elapsed)
			}
		})
	}
}
