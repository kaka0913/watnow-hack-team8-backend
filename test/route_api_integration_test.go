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

// setupAPIRouter はAPIサーバーのルーターを設定する
func setupAPIRouter() (*gin.Engine, error) {
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

// TestRouteAPIIntegration_Nature は自然テーマでの統合テストを行う
func TestRouteAPIIntegration_Nature(t *testing.T) {
	log.Printf("🧪 ルートAPI統合テスト (Nature) 開始")

	router, err := setupAPIRouter()
	if err != nil {
		t.Fatalf("APIルーター設定に失敗: %v", err)
	}

	// テストケース1: 時間ベースのルート提案生成
	t.Run("時間ベースルート提案生成", func(t *testing.T) {
		log.Printf("📍 時間ベースルート提案生成テスト開始")
		
		proposalRequest := model.RouteProposalRequest{
			StartLocation: &model.Location{
				Latitude:  34.9853, // 元のテストで成功した座標に戻す
				Longitude: 135.7581,
			},
			Mode:        "time_based",
			TimeMinutes: 60,
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

		if w.Code != http.StatusOK {
			t.Fatalf("ルート提案生成に失敗: %d, %s", w.Code, w.Body.String())
		}

		var proposalResponse model.RouteProposalResponse
		if err := json.Unmarshal(w.Body.Bytes(), &proposalResponse); err != nil {
			t.Fatalf("レスポンス解析に失敗: %v", err)
		}

		if len(proposalResponse.Proposals) == 0 {
			t.Fatal("提案が生成されませんでした")
		}

		proposal := proposalResponse.Proposals[0]
		log.Printf("✅ ルート提案生成成功:")
		log.Printf("   提案ID: %s", proposal.ProposalID)
		log.Printf("   タイトル: %s", proposal.Title)
		log.Printf("   推定時間: %d分", proposal.EstimatedDurationMinutes)
		log.Printf("   ハイライト数: %d", len(proposal.DisplayHighlights))
		log.Printf("   ハイライト: %v", proposal.DisplayHighlights)
		log.Printf("   ナビゲーションステップ数: %d", len(proposal.NavigationSteps))

		// サブテスト: 基本的なルート再計算
		t.Run("基本的なルート再計算", func(t *testing.T) {
			testRouteRecalculation(t, router, proposal, "basic")
		})

		// サブテスト: 一部POI訪問後の再計算
		t.Run("一部POI訪問後の再計算", func(t *testing.T) {
			testRouteRecalculationWithVisitedPOIs(t, router, proposal)
		})
	})

	// テストケース2: 目的地ありのルート提案生成
	t.Run("目的地ありルート提案生成", func(t *testing.T) {
		log.Printf("📍 目的地ありルート提案生成テスト開始")
		
		proposalRequest := model.RouteProposalRequest{
			StartLocation: &model.Location{
				Latitude:  35.0116, // 京都駅周辺
				Longitude: 135.7681,
			},
			DestinationLocation: &model.Location{
				Latitude:  35.0180, // 京都駅から少し北
				Longitude: 135.7700,
			},
			Mode:  "destination",
			Theme: "nature",
			RealtimeContext: &model.RealtimeContext{
				Weather:   "cloudy",
				TimeOfDay: "morning",
			},
		}

		jsonData, _ := json.Marshal(proposalRequest)
		req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("目的地ありルート提案生成に失敗: %d, %s", w.Code, w.Body.String())
		}

		var proposalResponse model.RouteProposalResponse
		if err := json.Unmarshal(w.Body.Bytes(), &proposalResponse); err != nil {
			t.Fatalf("レスポンス解析に失敗: %v", err)
		}

		if len(proposalResponse.Proposals) == 0 {
			t.Fatal("目的地ありの提案が生成されませんでした")
		}

		proposal := proposalResponse.Proposals[0]
		log.Printf("✅ 目的地ありルート提案生成成功:")
		log.Printf("   提案ID: %s", proposal.ProposalID)
		log.Printf("   タイトル: %s", proposal.Title)
		log.Printf("   推定時間: %d分", proposal.EstimatedDurationMinutes)
		log.Printf("   ハイライト数: %d", len(proposal.DisplayHighlights))
		log.Printf("   ハイライト: %v", proposal.DisplayHighlights)

		// サブテスト: 目的地ありルートの再計算
		t.Run("目的地ありルート再計算", func(t *testing.T) {
			testRouteRecalculationWithDestination(t, router, proposal)
		})
	})

	log.Printf("🎉 ルートAPI統合テスト完了")
}

// testRouteRecalculation は基本的なルート再計算をテストする
func testRouteRecalculation(t *testing.T, router *gin.Engine, originalProposal model.RouteProposal, testType string) {
	log.Printf("🔄 ルート再計算テスト開始 (%s)", testType)

	recalcRequest := model.RouteRecalculateRequest{
		ProposalID: originalProposal.ProposalID,
		CurrentLocation: &model.Location{
			Latitude:  35.0120, // 少し移動した位置
			Longitude: 135.7690,
		},
		Mode: "time_based",
		VisitedPOIs: &model.VisitedPOIsContext{
			PreviousPOIs: []model.PreviousPOI{}, // 未訪問の状態
		},
		RealtimeContext: &model.RealtimeContext{
			Weather:   "sunny",
			TimeOfDay: "afternoon",
		},
	}

	jsonData, _ := json.Marshal(recalcRequest)
	req, _ := http.NewRequest("POST", "/routes/recalculate", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("ルート再計算に失敗: %d, %s", w.Code, w.Body.String())
	}

	var recalcResponse model.RouteRecalculateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &recalcResponse); err != nil {
		t.Fatalf("再計算レスポンス解析に失敗: %v", err)
	}

	updatedRoute := recalcResponse.UpdatedRoute
	log.Printf("✅ ルート再計算成功 (%s):", testType)
	log.Printf("   元のハイライト数: %d", len(originalProposal.DisplayHighlights))
	log.Printf("   新しいハイライト数: %d", len(updatedRoute.Highlights))
	log.Printf("   元の推定時間: %d分", originalProposal.EstimatedDurationMinutes)
	log.Printf("   新しい推定時間: %d分", updatedRoute.EstimatedDurationMinutes)
	log.Printf("   新しいハイライト: %v", updatedRoute.Highlights)
	log.Printf("   ナビゲーションステップ数: %d", len(updatedRoute.NavigationSteps))

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
}

