package helper

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"errors"
	"time"
)

// POISearchHelper はPOI検索に関するヘルパー関数を提供する
type POISearchHelper struct {
	poiRepo repository.POIsRepository
}

// NewPOISearchHelper は新しいPOISearchHelperインスタンスを作成する
func NewPOISearchHelper(repo repository.POIsRepository) *POISearchHelper {
	return &POISearchHelper{
		poiRepo: repo,
	}
}

// FindNearestPOI は目的地に該当するPOIがないかを確認するために、指定座標に最も近いPOIを見つける
func (h *POISearchHelper) FindNearestPOI(ctx context.Context, location model.LatLng, categories []string) (*model.POI, error) {
	// 目的地周辺のPOIを検索（実際にあるカテゴリで段階的に検索）
	
	// 第1段階：観光名所で検索
	nearbyPOIs, err := h.poiRepo.FindNearbyByCategories(ctx, location, []string{"観光名所"}, 4000, 40)
	if err != nil {
		return nil, err
	}

	if len(nearbyPOIs) > 0 {
		return nearbyPOIs[0], nil
	}
	
	// 第2段階：店舗で検索
	nearbyPOIs, err = h.poiRepo.FindNearbyByCategories(ctx, location, []string{"店舗"}, 6000, 50)
	if err != nil {
		return nil, err
	}

	if len(nearbyPOIs) > 0 {
		return nearbyPOIs[0], nil
	}
	
	// 第3段階：寺院で検索
	nearbyPOIs, err = h.poiRepo.FindNearbyByCategories(ctx, location, []string{"寺院"}, 8000, 60)
	if err != nil {
		return nil, err
	}

	if len(nearbyPOIs) > 0 {
		return nearbyPOIs[0], nil
	}

	return nil, errors.New("目的地周辺にPOIが見つかりません")
}

// GetCategoriesForScenario はシナリオに応じて適切なPOIカテゴリを取得する
func (h *POISearchHelper) GetCategoriesForScenario(theme, scenario string) []string {
	return model.GetCategoriesForThemeAndScenario(theme, scenario)
}

// ValidateThemeAndScenario はテーマとシナリオの組み合わせが有効かチェックする
func (h *POISearchHelper) ValidateThemeAndScenario(theme, scenario string) bool {
	if !model.IsValidTheme(theme) {
		return false
	}
	if !model.IsValidScenario(scenario) {
		return false
	}
	
	// シナリオがテーマに属するかチェック
	validScenarios := model.GetScenariosForTheme(theme)
	for _, validScenario := range validScenarios {
		if validScenario == scenario {
			return true
		}
	}
	return false
}

// GetAvailableScenarios は指定されたテーマで利用可能なシナリオを取得する
func (h *POISearchHelper) GetAvailableScenarios(theme string) []string {
	return model.GetScenariosForTheme(theme)
}

// GetThemeAndScenarioNames は指定されたテーマとシナリオの日本語名を取得する
func (h *POISearchHelper) GetThemeAndScenarioNames(theme, scenario string) (string, string) {
	themeName := model.GetThemeJapaneseName(theme)
	scenarioName := model.GetScenarioJapaneseName(scenario)
	return themeName, scenarioName
}

// ValidateCombination は組み合わせが有効かどうかをチェックする
// 1. 同一POIの重複チェック（2個以上の重複は無効）
// 2. 所要時間制限チェック（歩行時間を長くし、滞在時間を短くするため制限を大幅に緩和）
func (h *POISearchHelper) ValidateCombination(combination []*model.POI, estimatedDuration time.Duration, isHealthLongDistance bool) bool {
	// 1. 同一POI重複チェック
	if hasDuplicatePOIs(combination) {
		return false
	}

	// 2. 所要時間制限チェック（歩行時間重視のため制限を大幅に緩和）
	maxDuration := 180 * time.Minute // 3時間に拡大（長時間歩行を重視）
	if isHealthLongDistance {
		// 健康テーマのロングコースの場合は制限なし
		return true
	}

	return estimatedDuration <= maxDuration
}

// hasDuplicatePOIs は組み合わせに2個以上の同一POIが含まれているかチェック
func hasDuplicatePOIs(combination []*model.POI) bool {
	poiCount := make(map[string]int)
	
	for _, poi := range combination {
		if poi != nil {
			poiCount[poi.ID]++
			if poiCount[poi.ID] >= 2 {
				return true
			}
		}
	}
	
	return false
}

// FilterValidCombinations は有効な組み合わせのみを返す
func (h *POISearchHelper) FilterValidCombinations(combinations [][]*model.POI, estimatedDurations []time.Duration, isHealthLongDistance bool) [][]*model.POI {
	var validCombinations [][]*model.POI
	
	for i, combination := range combinations {
		var duration time.Duration
		if i < len(estimatedDurations) {
			duration = estimatedDurations[i]
		}
		
		if h.ValidateCombination(combination, duration, isHealthLongDistance) {
			validCombinations = append(validCombinations, combination)
		}
	}
	
	return validCombinations
}
