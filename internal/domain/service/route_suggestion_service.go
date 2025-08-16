package service

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/infrastructure/maps"
	"context"
	"errors"
)

// RouteSuggestionService はテーマに応じたルート提案のオーケストレーションを行う
// 3つのPOIサービスと2つのPOI+目的地サービスを統合して提供する
type RouteSuggestionService interface {
	// SuggestRoutes はリクエストの内容に応じて適切なルート提案を行う
	// 単一のエントリーポイントで全ての条件（テーマ、シナリオ、目的地の有無）を受け付ける
	SuggestRoutes(ctx context.Context, req *model.SuggestionRequest) ([]*model.SuggestedRoute, error)

	// GetAvailableScenariosForTheme は指定されたテーマで利用可能なシナリオの一覧を取得する
	// フロントエンドでユーザーがシナリオを選択する際に使用される
	GetAvailableScenariosForTheme(theme string) ([]string, error)
}

type routeSuggestionService struct {
	strategies                      map[string]strategy.StrategyInterface
	threePOIService                *ThreePOIRouteSuggestionService
	twoPOIWithDestinationService   *TwoPOIWithDestinationRouteSuggestionService
}

func NewRouteSuggestionService(dp *maps.GoogleDirectionsProvider, repo repository.POIsRepository) RouteSuggestionService {
	strategies := map[string]strategy.StrategyInterface{
		model.ThemeGourmet:           strategy.NewGourmetStrategy(),
		model.ThemeNature:            strategy.NewNatureStrategy(repo),
		model.ThemeHistoryAndCulture: strategy.NewHistoryAndCultureStrategy(),
		model.ThemeHorror:            strategy.NewHorrorStrategy(),
	}
	
	routeBuilderHelper := NewRouteBuilderHelper()
	
	return &routeSuggestionService{
		strategies:                     strategies,
		threePOIService:               NewThreePOIRouteSuggestionService(dp, strategies, routeBuilderHelper),
		twoPOIWithDestinationService:  NewTwoPOIWithDestinationRouteSuggestionService(dp, strategies, routeBuilderHelper),
	}
}

// SuggestRoutes はリクエストの内容に応じて適切な処理を呼び出すディスパッチャの役割を担う
func (s *routeSuggestionService) SuggestRoutes(ctx context.Context, req *model.SuggestionRequest) ([]*model.SuggestedRoute, error) {
	// Step 1: 戦略を選択
	selectedStrategy, ok := s.strategies[req.Theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + req.Theme)
	}

	// Step 2: シナリオを決定
	scenariosToRun := req.GetScenarios()
	if !req.HasSpecificScenarios() {
		scenariosToRun = selectedStrategy.GetAvailableScenarios()
	}
	if len(scenariosToRun) == 0 {
		return nil, errors.New("利用可能なシナリオがありません")
	}

	// Step 3: 目的地の有無で処理を振り分け
	if req.HasDestination() {
		// 2つのPOI+目的地サービスを使用
		return s.twoPOIWithDestinationService.SuggestRoutesForMultipleScenariosWithDestination(ctx, req.Theme, scenariosToRun, req.UserLocation, *req.Destination)
	} else {
		// 3つのPOIサービスを使用
		return s.threePOIService.SuggestRoutesForMultipleScenarios(ctx, req.Theme, scenariosToRun, req.UserLocation)
	}
}

func (s *routeSuggestionService) GetAvailableScenariosForTheme(theme string) ([]string, error) {
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	return selectedStrategy.GetAvailableScenarios(), nil
}

// scenarioResult は並行処理用のチャネルとWaitGroupで使用される結果構造体
type scenarioResult struct {
	scenario string
	routes   []*model.SuggestedRoute
	err      error
}
