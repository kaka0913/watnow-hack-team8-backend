package service

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/infrastructure/maps"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RouteSuggestionService はルート提案のオーケストレーションを行う単一のサービス
type RouteSuggestionService interface {
	SuggestRoutes(ctx context.Context, req *model.SuggestionRequest) ([]*model.SuggestedRoute, error)
	GetAvailableScenariosForTheme(theme string) ([]string, error)
}

type routeSuggestionService struct {
	directionsProvider *maps.GoogleDirectionsProvider
	strategies         map[string]strategy.StrategyInterface
	poiRepo            repository.POIsRepository
	routeBuilderHelper *RouteBuilderHelper
}

func NewRouteSuggestionService(dp *maps.GoogleDirectionsProvider, repo repository.POIsRepository) RouteSuggestionService {
	// 各Strategyにrepoを注入
	strategies := map[string]strategy.StrategyInterface{
		model.ThemeGourmet:           strategy.NewGourmetStrategy(repo),
		model.ThemeNature:            strategy.NewNatureStrategy(repo),
		model.ThemeHistoryAndCulture: strategy.NewHistoryAndCultureStrategy(repo),
		model.ThemeHorror:            strategy.NewHorrorStrategy(repo),
	}
	return &routeSuggestionService{
		directionsProvider: dp,
		strategies:         strategies,
		poiRepo:            repo,
		routeBuilderHelper: NewRouteBuilderHelper(),
	}
}

// SuggestRoutes はリクエストに応じて処理を振り分ける単一のエントリーポイント
func (s *routeSuggestionService) SuggestRoutes(ctx context.Context, req *model.SuggestionRequest) ([]*model.SuggestedRoute, error) {
	selectedStrategy, ok := s.strategies[req.Theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + req.Theme)
	}

	scenariosToRun := req.GetScenarios()
	if !req.HasSpecificScenarios() {
		scenariosToRun = selectedStrategy.GetAvailableScenarios()
	}
	if len(scenariosToRun) == 0 {
		return nil, errors.New("利用可能なシナリオがありません")
	}

	// 目的地の有無に応じて、「組み合わせ取得」と「ルート最適化」のロジックを定義
	var combinationFinder combinationFinderFunc
	var routeOptimizer routeOptimizerFunc

	if req.HasDestination() {
		combinationFinder = func(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error) {
			return selectedStrategy.FindCombinationsWithDestination(ctx, scenario, userLocation, *req.Destination)
		}
		routeOptimizer = s.optimizeRouteWithDestination
	} else {
		combinationFinder = selectedStrategy.FindCombinations
		routeOptimizer = s.optimizeRoute
	}

	return s.executeScenariosInParallel(ctx, req.Theme, scenariosToRun, req.UserLocation, combinationFinder, routeOptimizer)
}

func (s *routeSuggestionService) GetAvailableScenariosForTheme(theme string) ([]string, error) {
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	return selectedStrategy.GetAvailableScenarios(), nil
}

// 振る舞いを定義する関数型
type combinationFinderFunc func(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error)
type routeOptimizerFunc func(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error)

// scenarioResult は並行処理の結果を格納する
type scenarioResult struct {
	routes []*model.SuggestedRoute
	err    error
}

