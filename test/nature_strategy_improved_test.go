package test

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/infrastructure/database"
	poisrepo "Team8-App/internal/repository"

	"github.com/joho/godotenv"
)

func TestNatureStrategyImproved(t *testing.T) {
	fmt.Println("🌿 Nature Strategy 改良版テスト（河原町中心・段階的検索）")
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

	t.Run("シナリオベース（改良版）", func(t *testing.T) {
		testImprovedScenarioBased(t, natureStrategy)
	})

	t.Run("目的地ベース（改良版・河原町周辺）", func(t *testing.T) {
		testImprovedDestinationBased(t, natureStrategy)
	})

	fmt.Println("============================================================")
	fmt.Println("🎉 Nature Strategy 改良版テスト完了")
}

func testImprovedScenarioBased(t *testing.T, natureStrategy strategy.StrategyInterface) {
	fmt.Println("\n🎯 改良版シナリオベーステスト（河原町中心）")
	
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
		
		// 組み合わせ詳細の表示と検証
		displayAndValidateCombinations(combinations, scenario)
	}
}

func testImprovedDestinationBased(t *testing.T, natureStrategy strategy.StrategyInterface) {
	fmt.Println("\n🎯 改良版目的地ベーステスト（河原町周辺・現実的距離）")
	
	// 河原町中心地点から開始（四条河原町交差点）
	startLocation := model.LatLng{Lat: 35.0047, Lng: 135.7700}
	
	// 河原町から現実的に歩ける目的地（1時間圏内）
	destinations := []struct {
		name         string
		location     model.LatLng
		walkTime     string
		description  string
	}{
		{
			"鴨川デルタ（三角州）", 
			model.LatLng{Lat: 35.0266, Lng: 135.7729}, 
			"約25分", 
			"鴨川と高野川の合流地点、自然豊か",
		},
		{
			"梅小路公園", 
			model.LatLng{Lat: 34.9925, Lng: 135.7442}, 
			"約30分", 
			"京都駅西側の大型公園",
		},
		{
			"京都御苑", 
			model.LatLng{Lat: 35.0251, Lng: 135.7625}, 
			"約20分", 
			"皇室ゆかりの広大な公園",
		},
		{
			"円山公園", 
			model.LatLng{Lat: 35.0036, Lng: 135.7810}, 
			"約18分", 
			"祇園に隣接する桜の名所",
		},
		{
			"白川地区", 
			model.LatLng{Lat: 35.0049, Lng: 135.7759}, 
			"約10分", 
			"歴史的な町並みと小川",
		},
	}
	
	scenarios := natureStrategy.GetAvailableScenarios()
	
	// 最初の2つの目的地のみテスト（時間短縮）
	for i, dest := range destinations {
		if i >= 2 { // 最初の2つのみ
			break
		}
		
		for _, scenario := range scenarios {
			fmt.Printf("\n🧪 シナリオ: %s → %s\n", scenario, dest.name)
			fmt.Printf("   📍 開始地点: 河原町中心（四条河原町） (%.4f, %.4f)\n", startLocation.Lat, startLocation.Lng)
			fmt.Printf("   🎯 目的地: %s (%.4f, %.4f)\n", dest.name, dest.location.Lat, dest.location.Lng)
			fmt.Printf("   🚶 徒歩時間: %s | %s\n", dest.walkTime, dest.description)
			
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
			
			// 組み合わせ詳細の表示と検証
			displayAndValidateCombinations(combinations, fmt.Sprintf("%s（%s行き）", scenario, dest.name))
		}
	}
}

func displayAndValidateCombinations(combinations [][]*model.POI, scenarioType string) {
	fmt.Printf("\n   📋 %s POI組み合わせ詳細:\n", scenarioType)
	
	if len(combinations) == 0 {
		fmt.Printf("   ⚠️  組み合わせが見つかりませんでした\n")
		return
	}
	
	// 最初の2組み合わせのみ表示
	maxDisplay := 2
	if len(combinations) < maxDisplay {
		maxDisplay = len(combinations)
	}
	
	for i := 0; i < maxDisplay; i++ {
		combination := combinations[i]
		fmt.Printf("   📍 組み合わせ %d:\n", i+1)
		
		var totalDistance float64
		var prevLocation *model.LatLng
		
		for j, poi := range combination {
			if poi == nil {
				continue
			}
			
			// POIタイプの判定
			poiIcon := "📍"
			if len(poi.Categories) > 0 {
				for _, cat := range poi.Categories {
					if cat == "公園" {
						poiIcon = "🌳"
						break
					} else if cat == "カフェ" {
						poiIcon = "☕"
						break
					} else if cat == "寺院" {
						poiIcon = "⛩️"
						break
					} else if cat == "ベーカリー" {
						poiIcon = "🥖"
						break
					} else if cat == "観光名所" {
						poiIcon = "🏛️"
						break
					}
				}
			}
			
			currentLocation := poi.ToLatLng()
			
			// 前のPOIからの距離を計算
			var distanceText string
			if prevLocation != nil {
				distance := math.Sqrt(math.Pow(currentLocation.Lat-prevLocation.Lat, 2)+math.Pow(currentLocation.Lng-prevLocation.Lng, 2)) * 111000 // 簡易距離計算（メートル）
				totalDistance += distance
				walkMinutes := int(distance / 80) // 80m/分の歩行速度
				distanceText = fmt.Sprintf(" [前から%.0fm・徒歩%d分]", distance, walkMinutes)
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
			
			fmt.Printf("%s\n", distanceText)
			prevLocation = &currentLocation
		}
		
		// 組み合わせの統計情報
		fmt.Printf("      💫 スポット数: %d箇所\n", len(combination))
		if totalDistance > 0 {
			totalWalkMinutes := int(totalDistance / 80)
			fmt.Printf("      🚶 総歩行距離: %.0fm（徒歩約%d分）\n", totalDistance, totalWalkMinutes)
			
			// 距離間隔の検証（ユーザー要望に合わせて歩行時間を重視）
			if totalDistance < 3000 { // 3km未満
				fmt.Printf("      ✅ 適切な距離間隔（歩行重視・長時間散歩）\n")
			} else if totalDistance < 6000 { // 6km未満
				fmt.Printf("      ⚠️  やや長距離（歩行重視により許容範囲）\n")
			} else {
				fmt.Printf("      ❌ 極端に長距離（3時間超過の可能性）\n")
			}
		}
	}
	
	if len(combinations) > maxDisplay {
		fmt.Printf("   ... (他 %d 組み合わせ)\n", len(combinations)-maxDisplay)
	}
}
