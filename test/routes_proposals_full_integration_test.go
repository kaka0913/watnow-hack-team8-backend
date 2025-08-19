package test

import (
	"bytes"
	"context"
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
	"Team8-App/internal/infrastructure/firestore"
	"Team8-App/internal/infrastructure/maps"
	"Team8-App/internal/repository"
	"Team8-App/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func TestRoutesProposalsFullIntegration(t *testing.T) {
	fmt.Println("🚀 POST /routes/proposals 完全統合テスト（Firestore保存込み）")
	fmt.Println("============================================================")

	// .envファイルの読み込み
	err := godotenv.Load("../.env")
	if err != nil {
		t.Logf("⚠️  .envファイルの読み込みに失敗: %v", err)
	}

	// 必要な環境変数のチェック
	requiredEnvVars := map[string]string{
		"GEMINI_API_KEY":        os.Getenv("GEMINI_API_KEY"),
		"GOOGLE_MAPS_API_KEY":   os.Getenv("GOOGLE_MAPS_API_KEY"),
		"FIRESTORE_PROJECT_ID":  os.Getenv("FIRESTORE_PROJECT_ID"),
		"POSTGRES_URL":          os.Getenv("POSTGRES_URL"),
	}

	for varName, value := range requiredEnvVars {
		if value == "" {
			t.Skipf("⚠️  %s が設定されていません。テストをスキップします。", varName)
		} else {
			fmt.Printf("✅ %s: 設定済み\n", varName)
		}
	}

	// 各種クライアントの初期化
	ctx := context.Background()

	// PostgreSQL
	postgresClient, err := database.NewPostgreSQLClient()
	if err != nil {
		t.Fatalf("❌ PostgreSQL接続エラー: %v", err)
	}
	defer postgresClient.Close()

	// Firestore
	firestoreClient, err := firestore.NewFirestoreClient(ctx, requiredEnvVars["FIRESTORE_PROJECT_ID"])
	if err != nil {
		t.Fatalf("❌ Firestore接続エラー: %v", err)
	}
	defer firestoreClient.Close()

	// Google Maps
	directionsProvider := maps.NewGoogleDirectionsProvider(requiredEnvVars["GOOGLE_MAPS_API_KEY"])

	// Gemini AI
	geminiClient := ai.NewGeminiClient(requiredEnvVars["GEMINI_API_KEY"])
	storyGenerator := ai.NewGeminiStoryRepository(geminiClient)

	fmt.Println("✅ 全データベース・API接続成功")

	// サービス・リポジトリ・ユースケースの初期化
	poiRepo := repository.NewPostgresPOIsRepository(postgresClient)
	routeSuggestionService := service.NewRouteSuggestionService(directionsProvider, poiRepo)
	firestoreRepo := repository.NewFirestoreRouteProposalRepository(firestoreClient.GetClient())

	routeProposalUseCase := usecase.NewRouteProposalUseCase(
		routeSuggestionService,
		firestoreRepo,
		storyGenerator,
	)

	// ハンドラーの初期化
	routeProposalHandler := handler.NewRouteProposalHandler(routeProposalUseCase)

	// Ginエンジンのセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/routes/proposals", routeProposalHandler.PostRouteProposals)
	router.GET("/routes/proposals/:id", routeProposalHandler.GetRouteProposal)

	t.Run("完全な時間ベース散歩テスト", func(t *testing.T) {
		testFullTimeBasedProposal(t, router)
	})

	t.Run("完全な目的地ベーステスト", func(t *testing.T) {
		testFullDestinationBasedProposal(t, router)
	})

	t.Run("Firestore保存・取得テスト", func(t *testing.T) {
		testFirestoreIntegration(t, router)
	})

	t.Run("パフォーマンステスト", func(t *testing.T) {
		testPerformanceMetrics(t, router)
	})

	fmt.Println("============================================================")
	fmt.Println("🎉 POST /routes/proposals 完全統合テスト完了")
}

