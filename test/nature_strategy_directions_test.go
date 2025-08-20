package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/infrastructure/database"
	poisrepo "Team8-App/internal/repository"

	"github.com/joho/godotenv"
)

func TestNatureStrategyDirectionsAPI(t *testing.T) {
	fmt.Println("🌿 Nature Strategy POI組み合わせ生成テスト（河原町中心）")
	fmt.Println("============================================================")

	// .envファイルの読み込み
	err := godotenv.Load("../.env")
	if err != nil {
		t.Logf("⚠️  .envファイルの読み込みに失敗: %v", err)
	}

	// データベース接続
	client, err := database.NewPostgreSQLClient()
	if err != nil {
		t.Fatalf("❌ PostgreSQL接続エラー: %v", err)
	}
	defer client.Close()

	fmt.Println("✅ データベース接続成功")

	// リポジトリとストラテジーの初期化
	poisRepo := poisrepo.NewPostgresPOIsRepository(client)
	natureStrategy := strategy.NewNatureStrategy(poisRepo)

	t.Run("目的地なし（シナリオベース）", func(t *testing.T) {
		testScenarioBased(t, natureStrategy)
	})

	t.Run("目的地あり（目的地ベース）", func(t *testing.T) {
		testDestinationBased(t, natureStrategy)
	})

	fmt.Println("============================================================")
	fmt.Println("🎉 Nature Strategy POI組み合わせ生成テスト完了（河原町中心）")
}

func testScenarioBased(t *testing.T, natureStrategy strategy.StrategyInterface) {
	fmt.Println("\n🎯 シナリオベーステスト（目的地なし）")
	
	// 河原町中心地点から開始（四条河原町交差点）
	startLocation := model.LatLng{Lat: 35.0047, Lng: 135.7700}
	
	// 利用可能なシナリオを取得
	scenarios := natureStrategy.GetAvailableScenarios()
	fmt.Printf("   🎭 利用可能なシナリオ: %v\n", scenarios)
	
	for _, scenario := range scenarios {
		fmt.Printf("\n🧪 シナリオ: %s\n", scenario)
		fmt.Printf("   📍 開始地点: 河原町中心（四条河原町） (%.4f, %.4f)\n", startLocation.Lat, startLocation.Lng)
		
		ctx := context.Background()
		startTime := time.Now()
		
		// POI組み合わせを生成
		combinations, err := natureStrategy.FindCombinations(ctx, scenario, startLocation)
		
		duration := time.Since(startTime)
		
		if err != nil {
			fmt.Printf("   ❌ POI組み合わせ生成エラー: %v\n", err)
			fmt.Printf("   ⏱️  実行時間: %v\n", duration)
			continue
		}
		
		fmt.Printf("   ✅ POI組み合わせ生成成功\n")
		fmt.Printf("   ⏱️  実行時間: %v\n", duration)
		fmt.Printf("   🎯 生成された組み合わせ数: %d\n", len(combinations))
		
		// 組み合わせ詳細の表示
		displayCombinations(combinations, scenario)
	}
}

