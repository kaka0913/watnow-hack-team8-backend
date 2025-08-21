package service

import (
	"Team8-App/internal/domain/model"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ParallelRouteOptimizer はGoogle Maps APIの並行処理による高速ルート最適化
type ParallelRouteOptimizer struct {
	directionsProvider interface {
		GetWalkingRoute(ctx context.Context, origin model.LatLng, waypoints ...model.LatLng) (*model.RouteDetails, error)
	}
	maxGoroutines int
}

// NewParallelRouteOptimizer は新しい並行ルート最適化インスタンスを作成
func NewParallelRouteOptimizer(directionsProvider interface {
	GetWalkingRoute(ctx context.Context, origin model.LatLng, waypoints ...model.LatLng) (*model.RouteDetails, error)
}) *ParallelRouteOptimizer {
	return &ParallelRouteOptimizer{
		directionsProvider: directionsProvider,
		maxGoroutines:      5, // 同時実行数を制限
	}
}

// RouteCandidate はルート候補
type RouteCandidate struct {
	Route     []*model.POI
	Duration  time.Duration
	Details   *model.RouteDetails
	Error     error
}

// OptimizeRouteParallel は複数ルート候補を並行で評価し、最適なルートを選択
func (p *ParallelRouteOptimizer) OptimizeRouteParallel(ctx context.Context, name string, userLocation model.LatLng, combinations [][]*model.POI) (*model.SuggestedRoute, error) {
	if len(combinations) == 0 {
		return nil, fmt.Errorf("ルート候補が存在しません")
	}

	log.Printf("🚀 並行ルート最適化開始: %d候補を並行評価", len(combinations))
	start := time.Now()

	// セマフォを使用して同時実行数を制限
	semaphore := make(chan struct{}, p.maxGoroutines)
	results := make(chan RouteCandidate, len(combinations))
	var wg sync.WaitGroup

	// 各ルート候補を並行で評価
	for i, combination := range combinations {
		wg.Add(1)
		go func(routeIndex int, route []*model.POI) {
			defer wg.Done()
			
			// セマフォを取得（同時実行数制限）
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			candidate := RouteCandidate{Route: route}

			// 有効性チェック
			if len(route) == 0 {
				candidate.Error = fmt.Errorf("空のルート")
				results <- candidate
				return
			}

			// waypointsを構築
			waypointLatLngs := make([]model.LatLng, len(route))
			for j, poi := range route {
				if poi == nil {
					candidate.Error = fmt.Errorf("nilなPOIが含まれています")
					results <- candidate
					return
				}
				waypointLatLngs[j] = poi.ToLatLng()
			}

			// Google Maps API呼び出し
			routeDetails, err := p.directionsProvider.GetWalkingRoute(ctx, userLocation, waypointLatLngs...)
			if err != nil {
				candidate.Error = fmt.Errorf("ルート%d取得失敗: %w", routeIndex, err)
				results <- candidate
				return
			}

			// 制約チェック（2時間以内）
			maxDurationMinutes := 120
			if int(routeDetails.TotalDuration.Minutes()) > maxDurationMinutes {
				candidate.Error = fmt.Errorf("ルート%d: 時間制限超過(%d分)", routeIndex, int(routeDetails.TotalDuration.Minutes()))
				results <- candidate
				return
			}

			candidate.Duration = routeDetails.TotalDuration
			candidate.Details = routeDetails
			results <- candidate

		}(i, combination)
	}

	// 別のgoroutineでwaitしてチャンネルを閉じる
	go func() {
		wg.Wait()
		close(results)
	}()

	// 結果を収集して最適なルートを選択
	var bestCandidate *RouteCandidate
	var shortestDuration = time.Duration(24 * time.Hour)
	successCount := 0
	errorCount := 0

	for candidate := range results {
		if candidate.Error != nil {
			errorCount++
			log.Printf("⚠️  ルート評価エラー: %v", candidate.Error)
			continue
		}

		successCount++
		if candidate.Duration < shortestDuration {
			shortestDuration = candidate.Duration
			bestCandidate = &candidate
		}
	}

	duration := time.Since(start)
	log.Printf("✅ 並行ルート最適化完了: %v (成功:%d, 失敗:%d)", duration, successCount, errorCount)

	if bestCandidate == nil {
		return nil, fmt.Errorf("すべてのルート候補で評価に失敗しました (成功:%d, 失敗:%d)", successCount, errorCount)
	}

	// 最適なルートを構築
	bestRoute := &model.SuggestedRoute{
		Name:          fmt.Sprintf("%s (%d分)", name, int(bestCandidate.Duration.Minutes())),
		Spots:         bestCandidate.Route,
		TotalDuration: bestCandidate.Duration,
		Polyline:      bestCandidate.Details.Polyline,
	}

	log.Printf("🏆 最適ルート選択: %d分, %d箇所", int(bestCandidate.Duration.Minutes()), len(bestCandidate.Route))
	return bestRoute, nil
}
