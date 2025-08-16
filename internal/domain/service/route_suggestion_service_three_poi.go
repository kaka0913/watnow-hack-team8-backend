package service

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/infrastructure/maps"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ThreePOIRouteSuggestionService は3つのPOIを巡るルート提案サービス
// スタート地点から3つのスポットを巡る4箇所のルートを提案する
type ThreePOIRouteSuggestionService struct {
	directionsProvider *maps.GoogleDirectionsProvider
	strategies         map[string]strategy.StrategyInterface
	routeBuilderHelper *RouteBuilderHelper
}

func NewThreePOIRouteSuggestionService(dp *maps.GoogleDirectionsProvider, strategies map[string]strategy.StrategyInterface, helper *RouteBuilderHelper) *ThreePOIRouteSuggestionService {
	return &ThreePOIRouteSuggestionService{
		directionsProvider: dp,
		strategies:         strategies,
		routeBuilderHelper: helper,
	}
}

// SuggestRoutesForMultipleScenarios は複数のシナリオから並行でルートを生成する
func (s *ThreePOIRouteSuggestionService) SuggestRoutesForMultipleScenarios(ctx context.Context, theme string, scenarios []string, userLocation model.LatLng) ([]*model.SuggestedRoute, error) {
	if len(scenarios) == 0 {
		return nil, errors.New("シナリオが指定されていません")
	}

	// テーマが有効かチェック
	_, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	resultsChan := make(chan scenarioResult, len(scenarios))
	var wg sync.WaitGroup

	// 各シナリオを並行処理で実行
	for _, scenario := range scenarios {
		wg.Add(1)
		go func(sc string) {
			defer wg.Done()
			routes, err := s.SuggestRoutesForScenario(ctx, theme, sc, userLocation)
			resultsChan <- scenarioResult{
				scenario: sc,
				routes:   routes,
				err:      err,
			}
		}(scenario)
	}

	// すべてのgoroutineの完了を待機
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// 結果を収集
	var allRoutes []*model.SuggestedRoute
	var errorMessages []string

	for result := range resultsChan {
		if result.err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("シナリオ '%s': %s", result.scenario, result.err.Error()))
		} else {
			allRoutes = append(allRoutes, result.routes...)
		}
	}

	// すべてのシナリオでエラーが発生した場合
	if len(allRoutes) == 0 {
		if len(errorMessages) > 0 {
			return nil, fmt.Errorf("すべてのシナリオでエラーが発生しました: %v", errorMessages)
		}
		return nil, errors.New("指定されたシナリオからルートを生成できませんでした")
	}

	return allRoutes, nil
}

// SuggestRoutesForScenario は順番が決まっていないPOIの組み合わせからルートを提案する
func (s *ThreePOIRouteSuggestionService) SuggestRoutesForScenario(ctx context.Context, theme string, scenario string, userLocation model.LatLng) ([]*model.SuggestedRoute, error) {
	// Step 1: 戦略を選択
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	var combinations [][]*model.POI
	var err error

	// Step 2: 戦略に組み合わせの生成を完全に委譲
	// 全ての戦略が統一されたインターフェースを持つため、テーマごとの分岐は不要
	combinations, err = selectedStrategy.FindCombinations(ctx, scenario, userLocation)
	if err != nil {
		return nil, err
	}

	if len(combinations) == 0 {
		return nil, errors.New("このシナリオに合うルートの組み合わせが見つかりませんでした")
	}

	// Step 3: 組み合わせからルート構築処理を実行（スタート地点 + 3つのスポット = 4箇所巡り）
	suggestedRoutes := s.buildRoutesFromCombinationsWithStartLocation(ctx, theme, scenario, userLocation, combinations)

	return suggestedRoutes, nil
}

// buildRoutesFromCombinationsWithStartLocation は、出発位置を考慮して複数の組み合わせから並行でルートを構築する
func (s *ThreePOIRouteSuggestionService) buildRoutesFromCombinationsWithStartLocation(ctx context.Context, theme string, scenario string, userLocation model.LatLng, combinations [][]*model.POI) []*model.SuggestedRoute {
	var suggestedRoutes []*model.SuggestedRoute
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, comb := range combinations {
		wg.Add(1)
		go func(index int, combination []*model.POI) {
			defer wg.Done()
			routeName := s.routeBuilderHelper.GenerateRouteName(theme, scenario, combination, index)
			route, err := s.optimizeAndBuildRouteFromUserLocation(ctx, routeName, userLocation, combination)
			if err == nil {
				mu.Lock()
				suggestedRoutes = append(suggestedRoutes, route)
				mu.Unlock()
			}
		}(i, comb)
	}
	wg.Wait()
	return suggestedRoutes
}

// optimizeAndBuildRouteFromUserLocation はユーザーの現在地（スタート地点）から3つのスポットを巡る最短ルートを決定する
// スタート地点 → POI1 → POI2 → POI3 の形式で4箇所を巡るルート
func (s *ThreePOIRouteSuggestionService) optimizeAndBuildRouteFromUserLocation(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
	if len(combination) != 3 {
		return nil, errors.New("組み合わせは3つのスポットである必要があります")
	}

	// 3つのスポットの全順列を生成 (3! = 6通り)
	permutations := s.routeBuilderHelper.GeneratePermutations(combination)

	var bestRoute *model.SuggestedRoute
	var shortestDuration time.Duration = 24 * time.Hour

	// 各順列でユーザー位置（スタート地点）からのルート計算を実行
	for _, perm := range permutations {
		// スタート地点 → スポット1 → スポット2 → スポット3のルートを計算
		waypointLatLngs := make([]model.LatLng, len(perm))
		for i, poi := range perm {
			waypointLatLngs[i] = poi.ToLatLng()
		}

		routeDetails, err := s.directionsProvider.GetWalkingRoute(ctx, userLocation, waypointLatLngs...)
		if err != nil {
			// このルートは計算できないのでスキップ
			continue
		}

		if routeDetails.TotalDuration < shortestDuration {
			shortestDuration = routeDetails.TotalDuration
			bestRoute = &model.SuggestedRoute{
				Name:          fmt.Sprintf("%s (%d分)", name, int(routeDetails.TotalDuration.Minutes())),
				Spots:         perm,
				TotalDuration: routeDetails.TotalDuration,
				Polyline:      routeDetails.Polyline,
			}
		}
	}

	if bestRoute == nil {
		return nil, errors.New("どの順列でもルート計算に失敗しました")
	}

	return bestRoute, nil
}
