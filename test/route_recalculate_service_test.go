package test

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/service"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/infrastructure/maps"
	postgres_repo "Team8-App/internal/repository"
	"context"
	"log"
	"os"
	"testing"
)

// TestRouteRecalculateService_Nature はnatureテーマでのルート再計算をテストする
func TestRouteRecalculateService_Nature(t *testing.T) {
	log.Printf("🧪 RouteRecalculateService (Nature) テスト開始")

	// テスト環境のセットアップ
	ctx := context.Background()
	
	// 環境変数の確認
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabasePassword := os.Getenv("SUPABASE_DB_PASSWORD")
	
	if googleMapsAPIKey == "" || supabaseURL == "" || supabasePassword == "" {
		t.Skip("必要な環境変数が設定されていません。統合テストをスキップします。")
	}
	
	// データベース接続の初期化
	dbClient, err := database.NewPostgreSQLClient()
	if err != nil {
		t.Fatalf("データベース接続に失敗しました: %v", err)
	}
	
	// リポジトリとサービスの初期化
	poiRepo := postgres_repo.NewPostgresPOIsRepository(dbClient)
	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)
	recalcService := service.NewRouteRecalculateService(directionsProvider, poiRepo)

	// テストケース1: 基本的なルート再計算
	t.Run("基本的なルート再計算", func(t *testing.T) {
		// テスト用の元の提案を作成
		originalProposal := createTestRouteProposal()
		
		// 再計算リクエストを作成
		request := &model.RouteRecalculateRequest{
			ProposalID: "test-proposal-123",
			CurrentLocation: &model.Location{
				Latitude:  34.9853,
				Longitude: 135.7581, // 京都市内
			},
			Mode: "time_based",
			VisitedPOIs: &model.VisitedPOIsContext{
				PreviousPOIs: []model.PreviousPOI{
					{
						POIId: "visited-poi-1",
						Name:  "訪問済み公園",
					},
				},
			},
			RealtimeContext: &model.RealtimeContext{
				Weather:   "sunny",
				TimeOfDay: "afternoon",
			},
		}

		// ルート再計算を実行
		response, err := recalcService.RecalculateRoute(ctx, request, originalProposal)
		if err != nil {
			t.Fatalf("ルート再計算でエラーが発生: %v", err)
		}

		// 結果の検証
		if response == nil {
			t.Fatal("レスポンスがnilです")
		}
		if response.UpdatedRoute == nil {
			t.Fatal("UpdatedRouteがnilです")
		}

		updatedRoute := response.UpdatedRoute
		log.Printf("✅ 再計算されたルート:")
		log.Printf("   タイトル: %s", updatedRoute.Title)
		log.Printf("   推定時間: %d分", updatedRoute.EstimatedDurationMinutes)
		log.Printf("   推定距離: %dメートル", updatedRoute.EstimatedDistanceMeters)
		log.Printf("   ハイライト数: %d", len(updatedRoute.Highlights))
		log.Printf("   ハイライト詳細: %v", updatedRoute.Highlights)
		log.Printf("   ナビゲーションステップ数: %d", len(updatedRoute.NavigationSteps))
		for i, step := range updatedRoute.NavigationSteps {
			if step.Type == "poi" {
				log.Printf("     %d. %s (POI)", i+1, step.Name)
			} else {
				log.Printf("     %d. %s", i+1, step.Description)
			}
		}

		// 基本的な検証
		if updatedRoute.EstimatedDurationMinutes <= 0 {
			t.Error("推定時間が0以下です")
		}
		if len(updatedRoute.NavigationSteps) == 0 {
			t.Error("ナビゲーションステップが空です")
		}
		if len(updatedRoute.Highlights) == 0 {
			t.Error("ハイライトが空です")
		}
		if updatedRoute.RoutePolyline == "" {
			t.Error("ルートポリラインが空です")
		}
	})

	// テストケース2: 目的地ありのルート再計算
	t.Run("目的地ありのルート再計算", func(t *testing.T) {
		originalProposal := createTestRouteProposal()
		
		request := &model.RouteRecalculateRequest{
			ProposalID: "test-proposal-456",
			CurrentLocation: &model.Location{
				Latitude:  34.9853,
				Longitude: 135.7581,
			},
			Mode: "destination",
			VisitedPOIs: &model.VisitedPOIsContext{
				PreviousPOIs: []model.PreviousPOI{
					{
						POIId: "visited-poi-1",
						Name:  "訪問済み公園",
					},
				},
			},
			DestinationLocation: &model.Location{
				Latitude:  34.9853,  // 現在地と同じ緯度
				Longitude: 135.7681, // 少しだけ東に移動（約1km程度）
			},
			RealtimeContext: &model.RealtimeContext{
				Weather:   "cloudy",
				TimeOfDay: "evening",
			},
		}

		response, err := recalcService.RecalculateRoute(ctx, request, originalProposal)
		if err != nil {
			t.Fatalf("目的地ありルート再計算でエラーが発生: %v", err)
		}

		if response == nil || response.UpdatedRoute == nil {
			t.Fatal("目的地ありの再計算結果が無効です")
		}

		log.Printf("✅ 目的地ありルート再計算成功:")
		log.Printf("   推定時間: %d分", response.UpdatedRoute.EstimatedDurationMinutes)
		log.Printf("   ハイライト数: %d", len(response.UpdatedRoute.Highlights))
		log.Printf("   ハイライト詳細: %v", response.UpdatedRoute.Highlights)
		log.Printf("   ナビゲーションステップ数: %d", len(response.UpdatedRoute.NavigationSteps))
		for i, step := range response.UpdatedRoute.NavigationSteps {
			if step.Type == "poi" {
				log.Printf("     %d. %s (POI)", i+1, step.Name)
			} else {
				log.Printf("     %d. %s", i+1, step.Description)
			}
		}
	})

	// テストケース3: サポートされていないテーマのエラーハンドリング
	t.Run("サポートされていないテーマのエラーハンドリング", func(t *testing.T) {
		unsupportedProposal := createTestRouteProposal()
		unsupportedProposal.Theme = "horror" // 現在サポートされていないテーマ
		
		request := &model.RouteRecalculateRequest{
			ProposalID: "test-proposal-789",
			CurrentLocation: &model.Location{
				Latitude:  34.9853,
				Longitude: 135.7581,
			},
			Mode: "time_based",
			VisitedPOIs: &model.VisitedPOIsContext{
				PreviousPOIs: []model.PreviousPOI{},
			},
		}

		_, err := recalcService.RecalculateRoute(ctx, request, unsupportedProposal)
		if err == nil {
			t.Error("サポートされていないテーマでエラーが発生しませんでした")
		}
		
		log.Printf("✅ 期待されるエラー: %v", err)
	})

	// テストケース4: サポートテーマの確認
	t.Run("サポートテーマの確認", func(t *testing.T) {
		supportedThemes := recalcService.GetSupportedThemes()
		log.Printf("✅ サポートされているテーマ: %v", supportedThemes)
		
		if len(supportedThemes) == 0 {
			t.Error("サポートされているテーマが空です")
		}
		
		hasNature := false
		for _, theme := range supportedThemes {
			if theme == model.ThemeNature {
				hasNature = true
				break
			}
		}
		if !hasNature {
			t.Error("natureテーマがサポートされていません")
		}
	})

	log.Printf("🎉 RouteRecalculateService テスト完了")
}