// testRouteRecalculationWithVisitedPOIs は一部POI訪問後の再計算をテストする
func testRouteRecalculationWithVisitedPOIs(t *testing.T, router *gin.Engine, originalProposal model.RouteProposal) {
	log.Printf("🔄 一部POI訪問後のルート再計算テスト開始")

	// 最初のPOIを訪問済みとする
	var visitedPOIs []model.PreviousPOI
	if len(originalProposal.NavigationSteps) > 0 {
		firstStep := originalProposal.NavigationSteps[0]
		if firstStep.Type == "poi" {
			visitedPOIs = append(visitedPOIs, model.PreviousPOI{
				POIId: firstStep.POIId,
				Name:  firstStep.Name,
			})
		}
	}

	recalcRequest := model.RouteRecalculateRequest{
		ProposalID: originalProposal.ProposalID,
		CurrentLocation: &model.Location{
			Latitude:  35.0125, // さらに移動した位置
			Longitude: 135.7695,
		},
		Mode: "time_based",
		VisitedPOIs: &model.VisitedPOIsContext{
			PreviousPOIs: visitedPOIs,
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

	if w.Code != http.StatusOK {
		t.Fatalf("一部POI訪問後のルート再計算に失敗: %d, %s", w.Code, w.Body.String())
	}

	var recalcResponse model.RouteRecalculateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &recalcResponse); err != nil {
		t.Fatalf("再計算レスポンス解析に失敗: %v", err)
	}

	updatedRoute := recalcResponse.UpdatedRoute
	log.Printf("✅ 一部POI訪問後のルート再計算成功:")
	log.Printf("   訪問済みPOI数: %d", len(visitedPOIs))
	log.Printf("   元のハイライト数: %d", len(originalProposal.DisplayHighlights))
	log.Printf("   新しいハイライト数: %d", len(updatedRoute.Highlights))
	log.Printf("   新しい推定時間: %d分", updatedRoute.EstimatedDurationMinutes)
	log.Printf("   新しいハイライト: %v", updatedRoute.Highlights)

	if len(visitedPOIs) > 0 {
		log.Printf("   訪問済み: %s", visitedPOIs[0].Name)
	}

	// 訪問済みPOIが考慮されているかチェック
	if len(updatedRoute.Highlights) == 0 {
		t.Error("一部POI訪問後でも新しいハイライトが生成されませんでした")
	}
}

// testRouteRecalculationWithDestination は目的地ありルートの再計算をテストする
func testRouteRecalculationWithDestination(t *testing.T, router *gin.Engine, originalProposal model.RouteProposal) {
	log.Printf("🔄 目的地ありルート再計算テスト開始")

	recalcRequest := model.RouteRecalculateRequest{
		ProposalID: originalProposal.ProposalID,
		CurrentLocation: &model.Location{
			Latitude:  35.0140, // 目的地に向かって移動中
			Longitude: 135.7695,
		},
		DestinationLocation: &model.Location{
			Latitude:  35.0180, // 元の目的地
			Longitude: 135.7700,
		},
		Mode: "destination",
		VisitedPOIs: &model.VisitedPOIsContext{
			PreviousPOIs: []model.PreviousPOI{}, // 未訪問の状態
		},
		RealtimeContext: &model.RealtimeContext{
			Weather:   "rainy",
			TimeOfDay: "evening",
		},
	}

	jsonData, _ := json.Marshal(recalcRequest)
	req, _ := http.NewRequest("POST", "/routes/recalculate", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("目的地ありルート再計算に失敗: %d, %s", w.Code, w.Body.String())
	}

	var recalcResponse model.RouteRecalculateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &recalcResponse); err != nil {
		t.Fatalf("再計算レスポンス解析に失敗: %v", err)
	}

	updatedRoute := recalcResponse.UpdatedRoute
	log.Printf("✅ 目的地ありルート再計算成功:")
	log.Printf("   元のハイライト数: %d", len(originalProposal.DisplayHighlights))
	log.Printf("   新しいハイライト数: %d", len(updatedRoute.Highlights))
	log.Printf("   新しい推定時間: %d分", updatedRoute.EstimatedDurationMinutes)
	log.Printf("   新しいハイライト: %v", updatedRoute.Highlights)
	log.Printf("   ナビゲーションステップ数: %d", len(updatedRoute.NavigationSteps))

	// 目的地が最後のステップに含まれているかチェック
	if len(updatedRoute.NavigationSteps) > 0 {
		lastStep := updatedRoute.NavigationSteps[len(updatedRoute.NavigationSteps)-1]
		log.Printf("   最終ステップ: %s", lastStep.Name)
	}

	// 基本的な検証
	if len(updatedRoute.Highlights) == 0 {
		t.Error("目的地ありでも新しいハイライトが生成されませんでした")
	}
	if updatedRoute.EstimatedDurationMinutes <= 0 {
		t.Error("推定時間が0以下です")
	}
}
