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

// GenerateRouteName はテーマ、シナリオ、組み合わせ、リアルタイムコンテキストからルート名を生成する
func (h *RouteBuilderHelper) GenerateRouteName(theme string, scenario string, combination []*model.POI, index int, realtimeContext *model.RealtimeContext) string {
	scenarioJapanese := model.GetScenarioJapaneseName(scenario)
	themeJapanese := model.GetThemeJapaneseName(theme)

	// TODO: ここで Gemini API 叩いて生成するからもっと複雑になる
	// リアルタイムコンテキスト（天気や時間帯）を考慮した名前生成ロジックを実装予定
	// 基本的なルート名を生成
	var baseName string
	if len(combination) >= 2 {
		baseName = fmt.Sprintf("【%s】%s - %sから%sへ", scenarioJapanese, themeJapanese, combination[0].Name, combination[1].Name)
	} else {
		baseName = fmt.Sprintf("【%s】%sルート_%d", scenarioJapanese, themeJapanese, index+1)
	}

	// リアルタイムコンテキストが提供されている場合、追加情報を含める
	if realtimeContext != nil {
		// 将来的にGemini APIでコンテキストを活用した名前生成を行う
		// 現在は基本的な情報のみ追加
		var contextSuffix string
		if realtimeContext.TimeOfDay != "" || realtimeContext.Weather != "" {
			contextSuffix = fmt.Sprintf(" (%s %s)", 
				getJapaneseTimeOfDay(realtimeContext.TimeOfDay),
				getJapaneseWeather(realtimeContext.Weather))
		}
		baseName += contextSuffix
	}

	return baseName
}

// getJapaneseTimeOfDay 英語の時間帯を日本語に変換
func getJapaneseTimeOfDay(timeOfDay string) string {
	switch timeOfDay {
	case "morning":
		return "朝"
	case "afternoon":
		return "午後"
	case "evening":
		return "夕方"
	default:
		return ""
	}
}

// getJapaneseWeather 英語の天気を日本語に変換
func getJapaneseWeather(weather string) string {
	switch weather {
	case "sunny":
		return "晴れ"
	case "cloudy":
		return "曇り"
	case "rainy":
		return "雨"
	default:
		return ""
	}
}
