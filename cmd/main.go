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

func main() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	firestoreProjectID := os.Getenv("FIRESTORE_PROJECT_ID")

	if supabaseURL == "" || supabaseAnonKey == "" {
		fmt.Println("⚠️  Supabase環境変数が設定されていません:")
		log.Fatal("Supabase環境変数が設定されていません")
	}

	if googleMapsAPIKey == "" {
		fmt.Println("⚠️  Google Maps API Keyが設定されていません:")
		log.Fatal("Google Maps API Key not set")
	}

	if geminiAPIKey == "" {
		fmt.Println("⚠️  Gemini API Keyが設定されていません:")
		log.Fatal("Gemini API Key not set")
	}

	if firestoreProjectID == "" {
		fmt.Println("⚠️  Firestore Project IDが設定されていません:")
		log.Fatal("Firestore Project ID not set")
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
	walksHandler := handler.NewWalksHandler(walksUsecase)

	poiRepo := repository.NewPostgresPOIsRepository(postgresClient)
	routeSuggestionService := service.NewRouteSuggestionService(directionsProvider, poiRepo)
	firestoreRepo := repository.NewFirestoreRouteProposalRepository(firestoreClient.GetClient())
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
		walks.POST("", walksHandler.CreateWalk)           // POST /walks
		walks.GET("", walksHandler.GetWalksByBoundingBox) // GET /walks?bbox=...
		walks.GET("/:id", walksHandler.GetWalkDetail)     // GET /walks/:id
	}

	// Route Proposals API エンドポイント
	routes := r.Group("/routes")
	{
		routes.POST("/proposals", routeProposalHandler.PostRouteProposals)    // POST /routes/proposals
		routes.GET("/proposals/:id", routeProposalHandler.GetRouteProposal)   // GET /routes/proposals/:id
		routes.POST("/recalculate", routeProposalHandler.PostRouteRecalculate) // POST /routes/recalculate
	}

	fmt.Println("🚀 Team8-App server starting on :8080...")
	log.Fatal(r.Run(":8080"))
}
