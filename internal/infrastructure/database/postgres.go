package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQLClient PostgreSQL直接接続クライアント
type PostgreSQLClient struct {
	DB *sql.DB
}

// NewPostgreSQLClient 新しいPostgreSQLクライアントを作成（リトライ機能付き）
func NewPostgreSQLClient() (*PostgreSQLClient, error) {
	return NewPostgreSQLClientWithRetry(3, 2*time.Second)
}

// NewPostgreSQLClientWithRetry リトライ機能付きのPostgreSQLクライアントを作成
func NewPostgreSQLClientWithRetry(maxRetries int, retryInterval time.Duration) (*PostgreSQLClient, error) {
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

	// Session Pooler最適化設定（IPv4対応、最も安定）
	connectionStrings := []string{
		// Session Pooler（推奨・安定接続確認済み）
		fmt.Sprintf(
			"host=aws-0-ap-northeast-1.pooler.supabase.com port=5432 user=postgres.%s password=%s dbname=postgres sslmode=require connect_timeout=15 pool_max_conns=20 pool_min_conns=2",
			strings.Split(host, ".")[0], supabasePassword,
		),
	}

	var db *sql.DB
	var err error
	var lastErr error

	// Session Pooler専用ループ（安定接続確認済み）
	for attempt := 1; attempt <= maxRetries; attempt++ {
		connStr := connectionStrings[0] // Session Poolerのみ使用
		connType := "Session Pooler (最適化・IPv4対応)"
		
		if attempt == 1 {
			fmt.Printf("Session Pooler接続開始: %s\n", connType)
		}
		
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				return nil, fmt.Errorf("Session Pooler接続失敗（%d回試行後）: %w", maxRetries, err)
			}
			fmt.Printf("接続試行 %d/%d 失敗: %v\n", attempt, maxRetries, err)
			time.Sleep(retryInterval)
			continue
		}

		// 接続テスト
		err = db.Ping()
		if err == nil {
			fmt.Printf("✅ Session Pooler接続成功（試行%d回目）\n", attempt)
			break
		}

		lastErr = err
		if attempt < maxRetries {
			fmt.Printf("接続試行 %d/%d 失敗: %v\n%v後にリトライします...\n", 
				attempt, maxRetries, err, retryInterval)
			db.Close()
			time.Sleep(retryInterval)
		} else {
			fmt.Printf("Session Pooler接続失敗（%d回試行後）: %v\n", maxRetries, err)
			db.Close()
		}
	}

	if err != nil {
		return nil, fmt.Errorf("Session Pooler接続に失敗（全試行完了）: %w", lastErr)
	}

	// Session Pooler最適化設定
	db.SetMaxOpenConns(20)  // Session Pooler推奨値
	db.SetMaxIdleConns(2)   // 最小アイドル接続
	db.SetConnMaxLifetime(10 * time.Minute) // 接続寿命延長

	return &PostgreSQLClient{
		DB: db,
	}, nil
}

// getConnectionType 接続文字列から接続タイプを取得
func getConnectionType(connStr string) string {
	port := getPortFromConnStr(connStr)
	
	// Session Pooler の判定
	if strings.Contains(connStr, "pooler.supabase.com") && port == "5432" && strings.Contains(connStr, "user=postgres.") {
		return "Session Pooler (推奨・IPv4対応)"
	}
	
	// Transaction Pooler の判定
	if strings.Contains(connStr, "pooler.supabase.com") && port == "6543" && strings.Contains(connStr, "user=postgres.") {
		return "Transaction Pooler (IPv4対応・PREPARE文制限)"
	}
	
	// 直接接続の判定
	if strings.Contains(connStr, "db.") && port == "5432" && strings.Contains(connStr, "user=postgres ") {
		return "Direct Connection (IPv6専用)"
	}
	
	// 従来のConnection Pooler
	if strings.Contains(connStr, "db.") && port == "6543" {
		return "Legacy Connection Pooler (フォールバック)"
	}
	
	// その他
	return fmt.Sprintf("Unknown Connection Type (ポート%s)", port)
}

// getPortFromConnStr 接続文字列からポート番号を抽出
func getPortFromConnStr(connStr string) string {
	if strings.Contains(connStr, "port=5432") {
		return "5432"
	} else if strings.Contains(connStr, "port=6543") {
		return "6543"
	}
	return "unknown"
}

// maskPassword パスワードをマスクする（ログ出力用）
func maskPassword(connStr string) string {
	// パスワード部分を***でマスク
	return "host=db.xxx.supabase.co port=6543 user=postgres password=*** dbname=postgres sslmode=require connect_timeout=10"
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

// HealthCheckWithRetry リトライ付きヘルスチェック
func (pc *PostgreSQLClient) HealthCheckWithRetry(maxRetries int, retryInterval time.Duration) error {
	if pc.DB == nil {
		return fmt.Errorf("PostgreSQLクライアントが初期化されていません")
	}

	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = pc.DB.Ping()
		if err == nil {
			return nil
		}

		if attempt < maxRetries {
			fmt.Printf("ヘルスチェック試行 %d/%d 失敗: %v\n%v後にリトライします...\n", 
				attempt, maxRetries, err, retryInterval)
			time.Sleep(retryInterval)
		}
	}

	return fmt.Errorf("ヘルスチェックに失敗（%d回試行後）: %w", maxRetries, err)
}

// IsConnectionAlive 接続が生きているかチェック
func (pc *PostgreSQLClient) IsConnectionAlive() bool {
	if pc.DB == nil {
		return false
	}
	return pc.DB.Ping() == nil
}

// Reconnect 再接続を試行
func (pc *PostgreSQLClient) Reconnect() error {
	if pc.DB != nil {
		pc.DB.Close()
	}

	newClient, err := NewPostgreSQLClientWithRetry(3, 2*time.Second)
	if err != nil {
		return err
	}

	pc.DB = newClient.DB
	return nil
}
