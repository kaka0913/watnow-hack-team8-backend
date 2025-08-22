package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

// NatureRouteTestResult natureテスト結果の構造体
type NatureRouteTestResult struct {
	TestName    string
	Location    string
	TimeMinutes int
	Success     bool
	StatusCode  int
	Duration    time.Duration
	Response    model.RouteProposalResponse
	Error       string
}

func TestNatureRoutesExhaustive(t *testing.T) {
	fmt.Println("🌿 Natureテーマ専用・網羅的ルート生成テスト")
	fmt.Println("============================================================")

	// .envファイルの読み込み
	err := godotenv.Load("../.env")
	if err != nil {
		t.Logf("⚠️  .envファイルの読み込みに失敗: %v", err)
	}

	// 必要な環境変数のチェック
	requiredEnvVars := map[string]string{
		"GEMINI_API_KEY":       os.Getenv("GEMINI_API_KEY"),
		"GOOGLE_MAPS_API_KEY":  os.Getenv("GOOGLE_MAPS_API_KEY"),
		"FIRESTORE_PROJECT_ID": os.Getenv("FIRESTORE_PROJECT_ID"),
		"POSTGRES_URL":         os.Getenv("POSTGRES_URL"),
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

	// ハンドラーの初期化（recalculateUseCaseにはnilを渡す）
	routeProposalHandler := handler.NewRouteProposalHandler(routeProposalUseCase, nil)

	// Ginエンジンのセットアップ
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/routes/proposals", routeProposalHandler.PostRouteProposals)

	// テスト実行
	runNatureTestsExhaustive(t, router)
}

func runNatureTestsExhaustive(t *testing.T, router *gin.Engine) {
	fmt.Println("\n🎯 Nature テーマ・包括的テスト開始")

	// テスト地点の定義（京都内の主要スポット）
	locations := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{"京都駅", 35.0047, 135.7700},
		{"四条河原町", 35.0028, 135.7708},
		{"祇園四条", 35.0036, 135.7786},
		{"清水寺", 34.9949, 135.7850},
		{"金閣寺", 35.0394, 135.7292},
		{"嵐山", 35.0116, 135.6761},
		{"京都御所", 35.0236, 135.7625},
		{"二条城", 35.0142, 135.7481},
		{"銀閣寺", 35.0270, 135.7988},
		{"伏見稲荷大社", 34.9672, 135.7727},
	}

	// テスト時間（分）
	timeOptions := []int{15, 30, 45, 60, 90, 120}

	// 結果を保存するスライス
	var results []NatureRouteTestResult

	testCount := 0
	for _, location := range locations {
		for _, timeMinutes := range timeOptions {
			testCount++
			fmt.Printf("\n🧪 テスト %d: %s -> %d分散歩 (nature)\n", testCount, location.name, timeMinutes)

			// リクエスト作成
			requestBody := model.RouteProposalRequest{
				StartLocation: &model.Location{
					Latitude:  location.lat,
					Longitude: location.lng,
				},
				Mode:        "time_based",
				TimeMinutes: timeMinutes,
				Theme:       "nature",
				RealtimeContext: &model.RealtimeContext{
					Weather:   "sunny",
					TimeOfDay: "afternoon",
				},
			}

			jsonData, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("POST", "/routes/proposals", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			startTime := time.Now()
			router.ServeHTTP(w, req)
			duration := time.Since(startTime)

			// レスポンス解析
			var response model.RouteProposalResponse
			var result NatureRouteTestResult
			
			if w.Code == http.StatusOK {
				json.Unmarshal(w.Body.Bytes(), &response)
				result = NatureRouteTestResult{
					TestName:    fmt.Sprintf("%s_%d分", location.name, timeMinutes),
					Location:    location.name,
					TimeMinutes: timeMinutes,
					Success:     true,
					StatusCode:  w.Code,
					Duration:    duration,
					Response:    response,
				}
			} else {
				result = NatureRouteTestResult{
					TestName:    fmt.Sprintf("%s_%d分", location.name, timeMinutes),
					Location:    location.name,
					TimeMinutes: timeMinutes,
					Success:     false,
					StatusCode:  w.Code,
					Duration:    duration,
					Error:       w.Body.String(),
				}
			}

			results = append(results, result)

			// 結果表示
			if result.Success {
				fmt.Printf("   ⏱️  実行時間: %.2f秒 | ステータス: %d\n", result.Duration.Seconds(), result.StatusCode)
				fmt.Printf("   ✅ 成功 | プロポーザル数: %d\n", len(result.Response.Proposals))
				if len(result.Response.Proposals) > 0 {
					firstProposal := result.Response.Proposals[0]
					fmt.Printf("   📝 タイトル: %s\n", firstProposal.Title)
					fmt.Printf("   ⏱️  予想時間: %d分 | ステップ数: %d\n",
						firstProposal.EstimatedDurationMinutes, len(firstProposal.NavigationSteps))

					// POI数をカウント
					poiCount := 0
					for _, step := range firstProposal.NavigationSteps {
						if step.Type == "poi" {
							poiCount++
						}
					}
					fmt.Printf("   📍 訪問POI数: %d箇所\n", poiCount)
				}
			} else {
				fmt.Printf("   ⏱️  実行時間: %.2f秒 | ステータス: %d\n", result.Duration.Seconds(), result.StatusCode)
				fmt.Printf("   ❌ エラー: %s\n", result.Error)
			}

			// レート制限のため少し待機
			time.Sleep(500 * time.Millisecond)
		}
	}

	// 結果をMarkdownファイルに出力
	writeNatureTestResultsToMarkdown(results)

	// 統計情報を表示
	successCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}

	fmt.Printf("\n📊 Nature テスト結果: %d/%d 成功 (成功率: %.1f%%)\n",
		successCount, len(results), float64(successCount)/float64(len(results))*100)
}

