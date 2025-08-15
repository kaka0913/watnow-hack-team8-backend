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

// RouteSuggestionService はテーマに応じたルート提案のオーケストレーションを行う
// スタート地点から3つのスポットを巡る4箇所のルートを提案する
type RouteSuggestionService interface {
	// SuggestRoutes はリクエストの内容に応じて適切なルート提案を行う
	// 単一のエントリーポイントで全ての条件（テーマ、シナリオ、目的地の有無）を受け付ける
	SuggestRoutes(ctx context.Context, req *model.SuggestionRequest) ([]*model.SuggestedRoute, error)

	// GetAvailableScenariosForTheme は指定されたテーマで利用可能なシナリオの一覧を取得する
	// フロントエンドでユーザーがシナリオを選択する際に使用される
	GetAvailableScenariosForTheme(theme string) ([]string, error)
}

type routeSuggestionService struct {
	directionsProvider *maps.GoogleDirectionsProvider
	strategies         map[string]strategy.StrategyInterface
	poiRepo            repository.POIsRepository
	routeBuilderHelper *RouteBuilderHelper
	poiSearchHelper    *POISearchHelper
}

func NewRouteSuggestionService(dp *maps.GoogleDirectionsProvider, repo repository.POIsRepository) RouteSuggestionService {
	return &routeSuggestionService{
		directionsProvider: dp,
		poiRepo:            repo,
		strategies: map[string]strategy.StrategyInterface{
			model.ThemeGourmet: strategy.NewGourmetStrategy(),
			model.ThemeNature:  strategy.NewNatureStrategy(),
			model.ThemeHistory: strategy.NewHistoryStrategy(),
		},
		routeBuilderHelper: NewRouteBuilderHelper(),
		poiSearchHelper:    NewPOISearchHelper(repo),
	}
}

// SuggestRoutes はリクエストの内容に応じて適切な処理を呼び出すディスパッチャの役割を担う
func (s *routeSuggestionService) SuggestRoutes(ctx context.Context, req *model.SuggestionRequest) ([]*model.SuggestedRoute, error) {
	// Step 1: 戦略を選択
	selectedStrategy, ok := s.strategies[req.Theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + req.Theme)
	}

	// Step 2: シナリオを決定（指定がなければテーマの全シナリオを対象とする）
	scenariosToRun := req.GetScenarios()
	if !req.HasSpecificScenarios() {
		scenariosToRun = selectedStrategy.GetAvailableScenarios()
	}
	if len(scenariosToRun) == 0 {
		return nil, errors.New("利用可能なシナリオがありません")
	}

	// Step 3: 目的地の有無で処理を振り分け
	if req.HasDestination() {
		// 目的地ありの並行処理を呼び出す
		return s.suggestRoutesForMultipleScenariosWithDestination(ctx, req.Theme, scenariosToRun, req.UserLocation, *req.Destination)
	} else {
		// 目的地なしの並行処理を呼び出す
		return s.suggestRoutesForMultipleScenarios(ctx, req.Theme, scenariosToRun, req.UserLocation)
	}
}

func (s *routeSuggestionService) GetAvailableScenariosForTheme(theme string) ([]string, error) {
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	return selectedStrategy.GetAvailableScenarios(), nil
}
// SuggestRoutesForMultipleScenarios は複数のシナリオから並行でルートを生成する
func (s *routeSuggestionService) suggestRoutesForMultipleScenarios(ctx context.Context, theme string, scenarios []string, userLocation model.LatLng) ([]*model.SuggestedRoute, error) {
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
			routes, err := s.suggestRoutesForScenario(ctx, theme, sc, userLocation)
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
func (s *routeSuggestionService) suggestRoutesForScenario(ctx context.Context, theme string, scenario string, userLocation model.LatLng) ([]*model.SuggestedRoute, error) {
	// Step 1: 戦略を選択
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	// Step 2: シナリオに合ったPOIカテゴリを取得し、候補を検索
	targetCategories := selectedStrategy.GetTargetCategories(scenario)
	if len(targetCategories) == 0 {
		return nil, errors.New("シナリオに該当するカテゴリがありません: " + scenario)
	}

	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, targetCategories, 1500, 30)
	if err != nil {
		return nil, fmt.Errorf("POI候補の取得に失敗しました: %w", err)
	}
	if len(candidates) < 3 {
		return nil, errors.New("周辺に見つかったスポットが3件未満です")
	}

	// Step 3: 戦略を使用して3つのスポットの組み合わせを取得
	combinations := selectedStrategy.FindCombinations(scenario, candidates)
	if len(combinations) == 0 {
		return nil, errors.New("このシナリオに合うルートの組み合わせが見つかりませんでした")
	}

	// TODO: Step3 と Step4の処理は分けるのがあっているのか？
	// Step 4: 組み合わせからルート構築処理を実行（スタート地点 + 3つのスポット = 4箇所巡り）
	suggestedRoutes := s.buildRoutesFromCombinationsWithStartLocation(ctx, theme, scenario, userLocation, combinations)

	return suggestedRoutes, nil
}

