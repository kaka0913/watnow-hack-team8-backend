package model

// ThemeConstants はアプリケーションで使用するテーマの定数
const (
	ThemeGourmet           = "gourmet"
	ThemeNature            = "nature"
	ThemeHistoryAndCulture = "history_and_culture"
	ThemeHorror            = "horror"
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

	// ホラーテーマのシナリオ
	ScenarioGhostTour    = "ghost_tour"
	ScenarioHauntedRuins = "haunted_ruins"
	ScenarioCursedNature = "cursed_nature"
	ScenarioCemeteryWalk = "cemetery_walk"
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
	ScenarioGhostTour:    "心霊スポット巡り",
	ScenarioHauntedRuins: "廃墟探索",
	ScenarioCursedNature: "呪いの自然",
	ScenarioCemeteryWalk: "墓地・慰霊散歩",
}

// ThemeNameMap はテーマIDから日本語名へのマッピング
var ThemeNameMap = map[string]string{
	ThemeGourmet:           "グルメ",
	ThemeNature:            "自然",
	ThemeHistoryAndCulture: "歴史・文化探訪",
	ThemeHorror:            "ホラー",
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

// GetHistoryAndCultureScenarios は歴史・文化探訪テーマのシナリオ一覧を取得する
func GetHistoryAndCultureScenarios() []string {
	return []string{
		ScenarioTempleShrine,
		ScenarioMuseumTour,
		ScenarioOldTown,
		ScenarioCulturalWalk,
	}
}

// GetHorrorScenarios はホラーテーマのシナリオ一覧を取得する
func GetHorrorScenarios() []string {
	return []string{
		ScenarioGhostTour,
		ScenarioHauntedRuins,
		ScenarioCursedNature,
		ScenarioCemeteryWalk,
	}
}

// GetAllThemes は全テーマの一覧を取得する
func GetAllThemes() []string {
	return []string{
		ThemeGourmet,
		ThemeNature,
		ThemeHistoryAndCulture,
		ThemeHorror,
	}
}

// GetAllScenarios は全シナリオの一覧を取得する
func GetAllScenarios() []string {
	scenarios := make([]string, 0)
	scenarios = append(scenarios, GetGourmetScenarios()...)
	scenarios = append(scenarios, GetNatureScenarios()...)
	scenarios = append(scenarios, GetHistoryAndCultureScenarios()...)
	scenarios = append(scenarios, GetHorrorScenarios()...)
	return scenarios
}

// シナリオ固有のPOIカテゴリマッピング
var ScenarioCategoriesMap = map[string][]string{
	// グルメシナリオ
	ScenarioCafeHopping:  {"カフェ", "ベーカリー", "観光名所"},
	ScenarioBakeryTour:   {"ベーカリー", "カフェ", "店舗"},
	ScenarioLocalGourmet: {"レストラン", "店舗", "観光名所"},
	ScenarioSweetJourney: {"ベーカリー", "カフェ", "スイーツ", "店舗"},

	// 自然シナリオ
	ScenarioParkTour:     {"公園", "観光名所", "ベーカリー", "カフェ"},
	ScenarioRiverside:    {"カフェ", "観光名所", "公園", "自然スポット"},
	ScenarioTempleNature: {"寺院", "公園", "観光名所", "店舗"},

	// 歴史・文化シナリオ
	ScenarioTempleShrine: {"寺院", "観光名所"},
	ScenarioMuseumTour:   {"博物館", "美術館・ギャラリー", "観光名所"},
	ScenarioOldTown:      {"観光名所", "店舗", "文化施設"},
	ScenarioCulturalWalk: {"観光名所", "博物館", "美術館・ギャラリー", "文化施設"},

	// ホラーシナリオ
	ScenarioGhostTour:    {"観光名所", "寺院", "店舗"},
	ScenarioHauntedRuins: {"観光名所", "廃墟"},
	ScenarioCursedNature: {"墓地", "寺院", "公園", "自然スポット"},
	ScenarioCemeteryWalk: {"墓地", "寺院", "観光名所"},
}

// テーマごとのPOIカテゴリマッピング
var ThemeCategoriesMap = map[string][]string{
	ThemeGourmet: {
		"カフェ", "ベーカリー", "レストラン", "スイーツ", "店舗", "観光名所",
	},
	ThemeNature: {
		"カフェ", "観光名所", "公園", "寺院", "自然スポット", "ベーカリー", "店舗",
	},
	ThemeHistoryAndCulture: {
		"寺院", "博物館", "美術館・ギャラリー", "観光名所", "店舗", "文化施設",
	},
	ThemeHorror: {
		"観光名所", "寺院", "公園", "廃墟", "墓地", "自然スポット", "店舗",
	},
}

// GetScenarioCategories はシナリオ固有のPOIカテゴリ一覧を取得する
func GetScenarioCategories(scenario string) []string {
	if categories, ok := ScenarioCategoriesMap[scenario]; ok {
		return categories
	}
	return nil // シナリオ固有のカテゴリがない場合はnilを返す
}

// GetThemeCategories はテーマで使用するPOIカテゴリ一覧を取得する
func GetThemeCategories(theme string) []string {
	if categories, ok := ThemeCategoriesMap[theme]; ok {
		return categories
	}
	// デフォルトは全カテゴリ
	return []string{"カフェ", "観光名所", "公園", "博物館", "レストラン", "店舗", "ホテル", "寺院", "自然スポット", "ベーカリー", "美術館・ギャラリー"}
}

// GetGourmetCategories はグルメテーマのPOIカテゴリ一覧を取得する
func GetGourmetCategories() []string {
	return ThemeCategoriesMap[ThemeGourmet]
}

// GetNatureCategories は自然テーマのPOIカテゴリ一覧を取得する
func GetNatureCategories() []string {
	return ThemeCategoriesMap[ThemeNature]
}

// GetHistoryAndCultureCategories は歴史・文化探訪テーマのPOIカテゴリ一覧を取得する
func GetHistoryAndCultureCategories() []string {
	return ThemeCategoriesMap[ThemeHistoryAndCulture]
}

// GetHorrorCategories はホラーテーマのPOIカテゴリ一覧を取得する
func GetHorrorCategories() []string {
	return ThemeCategoriesMap[ThemeHorror]
}

// GetCategoriesForThemeAndScenario はテーマとシナリオに応じて最適なカテゴリを取得する
// シナリオ固有のカテゴリがある場合はそれを、ない場合はテーマのカテゴリを返す
func GetCategoriesForThemeAndScenario(theme, scenario string) []string {
	// シナリオ固有のカテゴリがある場合は優先
	if scenarioCategories := GetScenarioCategories(scenario); len(scenarioCategories) > 0 {
		return scenarioCategories
	}
	// シナリオ固有のカテゴリがない場合はテーマのカテゴリを返す
	return GetThemeCategories(theme)
}

// IsValidTheme は有効なテーマかどうかをチェックする
func IsValidTheme(theme string) bool {
	_, ok := ThemeCategoriesMap[theme]
	return ok
}

// IsValidScenario は有効なシナリオかどうかをチェックする
func IsValidScenario(scenario string) bool {
	_, ok := ScenarioNameMap[scenario]
	return ok
}

// GetScenariosForTheme は指定されたテーマのシナリオ一覧を取得する
func GetScenariosForTheme(theme string) []string {
	switch theme {
	case ThemeGourmet:
		return GetGourmetScenarios()
	case ThemeNature:
		return GetNatureScenarios()
	case ThemeHistoryAndCulture:
		return GetHistoryAndCultureScenarios()
	case ThemeHorror:
		return GetHorrorScenarios()
	default:
		return []string{}
	}
}