func testDestinationBased(t *testing.T, natureStrategy strategy.StrategyInterface) {
	fmt.Println("\n🎯 目的地ベーステスト（目的地あり）")
	
	// 河原町中心地点から開始（四条河原町交差点）
	startLocation := model.LatLng{Lat: 35.0047, Lng: 135.7700}
	
	// 河原町から1時間程度で回れる目的地（徒歩圏内）
	destinations := []struct {
		name     string
		location model.LatLng
		walkTime string
	}{
		{"祇園・八坂神社", model.LatLng{Lat: 35.0036, Lng: 135.7786}, "約15分"},
		{"錦市場", model.LatLng{Lat: 35.0049, Lng: 135.7661}, "約5分"},
		{"先斗町", model.LatLng{Lat: 35.0037, Lng: 135.7707}, "約3分"},
		{"鴨川公園（三条大橋）", model.LatLng{Lat: 35.0098, Lng: 135.7732}, "約8分"},
		{"新京極・寺町通り", model.LatLng{Lat: 35.0057, Lng: 135.7684}, "約2分"},
	}
	
	scenarios := natureStrategy.GetAvailableScenarios()
	
	for _, dest := range destinations {
		for _, scenario := range scenarios {
			fmt.Printf("\n🧪 シナリオ: %s → %s\n", scenario, dest.name)
			fmt.Printf("   📍 開始地点: 河原町中心（四条河原町） (%.4f, %.4f)\n", startLocation.Lat, startLocation.Lng)
			fmt.Printf("   🎯 目的地: %s (%.4f, %.4f) [徒歩%s]\n", dest.name, dest.location.Lat, dest.location.Lng, dest.walkTime)
			
			ctx := context.Background()
			startTime := time.Now()
			
			// 目的地ありPOI組み合わせを生成
			combinations, err := natureStrategy.FindCombinationsWithDestination(ctx, scenario, startLocation, dest.location)
			
			duration := time.Since(startTime)
			
			if err != nil {
				fmt.Printf("   ❌ POI組み合わせ生成エラー: %v\n", err)
				fmt.Printf("   ⏱️  実行時間: %v\n", duration)
				continue
			}
			
			fmt.Printf("   ✅ POI組み合わせ生成成功\n")
			fmt.Printf("   ⏱️  実行時間: %v\n", duration)
			fmt.Printf("   🎯 生成された組み合わせ数: %d\n", len(combinations))
			
			// 組み合わせ詳細の表示
			displayCombinations(combinations, fmt.Sprintf("%s（%s行き）", scenario, dest.name))
		}
	}
}

func displayCombinations(combinations [][]*model.POI, scenarioType string) {
	fmt.Printf("\n   📋 %s POI組み合わせ詳細:\n", scenarioType)
	
	if len(combinations) == 0 {
		fmt.Printf("   ⚠️  組み合わせが見つかりませんでした\n")
		return
	}
	
	// 最初の3組み合わせのみ表示
	maxDisplay := 3
	if len(combinations) < maxDisplay {
		maxDisplay = len(combinations)
	}
	
	for i := 0; i < maxDisplay; i++ {
		combination := combinations[i]
		fmt.Printf("   📍 組み合わせ %d:\n", i+1)
		
		for j, poi := range combination {
			if poi == nil {
				continue
			}
			
			// POIタイプの判定
			poiIcon := "📍"
			if containsCategory(poi.Categories, "公園") {
				poiIcon = "🌳"
			} else if containsCategory(poi.Categories, "カフェ") {
				poiIcon = "☕"
			} else if containsCategory(poi.Categories, "寺院") {
				poiIcon = "⛩️"
			} else if containsCategory(poi.Categories, "ベーカリー") {
				poiIcon = "🥖"
			} else if containsCategory(poi.Categories, "観光名所") {
				poiIcon = "🏛️"
			}
			
			fmt.Printf("      %d. %s %s", j+1, poiIcon, poi.Name)
			
			if poi.Rate > 0 {
				fmt.Printf(" (評価: %.1f)", poi.Rate)
			}
			
			// カテゴリの表示
			if len(poi.Categories) > 0 {
				fmt.Printf(" [%s]", strings.Join(poi.Categories, ", "))
			}
			
			// 位置情報の表示
			if poi.Location != nil && poi.Location.Coordinates != nil && len(poi.Location.Coordinates) >= 2 {
				lng, lat := poi.Location.Coordinates[0], poi.Location.Coordinates[1]
				fmt.Printf(" (%.4f, %.4f)", lat, lng)
			}
			
			fmt.Println()
		}
		
		fmt.Printf("      � スポット数: %d箇所\n", len(combination))
	}
	
	if len(combinations) > maxDisplay {
		fmt.Printf("   ... (他 %d 組み合わせ)\n", len(combinations)-maxDisplay)
	}
}

func containsCategory(categories []string, target string) bool {
	for _, category := range categories {
		if category == target {
			return true
		}
	}
	return false
}