// 並行処理用のチャネルとWaitGroup
type scenarioResult struct {
	scenario string
	routes   []*model.SuggestedRoute
	err      error
}

// SuggestRoutesForMultipleScenariosWithDestination は複数のシナリオで目的地を指定してルート提案を行う
func (s *routeSuggestionService) suggestRoutesForMultipleScenariosWithDestination(ctx context.Context, theme string, scenarios []string, userLocation model.LatLng, destination model.LatLng) ([]*model.SuggestedRoute, error) {
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
			routes, err := s.suggestRoutesWithDestination(ctx, theme, sc, userLocation, destination)
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
func (s *routeSuggestionService) suggestRoutesWithDestination(ctx context.Context, theme string, scenario string, userLocation model.LatLng, destination model.LatLng) ([]*model.SuggestedRoute, error) {
	// Step 1: 戦略を選択
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	// Step 2: 目的地周辺のPOIから目的地のPOIを特定
	destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destination)
	if err != nil {
		return nil, fmt.Errorf("目的地周辺のPOIが見つかりません: %w", err)
	}

	// Step 3: シナリオに合ったPOIカテゴリを取得し、候補を検索（目的地は除外）
	targetCategories := selectedStrategy.GetTargetCategories(scenario)
	if len(targetCategories) == 0 {
		return nil, errors.New("シナリオに該当するカテゴリがありません: " + scenario)
	}

	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, targetCategories, 1500, 30)
	if err != nil {
		return nil, fmt.Errorf("POI候補の取得に失敗しました: %w", err)
	}

	// Step 4: 候補から目的地POIを除外
	filteredCandidates := s.routeBuilderHelper.RemovePOIFromSlice(candidates, destinationPOI)
	if len(filteredCandidates) < 2 {
		return nil, errors.New("目的地までの途中に立ち寄るスポットが2件未満です")
	}

	// Step 5: 目的地を含むルート組み合わせを取得
	combinations := selectedStrategy.FindCombinationsWithDestination(scenario, destinationPOI, filteredCandidates)
	if len(combinations) == 0 {
		return nil, errors.New("目的地を含むルートの組み合わせが見つかりませんでした")
	}

	// TODO: Step5 と Step6の処理は分けるのがあっているのか？
	// Step 6: 組み合わせからルート構築処理を実行（目的地が固定されているので、ユーザー位置から順番にルート計算）
	suggestedRoutes := s.buildRoutesWithDestinationFromUserLocation(ctx, theme, scenario, userLocation, combinations)

	return suggestedRoutes, nil
}

// buildRoutesFromCombinationsWithStartLocation は、出発位置を考慮して複数の組み合わせから並行でルートを構築する
func (s *routeSuggestionService) buildRoutesFromCombinationsWithStartLocation(ctx context.Context, theme string, scenario string, userLocation model.LatLng, combinations [][]*model.POI) []*model.SuggestedRoute {
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
func (s *routeSuggestionService) optimizeAndBuildRouteFromUserLocation(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
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

// buildRoutesWithDestinationFromUserLocation は目的地が固定された組み合わせからユーザー位置を起点とするルートを構築する
func (s *routeSuggestionService) buildRoutesWithDestinationFromUserLocation(ctx context.Context, theme string, scenario string, userLocation model.LatLng, combinations [][]*model.POI) []*model.SuggestedRoute {
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
func (s *routeSuggestionService) optimizeAndBuildRouteFromUserLocationToDestination(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
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
