# ドメイン層 API リファレンス

このドキュメントは、ドメイン層の詳細実装を参照せずに利用できるよう、インターフェイスとデータ型を整理したリファレンスです。

## 目次

1. [インターフェイス一覧](#インターフェイス一覧)
2. [データモデル](#データモデル)
3. [定数・列挙型](#定数列挙型)
4. [使用例](#使用例)

---

## インターフェイス一覧

### 1. RouteSuggestionService

**概要**: ルート提案のメインサービス

```go
type RouteSuggestionService interface {
    SuggestRoutes(ctx context.Context, req *model.SuggestionRequest) ([]*model.SuggestedRoute, error)
    GetAvailableScenariosForTheme(theme string) ([]string, error)
}
```

**メソッド詳細**:
- `SuggestRoutes`: リクエストに基づいてルート提案を生成
- `GetAvailableScenariosForTheme`: 指定テーマで利用可能なシナリオ一覧を取得

---

### 2. StrategyInterface

**概要**: テーマ別のルート生成戦略

```go
type StrategyInterface interface {
    GetAvailableScenarios() []string
    FindCombinations(ctx context.Context, scenario string, userLocation model.LatLng) ([][]*model.POI, error)
    FindCombinationsWithDestination(ctx context.Context, scenario string, userLocation model.LatLng, destination model.LatLng) ([][]*model.POI, error)
}
```

**メソッド詳細**:
- `GetAvailableScenarios`: 戦略で対応するシナリオ一覧を取得
- `FindCombinations`: 通常モード（目的地なし）でのPOI組み合わせ生成
- `FindCombinationsWithDestination`: 目的地ありモードでのPOI組み合わせ生成

---

### 3. POIsRepository

**概要**: POI（Point of Interest）データアクセス

```go
type POIsRepository interface {
    GetByID(ctx context.Context, id string) (*model.POI, error)
    GetByGridCellID(ctx context.Context, gridCellID int) ([]model.POI, error)
    GetNearbyPOIs(ctx context.Context, lat, lng float64, radiusMeters int) ([]model.POI, error)
    GetByCategories(ctx context.Context, categories []string, lat, lng float64, radiusMeters int) ([]model.POI, error)
    GetByCategory(ctx context.Context, category string, lat, lng float64, radiusMeters int) ([]model.POI, error)
    GetByRatingRange(ctx context.Context, minRating float64, lat, lng float64, radiusMeters int) ([]model.POI, error)
    Create(ctx context.Context, poi *model.POI) error
    Update(ctx context.Context, poi *model.POI) error
    Delete(ctx context.Context, id string) error
    BulkCreate(ctx context.Context, pois []model.POI) error
    FindNearbyByCategories(ctx context.Context, location model.LatLng, categories []string, radiusMeters int, limit int) ([]*model.POI, error)
    FindNearbyByCategoriesIncludingHorror(ctx context.Context, location model.LatLng, categories []string, radiusMeters int, limit int) ([]*model.POI, error)
}
```

---

### 4. GridCellsRepository

**概要**: グリッドセルデータアクセス

```go
type GridCellsRepository interface {
    GetByID(ctx context.Context, id int) (*model.GridCell, error)
    GetContainingPoint(ctx context.Context, lat, lng float64) (*model.GridCell, error)
    GetByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.GridCell, error)
    Create(ctx context.Context, gridCell *model.GridCell) error
    Update(ctx context.Context, gridCell *model.GridCell) error
    Delete(ctx context.Context, id int) error
    GetAll(ctx context.Context) ([]model.GridCell, error)
}
```

---

### 5. WalksRepository

**概要**: 散歩記録データアクセス

```go
type WalksRepository interface {
    Create(ctx context.Context, walk *model.Walk) error
    GetByID(ctx context.Context, id string) (*model.Walk, error)
    GetWalksByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.WalkSummary, error)
    GetWalkDetail(ctx context.Context, id string) (*model.WalkDetail, error)
    GetAll(ctx context.Context) ([]model.Walk, error)
}
```

---

## データモデル

### 基本データ型

#### LatLng
**用途**: 緯度経度座標

```go
type LatLng struct {
    Lat float64 `json:"lat"`
    Lng float64 `json:"lng"`
}
```

#### Location
**用途**: 位置情報（APIリクエスト/レスポンス用）

```go
type Location struct {
    Latitude  float64 `json:"latitude" validate:"required,min=-90,max=90"`
    Longitude float64 `json:"longitude" validate:"required,min=-180,max=180"`
}
```

#### Geometry
**用途**: PostGIS GEOMETRY型対応

```go
type Geometry struct {
    Type        string    `json:"type"`
    Coordinates []float64 `json:"coordinates"` // [longitude, latitude]
}
```

---

### コアモデル

#### POI (Point of Interest)
**用途**: 興味のあるスポット情報

```go
type POI struct {
    ID         string    `json:"id" db:"id"`
    Name       string    `json:"name" db:"name"`
    Location   *Geometry `json:"location" db:"location"`
    Categories []string  `json:"categories" db:"categories"`
    GridCellID int       `json:"grid_cell_id" db:"grid_cell_id"`
    Rate       float64   `json:"rate" db:"rate"`
    URL        *string   `json:"url,omitempty" db:"url"`
}
```

**主要メソッド**:
- `ToLatLng() LatLng`: 位置情報をLatLng型に変換
- `GetURL() string`: URLを安全に取得
- `SetURL(url string)`: URLを設定
- `HasURL() bool`: URLが設定されているかチェック

#### SuggestedRoute
**用途**: 提案されたルート情報

```go
type SuggestedRoute struct {
    Name          string
    Spots         []*POI
    TotalDuration time.Duration
    Polyline      string
}
```

#### SuggestionRequest
**用途**: ルート提案リクエスト

```go
type SuggestionRequest struct {
    UserLocation LatLng   // 必須：スタート地点
    Theme        string   // 必須：テーマ
    Scenarios    []string // オプション：シナリオ指定
    Destination  *LatLng  // オプション：目的地
}
```

**主要メソッド**:
- `HasDestination() bool`: 目的地が指定されているかチェック
- `GetScenarios() []string`: シナリオリストを取得
- `HasSpecificScenarios() bool`: 特定シナリオが指定されているかチェック

---

### 散歩記録関連

#### Walk
**用途**: 散歩記録の完全データ

```go
type Walk struct {
    ID              string      `json:"id" db:"id"`
    Title           string      `json:"title" db:"title"`
    Area            string      `json:"area" db:"area"`
    Description     string      `json:"description" db:"description"`
    Theme           string      `json:"theme" db:"theme"`
    POIIds          []string    `json:"poi_ids" db:"poi_ids"`
    Tags            []string    `json:"tags" db:"tags"`
    DurationMinutes int         `json:"duration_minutes" db:"duration_minutes"`
    DistanceMeters  int         `json:"distance_meters" db:"distance_meters"`
    RoutePolyline   string      `json:"route_polyline" db:"route_polyline"`
    Impressions     string      `json:"impressions" db:"impressions"`
    StartLocation   *Location   `json:"start_location" db:"start_location"`
    EndLocation     *Location   `json:"end_location" db:"end_location"`
    RouteBounds     *GeoPolygon `json:"route_bounds,omitempty" db:"route_bounds"`
    CreatedAt       time.Time   `json:"created_at" db:"created_at"`
}
```

#### WalkSummary
**用途**: 散歩記録のサマリー情報

```go
type WalkSummary struct {
    ID              string    `json:"id"`
    Title           string    `json:"title"`
    AreaName        string    `json:"area_name"`
    Date            string    `json:"date"`
    Summary         string    `json:"summary"`
    DurationMinutes int       `json:"duration_minutes"`
    DistanceMeters  int       `json:"distance_meters"`
    Tags            []string  `json:"tags"`
    StartLocation   *Location `json:"start_location"`
    EndLocation     *Location `json:"end_location"`
    RoutePolyline   string    `json:"route_polyline"`
}
```

#### WalkDetail
**用途**: 散歩記録の詳細情報

```go
type WalkDetail struct {
    ID              string           `json:"id"`
    Title           string           `json:"title"`
    AreaName        string           `json:"area_name"`
    Date            string           `json:"date"`
    Description     string           `json:"description"`
    Theme           string           `json:"theme"`
    DurationMinutes int              `json:"duration_minutes"`
    DistanceMeters  int              `json:"distance_meters"`
    RoutePolyline   string           `json:"route_polyline"`
    Tags            []string         `json:"tags"`
    NavigationSteps []NavigationStep `json:"navigation_steps"`
}
```

---

### API関連モデル

#### RouteProposalRequest
**用途**: ルート提案APIリクエスト

```go
type RouteProposalRequest struct {
    StartLocation       *Location        `json:"start_location" validate:"required"`
    DestinationLocation *Location        `json:"destination_location"`
    Mode                string           `json:"mode" validate:"required,oneof=destination time_based"`
    TimeMinutes         int              `json:"time_minutes"`
    Theme               string           `json:"theme" validate:"required"`
    RealtimeContext     *RealtimeContext `json:"realtime_context"`
}
```

#### RouteProposal
**用途**: ルート提案内容

```go
type RouteProposal struct {
    ProposalID               string           `json:"proposal_id"`
    Title                    string           `json:"title"`
    EstimatedDurationMinutes int              `json:"estimated_duration_minutes"`
    EstimatedDistanceMeters  int              `json:"estimated_distance_meters"`
    Theme                    string           `json:"theme"`
    DisplayHighlights        []string         `json:"display_highlights"`
    NavigationSteps          []NavigationStep `json:"navigation_steps"`
    RoutePolyline            string           `json:"route_polyline"`
    GeneratedStory           string           `json:"generated_story"`
}
```

#### NavigationStep
**用途**: ナビゲーションステップ

```go
type NavigationStep struct {
    Type                 string  `json:"type"`                    // "navigation" or "poi"
    Name                 string  `json:"name,omitempty"`          // POI名（typeがpoiの場合）
    POIId                string  `json:"poi_id,omitempty"`        // POI ID（typeがpoiの場合）
    Description          string  `json:"description"`             // 説明
    Latitude             float64 `json:"latitude,omitempty"`      // 緯度（typeがpoiの場合）
    Longitude            float64 `json:"longitude,omitempty"`     // 経度（typeがpoiの場合）
    DistanceToNextMeters int     `json:"distance_to_next_meters"` // 次のステップまでの距離
}
```

---

### その他のモデル

#### GridCell
**用途**: グリッドセル情報

```go
type GridCell struct {
    ID       int       `json:"id" db:"id"`
    Geometry *Geometry `json:"geometry" db:"geometry"`
    Geohash  string    `json:"geohash" db:"geohash"`
}
```

#### RealtimeContext
**用途**: リアルタイムコンテキスト情報

```go
type RealtimeContext struct {
    Weather   string `json:"weather"`     // "sunny", "cloudy", "rainy"など
    TimeOfDay string `json:"time_of_day"` // "morning", "afternoon", "evening"
}
```

---

## 定数・列挙型

### テーマ定数

```go
const (
    ThemeGourmet           = "gourmet"
    ThemeNature            = "nature"
    ThemeHistoryAndCulture = "history_and_culture"
    ThemeHorror            = "horror"
)
```

### シナリオ定数

#### グルメテーマ
```go
const (
    ScenarioCafeHopping  = "cafe_hopping"   // カフェ巡り
    ScenarioBakeryTour   = "bakery_tour"    // ベーカリー巡り
    ScenarioLocalGourmet = "local_gourmet"  // 地元グルメ
    ScenarioSweetJourney = "sweet_journey"  // スイーツ巡り
)
```

#### 自然テーマ
```go
const (
    ScenarioParkTour     = "park_tour"      // 公園巡り
    ScenarioRiverside    = "riverside"      // 河川敷散歩
    ScenarioTempleNature = "temple_nature"  // 寺社と自然
)
```

#### 歴史・文化テーマ
```go
const (
    ScenarioTempleShrine = "temple_shrine"  // 寺社仏閣巡り
    ScenarioMuseumTour   = "museum_tour"    // 博物館巡り
    ScenarioOldTown      = "old_town"       // 古い街並み散策
    ScenarioCulturalWalk = "cultural_walk"  // 文化的散歩
)
```

#### ホラーテーマ
```go
const (
    ScenarioGhostTour    = "ghost_tour"     // 心霊スポット巡り
    ScenarioHauntedRuins = "haunted_ruins"  // 廃墟探索
    ScenarioCursedNature = "cursed_nature"  // 呪いの自然
    ScenarioCemeteryWalk = "cemetery_walk"  // 墓地・慰霊散歩
)
```

### ヘルパー関数

```go
// テーマとシナリオの日本語名取得
func GetScenarioJapaneseName(scenario string) string
func GetThemeJapaneseName(theme string) string

// シナリオ一覧取得
func GetGourmetScenarios() []string
func GetNatureScenarios() []string
func GetHistoryAndCultureScenarios() []string
func GetHorrorScenarios() []string
func GetAllThemes() []string
func GetAllScenarios() []string
```

---

## 使用例

### 1. ルート提案の基本的な使用

```go
// サービスの初期化（DIコンテナから取得）
routeService := container.GetRouteSuggestionService()

// リクエストの作成
req := &model.SuggestionRequest{
    UserLocation: model.LatLng{Lat: 35.6762, Lng: 139.6503}, // 東京駅
    Theme:        model.ThemeGourmet,
    Scenarios:    []string{model.ScenarioCafeHopping},
}

// ルート提案の実行
routes, err := routeService.SuggestRoutes(ctx, req)
if err != nil {
    return err
}

// 結果の処理
for _, route := range routes {
    fmt.Printf("ルート: %s (所要時間: %v)\n", route.Name, route.TotalDuration)
    for i, poi := range route.Spots {
        fmt.Printf("  %d. %s\n", i+1, poi.Name)
    }
}
```

### 2. 目的地ありモードの使用

```go
destination := model.LatLng{Lat: 35.6586, Lng: 139.7454} // 新宿

req := &model.SuggestionRequest{
    UserLocation: model.LatLng{Lat: 35.6762, Lng: 139.6503},
    Theme:        model.ThemeNature,
    Destination:  &destination, // 目的地を指定
}

routes, err := routeService.SuggestRoutes(ctx, req)
```

### 3. 利用可能なシナリオの取得

```go
scenarios, err := routeService.GetAvailableScenariosForTheme(model.ThemeNature)
if err != nil {
    return err
}

for _, scenario := range scenarios {
    japName := model.GetScenarioJapaneseName(scenario)
    fmt.Printf("%s (%s)\n", japName, scenario)
}
```

### 4. POI検索の使用

```go
// リポジトリの取得（DIコンテナから）
poiRepo := container.GetPOIsRepository()

// カテゴリ別POI検索
location := model.LatLng{Lat: 35.6762, Lng: 139.6503}
pois, err := poiRepo.FindNearbyByCategories(
    ctx, 
    location,
    []string{"cafe", "bakery"},
    1500, // 半径1.5km
    10,   // 最大10件
)
```

---

## 備考

### エラーハンドリング
- 全てのメソッドは`error`を返すため、適切なエラーハンドリングが必要
- ビジネスロジックエラーと技術的なエラーを区別して処理

### パフォーマンス考慮事項
- 並行処理に対応したメソッドが多いため、`context.Context`を適切に利用
- リポジトリメソッドはデータベースアクセスを伴うため、適切なタイムアウト設定を推奨

### 依存関係
- `context`パッケージ: 全てのメソッドで必要
- `time`パッケージ: 所要時間計算で利用
- PostGIS: 地理情報処理に利用
