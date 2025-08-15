package strategy

import "Team8-App/internal/domain/model"

// StrategyInterface は、POI候補リストからテーマに合った組み合わせを見つける戦略のインターフェース
type StrategyInterface interface {
	// そのテーマで利用可能なシナリオ一覧を取得する
	GetAvailableScenarios() []string
	
	// 指定されたシナリオで検索対象となるPOIカテゴリを取得する
	GetTargetCategories(scenario string) []string
	
	// 指定されたシナリオで候補POIから組み合わせを見つける（目的地なしの場合）
	FindCombinations(scenario string, candidates []*model.POI) [][]*model.POI
	
	// 指定されたシナリオで目的地を含むルート組み合わせを見つける
	FindCombinationsWithDestination(scenario string, destinationPOI *model.POI, candidates []*model.POI) [][]*model.POI
}
