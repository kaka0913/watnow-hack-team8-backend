package service

import (
	"Team8-App/internal/domain/helper"
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/infrastructure/maps"
	"context"
	"errors"
	"fmt"
	"log"
)

// RouteRecalculateService はルート再計算のドメインサービス
type RouteRecalculateService interface {
	RecalculateRoute(ctx context.Context, req *model.RouteRecalculateRequest, originalProposal *model.RouteProposal) (*model.RouteRecalculateResponse, error)
	GetSupportedThemes() []string
}

type routeRecalculateService struct {
	directionsProvider *maps.GoogleDirectionsProvider
	strategies         map[string]strategy.StrategyInterface
	poiRepo            repository.POIsRepository
	poiSearchHelper    *helper.POISearchHelper
}

// NewRouteRecalculateService は新しいRouteRecalculateServiceインスタンスを作成
func NewRouteRecalculateService(
	dp *maps.GoogleDirectionsProvider,
	repo repository.POIsRepository,
) RouteRecalculateService {
	// 自然テーマのみ対応（将来的に拡張可能）
	strategies := map[string]strategy.StrategyInterface{
		model.ThemeNature: strategy.NewNatureStrategy(repo),
	}
	return &routeRecalculateService{
		directionsProvider: dp,
		strategies:         strategies,
		poiRepo:            repo,
		poiSearchHelper:    helper.NewPOISearchHelper(repo),
	}
}

// GetSupportedThemes は対応しているテーマ一覧を取得
func (s *routeRecalculateService) GetSupportedThemes() []string {
	themes := make([]string, 0, len(s.strategies))
	for theme := range s.strategies {
		themes = append(themes, theme)
	}
	return themes
}

// RecalculateRoute はルート再計算の主要処理
func (s *routeRecalculateService) RecalculateRoute(ctx context.Context, req *model.RouteRecalculateRequest, originalProposal *model.RouteProposal) (*model.RouteRecalculateResponse, error) {
	log.Printf("🔄 ルート再計算開始 (ProposalID: %s)", req.ProposalID)

	// テーマサポートチェック
	if !s.isThemeSupported(originalProposal.Theme) {
		return nil, fmt.Errorf("現在は%vテーマのみ再計算に対応しています（指定テーマ: %s）", s.GetSupportedThemes(), originalProposal.Theme)
	}

	// Step 1: 元の提案をコンテキストに設定
	recalcContext := &model.RouteRecalculateContext{
		OriginalProposal: originalProposal,
	}

	// Step 2: 未訪問のPOIを特定
	remainingPOIs, err := s.identifyRemainingPOIs(originalProposal, req.VisitedPOIs.PreviousPOIs)
	if err != nil {
		return nil, fmt.Errorf("未訪問POI特定に失敗: %w", err)
	}
	recalcContext.RemainingPOIs = remainingPOIs

	// Step 3: 新しい中継スポットを探索
	newDiscovery, err := s.exploreNewSpot(ctx, req.CurrentLocation, remainingPOIs, originalProposal.Theme)
	if err != nil {
		return nil, fmt.Errorf("新しいスポット探索に失敗: %w", err)
	}
	recalcContext.NewDiscoveryPOI = newDiscovery

	// Step 4: 新しいルートを生成
	updatedRoute, err := s.generateNewRoute(ctx, req.CurrentLocation, req.DestinationLocation, recalcContext)
	if err != nil {
		return nil, fmt.Errorf("新しいルート生成に失敗: %w", err)
	}

	log.Printf("✅ ルート再計算完了")
	return &model.RouteRecalculateResponse{
		UpdatedRoute: updatedRoute,
	}, nil
}

// isThemeSupported はテーマがサポートされているかチェック
func (s *routeRecalculateService) isThemeSupported(theme string) bool {
	_, supported := s.strategies[theme]
	return supported
}

