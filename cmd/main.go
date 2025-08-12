package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"Team8-App/internal/usecase"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/handler"
	"Team8-App/internal/repository"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("Warning: .env file not found, using system environment variables")
	}

	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")

	if supabaseURL == "" || supabaseAnonKey == "" {
		fmt.Println("⚠️  環境変数が設定されていません:")
		fmt.Println("必要な環境変数:")
		fmt.Println("\n.envファイルを作成するか、環境変数を設定してください")
		log.Fatal("Environment variables not set")
	}

	fmt.Println("Initializing Supabase client...")
	supabaseClient, err := database.NewSupabaseClient()
	if err != nil {
		log.Fatalf("Supabaseクライアント初期化失敗: %v", err)
	}

	fmt.Println("Performing Supabase health check...")
	if err := supabaseClient.HealthCheck(); err != nil {
		log.Fatalf("Supabaseヘルスチェック失敗: %v", err)
	}
	fmt.Println("✅ Supabase connection successful!")

	fmt.Println("Setting up dependency injection...")
	walksRepo := repository.NewSupabaseWalksRepository(supabaseClient)
	walksUsecase := usecase.NewWalksUsecase(walksRepo)
	walksHandler := handler.NewWalksHandler(walksUsecase)

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

	fmt.Println("🚀 Team8-App server starting on :8080...")
	log.Fatal(r.Run(":8080"))
}