// writeNatureTestResultsToMarkdown 結果をMarkdownファイルに書き込む
func writeNatureTestResultsToMarkdown(results []NatureRouteTestResult) {
	var md strings.Builder
	
	md.WriteString("# Natureテーマ ルート生成テスト結果\n\n")
	md.WriteString(fmt.Sprintf("実行日時: %s\n\n", time.Now().Format("2006年01月02日 15:04:05")))
	
	// 統計情報
	successCount := 0
	totalCount := len(results)
	for _, result := range results {
		if result.Success {
			successCount++
		}
	}
	
	md.WriteString("## 📊 テスト統計\n\n")
	md.WriteString(fmt.Sprintf("- **総テスト数**: %d\n", totalCount))
	md.WriteString(fmt.Sprintf("- **成功数**: %d\n", successCount))
	md.WriteString(fmt.Sprintf("- **失敗数**: %d\n", totalCount-successCount))
	md.WriteString(fmt.Sprintf("- **成功率**: %.1f%%\n\n", float64(successCount)/float64(totalCount)*100))
	
	// 成功したテストの詳細
	md.WriteString("## ✅ 成功したルート\n\n")
	
	for _, testResult := range results {
		if !testResult.Success {
			continue
		}
		
		md.WriteString(fmt.Sprintf("### %s (%d分散歩)\n\n", testResult.Location, testResult.TimeMinutes))
		
		response := testResult.Response
		if len(response.Proposals) == 0 {
			md.WriteString("プロポーザルが見つかりませんでした。\n\n")
			continue
		}
		
		for i, proposal := range response.Proposals {
			md.WriteString(fmt.Sprintf("#### ルート %d: %s\n\n", i+1, proposal.Title))
			md.WriteString(fmt.Sprintf("- **プロポーザルID**: %s\n", proposal.ProposalID))
			md.WriteString(fmt.Sprintf("- **予想時間**: %d分\n", proposal.EstimatedDurationMinutes))
			md.WriteString(fmt.Sprintf("- **予想距離**: %dm\n", proposal.EstimatedDistanceMeters))
			md.WriteString(fmt.Sprintf("- **ステップ数**: %d\n", len(proposal.NavigationSteps)))
			md.WriteString(fmt.Sprintf("- **テーマ**: %s\n", proposal.Theme))
			
			// ハイライト
			if len(proposal.DisplayHighlights) > 0 {
				md.WriteString("- **ハイライト**:\n")
				for _, highlight := range proposal.DisplayHighlights {
					md.WriteString(fmt.Sprintf("  - %s\n", highlight))
				}
			}
			
			if proposal.GeneratedStory != "" {
				md.WriteString(fmt.Sprintf("- **物語**: %s\n\n", proposal.GeneratedStory))
			}
			
			// POIの詳細
			md.WriteString("**訪問POI:**\n\n")
			poiFound := false
			for j, step := range proposal.NavigationSteps {
				if step.Type == "poi" {
					poiFound = true
					md.WriteString(fmt.Sprintf("%d. **%s**\n", j+1, step.Name))
					if step.POIId != "" {
						md.WriteString(fmt.Sprintf("   - **POI ID**: %s\n", step.POIId))
					}
					if step.Description != "" {
						md.WriteString(fmt.Sprintf("   - **説明**: %s\n", step.Description))
					}
					md.WriteString(fmt.Sprintf("   - **位置**: %.6f, %.6f\n", step.Latitude, step.Longitude))
					if step.DistanceToNextMeters > 0 {
						md.WriteString(fmt.Sprintf("   - **次への距離**: %dm\n", step.DistanceToNextMeters))
					}
					md.WriteString("\n")
				}
			}
			
			if !poiFound {
				md.WriteString("POIが見つかりませんでした。\n\n")
			}
			
			md.WriteString("---\n\n")
		}
	}
	
	// 失敗したテストの詳細
	md.WriteString("## ❌ 失敗したルート\n\n")
	
	failureFound := false
	for _, testResult := range results {
		if testResult.Success {
			continue
		}
		
		failureFound = true
		md.WriteString(fmt.Sprintf("### %s (%d分散歩)\n\n", testResult.Location, testResult.TimeMinutes))
		md.WriteString(fmt.Sprintf("- **エラー**: %s\n", testResult.Error))
		md.WriteString(fmt.Sprintf("- **ステータスコード**: %d\n", testResult.StatusCode))
		md.WriteString(fmt.Sprintf("- **実行時間**: %.2f秒\n\n", testResult.Duration.Seconds()))
	}
	
	if !failureFound {
		md.WriteString("失敗したルートはありませんでした。🎉\n\n")
	}
	
	// ファイルに書き込み
	filename := fmt.Sprintf("/Users/kaka/dev/Go/Team8-App/test/nature_route_test_results_%s.md", 
		time.Now().Format("20060102_150405"))
	
	err := os.WriteFile(filename, []byte(md.String()), 0644)
	if err != nil {
		fmt.Printf("❌ Markdownファイルの書き込みに失敗: %v\n", err)
		return
	}
	
	fmt.Printf("📝 結果をMarkdownファイルに保存しました: %s\n", filename)
}
