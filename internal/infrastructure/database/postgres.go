package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// PostgreSQLClient PostgreSQL直接接続クライアント
type PostgreSQLClient struct {
	DB *sql.DB
}

// NewPostgreSQLClient 新しいPostgreSQLクライアントを作成
func NewPostgreSQLClient() (*PostgreSQLClient, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabasePassword := os.Getenv("SUPABASE_DB_PASSWORD")

	if supabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL環境変数が設定されていません")
	}
	if supabasePassword == "" {
		return nil, fmt.Errorf("SUPABASE_DB_PASSWORD環境変数が設定されていません")
	}

	// SupabaseのURLからホスト名を抽出 (https://xxx.supabase.co -> xxx.supabase.co)
	host := supabaseURL[8:] // "https://"を除去

	// SupabaseのPostgreSQL接続文字列を構築（ポート6543を使用）
	connStr := fmt.Sprintf(
		"host=db.%s port=6543 user=postgres password=%s dbname=postgres sslmode=require",
		host, supabasePassword,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL接続の初期化に失敗: %w", err)
	}

	// 接続テスト
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("PostgreSQLへの接続に失敗: %w", err)
	}

	return &PostgreSQLClient{
		DB: db,
	}, nil
}

// Close データベース接続を閉じる
func (pc *PostgreSQLClient) Close() error {
	if pc.DB != nil {
		return pc.DB.Close()
	}
	return nil
}

// HealthCheck データベース接続のヘルスチェック
func (pc *PostgreSQLClient) HealthCheck() error {
	if pc.DB == nil {
		return fmt.Errorf("PostgreSQLクライアントが初期化されていません")
	}
	return pc.DB.Ping()
}