// identifyRemainingPOIs は未訪問のPOIを特定
func (s *routeRecalculateService) identifyRemainingPOIs(originalProposal *model.RouteProposal, visitedPOIs []model.PreviousPOI) ([]*model.POI, error) {
	log.Printf("📍 未訪問POI特定中...")

	// 元の提案からPOI型のNavigationStepを抽出
	var originalPOIs []*model.POI
	for _, step := range originalProposal.NavigationSteps {
		if step.Type == "poi" {
			// NavigationStepからPOIオブジェクトを再構築
			poi := &model.POI{
				ID:   step.POIId,
				Name: step.Name,
				Location: &model.Geometry{
					Type:        "Point",
					Coordinates: []float64{step.Longitude, step.Latitude},
				},
			}
			originalPOIs = append(originalPOIs, poi)
		}
	}

	// 訪問済みPOIのIDセットを作成
	visitedSet := make(map[string]bool)
	for _, visited := range visitedPOIs {
		visitedSet[visited.POIId] = true
	}

	// 未訪問のPOIをフィルタリング
	var remainingPOIs []*model.POI
	for _, poi := range originalPOIs {
		if !visitedSet[poi.ID] {
			remainingPOIs = append(remainingPOIs, poi)
		}
	}

	log.Printf("📊 未訪問POI: %d件", len(remainingPOIs))
	return remainingPOIs, nil
}

// exploreNewSpot は新しい中継スポットを探索
func (s *routeRecalculateService) exploreNewSpot(ctx context.Context, currentLocation *model.Location, remainingPOIs []*model.POI, theme string) (*model.POI, error) {
	log.Printf("🔍 新しいスポット探索中...")

	if len(remainingPOIs) == 0 {
		return nil, errors.New("未訪問POIがないため新しいスポットを探索できません")
	}

	// 探索エリアを決定（現在地と次のPOIの間）
	currentLatLng := model.LatLng{
		Lat: currentLocation.Latitude,
		Lng: currentLocation.Longitude,
	}
	nextPOI := remainingPOIs[0] // 最初の未訪問POI
	nextLatLng := nextPOI.ToLatLng()

	// 自然テーマのカテゴリで探索
	categories := model.GetNatureCategories()
	
	// 中間地点を計算
	midLatLng := model.LatLng{
		Lat: (currentLatLng.Lat + nextLatLng.Lat) / 2,
		Lng: (currentLatLng.Lng + nextLatLng.Lng) / 2,
	}

	// 中間地点周辺でPOIを探索
	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, midLatLng, categories, 1000, 10)
	if err != nil {
		return nil, fmt.Errorf("新しいPOI探索に失敗: %w", err)
	}

	if len(candidates) == 0 {
		// より広い範囲で再検索
		candidates, err = s.poiRepo.FindNearbyByCategories(ctx, midLatLng, []string{"店舗", "観光名所"}, 2000, 15)
		if err != nil {
			return nil, fmt.Errorf("広範囲POI探索に失敗: %w", err)
		}
	}

	// 既存のPOIを除外
	var filteredCandidates []*model.POI
	for _, candidate := range candidates {
		isExisting := false
		for _, remaining := range remainingPOIs {
			if candidate.ID == remaining.ID {
				isExisting = true
				break
			}
		}
		if !isExisting {
			filteredCandidates = append(filteredCandidates, candidate)
		}
	}

	if len(filteredCandidates) == 0 {
		return nil, errors.New("新しい発見スポットが見つかりませんでした")
	}

	// 評価の高いPOIを選択
	newDiscovery := helper.FindHighestRated(filteredCandidates)
	log.Printf("✨ 新しい発見: %s", newDiscovery.Name)

	return newDiscovery, nil
}

