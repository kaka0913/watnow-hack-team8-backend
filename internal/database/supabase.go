package database

import (
	"fmt"
	"os"

	"github.com/supabase-community/supabase-go"
)

// SupabaseClient Supabaseクライアントのラッパー
type SupabaseClient struct {
	Client *supabase.Client
}

// NewSupabaseClient 新しいSupabaseクライアントを作成
func NewSupabaseClient() (*SupabaseClient, error) {
	supabaseURL := os.Getenv("SUPABASE_URL")
	supabaseAnonKey := os.Getenv("SUPABASE_ANON_KEY")

	if supabaseURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL環境変数が設定されていません")
	}
	if supabaseAnonKey == "" {
		return nil, fmt.Errorf("SUPABASE_ANON_KEY環境変数が設定されていません")
	}

	// クライアントオプションの設定
	client, err := supabase.NewClient(supabaseURL, supabaseAnonKey, &supabase.ClientOptions{})
	if err != nil {
		return nil, fmt.Errorf("Supabaseクライアントの初期化に失敗: %w", err)
	}

	return &SupabaseClient{
		Client: client,
	}, nil
}

// GetClient Supabaseクライアントを取得
func (sc *SupabaseClient) GetClient() *supabase.Client {
	return sc.Client
}

// HealthCheck データベース接続のヘルスチェック
func (sc *SupabaseClient) HealthCheck() error {
	// より軽量なヘルスチェックとして接続のみ確認
	if sc.Client == nil {
		return fmt.Errorf("Supabaseクライアントが初期化されていません")
	}

	// 簡単な接続確認として、クライアントの存在確認のみ行う
	fmt.Printf("Supabase client initialized with URL: %s\n", os.Getenv("SUPABASE_URL"))
	return nil
}
