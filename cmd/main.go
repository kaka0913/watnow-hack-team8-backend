package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"Team8-App/internal/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
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

	// HTTPハンドラーの設定
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/team", teamHandler)

	fmt.Println("Team8-App server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to Team8-App!")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "service": "Team8-App"}`)
}

func teamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"team": "Team8", "members": ["Member1", "Member2", "Member3"]}`)
}