// executeScenariosInParallel は並行処理の骨格を担う共通ヘルパー
func (s *routeSuggestionService) executeScenariosInParallel(
	ctx context.Context,
	theme string,
	scenarios []string,
	userLocation model.LatLng,
	findCombinations combinationFinderFunc, // 組み合わせ取得ロジックを引数で受け取る
	optimizeRoute routeOptimizerFunc, // ルート最適化ロジックを引数で受け取る
) ([]*model.SuggestedRoute, error) {

	resultsChan := make(chan scenarioResult, len(scenarios))
	var wg sync.WaitGroup

	for _, scenario := range scenarios {
		wg.Add(1)
		go func(sc string) {
			defer wg.Done()
			// 1. 組み合わせを取得
			combinations, err := findCombinations(ctx, sc, userLocation)
			if err != nil {
				resultsChan <- scenarioResult{err: fmt.Errorf("シナリオ '%s': %w", sc, err)}
				return
			}
			if len(combinations) == 0 {
				// エラーではないがルートがない場合
				resultsChan <- scenarioResult{}
				return
			}
			// 2. 組み合わせからルートを並行構築
			routes := s.buildRoutesFromCombinations(ctx, theme, sc, userLocation, combinations, optimizeRoute)
			resultsChan <- scenarioResult{routes: routes}
		}(scenario)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// 結果収集ロジック
	var allRoutes []*model.SuggestedRoute
	var errorMessages []string
	for result := range resultsChan {
		if result.err != nil {
			errorMessages = append(errorMessages, result.err.Error())
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

// buildRoutesFromCombinations はルート構築の並行処理を行う
func (s *routeSuggestionService) buildRoutesFromCombinations(
	ctx context.Context,
	theme, scenario string,
	userLocation model.LatLng,
	combinations [][]*model.POI,
	optimizeRoute routeOptimizerFunc, // 最適化関数を引数で受け取る
) []*model.SuggestedRoute {
	var suggestedRoutes []*model.SuggestedRoute
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, comb := range combinations {
		wg.Add(1)
		go func(index int, combination []*model.POI) {
			defer wg.Done()
			routeName := s.routeBuilderHelper.GenerateRouteName(theme, scenario, combination, index)
			// 渡された最適化関数を実行
			route, err := optimizeRoute(ctx, routeName, userLocation, combination)
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

//------------------------------------------------------------------------------
// ### 2つのルート最適化ロジック
//------------------------------------------------------------------------------

// optimizeRoute は目的地なしのルートを最適化する
func (s *routeSuggestionService) optimizeRoute(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
	// POI数の検証（最低2箇所必要）
	if len(combination) < 2 {
		return nil, errors.New("ルート生成には最低2箇所のスポットが必要です")
	}
	
	// nilPOIのチェック
	validPOIs := make([]*model.POI, 0, len(combination))
	for _, poi := range combination {
		if poi != nil {
			validPOIs = append(validPOIs, poi)
		}
	}
	
	if len(validPOIs) < 2 {
		return nil, errors.New("有効なスポットが不足しています（最低2箇所必要）")
	}
	
	// 2箇所の場合は順列なし、3箇所以上の場合は順列生成
	var routesToTry [][]*model.POI
	if len(validPOIs) == 2 {
		routesToTry = [][]*model.POI{validPOIs}
	} else {
		routesToTry = s.routeBuilderHelper.GeneratePermutations(validPOIs)
	}
	
	var bestRoute *model.SuggestedRoute
	var shortestDuration = time.Duration(24 * time.Hour)

	for _, route := range routesToTry {
		waypointLatLngs := make([]model.LatLng, len(route))
		for i, poi := range route {
			waypointLatLngs[i] = poi.ToLatLng()
		}
		routeDetails, err := s.directionsProvider.GetWalkingRoute(ctx, userLocation, waypointLatLngs...)
		if err != nil {
			continue
		}

		// 所要時間制限チェック（1時間30分以内）
		maxDuration := 90 * time.Minute
		if routeDetails.TotalDuration > maxDuration {
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
		return nil, errors.New("制限時間内でルート計算に成功した順列がありませんでした")
	}
	return bestRoute, nil
}

// optimizeRouteWithDestination は目的地ありのルートを最適化する
func (s *routeSuggestionService) optimizeRouteWithDestination(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
	// POI数の検証（最低2箇所必要、最後が目的地）
	if len(combination) < 2 {
		return nil, errors.New("目的地ありルート生成には最低2箇所のスポットが必要です")
	}
	
	// nilPOIのチェック
	validPOIs := make([]*model.POI, 0, len(combination))
	for _, poi := range combination {
		if poi != nil {
			validPOIs = append(validPOIs, poi)
		}
	}
	
	if len(validPOIs) < 2 {
		return nil, errors.New("有効なスポットが不足しています（最低2箇所必要）")
	}
	
	// 最後のPOIを目的地として扱う
	destination := validPOIs[len(validPOIs)-1]
	waypoints := validPOIs[:len(validPOIs)-1]
	
	// 経由地が1つの場合は順列なし、複数の場合は順列生成
	var routesToTry [][]*model.POI
	if len(waypoints) == 1 {
		routesToTry = [][]*model.POI{append(waypoints, destination)}
	} else {
		waypointPermutations := s.routeBuilderHelper.GeneratePermutations(waypoints)
		for _, perm := range waypointPermutations {
			routesToTry = append(routesToTry, append(perm, destination))
		}
	}
	
	var bestRoute *model.SuggestedRoute
	var shortestDuration = time.Duration(24 * time.Hour)

	for _, route := range routesToTry {
		waypointLatLngs := make([]model.LatLng, len(route))
		for i, poi := range route {
			waypointLatLngs[i] = poi.ToLatLng()
		}
		routeDetails, err := s.directionsProvider.GetWalkingRoute(ctx, userLocation, waypointLatLngs...)
		if err != nil {
			continue
		}

		// 所要時間制限チェック（1時間30分以内）
		maxDuration := 90 * time.Minute
		if routeDetails.TotalDuration > maxDuration {
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
		return nil, errors.New("制限時間内で目的地へのルート計算に成功した順列がありませんでした")
	}
	return bestRoute, nil
}
