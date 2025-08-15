package model

// ThemeConstants はアプリケーションで使用するテーマの定数
const (
	ThemeGourmet = "gourmet"
	ThemeNature  = "nature"
	ThemeHistory = "history"
)

// ScenarioConstants はアプリケーションで使用するシナリオの定数
const (
	// グルメテーマのシナリオ
	ScenarioCafeHopping  = "cafe_hopping"
	ScenarioBakeryTour   = "bakery_tour"
	ScenarioLocalGourmet = "local_gourmet"
	ScenarioSweetJourney = "sweet_journey"

	// 自然テーマのシナリオ
	ScenarioParkTour     = "park_tour"
	ScenarioRiverside    = "riverside"
	ScenarioTempleNature = "temple_nature"

	// 歴史テーマのシナリオ
	ScenarioTempleShrine = "temple_shrine"
	ScenarioMuseumTour   = "museum_tour"
	ScenarioOldTown      = "old_town"
	ScenarioCulturalWalk = "cultural_walk"
)

// ScenarioNameMap はシナリオIDから日本語名へのマッピング
var ScenarioNameMap = map[string]string{
	ScenarioParkTour:     "公園巡り",
	ScenarioRiverside:    "河川敷散歩",
	ScenarioTempleNature: "寺社と自然",
	ScenarioCafeHopping:  "カフェ巡り",
	ScenarioBakeryTour:   "ベーカリー巡り",
	ScenarioLocalGourmet: "地元グルメ",
	ScenarioSweetJourney: "スイーツ巡り",
	ScenarioTempleShrine: "寺社仏閣巡り",
	ScenarioMuseumTour:   "博物館巡り",
	ScenarioOldTown:      "古い街並み散策",
	ScenarioCulturalWalk: "文化的散歩",
}

// ThemeNameMap はテーマIDから日本語名へのマッピング
var ThemeNameMap = map[string]string{
	ThemeGourmet: "グルメ",
	ThemeNature:  "自然",
	ThemeHistory: "歴史",
}

// GetScenarioJapaneseName はシナリオIDから日本語名を取得する
func GetScenarioJapaneseName(scenario string) string {
	if name, ok := ScenarioNameMap[scenario]; ok {
		return name
	}
	return scenario // デフォルトはそのまま返す
}

// GetThemeJapaneseName はテーマIDから日本語名を取得する
func GetThemeJapaneseName(theme string) string {
	if name, ok := ThemeNameMap[theme]; ok {
		return name
	}
	return theme // デフォルトはそのまま返す
}

// GetGourmetScenarios はグルメテーマのシナリオ一覧を取得する
func GetGourmetScenarios() []string {
	return []string{
		ScenarioCafeHopping,
		ScenarioBakeryTour,
		ScenarioLocalGourmet,
		ScenarioSweetJourney,
	}
}

// GetNatureScenarios は自然テーマのシナリオ一覧を取得する
func GetNatureScenarios() []string {
	return []string{
		ScenarioParkTour,
		ScenarioRiverside,
		ScenarioTempleNature,
	}
}

// GetHistoryScenarios は歴史テーマのシナリオ一覧を取得する
func GetHistoryScenarios() []string {
	return []string{
		ScenarioTempleShrine,
		ScenarioMuseumTour,
		ScenarioOldTown,
		ScenarioCulturalWalk,
	}
}

// GetAllThemes は全テーマの一覧を取得する
func GetAllThemes() []string {
	return []string{
		ThemeGourmet,
		ThemeNature,
		ThemeHistory,
	}
}

// GetAllScenarios は全シナリオの一覧を取得する
func GetAllScenarios() []string {
	scenarios := make([]string, 0)
	scenarios = append(scenarios, GetGourmetScenarios()...)
	scenarios = append(scenarios, GetNatureScenarios()...)
	scenarios = append(scenarios, GetHistoryScenarios()...)
	return scenarios
}