// generateNewRoute は新しいルートを生成
func (s *routeRecalculateService) generateNewRoute(ctx context.Context, currentLocation *model.Location, destinationLocation *model.Location, recalcContext *model.RouteRecalculateContext) (*model.UpdatedRoute, error) {
	log.Printf("🗺️ 新しいルート生成中...")

	// 新しい経由地リストを作成
	var newCombination []*model.POI
	
	// 新しい発見を最初に追加
	if recalcContext.NewDiscoveryPOI != nil {
		newCombination = append(newCombination, recalcContext.NewDiscoveryPOI)
	}
	
	// 残りの未訪問POIを追加
	newCombination = append(newCombination, recalcContext.RemainingPOIs...)

	// 目的地が指定されている場合は、目的地周辺のPOIを最後に追加
	if destinationLocation != nil {
		destinationLatLng := model.LatLng{
			Lat: destinationLocation.Latitude,
			Lng: destinationLocation.Longitude,
		}
		destinationPOI, err := s.poiSearchHelper.FindNearestPOI(ctx, destinationLatLng, model.GetNatureCategories())
		if err == nil {
			newCombination = append(newCombination, destinationPOI)
		}
	}

	if len(newCombination) == 0 {
		return nil, errors.New("新しいルートの経由地が見つかりませんでした")
	}

	// ルート最適化を実行
	currentLatLng := model.LatLng{
		Lat: currentLocation.Latitude,
		Lng: currentLocation.Longitude,
	}

	// 目的地の有無に応じてルート最適化方法を選択
	var optimizedRoute *model.SuggestedRoute
	var err error
	
	if destinationLocation != nil {
		optimizedRoute, err = s.optimizeRouteWithDestination(ctx, "再計算ルート", currentLatLng, newCombination)
	} else {
		optimizedRoute, err = s.optimizeRoute(ctx, "再計算ルート", currentLatLng, newCombination)
	}
	
	if err != nil {
		return nil, fmt.Errorf("ルート最適化に失敗: %w", err)
	}

	// NavigationStepsを生成
	var navigationSteps []model.NavigationStep
	for i, poi := range optimizedRoute.Spots {
		poiLatLng := poi.ToLatLng()
		var distanceToNext int
		if i < len(optimizedRoute.Spots)-1 {
			distanceToNext = s.calculateDistanceToNext(optimizedRoute.Spots, i)
		}
		
		step := model.NavigationStep{
			Type:                 "poi",
			Name:                 poi.Name,
			POIId:                poi.ID,
			Description:          fmt.Sprintf("%sを発見", poi.Name),
			Latitude:             poiLatLng.Lat,
			Longitude:            poiLatLng.Lng,
			DistanceToNextMeters: distanceToNext,
		}
		navigationSteps = append(navigationSteps, step)
	}

	// ハイライトを生成
	var highlights []string
	for _, poi := range optimizedRoute.Spots {
		highlights = append(highlights, poi.Name)
	}

	// 更新されたルート情報を返す
	updatedRoute := &model.UpdatedRoute{
		Title:                    "新たな発見の散歩道", // 仮のタイトル（物語生成で更新される）
		EstimatedDurationMinutes: int(optimizedRoute.TotalDuration.Minutes()),
		EstimatedDistanceMeters:  s.calculateTotalDistance(optimizedRoute.Spots),
		Highlights:               highlights,
		NavigationSteps:          navigationSteps,
		RoutePolyline:            optimizedRoute.Polyline,
		GeneratedStory:           "", // 物語は後で生成される
	}

	log.Printf("📊 新ルート: %d分, %d箇所", updatedRoute.EstimatedDurationMinutes, len(optimizedRoute.Spots))
	return updatedRoute, nil
}

//------------------------------------------------------------------------------
// ### ルート最適化ロジック（route_suggestion_serviceから参考）
//------------------------------------------------------------------------------

