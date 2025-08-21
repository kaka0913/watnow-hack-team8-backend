package test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/service"
	"Team8-App/internal/infrastructure/ai"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/infrastructure/firestore"
	"Team8-App/internal/infrastructure/maps"
	"Team8-App/internal/repository"
	"Team8-App/internal/usecase"
)

// TestDetailedPerformanceAnalysis は詳細なパフォーマンス分析テスト
func TestDetailedPerformanceAnalysis(t *testing.T) {
	log.Printf("🔍 詳細なパフォーマンス分析開始")

	// 環境変数読み込み
	if err := godotenv.Load("../.env"); err != nil {
		t.Fatalf(".env file not found: %v", err)
	}

	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	firestoreProjectID := os.Getenv("FIRESTORE_PROJECT_ID")

	if googleMapsAPIKey == "" || geminiAPIKey == "" || firestoreProjectID == "" {
		t.Fatalf("Required environment variables not set")
	}

	// コンポーネント初期化
	postgresClient, _ := database.NewPostgreSQLClient()
	ctx := context.Background()
	firestoreClient, _ := firestore.NewFirestoreClient(ctx, firestoreProjectID)
	defer firestoreClient.Close()

	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)
	geminiClient := ai.NewGeminiClient(geminiAPIKey)
	storyGenerationRepo := ai.NewGeminiStoryRepository(geminiClient)

	poiRepo := repository.NewPostgresPOIsRepository(postgresClient)
	routeRecalculateService := service.NewRouteRecalculateService(directionsProvider, poiRepo)
	firestoreRepo := repository.NewFirestoreRouteProposalRepository(firestoreClient.GetClient())
	routeRecalculateUseCase := usecase.NewRouteRecalculateUseCase(routeRecalculateService, firestoreRepo, storyGenerationRepo)

	// 詳細分析: 各ステップを個別に測定
	realProposalID := "temp_prop_00013c54-b541-429e-93fb-836d223b4f88"

	recalcRequest := model.RouteRecalculateRequest{
		ProposalID: realProposalID,
		CurrentLocation: &model.Location{
			Latitude:  34.9853,
			Longitude: 135.7581,
		},
		Mode: "time_based",
		VisitedPOIs: &model.VisitedPOIsContext{
			PreviousPOIs: []model.PreviousPOI{},
		},
		RealtimeContext: &model.RealtimeContext{
			Weather:   "sunny",
			TimeOfDay: "afternoon",
		},
	}

	// Step 1: Firestore取得のテスト
	log.Printf("📊 Step 1: Firestore取得テスト")
	firestoreStart := time.Now()
	_, err := firestoreRepo.GetRouteProposal(ctx, realProposalID)
	firestoreDuration := time.Since(firestoreStart)
	log.Printf("   Firestore取得時間: %v", firestoreDuration)
	if err != nil {
		t.Fatalf("Firestore取得失敗: %v", err)
	}

	// Step 2: POI検索のテスト
	log.Printf("📊 Step 2: POI検索テスト")
	poiStart := time.Now()
	categories := []string{"観光名所", "店舗", "寺院", "公園"}
	pois, err := poiRepo.GetByCategories(ctx, categories, 34.9853, 135.7581, 1000)
	poiDuration := time.Since(poiStart)
	log.Printf("   POI検索時間: %v (検索結果: %d件)", poiDuration, len(pois))
	if err != nil {
		log.Printf("   POI検索エラー: %v", err)
	}

	// Step 3: Google Maps Directions APIのテスト
	log.Printf("📊 Step 3: Google Directions APIテスト")
	if len(pois) >= 2 {
		directionsStart := time.Now()
		origin := pois[0].ToLatLng()
		waypoints := []model.LatLng{pois[1].ToLatLng()}
		
		_, err := directionsProvider.GetWalkingRoute(ctx, origin, waypoints...)
		directionsDuration := time.Since(directionsStart)
		log.Printf("   Google Directions API時間: %v", directionsDuration)
		if err != nil {
			log.Printf("   Google Directions APIエラー: %v", err)
		}
	}

	// Step 4: Gemini API単体テスト
	log.Printf("📊 Step 4: Gemini API単体テスト")
	geminiStart := time.Now()
	prompt := "京都の自然をテーマにした散歩ルートのタイトルと物語を生成してください。簡潔に200文字程度で。"
	_, err = geminiClient.GenerateContent(ctx, prompt)
	geminiDuration := time.Since(geminiStart)
	log.Printf("   Gemini API時間: %v", geminiDuration)
	if err != nil {
		log.Printf("   Gemini APIエラー: %v", err)
	}

	// Step 5: 複数のGoogle Maps API呼び出しテスト
	log.Printf("📊 Step 5: 複数Maps API呼び出しテスト")
	if len(pois) >= 4 {
		multiApiStart := time.Now()
		
		// 順次処理
		sequentialStart := time.Now()
		for i := 0; i < 3; i++ {
			origin := pois[i].ToLatLng()
			waypoints := []model.LatLng{pois[i+1].ToLatLng()}
			_, _ = directionsProvider.GetWalkingRoute(ctx, origin, waypoints...)
		}
		sequentialDuration := time.Since(sequentialStart)
		
		multiApiDuration := time.Since(multiApiStart)
		log.Printf("   複数Maps API呼び出し時間: %v (順次実行)", sequentialDuration)
		log.Printf("   合計時間: %v", multiApiDuration)
	}

	// 総合テスト（実際のUseCase実行）
	log.Printf("📊 総合テスト: 実際のUseCase実行")
	usecaseStart := time.Now()
	response, err := routeRecalculateUseCase.RecalculateRoute(ctx, &recalcRequest)
	usecaseDuration := time.Since(usecaseStart)
	log.Printf("   総UseCase実行時間: %v", usecaseDuration)
	
	if err != nil {
		t.Fatalf("UseCase実行失敗: %v", err)
	}

	log.Printf("✅ 詳細分析完了:")
	log.Printf("   生成されたハイライト数: %d", len(response.UpdatedRoute.Highlights))
	log.Printf("   推定時間: %d分", response.UpdatedRoute.EstimatedDurationMinutes)

	// パフォーマンス改善提案の分析
	log.Printf("\n💡 詳細改善提案:")
	if firestoreDuration > 50*time.Millisecond {
		log.Printf("  🔥 Firestore最適化: %v → キャッシュ化を検討", firestoreDuration)
	}
	if poiDuration > 100*time.Millisecond {
		log.Printf("  🔥 POI検索最適化: %v → インデックス最適化を検討", poiDuration)
	}
	if geminiDuration > 1*time.Second {
		log.Printf("  🔥 Gemini API最適化: %v → プロンプト短縮・並行化を検討", geminiDuration)
	}
}