func testFullTimeBasedProposal(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 完全な時間ベース散歩テスト（全機能統合）")

	request := model.RouteProposalRequest{
		StartLocation: &model.Location{
			Latitude:  35.0047, // 四条河原町
			Longitude: 135.7700,
		},
		DestinationLocation: nil, // 目的地なし
		Mode:                "time_based",
		TimeMinutes:         60, // 1時間散歩
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
	fmt.Printf("   ☀️ 天気: %s, 時間帯: %s\n", 
		request.RealtimeContext.Weather, request.RealtimeContext.TimeOfDay)

	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("❌ JSONマーシャリングエラー: %v", err)
	}

	req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	
	// 詳細なタイミング測定
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
	displayDetailedProposals(response.Proposals, "完全統合時間ベース散歩")

	// パフォーマンス分析
	analyzePerformance(totalDuration, len(response.Proposals))
}

func testFullDestinationBasedProposal(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 完全な目的地ベーステスト（全機能統合）")

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
	fmt.Printf("   🌿 テーマ: %s\n", request.Theme)
	fmt.Printf("   ☁️ 天気: %s, 時間帯: %s\n", 
		request.RealtimeContext.Weather, request.RealtimeContext.TimeOfDay)

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

	displayDetailedProposals(response.Proposals, "完全統合目的地ベース")
	analyzePerformance(totalDuration, len(response.Proposals))
}

func testFirestoreIntegration(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 Firestore保存・取得統合テスト")

	// 1. プロポーザルを生成してFirestoreに保存
	request := model.RouteProposalRequest{
		StartLocation: &model.Location{
			Latitude:  35.0047,
			Longitude: 135.7700,
		},
		Mode:        "time_based",
		TimeMinutes: 30,
		Theme:       "nature",
		RealtimeContext: &model.RealtimeContext{
			Weather:   "sunny",
			TimeOfDay: "afternoon",
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("❌ JSONマーシャリングエラー: %v", err)
	}

	req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	
	fmt.Println("   ⏱️  プロポーザル生成・Firestore保存開始...")
	saveStartTime := time.Now()
	
	router.ServeHTTP(w, req)
	
	saveDuration := time.Since(saveStartTime)
	fmt.Printf("   ⏱️  保存完了時間: %.2f秒\n", saveDuration.Seconds())

	if w.Code != http.StatusOK {
		fmt.Printf("   ❌ 保存エラー: %s\n", w.Body.String())
		t.Errorf("プロポーザル生成・保存失敗: %d", w.Code)
		return
	}

	var response model.RouteProposalResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("❌ レスポンスのJSONパースエラー: %v", err)
		return
	}

	if len(response.Proposals) == 0 {
		t.Error("❌ プロポーザルが生成されませんでした")
		return
	}

	firstProposal := response.Proposals[0]
	fmt.Printf("   ✅ Firestore保存成功\n")
	fmt.Printf("   🆔 ProposalID: %s\n", firstProposal.ProposalID)
	fmt.Printf("   📝 タイトル: %s\n", firstProposal.Title)

	// 2. 保存されたプロポーザルをFirestoreから取得
	fmt.Println("\n   ⏱️  Firestore取得テスト開始...")
	getReq, _ := http.NewRequest("GET", "/routes/proposals/"+firstProposal.ProposalID, nil)
	getW := httptest.NewRecorder()
	
	getStartTime := time.Now()
	router.ServeHTTP(getW, getReq)
	getDuration := time.Since(getStartTime)
	
	fmt.Printf("   ⏱️  取得完了時間: %.2f秒\n", getDuration.Seconds())
	fmt.Printf("   📊 取得HTTPステータス: %d\n", getW.Code)

	if getW.Code != http.StatusOK {
		fmt.Printf("   ❌ 取得エラー: %s\n", getW.Body.String())
		t.Errorf("プロポーザル取得失敗: %d", getW.Code)
		return
	}

	var retrievedProposal model.RouteProposal
	err = json.Unmarshal(getW.Body.Bytes(), &retrievedProposal)
	if err != nil {
		t.Errorf("❌ 取得レスポンスのJSONパースエラー: %v", err)
		return
	}

	fmt.Printf("   ✅ Firestore取得成功\n")
	fmt.Printf("   📝 取得タイトル: %s\n", retrievedProposal.Title)
	fmt.Printf("   📖 物語文字数: %d文字\n", len(retrievedProposal.GeneratedStory))

	// データ整合性の確認
	if firstProposal.Title != retrievedProposal.Title {
		t.Errorf("❌ タイトル不一致: 保存=%s, 取得=%s", firstProposal.Title, retrievedProposal.Title)
	} else {
		fmt.Printf("   ✅ データ整合性確認完了\n")
	}

	fmt.Printf("   📊 Firestore往復時間: %.2f秒\n", (saveDuration + getDuration).Seconds())
}