// createTestRouteProposal はテスト用のRouteProposalを作成する
func createTestRouteProposal() *model.RouteProposal {
	return &model.RouteProposal{
		ProposalID:               "test-proposal",
		Theme:                    model.ThemeNature,
		Title:                    "テスト用自然散歩道",
		EstimatedDurationMinutes: 60,
		EstimatedDistanceMeters:  2000,
		DisplayHighlights:        []string{"テスト公園A", "テスト公園B", "テストカフェ"},
		NavigationSteps: []model.NavigationStep{
			{
				Type:        "poi",
				Name:        "テスト公園A",
				POIId:       "test-poi-1",
				Description: "美しい公園です",
				Latitude:    34.9753,
				Longitude:   135.7481,
				DistanceToNextMeters: 500,
			},
			{
				Type:        "poi",
				Name:        "テスト公園B",
				POIId:       "test-poi-2",
				Description: "静かな公園です",
				Latitude:    34.9803,
				Longitude:   135.7531,
				DistanceToNextMeters: 300,
			},
			{
				Type:        "poi",
				Name:        "テストカフェ",
				POIId:       "test-poi-3",
				Description: "落ち着けるカフェです",
				Latitude:    34.9853,
				Longitude:   135.7581,
				DistanceToNextMeters: 0,
			},
		},
		RoutePolyline:  "test_polyline_data",
		GeneratedStory: "テスト用の物語です",
	}
}

