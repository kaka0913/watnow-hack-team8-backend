package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/service"
	"Team8-App/internal/handler"
	"Team8-App/internal/infrastructure/ai"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/infrastructure/maps"
	"Team8-App/internal/repository"
	"Team8-App/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func TestRoutesProposalsIntegrationCore(t *testing.T) {
	fmt.Println("🚀 POST /routes/proposals コア統合テスト（Firestore除く）")
	fmt.Println("============================================================")

	// .envファイルの読み込み
	err := godotenv.Load("../.env")
	if err != nil {
		t.Logf("⚠️  .envファイルの読み込みに失敗: %v", err)
	}

	// 必要な環境変数のチェック（最小限）
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")

	if geminiAPIKey == "" {
		t.Skip("⚠️  GEMINI_API_KEY が設定されていません。テストをスキップします。")
	}
	if googleMapsAPIKey == "" {
		t.Skip("⚠️  GOOGLE_MAPS_API_KEY が設定されていません。テストをスキップします。")
	}

	fmt.Printf("✅ GEMINI_API_KEY: 設定済み\n")
	fmt.Printf("✅ GOOGLE_MAPS_API_KEY: 設定済み\n")

	// PostgreSQL接続
	postgresClient, err := database.NewPostgreSQLClient()
	if err != nil {
		t.Fatalf("❌ PostgreSQL接続エラー: %v", err)
	}
	defer postgresClient.Close()

	// Google Maps
	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)

	// Gemini AI
	geminiClient := ai.NewGeminiClient(geminiAPIKey)
	storyGenerator := ai.NewGeminiStoryRepository(geminiClient)

	fmt.Println("✅ コアサービス接続成功（PostgreSQL + Google Maps + Gemini）")

	// サービス・リポジトリの初期化（Firestoreモック）
	poiRepo := repository.NewPostgresPOIsRepository(postgresClient)
	routeSuggestionService := service.NewRouteSuggestionService(directionsProvider, poiRepo)

	// 実際のFirestoreリポジトリを作成（nil clientでモック動作）
	mockFirestoreRepo := repository.NewFirestoreRouteProposalRepository(nil)

	routeProposalUseCase := usecase.NewRouteProposalUseCase(
		routeSuggestionService,
		mockFirestoreRepo,
		storyGenerator,
	)

	// ハンドラーの初期化
	routeProposalHandler := handler.NewRouteProposalHandler(routeProposalUseCase)

	// Ginエンジンのセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/routes/proposals", routeProposalHandler.PostRouteProposals)

	t.Run("河原町30分散歩（コア機能）", func(t *testing.T) {
		testCoreIntegration30Min(t, router)
	})

	t.Run("河原町→祇園（目的地ベース）", func(t *testing.T) {
		testCoreIntegrationDestination(t, router)
	})

	t.Run("パフォーマンス測定（複数パターン）", func(t *testing.T) {
		testCorePerformance(t, router)
	})

	fmt.Println("============================================================")
	fmt.Printf("🎉 POST /routes/proposals コア統合テスト完了\n")
	fmt.Printf("📊 レスポンス時間の目安: %.1f-%.1f秒\n", 15.0, 45.0)
}

func testCoreIntegration30Min(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 河原町30分散歩（コア機能統合テスト）")

	request := model.RouteProposalRequest{
		StartLocation: &model.Location{
			Latitude:  35.0047, // 四条河原町
			Longitude: 135.7700,
		},
		DestinationLocation: nil,
		Mode:                "time_based",
		TimeMinutes:         30, // 30分散歩
		Theme:               "nature",
		RealtimeContext: &model.RealtimeContext{
			Weather:   "sunny",
			TimeOfDay: "afternoon",
		},
	}

	fmt.Printf("   📍 開始地点: 四条河原町 (%.4f, %.4f)\n", 
		request.StartLocation.Latitude, request.StartLocation.Longitude)
	fmt.Printf("   ⏱️  散歩時間: %d分\n", request.TimeMinutes)
	fmt.Printf("   🌿 テーマ: %s\n", request.Theme)

	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("❌ JSONマーシャリングエラー: %v", err)
	}

	req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	
	fmt.Println("   ⏱️  API呼び出し開始...")
	startTime := time.Now()
	
	router.ServeHTTP(w, req)
	
	totalDuration := time.Since(startTime)
	fmt.Printf("   ⏱️  総実行時間: %.2f秒\n", totalDuration.Seconds())
	fmt.Printf("   📊 HTTPステータス: %d\n", w.Code)

	if w.Code != http.StatusOK {
		fmt.Printf("   ❌ エラーレスポンス: %s\n", w.Body.String())
		t.Errorf("期待していたHTTPステータス: 200, 実際: %d", w.Code)
		return
	}

	var response model.RouteProposalResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("❌ レスポンスのJSONパースエラー: %v", err)
		return
	}

	fmt.Printf("   ✅ API成功\n")
	fmt.Printf("   🎯 生成されたプロポーザル数: %d\n", len(response.Proposals))

	// 詳細な結果表示
	displayCoreProposals(response.Proposals, "30分散歩")
	analyzeCorePerformance(totalDuration, len(response.Proposals))
}