func testPerformanceMetrics(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 パフォーマンス指標測定")

	testCases := []struct {
		name        string
		timeMinutes int
		theme       string
	}{
		{"短時間散歩", 15, "nature"},
		{"中時間散歩", 45, "nature"},
		{"長時間散歩", 90, "nature"},
	}

	var totalDurations []time.Duration
	var totalProposals int

	for _, testCase := range testCases {
		fmt.Printf("\n   🧪 %s (%d分)\n", testCase.name, testCase.timeMinutes)

		request := model.RouteProposalRequest{
			StartLocation: &model.Location{
				Latitude:  35.0047,
				Longitude: 135.7700,
			},
			Mode:        "time_based",
			TimeMinutes: testCase.timeMinutes,
			Theme:       testCase.theme,
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

		totalDurations = append(totalDurations, duration)

		fmt.Printf("      ⏱️  実行時間: %.2f秒\n", duration.Seconds())
		fmt.Printf("      📊 ステータス: %d\n", w.Code)

		if w.Code == http.StatusOK {
			var response model.RouteProposalResponse
			json.Unmarshal(w.Body.Bytes(), &response)
			proposalCount := len(response.Proposals)
			totalProposals += proposalCount
			fmt.Printf("      🎯 プロポーザル数: %d\n", proposalCount)
		}
	}

	// 総合パフォーマンス分析
	fmt.Println("\n   📊 総合パフォーマンス分析:")
	var totalTime time.Duration
	for _, d := range totalDurations {
		totalTime += d
	}
	avgTime := totalTime / time.Duration(len(totalDurations))
	
	fmt.Printf("      ⏱️  平均実行時間: %.2f秒\n", avgTime.Seconds())
	fmt.Printf("      🎯 総プロポーザル数: %d\n", totalProposals)
	fmt.Printf("      📈 プロポーザル生成効率: %.2f件/秒\n", 
		float64(totalProposals)/totalTime.Seconds())
}

func displayDetailedProposals(proposals []model.RouteProposal, testType string) {
	fmt.Printf("\n   📋 %s プロポーザル詳細:\n", testType)

	if len(proposals) == 0 {
		fmt.Printf("   ⚠️  プロポーザルが見つかりませんでした\n")
		return
	}

	// 最初のプロポーザルのみ詳細表示
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
	if len(proposal.GeneratedStory) > 200 {
		fmt.Printf("      📖 物語抜粋: %s...\n", proposal.GeneratedStory[:200])
	} else {
		fmt.Printf("      📖 物語: %s\n", proposal.GeneratedStory)
	}

	if len(proposals) > 1 {
		fmt.Printf("   ... (他 %d プロポーザル)\n", len(proposals)-1)
	}
}

func analyzePerformance(duration time.Duration, proposalCount int) {
	fmt.Printf("\n   📊 パフォーマンス分析:\n")
	fmt.Printf("      ⏱️  総実行時間: %.2f秒\n", duration.Seconds())
	fmt.Printf("      🎯 生成プロポーザル数: %d件\n", proposalCount)
	
	if proposalCount > 0 {
		avgTimePerProposal := duration.Seconds() / float64(proposalCount)
		fmt.Printf("      📈 1プロポーザルあたり: %.2f秒\n", avgTimePerProposal)
	}
	
	// パフォーマンス評価
	if duration.Seconds() < 30 {
		fmt.Printf("      ✅ 高速（30秒未満）\n")
	} else if duration.Seconds() < 60 {
		fmt.Printf("      ⚠️  標準（30-60秒）\n")
	} else {
		fmt.Printf("      ❌ 低速（60秒以上）\n")
	}
}
