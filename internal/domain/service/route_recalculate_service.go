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
	"math"
)

// RouteRecalculateService はルート再計算のドメインサービス
type RouteRecalculateService interface {
	RecalculateRoute(ctx context.Context, req *model.RouteRecalculateRequest, originalProposal *model.RouteProposal) (*model.RouteRecalculateResponse, error)
	GetSupportedThemes() []string
}

type routeRecalculateService struct {
	directionsProvider  *maps.GoogleDirectionsProvider
	strategies          map[string]strategy.StrategyInterface
	poiRepo             repository.POIsRepository
	poiSearchHelper     *helper.POISearchHelper
	parallelOptimizer   *ParallelRouteOptimizer
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
	parallelOptimizer := NewParallelRouteOptimizer(dp)
	return &routeRecalculateService{
		directionsProvider: dp,
		strategies:         strategies,
		poiRepo:            repo,
		poiSearchHelper:    helper.NewPOISearchHelper(repo),
		parallelOptimizer:  parallelOptimizer,
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
	newDiscoveries, err := s.exploreNewSpot(ctx, req.CurrentLocation, remainingPOIs, originalProposal.Theme, originalProposal)
	if err != nil {
		return nil, fmt.Errorf("新しいスポット探索に失敗: %w", err)
	}
	recalcContext.NewDiscoveryPOIs = newDiscoveries

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
func (s *routeRecalculateService) exploreNewSpot(ctx context.Context, currentLocation *model.Location, remainingPOIs []*model.POI, theme string, originalProposal *model.RouteProposal) ([]*model.POI, error) {
	log.Printf("🔍 新しいスポット探索中...")

	if len(remainingPOIs) == 0 {
		return nil, errors.New("未訪問POIがないため新しいスポットを探索できません")
	}

	// 元の提案の総物件数と時間を取得
	originalTotalSpots := len(originalProposal.DisplayHighlights)
	originalDurationMinutes := originalProposal.EstimatedDurationMinutes
	currentVisitedSpots := originalTotalSpots - len(remainingPOIs) // 既に訪問した物件数
	
	// 新しく探索する物件数を決定
	// 元の物件数を基準に、時間制約と探索効率を考慮して決定
	var neededNewSpots int
	
	// 時間制約を考慮した最大追加物件数
	maxNewSpots := 1
	if originalDurationMinutes <= 90 {
		maxNewSpots = 2
	} else if originalDurationMinutes <= 120 {
		maxNewSpots = 3
	}
	
	// 残りの物件数が少ない場合は多めに追加、多い場合は少なめに追加
	if len(remainingPOIs) <= 2 {
		neededNewSpots = maxNewSpots // 残り物件が少ないので最大まで追加
	} else if len(remainingPOIs) <= 4 {
		neededNewSpots = maxNewSpots - 1 // 中程度なので少し控えめ
	} else {
		neededNewSpots = 1 // 残り物件が多いので最小限追加
	}
	
	// 最低1件は追加
	if neededNewSpots <= 0 {
		neededNewSpots = 1
	}
	
	log.Printf("📊 物件数調整: 元の総数=%d, 元の時間=%d分, 現在の訪問済み=%d, 残り=%d, 追加予定=%d, 最大追加=%d", 
		originalTotalSpots, originalDurationMinutes, currentVisitedSpots, len(remainingPOIs), neededNewSpots, maxNewSpots)

	// 探索エリアを決定（現在地と次のPOIの間）
	currentLatLng := model.LatLng{
		Lat: currentLocation.Latitude,
		Lng: currentLocation.Longitude,
	}
	nextPOI := remainingPOIs[0] // 最初の未訪問POI
	nextLatLng := nextPOI.ToLatLng()
	// 中間地点を計算
	midLatLng := model.LatLng{
		Lat: (currentLatLng.Lat + nextLatLng.Lat) / 2,
		Lng: (currentLatLng.Lng + nextLatLng.Lng) / 2,
	}

	// テーマに対応するStrategyを取得し、新しいスポットを探索
	strategy, exists := s.strategies[theme]
	if !exists {
		return nil, fmt.Errorf("対応していないテーマです: %s", theme)
	}

	// Strategyに新しいスポットの探索を委譲
	candidates, err := strategy.ExploreNewSpots(ctx, midLatLng)
	if err != nil {
		return nil, fmt.Errorf("新しいPOI探索に失敗: %w", err)
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

	// 必要な数だけ新しいスポットを選択
	var selectedSpots []*model.POI
	for i := 0; i < neededNewSpots && i < len(filteredCandidates); i++ {
		selectedSpots = append(selectedSpots, filteredCandidates[i])
	}

	if len(selectedSpots) > 0 {
		log.Printf("✨ 新しい発見: %d件の物件を追加", len(selectedSpots))
		for i, spot := range selectedSpots {
			log.Printf("   %d. %s", i+1, spot.Name)
		}
	}

	return selectedSpots, nil
}

// generateNewRoute は新しいルートを生成
func (s *routeRecalculateService) generateNewRoute(ctx context.Context, currentLocation *model.Location, destinationLocation *model.Location, recalcContext *model.RouteRecalculateContext) (*model.UpdatedRoute, error) {
	log.Printf("🗺️ 新しいルート生成中...")

	// 新しい経由地リストを作成
	var newCombination []*model.POI
	
	// 新しい発見されたPOIを最初に追加
	if len(recalcContext.NewDiscoveryPOIs) > 0 {
		newCombination = append(newCombination, recalcContext.NewDiscoveryPOIs...)
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

	// 並行最適化を使用
	return s.parallelOptimizer.OptimizeRouteParallel(ctx, name, userLocation, routesToTry)
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
	
	// 最後のPOIを目的地として固定し、それ以外の順列を生成
	var routesToTry [][]*model.POI
	if len(validPOIs) == 1 {
		// 1箇所の場合は目的地のみ
		routesToTry = [][]*model.POI{validPOIs}
	} else {
		// 最後のPOI（目的地）以外の順列を生成
		destinationPOI := validPOIs[len(validPOIs)-1]
		intermediatePOIs := validPOIs[:len(validPOIs)-1]
		
		if len(intermediatePOIs) == 0 {
			routesToTry = [][]*model.POI{{destinationPOI}}
		} else {
			intermediatePermutations := s.generatePermutations(intermediatePOIs)
			routesToTry = make([][]*model.POI, len(intermediatePermutations))
			for i, perm := range intermediatePermutations {
				route := make([]*model.POI, len(perm)+1)
				copy(route, perm)
				route[len(route)-1] = destinationPOI
				routesToTry[i] = route
			}
		}
	}
	
	// 並行最適化を使用
	return s.parallelOptimizer.OptimizeRouteParallel(ctx, name, userLocation, routesToTry)
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

// calculateDistanceToNext は次のPOIまでの距離を計算する
func (s *routeRecalculateService) calculateDistanceToNext(spots []*model.POI, currentIndex int) int {
	if currentIndex >= len(spots)-1 {
		return 0 // 最後のスポットの場合
	}
	
	current := spots[currentIndex].ToLatLng()
	next := spots[currentIndex+1].ToLatLng()
	
	// Haversine公式を使用して距離を計算
	return s.calculateHaversineDistance(current, next)
}

// calculateTotalDistance は総距離を計算する
func (s *routeRecalculateService) calculateTotalDistance(spots []*model.POI) int {
	if len(spots) <= 1 {
		return 0
	}
	
	totalDistance := 0
	for i := 0; i < len(spots)-1; i++ {
		current := spots[i].ToLatLng()
		next := spots[i+1].ToLatLng()
		totalDistance += s.calculateHaversineDistance(current, next)
	}
	
	return totalDistance
}

// calculateHaversineDistance はHaversine公式を使用して2点間の距離をメートルで計算
func (s *routeRecalculateService) calculateHaversineDistance(point1, point2 model.LatLng) int {
	const earthRadius = 6371000 // 地球の半径（メートル）
	
	// 度をラジアンに変換
	lat1Rad := point1.Lat * (3.14159265359 / 180)
	lon1Rad := point1.Lng * (3.14159265359 / 180)
	lat2Rad := point2.Lat * (3.14159265359 / 180)
	lon2Rad := point2.Lng * (3.14159265359 / 180)
	
	// 差分を計算
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad
	
	// Haversine公式
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + 
		 math.Cos(lat1Rad)*math.Cos(lat2Rad)*
		 math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	distance := earthRadius * c
	return int(distance)
}
