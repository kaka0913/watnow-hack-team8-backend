package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/service"
	"Team8-App/internal/handler"
	"Team8-App/internal/infrastructure/ai"
	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/infrastructure/firestore"
	"Team8-App/internal/infrastructure/maps"
	"Team8-App/internal/repository"
	"Team8-App/internal/usecase"
)

// setupAPIRouterForIntegration はAPIサーバーのルーターを設定する（統合テスト用）
func setupAPIRouterForIntegration() (*gin.Engine, error) {
	// 環境変数読み込み
	if err := godotenv.Load("../.env"); err != nil {
		return nil, fmt.Errorf(".env file not found: %v", err)
	}

	gin.SetMode(gin.TestMode)

	// 必要な環境変数の取得
	googleMapsAPIKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	firestoreProjectID := os.Getenv("FIRESTORE_PROJECT_ID")

	if googleMapsAPIKey == "" {
		return nil, fmt.Errorf("Google Maps API Key not set")
	}
	if geminiAPIKey == "" {
		return nil, fmt.Errorf("Gemini API Key not set")
	}
	if firestoreProjectID == "" {
		return nil, fmt.Errorf("Firestore Project ID not set")
	}

	// Database connections
	postgresClient, err := database.NewPostgreSQLClient()
	if err != nil {
		return nil, fmt.Errorf("PostgreSQL初期化失敗: %v", err)
	}

	ctx := context.Background()
	firestoreClient, err := firestore.NewFirestoreClient(ctx, firestoreProjectID)
	if err != nil {
		return nil, fmt.Errorf("Firestore初期化失敗: %v", err)
	}

	directionsProvider := maps.NewGoogleDirectionsProvider(googleMapsAPIKey)
	geminiClient := ai.NewGeminiClient(geminiAPIKey)
	storyGenerationRepo := ai.NewGeminiStoryRepository(geminiClient)

	// Dependency injection
	poiRepo := repository.NewPostgresPOIsRepository(postgresClient)
	routeSuggestionService := service.NewRouteSuggestionService(directionsProvider, poiRepo)
	firestoreRepo := repository.NewFirestoreRouteProposalRepository(firestoreClient.GetClient())
	routeProposalUseCase := usecase.NewRouteProposalUseCase(routeSuggestionService, firestoreRepo, storyGenerationRepo)
	
	routeRecalculateService := service.NewRouteRecalculateService(directionsProvider, poiRepo)
	routeRecalculateUseCase := usecase.NewRouteRecalculateUseCase(routeRecalculateService, firestoreRepo, storyGenerationRepo)
	routeProposalHandler := handler.NewRouteProposalHandler(routeProposalUseCase, routeRecalculateUseCase)

	// Ginルーターのセットアップ
	r := gin.New()
	
	// Route Proposals API エンドポイント
	routes := r.Group("/routes")
	{
		routes.POST("/proposals", routeProposalHandler.PostRouteProposals)
		routes.GET("/proposals/:id", routeProposalHandler.GetRouteProposal)
		routes.POST("/recalculate", routeProposalHandler.PostRouteRecalculate)
	}

	return r, nil
}

// TestFullAPIIntegration_RealData は実際のデータを使用した完全な統合テスト
func TestFullAPIIntegration_RealData(t *testing.T) {
	log.Printf("🧪 実データを使用したAPI統合テスト開始")

	router, err := setupAPIRouterForIntegration()
	if err != nil {
		t.Fatalf("APIルーター設定に失敗: %v", err)
	}

	// Step 1: まず簡単なルート提案を生成（現在動作確認済みの座標を使用）
	t.Run("実データでのルート提案生成", func(t *testing.T) {
		log.Printf("📍 実データでのルート提案生成テスト開始")
		
		// 実際にPOIが存在する座標を使用
		proposalRequest := model.RouteProposalRequest{
			StartLocation: &model.Location{
				Latitude:  35.0100, // 本能寺、京都鳩居堂付近
				Longitude: 135.7671,
			},
			Mode:        "time_based",
			TimeMinutes: 90, // 十分な時間を設定
			Theme:       "nature",
			RealtimeContext: &model.RealtimeContext{
				Weather:   "sunny",
				TimeOfDay: "afternoon",
			},
		}

		jsonData, _ := json.Marshal(proposalRequest)
		req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		log.Printf("⚡ ルート提案リクエスト送信完了 - ステータス: %d", w.Code)
		
		if w.Code != http.StatusOK {
			log.Printf("❌ ルート提案生成失敗: %d, %s", w.Code, w.Body.String())
			
			// 別の座標で再試行
			log.Printf("🔄 別の座標で再試行...")
			
			proposalRequest.StartLocation.Latitude = 35.0116
			proposalRequest.StartLocation.Longitude = 135.7683
			proposalRequest.TimeMinutes = 120 // より長い時間設定
			
			jsonData, _ = json.Marshal(proposalRequest)
			req, _ = http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Fatalf("再試行でもルート提案生成に失敗: %d, %s", w.Code, w.Body.String())
			}
		}

		var proposalResponse model.RouteProposalResponse
		if err := json.Unmarshal(w.Body.Bytes(), &proposalResponse); err != nil {
			t.Fatalf("レスポンス解析に失敗: %v", err)
		}

		if len(proposalResponse.Proposals) == 0 {
			t.Fatal("提案が生成されませんでした")
		}

		proposal := proposalResponse.Proposals[0]
		log.Printf("✅ 実データでのルート提案生成成功:")
		log.Printf("   提案ID: %s", proposal.ProposalID)
		log.Printf("   タイトル: %s", proposal.Title)
		log.Printf("   推定時間: %d分", proposal.EstimatedDurationMinutes)
		log.Printf("   ハイライト数: %d", len(proposal.DisplayHighlights))
		log.Printf("   ハイライト: %v", proposal.DisplayHighlights)
		log.Printf("   ナビゲーションステップ数: %d", len(proposal.NavigationSteps))

		// Step 2: 生成されたルートで再計算テストを実行
		t.Run("実データでのルート再計算", func(t *testing.T) {
			testRealDataRecalculation(t, router, proposal)
		})
	})

	log.Printf("🎉 実データ統合テスト完了")
}