func testCoreIntegrationDestination(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 河原町→祇園（目的地ベース統合テスト）")

	request := model.RouteProposalRequest{
		StartLocation: &model.Location{
			Latitude:  35.0047, // 四条河原町
			Longitude: 135.7700,
		},
		DestinationLocation: &model.Location{
			Latitude:  35.0036, // 八坂神社
			Longitude: 135.7786,
		},
		Mode:  "destination",
		Theme: "nature",
		RealtimeContext: &model.RealtimeContext{
			Weather:   "cloudy",
			TimeOfDay: "morning",
		},
	}

	fmt.Printf("   📍 開始地点: 四条河原町 (%.4f, %.4f)\n", 
		request.StartLocation.Latitude, request.StartLocation.Longitude)
	fmt.Printf("   🎯 目的地: 八坂神社 (%.4f, %.4f)\n", 
		request.DestinationLocation.Latitude, request.DestinationLocation.Longitude)

	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("❌ JSONマーシャリングエラー: %v", err)
	}

	req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	
	fmt.Println("   ⏱️  API呼び出し開始...")
	startTime := time.Now()
	
	router.ServeHTTP(w, req)
	
	totalDuration := time.Since(startTime)
	fmt.Printf("   ⏱️  総実行時間: %.2f秒\n", totalDuration.Seconds())
	fmt.Printf("   📊 HTTPステータス: %d\n", w.Code)

	if w.Code != http.StatusOK {
		fmt.Printf("   ❌ エラーレスポンス: %s\n", w.Body.String())
		t.Errorf("期待していたHTTPステータス: 200, 実際: %d", w.Code)
		return
	}

	var response model.RouteProposalResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("❌ レスポンスのJSONパースエラー: %v", err)
		return
	}

	fmt.Printf("   ✅ API成功\n")
	fmt.Printf("   🎯 生成されたプロポーザル数: %d\n", len(response.Proposals))

	displayCoreProposals(response.Proposals, "目的地ベース")
	analyzeCorePerformance(totalDuration, len(response.Proposals))
}

func testCorePerformance(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 パフォーマンス測定（複数パターン）")

	testCases := []struct {
		name        string
		timeMinutes int
		description string
	}{
		{"短時間散歩", 15, "クイック散歩"},
		{"標準散歩", 45, "標準的な散歩時間"},
		{"長時間散歩", 90, "ゆったり散歩"},
	}

	var allDurations []time.Duration
	var totalProposals int

	for _, testCase := range testCases {
		fmt.Printf("\n   🧪 %s (%d分) - %s\n", testCase.name, testCase.timeMinutes, testCase.description)

		request := model.RouteProposalRequest{
			StartLocation: &model.Location{
				Latitude:  35.0047,
				Longitude: 135.7700,
			},
			Mode:        "time_based",
			TimeMinutes: testCase.timeMinutes,
			Theme:       "nature",
			RealtimeContext: &model.RealtimeContext{
				Weather:   "sunny",
				TimeOfDay: "afternoon",
			},
		}

		jsonData, _ := json.Marshal(request)
		req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		startTime := time.Now()
		router.ServeHTTP(w, req)
		duration := time.Since(startTime)

		allDurations = append(allDurations, duration)

		fmt.Printf("      ⏱️  実行時間: %.2f秒\n", duration.Seconds())
		fmt.Printf("      📊 ステータス: %d\n", w.Code)

		if w.Code == http.StatusOK {
			var response model.RouteProposalResponse
			json.Unmarshal(w.Body.Bytes(), &response)
			proposalCount := len(response.Proposals)
			totalProposals += proposalCount
			fmt.Printf("      🎯 プロポーザル数: %d\n", proposalCount)

			// 最初のプロポーザルの概要
			if len(response.Proposals) > 0 {
				proposal := response.Proposals[0]
				fmt.Printf("      📝 タイトル例: %s\n", proposal.Title)
				fmt.Printf("      📖 物語文字数: %d文字\n", len(proposal.GeneratedStory))
			}
		}
	}

	// 総合パフォーマンス分析
	fmt.Println("\n   📊 総合パフォーマンス分析:")
	var totalTime time.Duration
	minTime := allDurations[0]
	maxTime := allDurations[0]
	
	for _, d := range allDurations {
		totalTime += d
		if d < minTime {
			minTime = d
		}
		if d > maxTime {
			maxTime = d
		}
	}
	
	avgTime := totalTime / time.Duration(len(allDurations))
	
	fmt.Printf("      ⏱️  平均実行時間: %.2f秒\n", avgTime.Seconds())
	fmt.Printf("      ⏱️  最短実行時間: %.2f秒\n", minTime.Seconds())
	fmt.Printf("      ⏱️  最長実行時間: %.2f秒\n", maxTime.Seconds())
	fmt.Printf("      🎯 総プロポーザル数: %d\n", totalProposals)
	fmt.Printf("      📈 プロポーザル生成効率: %.2f件/秒\n", 
		float64(totalProposals)/totalTime.Seconds())

	// レスポンス時間の評価
	fmt.Println("\n   📋 パフォーマンス評価:")
	if avgTime.Seconds() < 20 {
		fmt.Printf("      ✅ 優秀（20秒未満）- ユーザー体験良好\n")
	} else if avgTime.Seconds() < 40 {
		fmt.Printf("      ⚠️  標準（20-40秒）- 許容範囲\n")
	} else {
		fmt.Printf("      ❌ 改善要（40秒以上）- 最適化が必要\n")
	}
}

