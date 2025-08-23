package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"Team8-App/internal/domain/service"
	"Team8-App/internal/handler"
	"Team8-App/internal/infrastructure/ai"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/infrastructure/firestore"
	"Team8-App/internal/infrastructure/maps"
	"Team8-App/internal/repository"
	"Team8-App/internal/usecase"
)

// maskEnvVar は環境変数の値をマスクして表示する
func maskEnvVar(value string) string {
	if value == "" {
		return "❌ 未設定"
	}
	if len(value) <= 8 {
		return "✅ ****"
	}
	return fmt.Sprintf("✅ %s****%s", value[:4], value[len(value)-4:])
}

func main() {
	// Cloud Run環境の検出
	isCloudRun := os.Getenv("K_SERVICE") != "" || os.Getenv("PORT") != ""
	
	// 開発環境では.envファイルを読み込み、本番環境ではシステム環境変数を使用
	if err := godotenv.Load(".env"); err != nil {
		if isCloudRun {
			log.Printf("☁️  Cloud Run環境: システム環境変数を使用")
		} else {
			log.Printf("📝 .envファイルが見つかりません。システム環境変数を使用します")
		}
	} else {
		log.Printf("📝 .envファイルを読み込みました")
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	firestoreProjectID := os.Getenv("FIRESTORE_PROJECT_ID")

	// 環境変数の設定状況をログ出力
	log.Printf("🔧 環境変数設定状況:")
	log.Printf("   SUPABASE_URL: %s", maskEnvVar(supabaseURL))
	log.Printf("   SUPABASE_ANON_KEY: %s", maskEnvVar(supabaseAnonKey))
	log.Printf("   GOOGLE_MAPS_API_KEY: %s", maskEnvVar(googleMapsAPIKey))
	log.Printf("   GEMINI_API_KEY: %s", maskEnvVar(geminiAPIKey))
	log.Printf("   FIRESTORE_PROJECT_ID: %s", maskEnvVar(firestoreProjectID))

	if supabaseURL == "" || supabaseAnonKey == "" {
		log.Fatal("❌ Supabase環境変数が設定されていません")
	}

	if googleMapsAPIKey == "" {
		log.Fatal("❌ Google Maps API Keyが設定されていません")
	}

	if geminiAPIKey == "" {
		log.Fatal("❌ Gemini API Keyが設定されていません")
	}

	if firestoreProjectID == "" {
		log.Fatal("❌ Firestore Project IDが設定されていません")
	}
	// Database connections
	supabaseClient, err := database.NewSupabaseClient()
	if err != nil {
		log.Fatalf("Supabase初期化失敗: %v", err)
	}
	if err := supabaseClient.HealthCheck(); err != nil {
		log.Fatalf("Supabaseヘルスチェック失敗: %v", err)
	}

	postgresClient, err := database.NewPostgreSQLClient()
	if err != nil {
		log.Fatalf("PostgreSQL初期化失敗: %v", err)
	}
	defer postgresClient.Close()
	if err := postgresClient.HealthCheck(); err != nil {
		log.Fatalf("PostgreSQLヘルスチェック失敗: %v", err)
	}

	ctx := context.Background()
	firestoreClient, err := firestore.NewFirestoreClient(ctx, firestoreProjectID)
	if err != nil {
		log.Fatalf("Firestore初期化失敗: %v", err)
	}
	defer firestoreClient.Close()

	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)
	geminiClient := ai.NewGeminiClient(geminiAPIKey)
	storyGenerationRepo := ai.NewGeminiStoryRepository(geminiClient)

	// Dependency injection
	walksRepo := repository.NewSupabaseWalksRepository(supabaseClient)
	walksUsecase := usecase.NewWalksUsecase(walksRepo)
	firestoreRepo := repository.NewFirestoreRouteProposalRepository(firestoreClient.GetClient())
	walksHandler := handler.NewWalksHandler(walksUsecase, firestoreRepo)

	poiRepo := repository.NewPostgresPOIsRepository(postgresClient)
	routeSuggestionService := service.NewRouteSuggestionService(directionsProvider, poiRepo)
	routeProposalUseCase := usecase.NewRouteProposalUseCase(routeSuggestionService, firestoreRepo, storyGenerationRepo)
	
	routeRecalculateService := service.NewRouteRecalculateService(directionsProvider, poiRepo)
	routeRecalculateUseCase := usecase.NewRouteRecalculateUseCase(routeRecalculateService, firestoreRepo, storyGenerationRepo)
	routeProposalHandler := handler.NewRouteProposalHandler(routeProposalUseCase, routeRecalculateUseCase)

	// Ginルーターのセットアップ
	r := gin.Default()
	// ヘルスチェックエンドポイント
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "Team8-App",
		})
	})

	// Walks API エンドポイント
	walks := r.Group("/walks")
	{
		walks.POST("", walksHandler.CreateWalk)     // POST /walks
		walks.GET("", walksHandler.GetWalks)        // GET /walks - Firestoreから全てのルート提案を取得
		walks.GET("/:id", walksHandler.GetWalkDetail) // GET /walks/:id
	}

	// Route Proposals API エンドポイント
	routes := r.Group("/routes")
	{
		routes.POST("/proposals", routeProposalHandler.PostRouteProposals)    // POST /routes/proposals
		routes.GET("/proposals/:id", routeProposalHandler.GetRouteProposal)   // GET /routes/proposals/:id
		routes.POST("/recalculate", routeProposalHandler.PostRouteRecalculate) // POST /routes/recalculate
	}

	// Cloud RunのPORT環境変数を取得（デフォルト8080）
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	fmt.Printf("🚀 Team8-App server starting on :%s...\n", port)
	log.Fatal(r.Run(":" + port))
}
