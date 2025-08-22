package strategy

import (
	"context"
	"Team8-App/internal/domain/model"
)

// StrategyInterface は、POI候補リストからテーマに合った組み合わせを見つける戦略のインターフェース
type StrategyInterface interface {
	// 利用可能なシナリオ一覧を取得
	GetAvailableScenarios() []string
	
	// シナリオに基づいてルート組み合わせを生成する
	// 戦略が自分でPOI検索から組み合わせ生成まで全て行う
	FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error)
	
	// 目的地を含むルート組み合わせを生成する
	// 戦略が自分でPOI検索から組み合わせ生成まで全て行う
	FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error)
	
	// ルート再計算用の新しいスポット探索
	// テーマ固有の段階的検索パターンを使用して新しいPOIを探索する
	ExploreNewSpots(ctx context.Context, searchLocation model.LatLng) ([]*model.POI, error)
}