// optimizeRoute は目的地なしのルートを最適化する
func (s *routeRecalculateService) optimizeRoute(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
	// POI数の検証（最低1箇所必要）
	if len(combination) < 1 {
		return nil, errors.New("ルート生成には最低1箇所のスポットが必要です")
	}
	
	// nilPOIのチェック
	validPOIs := make([]*model.POI, 0, len(combination))
	for _, poi := range combination {
		if poi != nil {
			validPOIs = append(validPOIs, poi)
		}
	}
	
	if len(validPOIs) < 1 {
		return nil, errors.New("有効なスポットが不足しています")
	}
	
	// 1箇所の場合は順列なし、2箇所以上の場合は順列生成
	var routesToTry [][]*model.POI
	if len(validPOIs) == 1 {
		routesToTry = [][]*model.POI{validPOIs}
	} else {
		routesToTry = s.generatePermutations(validPOIs)
	}
	
	var bestRoute *model.SuggestedRoute
	var shortestDuration = 24 * 60 * 60 // 24時間を秒で表現

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
		maxDurationMinutes := 90
		if int(routeDetails.TotalDuration.Minutes()) > maxDurationMinutes {
			continue
		}

		if int(routeDetails.TotalDuration.Seconds()) < shortestDuration {
			shortestDuration = int(routeDetails.TotalDuration.Seconds())
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
func (s *routeRecalculateService) optimizeRouteWithDestination(ctx context.Context, name string, userLocation model.LatLng, combination []*model.POI) (*model.SuggestedRoute, error) {
	// POI数の検証（最低1箇所必要、最後が目的地）
	if len(combination) < 1 {
		return nil, errors.New("目的地ありルート生成には最低1箇所のスポットが必要です")
	}
	
	// nilPOIのチェック
	validPOIs := make([]*model.POI, 0, len(combination))
	for _, poi := range combination {
		if poi != nil {
			validPOIs = append(validPOIs, poi)
		}
	}
	
	if len(validPOIs) < 1 {
		return nil, errors.New("有効なスポットが不足しています")
	}
	
	// 最後のPOIを目的地として扱う
	destination := validPOIs[len(validPOIs)-1]
	waypoints := validPOIs[:len(validPOIs)-1]
	
	// 経由地が0の場合は目的地のみ、1つ以上の場合は順列生成
	var routesToTry [][]*model.POI
	if len(waypoints) == 0 {
		routesToTry = [][]*model.POI{{destination}}
	} else if len(waypoints) == 1 {
		routesToTry = [][]*model.POI{append(waypoints, destination)}
	} else {
		waypointPermutations := s.generatePermutations(waypoints)
		for _, perm := range waypointPermutations {
			routesToTry = append(routesToTry, append(perm, destination))
		}
	}
	
	var bestRoute *model.SuggestedRoute
	var shortestDuration = 24 * 60 * 60 // 24時間を秒で表現

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
		maxDurationMinutes := 90
		if int(routeDetails.TotalDuration.Minutes()) > maxDurationMinutes {
			continue
		}

		if int(routeDetails.TotalDuration.Seconds()) < shortestDuration {
			shortestDuration = int(routeDetails.TotalDuration.Seconds())
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

//------------------------------------------------------------------------------
// ### ヘルパーメソッド
//------------------------------------------------------------------------------

// generatePermutations はPOIの順列を生成する（route_suggestion_serviceから参考）
func (s *routeRecalculateService) generatePermutations(pois []*model.POI) [][]*model.POI {
	if len(pois) <= 1 {
		return [][]*model.POI{pois}
	}
	
	var result [][]*model.POI
	for i, poi := range pois {
		remaining := make([]*model.POI, 0, len(pois)-1)
		remaining = append(remaining, pois[:i]...)
		remaining = append(remaining, pois[i+1:]...)
		
		subPerms := s.generatePermutations(remaining)
		for _, subPerm := range subPerms {
			perm := make([]*model.POI, 0, len(pois))
			perm = append(perm, poi)
			perm = append(perm, subPerm...)
			result = append(result, perm)
		}
	}
	return result
}

// calculateDistanceToNext は次のPOIまでの距離を計算する（仮実装）
func (s *routeRecalculateService) calculateDistanceToNext(spots []*model.POI, currentIndex int) int {
	// 仮の実装：固定値を返す
	// 実際はGoogleMapsAPIやGeographyライブラリを使用して正確な距離を計算
	return 200
}

// calculateTotalDistance は総距離を計算する（仮実装）
func (s *routeRecalculateService) calculateTotalDistance(spots []*model.POI) int {
	// 仮の実装：POI数 × 平均距離
	// 実際はより正確な計算が必要
	return len(spots) * 500
}