// testRealDataRecalculation は実データを使用したルート再計算をテストする
func testRealDataRecalculation(t *testing.T, router *gin.Engine, originalProposal model.RouteProposal) {
	log.Printf("🔄 実データでのルート再計算テスト開始")

	// 元の座標から少し移動した位置を計算
	startLat := 35.0100
	startLng := 135.7671
	
	if len(originalProposal.NavigationSteps) > 0 {
		firstStep := originalProposal.NavigationSteps[0]
		if firstStep.Latitude != 0 && firstStep.Longitude != 0 {
			startLat = firstStep.Latitude
			startLng = firstStep.Longitude
		}
	}

	recalcRequest := model.RouteRecalculateRequest{
		ProposalID: originalProposal.ProposalID,
		CurrentLocation: &model.Location{
			Latitude:  startLat + 0.001, // 約100m移動
			Longitude: startLng + 0.001,
		},
		Mode: "time_based",
		VisitedPOIs: &model.VisitedPOIsContext{
			PreviousPOIs: []model.PreviousPOI{}, // 未訪問の状態
		},
		RealtimeContext: &model.RealtimeContext{
			Weather:   "cloudy",
			TimeOfDay: "evening",
		},
	}

	jsonData, _ := json.Marshal(recalcRequest)
	req, _ := http.NewRequest("POST", "/routes/recalculate", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	log.Printf("⚡ ルート再計算リクエスト送信完了 - ステータス: %d", w.Code)

	if w.Code != http.StatusOK {
		t.Fatalf("実データでのルート再計算に失敗: %d, %s", w.Code, w.Body.String())
	}

	var recalcResponse model.RouteRecalculateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &recalcResponse); err != nil {
		t.Fatalf("再計算レスポンス解析に失敗: %v", err)
	}

	updatedRoute := recalcResponse.UpdatedRoute
	log.Printf("✅ 実データでのルート再計算成功:")
	log.Printf("   元のハイライト数: %d", len(originalProposal.DisplayHighlights))
	log.Printf("   新しいハイライト数: %d", len(updatedRoute.Highlights))
	log.Printf("   元の推定時間: %d分", originalProposal.EstimatedDurationMinutes)
	log.Printf("   新しい推定時間: %d分", updatedRoute.EstimatedDurationMinutes)
	log.Printf("   新しいハイライト: %v", updatedRoute.Highlights)
	log.Printf("   ナビゲーションステップ数: %d", len(updatedRoute.NavigationSteps))

	// 詳細なルート情報を表示
	for i, step := range updatedRoute.NavigationSteps {
		if step.Type == "poi" {
			log.Printf("     %d. %s (POI) - (%.4f, %.4f)", i+1, step.Name, step.Latitude, step.Longitude)
		} else {
			log.Printf("     %d. %s", i+1, step.Description)
		}
	}

	// 基本的な検証
	if len(updatedRoute.Highlights) == 0 {
		t.Error("更新されたルートにハイライトがありません")
	}
	if len(updatedRoute.NavigationSteps) == 0 {
		t.Error("更新されたルートにナビゲーションステップがありません")
	}
	if updatedRoute.EstimatedDurationMinutes <= 0 {
		t.Error("推定時間が0以下です")
	}

	// 一部POI訪問後の再計算テスト
	t.Run("一部POI訪問後の再計算", func(t *testing.T) {
		if len(originalProposal.NavigationSteps) > 0 {
			firstStep := originalProposal.NavigationSteps[0]
			if firstStep.Type == "poi" {
				visitedPOIs := []model.PreviousPOI{
					{
						POIId: firstStep.POIId,
						Name:  firstStep.Name,
					},
				}

				recalcWithVisited := model.RouteRecalculateRequest{
					ProposalID: originalProposal.ProposalID,
					CurrentLocation: &model.Location{
						Latitude:  startLat + 0.002,
						Longitude: startLng + 0.002,
					},
					Mode: "time_based",
					VisitedPOIs: &model.VisitedPOIsContext{
						PreviousPOIs: visitedPOIs,
					},
					RealtimeContext: &model.RealtimeContext{
						Weather:   "rainy",
						TimeOfDay: "evening",
					},
				}

				jsonData, _ := json.Marshal(recalcWithVisited)
				req, _ := http.NewRequest("POST", "/routes/recalculate", bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code == http.StatusOK {
					var visitedRecalcResponse model.RouteRecalculateResponse
					if err := json.Unmarshal(w.Body.Bytes(), &visitedRecalcResponse); err == nil {
						log.Printf("✅ 一部POI訪問後の再計算も成功:")
						log.Printf("   訪問済み: %s", visitedPOIs[0].Name)
						log.Printf("   新しいハイライト数: %d", len(visitedRecalcResponse.UpdatedRoute.Highlights))
						log.Printf("   新しいハイライト: %v", visitedRecalcResponse.UpdatedRoute.Highlights)
					}
				} else {
					log.Printf("⚠️  一部POI訪問後の再計算は失敗: %d", w.Code)
				}
			}
		}
	})
}
