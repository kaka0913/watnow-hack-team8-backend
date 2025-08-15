package service

import (
	"Team8-App/internal/domain/model"
	"fmt"
)

// RouteBuilderHelper はルート構築に関するヘルパー関数を提供する
type RouteBuilderHelper struct{}

// NewRouteBuilderHelper は新しいRouteBuilderHelperインスタンスを作成する
func NewRouteBuilderHelper() *RouteBuilderHelper {
	return &RouteBuilderHelper{}
}

// GeneratePermutations は3つのPOIの全順列を生成する
func (h *RouteBuilderHelper) GeneratePermutations(pois []*model.POI) [][]*model.POI {
	if len(pois) != 3 {
		return nil
	}

	// 3! = 6通りの順列を明示的に生成
	return [][]*model.POI{
		{pois[0], pois[1], pois[2]}, // ABC
		{pois[0], pois[2], pois[1]}, // ACB
		{pois[1], pois[0], pois[2]}, // BAC
		{pois[1], pois[2], pois[0]}, // BCA
		{pois[2], pois[0], pois[1]}, // CAB
		{pois[2], pois[1], pois[0]}, // CBA
	}
}

// RemovePOIFromSlice はスライスから特定のPOIを除外する（POISの候補から目的地を除外するために使用する）
func (h *RouteBuilderHelper) RemovePOIFromSlice(pois []*model.POI, target *model.POI) []*model.POI {
	var result []*model.POI
	for _, poi := range pois {
		if poi.ID != target.ID {
			result = append(result, poi)
		}
	}
	return result
}

// GenerateRouteName はテーマ、シナリオ、組み合わせからルート名を生成する
func (h *RouteBuilderHelper) GenerateRouteName(theme string, scenario string, combination []*model.POI, index int) string {
	scenarioJapanese := model.GetScenarioJapaneseName(scenario)
	themeJapanese := model.GetThemeJapaneseName(theme)

	// TODO: ここで Gemini API 叩いて生成するからもっと複雑になる
	if len(combination) >= 2 {
		return fmt.Sprintf("【%s】%s - %sから%sへ", scenarioJapanese, themeJapanese, combination[0].Name, combination[1].Name)
	}
	return fmt.Sprintf("【%s】%sルート_%d", scenarioJapanese, themeJapanese, index+1)
}
