package service

import (
	"Team8-App/internal/domain/model"
	"fmt"
)

// generateTemporaryRouteName は仮のルート名を生成する（実際のタイトルはUsecaseでGemini APIにより生成）
func generateTemporaryRouteName(theme string, scenario string, combination []*model.POI, index int) string {
	scenarioJapanese := model.GetScenarioJapaneseName(scenario)
	themeJapanese := model.GetThemeJapaneseName(theme)

	if len(combination) >= 2 {
		return fmt.Sprintf("【%s】%s - %sから%sへ", scenarioJapanese, themeJapanese, combination[0].Name, combination[1].Name)
	} else {
		return fmt.Sprintf("【%s】%sルート_%d", scenarioJapanese, themeJapanese, index+1)
	}
}

// GeneratePermutations は3つのPOIの全順列を生成する
func generatePermutations(pois []*model.POI) [][]*model.POI {
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
func removePOIFromSlice(pois []*model.POI, target *model.POI) []*model.POI {
	var result []*model.POI
	for _, poi := range pois {
		if poi.ID != target.ID {
			result = append(result, poi)
		}
	}
	return result
}
