package test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/service"
	"Team8-App/internal/handler"
	"Team8-App/internal/infrastructure/ai"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/infrastructure/firestore"
	"Team8-App/internal/infrastructure/maps"
	"Team8-App/internal/repository"
	"Team8-App/internal/usecase"
)

// TestPerformanceAnalysis はルート再計算のパフォーマンス分析テスト
func TestPerformanceAnalysis(t *testing.T) {
	log.Printf("🔍 ルート再計算パフォーマンス分析開始")

	// 環境変数読み込み
	if err := godotenv.Load("../.env"); err != nil {
		t.Fatalf(".env file not found: %v", err)
	}

	gin.SetMode(gin.TestMode)

	// 必要な環境変数の取得
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	firestoreProjectID := os.Getenv("FIRESTORE_PROJECT_ID")

	if googleMapsAPIKey == "" || geminiAPIKey == "" || firestoreProjectID == "" {
		t.Fatalf("Required environment variables not set")
	}

	// 各コンポーネントの初期化時間を測定
	startTime := time.Now()

	// Database connections
	dbStart := time.Now()
	postgresClient, err := database.NewPostgreSQLClient()
	if err != nil {
		t.Fatalf("PostgreSQL初期化失敗: %v", err)
	}
	dbDuration := time.Since(dbStart)
	log.Printf("📊 PostgreSQL初期化時間: %v", dbDuration)

	firestoreStart := time.Now()
	ctx := context.Background()
	firestoreClient, err := firestore.NewFirestoreClient(ctx, firestoreProjectID)
	if err != nil {
		t.Fatalf("Firestore初期化失敗: %v", err)
	}
	defer firestoreClient.Close()
	firestoreDuration := time.Since(firestoreStart)
	log.Printf("📊 Firestore初期化時間: %v", firestoreDuration)

	// API clients
	apiStart := time.Now()
	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)
	geminiClient := ai.NewGeminiClient(geminiAPIKey)
	storyGenerationRepo := ai.NewGeminiStoryRepository(geminiClient)
	apiDuration := time.Since(apiStart)
	log.Printf("📊 API clients初期化時間: %v", apiDuration)

	// Dependency injection
	diStart := time.Now()
	poiRepo := repository.NewPostgresPOIsRepository(postgresClient)
	routeSuggestionService := service.NewRouteSuggestionService(directionsProvider, poiRepo)
	firestoreRepo := repository.NewFirestoreRouteProposalRepository(firestoreClient.GetClient())
	routeProposalUseCase := usecase.NewRouteProposalUseCase(routeSuggestionService, firestoreRepo, storyGenerationRepo)
	
	routeRecalculateService := service.NewRouteRecalculateService(directionsProvider, poiRepo)
	routeRecalculateUseCase := usecase.NewRouteRecalculateUseCase(routeRecalculateService, firestoreRepo, storyGenerationRepo)
	routeProposalHandler := handler.NewRouteProposalHandler(routeProposalUseCase, routeRecalculateUseCase)
	diDuration := time.Since(diStart)
	log.Printf("📊 Dependency injection時間: %v", diDuration)

	// Ginルーターのセットアップ
	routerStart := time.Now()
	r := gin.New()
	
	// Route Proposals API エンドポイント
	routes := r.Group("/routes")
	{
		routes.POST("/proposals", routeProposalHandler.PostRouteProposals)
		routes.GET("/proposals/:id", routeProposalHandler.GetRouteProposal)
		routes.POST("/recalculate", routeProposalHandler.PostRouteRecalculate)
	}
	routerDuration := time.Since(routerStart)
	log.Printf("📊 Router setup時間: %v", routerDuration)

	totalInitDuration := time.Since(startTime)
	log.Printf("📊 総初期化時間: %v", totalInitDuration)

	// 実際のAPI呼び出しのパフォーマンス測定
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

	jsonData, _ := json.Marshal(recalcRequest)

	// 複数回測定して平均を取る
	iterations := 3
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		log.Printf("🔄 実行 %d/%d", i+1, iterations)
		
		apiStart := time.Now()
		req, _ := http.NewRequest("POST", "/routes/recalculate", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		apiDuration := time.Since(apiStart)

		log.Printf("📊 API実行時間 %d: %v (ステータス: %d)", i+1, apiDuration, w.Code)
		
		if w.Code != http.StatusOK {
			log.Printf("❌ API失敗: %s", w.Body.String())
			continue
		}

		totalDuration += apiDuration

		// レスポンス詳細
		var recalcResponse model.RouteRecalculateResponse
		if err := json.Unmarshal(w.Body.Bytes(), &recalcResponse); err == nil {
			log.Printf("   生成されたハイライト数: %d", len(recalcResponse.UpdatedRoute.Highlights))
			log.Printf("   推定時間: %d分", recalcResponse.UpdatedRoute.EstimatedDurationMinutes)
		}

		// 少し間隔を空ける
		time.Sleep(500 * time.Millisecond)
	}

	avgDuration := totalDuration / time.Duration(iterations)
	log.Printf("📊 平均API実行時間: %v", avgDuration)

	// パフォーマンス分析結果
	log.Printf("\n🎯 パフォーマンス分析結果:")
	log.Printf("  初期化:")
	log.Printf("    - PostgreSQL: %v", dbDuration)
	log.Printf("    - Firestore: %v", firestoreDuration)
	log.Printf("    - API clients: %v", apiDuration)
	log.Printf("    - DI: %v", diDuration)
	log.Printf("    - Router: %v", routerDuration)
	log.Printf("    - 合計: %v", totalInitDuration)
	log.Printf("  実行:")
	log.Printf("    - 平均API時間: %v", avgDuration)

	// パフォーマンス改善提案
	log.Printf("\n💡 改善提案:")
	if dbDuration > 100*time.Millisecond {
		log.Printf("  - PostgreSQL接続プールの最適化を検討")
	}
	if firestoreDuration > 100*time.Millisecond {
		log.Printf("  - Firestore接続の最適化を検討")
	}
	if avgDuration > 2*time.Second {
		log.Printf("  - 外部API呼び出しの並行化を検討")
		log.Printf("  - キャッシュ機構の導入を検討")
	}
}
