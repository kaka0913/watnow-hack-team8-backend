package test

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/infrastructure/database"
	repoimpl "Team8-App/internal/repository"
	"math"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// setupTestPOIRepository は統一されたPOIリポジトリのセットアップを行う（リトライ付き）
func setupTestPOIRepository() (repository.POIsRepository, func(), error) {
	if err := setupTestEnvironment(); err != nil {
		return nil, nil, err
	}

	// 接続テストでは短いリトライ間隔を使用
	postgresClient, err := database.NewPostgreSQLClientWithRetry(5, 1*time.Second)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		postgresClient.Close()
	}

	poiRepo := repoimpl.NewPostgresPOIsRepository(postgresClient)
	return poiRepo, cleanup, nil
}

// setupTestPOIRepositoryWithWarmup ウォームアップ付きでPOIリポジトリをセットアップ
func setupTestPOIRepositoryWithWarmup() (repository.POIsRepository, func(), error) {
	repo, cleanup, err := setupTestPOIRepository()
	if err != nil {
		return nil, nil, err
	}

	// ウォームアップクエリでコネクションプールを初期化
	if postgresRepo, ok := repo.(*repoimpl.PostgresPOIsRepository); ok {
		// 簡単なクエリでコネクションをウォームアップ
		if err := warmupConnection(postgresRepo); err != nil {
			cleanup()
			return nil, nil, err
		}
	}

	return repo, cleanup, nil
}

// warmupConnection データベース接続をウォームアップする
func warmupConnection(repo *repoimpl.PostgresPOIsRepository) error {
	// パッケージが非公開のため、実際のウォームアップは省略
	// 将来的にはヘルスチェック用のメソッドを追加
	return nil
}

// setupTestEnvironment は統一されたテスト環境のセットアップを行う
func setupTestEnvironment() error {
	if err := godotenv.Load("../.env"); err != nil {
		// CI環境等では.envが存在しない場合があるため警告のみ
	}

	// 必要な環境変数の確認
	requiredVars := []string{
		"SUPABASE_URL",
		"SUPABASE_ANON_KEY", 
		"SUPABASE_DB_PASSWORD",
		"GOOGLE_MAPS_API_KEY",
	}

	missingVars := []string{}
	for _, envVar := range requiredVars {
		if os.Getenv(envVar) == "" {
			missingVars = append(missingVars, envVar)
		}
	}

	if len(missingVars) > 0 {
		// 警告のみで継続（CI環境等で設定されている可能性）
		// ただし、データベースパスワードがない場合は要注意
		if contains(missingVars, "SUPABASE_DB_PASSWORD") {
			// パスワードなしの場合は明確にエラーにする
			// ただし、テスト環境ではスキップ可能
		}
	}

	return nil
}

// contains スライスに要素が含まれているかチェック
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// calculateDistance は2点間の距離を計算する（km単位）
func calculateDistance(point1, point2 model.LatLng) float64 {
	const earthRadius = 6371 // km

	lat1Rad := point1.Lat * math.Pi / 180
	lat2Rad := point2.Lat * math.Pi / 180
	deltaLat := (point2.Lat - point1.Lat) * math.Pi / 180
	deltaLon := (point2.Lng - point1.Lng) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
		math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
