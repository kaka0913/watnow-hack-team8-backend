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

// TwoPOIWithDestinationRouteSuggestionService は2つのPOI+目的地を巡るルート提案サービス
// スタート地点 → POI1 → POI2 → 目的地 の形式で4箇所を巡るルート
type TwoPOIWithDestinationRouteSuggestionService struct {
	directionsProvider *maps.GoogleDirectionsProvider
	strategies         map[string]strategy.StrategyInterface
	routeBuilderHelper *RouteBuilderHelper
}

func NewTwoPOIWithDestinationRouteSuggestionService(dp *maps.GoogleDirectionsProvider, strategies map[string]strategy.StrategyInterface, helper *RouteBuilderHelper) *TwoPOIWithDestinationRouteSuggestionService {
	return &TwoPOIWithDestinationRouteSuggestionService{
		directionsProvider: dp,
		strategies:         strategies,
		routeBuilderHelper: helper,
	}
}

// SuggestRoutesForMultipleScenariosWithDestination は複数のシナリオで目的地を指定してルート提案を行う
func (s *TwoPOIWithDestinationRouteSuggestionService) SuggestRoutesForMultipleScenariosWithDestination(ctx context.Context, theme string, scenarios []string, userLocation model.LatLng, destination model.LatLng) ([]*model.SuggestedRoute, error) {
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
			routes, err := s.SuggestRoutesWithDestination(ctx, theme, sc, userLocation, destination)
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

// SuggestRoutesWithDestination は目的地を指定してルート提案を行う
// スタート地点 → POI1 → POI2 → 目的地 の形式で4箇所を巡るルート
func (s *TwoPOIWithDestinationRouteSuggestionService) SuggestRoutesWithDestination(ctx context.Context, theme string, scenario string, userLocation model.LatLng, destination model.LatLng) ([]*model.SuggestedRoute, error) {
	// Step 1: 戦略を選択
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	var combinations [][]*model.POI
	var err error

	// Step 2: 戦略に組み合わせの生成を完全に委譲
	// 全ての戦略が統一されたインターフェースを持つため、テーマごとの分岐は不要
	combinations, err = selectedStrategy.FindCombinationsWithDestination(ctx, scenario, userLocation, destination)
	if err != nil {
		return nil, err
	}

	if len(combinations) == 0 {
		return nil, errors.New("目的地を含むルートの組み合わせが見つかりませんでした")
	}

	// Step 3: 組み合わせからルート構築処理を実行（目的地が固定されているので、ユーザー位置から順番にルート計算）
	suggestedRoutes := s.buildRoutesWithDestinationFromUserLocation(ctx, theme, scenario, userLocation, combinations)

	return suggestedRoutes, nil
}

// buildRoutesWithDestinationFromUserLocation は目的地が固定された組み合わせからユーザー位置を起点とするルートを構築する
func (s *TwoPOIWithDestinationRouteSuggestionService) buildRoutesWithDestinationFromUserLocation(ctx context.Context, theme string, scenario string, userLocation model.LatLng, combinations [][]*model.POI) []*model.SuggestedRoute {
	var suggestedRoutes []*model.SuggestedRoute
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, comb := range combinations {
		wg.Add(1)
		go func(index int, combination []*model.POI) {
			defer wg.Done()
			// TODO: ここでかなり叩くことになると思うので、パフォーマンスに注意して検討
			routeName := s.routeBuilderHelper.GenerateRouteName(theme, scenario, combination, index)
			// 目的地が固定されているので、ユーザー位置から順番にルート計算
			route, err := s.optimizeAndBuildRouteFromUserLocationToDestination(ctx, routeName, userLocation, combination)
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

// optimizeAndBuildRouteFromUserLocationToDestination はスタート地点から目的地への最適化ルートを構築する
// スタート地点 → POI1 → POI2 → 目的地 の形式で4箇所を巡るルート
func (s *TwoPOIWithDestinationRouteSuggestionService) optimizeAndBuildRouteFromUserLocationToDestination(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
	if len(combination) != 3 {
		return nil, errors.New("組み合わせは3つのスポットである必要があります")
	}

	// 最後のPOIが目的地として固定されているので、最初の2つのPOIの順序のみ最適化
	// combination = [POI1, POI2, destination] の形式
	poi1, poi2, destination := combination[0], combination[1], combination[2]

	// 2通りの順序を試す: スタート地点 → poi1 → poi2 → destination vs スタート地点 → poi2 → poi1 → destination
	routes := [][]*model.POI{
		{poi1, poi2, destination},
		{poi2, poi1, destination},
	}

	var bestRoute *model.SuggestedRoute
	var shortestDuration time.Duration = 24 * time.Hour

	for _, route := range routes {
		waypointLatLngs := make([]model.LatLng, len(route))
		for i, poi := range route {
			waypointLatLngs[i] = poi.ToLatLng()
		}

		routeDetails, err := s.directionsProvider.GetWalkingRoute(ctx, userLocation, waypointLatLngs...)
		if err != nil {
			continue
		}

		if routeDetails.TotalDuration < shortestDuration {
			shortestDuration = routeDetails.TotalDuration
			bestRoute = &model.SuggestedRoute{
				Name:          fmt.Sprintf("%s (%d分)", name, int(routeDetails.TotalDuration.Minutes())),
				Spots:         route,
				TotalDuration: routeDetails.TotalDuration,
				Polyline:      routeDetails.Polyline,
			}
		}
	}

	if bestRoute == nil {
		return nil, errors.New("目的地へのルート計算に失敗しました")
	}

	return bestRoute, nil
}