func displayCoreProposals(proposals []model.RouteProposal, testType string) {
	fmt.Printf("\n   📋 %s プロポーザル詳細:\n", testType)

	if len(proposals) == 0 {
		fmt.Printf("   ⚠️  プロポーザルが見つかりませんでした\n")
		return
	}

	// 最初のプロポーザルの詳細表示
	proposal := proposals[0]
	fmt.Printf("   📍 プロポーザル 1:\n")
	fmt.Printf("      🆔 ID: %s\n", proposal.ProposalID)
	fmt.Printf("      📝 タイトル: %s (%d文字)\n", proposal.Title, len(proposal.Title))
	fmt.Printf("      ⏱️  予想時間: %d分\n", proposal.EstimatedDurationMinutes)
	fmt.Printf("      📏 予想距離: %dm\n", proposal.EstimatedDistanceMeters)
	fmt.Printf("      🌿 テーマ: %s\n", proposal.Theme)
	fmt.Printf("      ✨ ハイライト数: %d\n", len(proposal.DisplayHighlights))
	fmt.Printf("      🗺️  ナビステップ数: %d\n", len(proposal.NavigationSteps))
	fmt.Printf("      📖 物語文字数: %d文字\n", len(proposal.GeneratedStory))

	// 物語の一部を表示
	if len(proposal.GeneratedStory) > 150 {
		fmt.Printf("      📖 物語抜粋: %s...\n", proposal.GeneratedStory[:150])
	} else {
		fmt.Printf("      📖 物語: %s\n", proposal.GeneratedStory)
	}

	// ハイライトの表示
	if len(proposal.DisplayHighlights) > 0 {
		fmt.Printf("      ✨ ハイライト:\n")
		for i, highlight := range proposal.DisplayHighlights {
			if i >= 3 { // 最初の3つのみ表示
				fmt.Printf("         ... (他%d件)\n", len(proposal.DisplayHighlights)-3)
				break
			}
			fmt.Printf("         - %s\n", highlight)
		}
	}

	if len(proposals) > 1 {
		fmt.Printf("   ... (他 %d プロポーザル)\n", len(proposals)-1)
	}
}

func analyzeCorePerformance(duration time.Duration, proposalCount int) {
	fmt.Printf("\n   📊 パフォーマンス分析:\n")
	fmt.Printf("      ⏱️  総実行時間: %.2f秒\n", duration.Seconds())
	fmt.Printf("      🎯 生成プロポーザル数: %d件\n", proposalCount)
	
	if proposalCount > 0 {
		avgTimePerProposal := duration.Seconds() / float64(proposalCount)
		fmt.Printf("      📈 1プロポーザルあたり: %.2f秒\n", avgTimePerProposal)
	}
	
	// パフォーマンス評価
	if duration.Seconds() < 20 {
		fmt.Printf("      ✅ 高速（20秒未満）- ユーザビリティ良好\n")
	} else if duration.Seconds() < 40 {
		fmt.Printf("      ⚠️  標準（20-40秒）- 許容範囲\n")
	} else {
		fmt.Printf("      ❌ 低速（40秒以上）- 改善が必要\n")
	}
}