// TestRouteRecalculateService_ExploreNewSpot は新しいスポット探索のテストを行う
func TestRouteRecalculateService_ExploreNewSpot(t *testing.T) {
	log.Printf("🧪 ExploreNewSpot テスト開始")

	ctx := context.Background()
	
	// 環境変数の確認
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabasePassword := os.Getenv("SUPABASE_DB_PASSWORD")
	
	if googleMapsAPIKey == "" || supabaseURL == "" || supabasePassword == "" {
		t.Skip("必要な環境変数が設定されていません。統合テストをスキップします。")
	}
	
	// データベース接続の初期化
	dbClient, err := database.NewPostgreSQLClient()
	if err != nil {
		t.Fatalf("データベース接続に失敗しました: %v", err)
	}
	
	poiRepo := postgres_repo.NewPostgresPOIsRepository(dbClient)
	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)
	
	// テスト用のサービスインスタンスを作成
	service := service.NewRouteRecalculateService(directionsProvider, poiRepo)
	
	t.Run("新しいスポット探索の統合テスト", func(t *testing.T) {
		// 実際のルート再計算を通して、新しいスポット探索の動作をテスト
		originalProposal := createTestRouteProposal()
		
		request := &model.RouteRecalculateRequest{
			ProposalID: "explore-test",
			CurrentLocation: &model.Location{
				Latitude:  34.9853,
				Longitude: 135.7581,
			},
			Mode: "time_based",
			VisitedPOIs: &model.VisitedPOIsContext{
				PreviousPOIs: []model.PreviousPOI{
					{
						POIId: "test-poi-1", // 最初のPOIを訪問済みとする
						Name:  "テスト公園A",
					},
				},
			},
		}

		response, err := service.RecalculateRoute(ctx, request, originalProposal)
		if err != nil {
			t.Fatalf("新しいスポット探索を含むルート再計算でエラー: %v", err)
		}

		if response == nil || response.UpdatedRoute == nil {
			t.Fatal("新しいスポット探索結果が無効です")
		}

		log.Printf("✅ 新しいスポット探索成功:")
		log.Printf("   見つかったスポット数: %d", len(response.UpdatedRoute.NavigationSteps))
		log.Printf("   ハイライト数: %d", len(response.UpdatedRoute.Highlights))
		log.Printf("   ハイライト詳細: %v", response.UpdatedRoute.Highlights)
		log.Printf("   推定時間: %d分", response.UpdatedRoute.EstimatedDurationMinutes)
		for i, step := range response.UpdatedRoute.NavigationSteps {
			if step.Type == "poi" {
				log.Printf("     %d. %s (POI)", i+1, step.Name)
			} else {
				log.Printf("     %d. %s", i+1, step.Description)
			}
		}

		// 新しいスポットが追加されているかチェック
		if len(response.UpdatedRoute.NavigationSteps) <= len(originalProposal.NavigationSteps)-1 {
			t.Error("新しいスポットが見つからなかった可能性があります")
		}
	})

	log.Printf("🎉 ExploreNewSpot テスト完了")
}
