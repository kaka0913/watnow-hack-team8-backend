### ドメイン層の完全なコード

#### `internal/domain/model/` - モデル定義

ドメインで扱う「モノ」を定義します。

```go
// internal/domain/model/route.go
package model

import "time"

// SuggestedRoute は最終的に提案される一つの完全なルート
type SuggestedRoute struct {
	Name          string    // 例：「美食家の密集カフェ巡り」
	Spots         []*POI    // ルートに含まれる3つのスポット（最適化された順序）
	TotalDuration time.Duration // 合計所要時間
	Polyline      string    // 地図に描画するためのエンコードされた経路情報
}

// RouteDetails はGoogle Maps APIから返される生のルート情報
// SuggestedRouteを構築するために使われる
type RouteDetails struct {
	TotalDuration time.Duration
	Polyline      string
}
```

```go
// internal/domain/model/pois.go
package model

// POI Point of Interest（興味のあるスポット）を表すモデル
type POI struct {
	ID         string    `json:"id" db:"id"`                     // ユニークなスポットID
	Name       string    `json:"name" db:"name"`                 // スポット名
	Location   *Geometry `json:"location" db:"location"`         // 位置情報（PostGIS GEOMETRY型）
	Categories []string  `json:"categories" db:"categories"`     // カテゴリ（複数対応）
	GridCellID int       `json:"grid_cell_id" db:"grid_cell_id"` // グリッドセルID
	Rate       float64   `json:"rate" db:"rate"`                 // 評価値
	URL        *string   `json:"url,omitempty" db:"url"`         // URL（NULLABLE）
}

// LatLng 緯度経度を表す基本的な型（経路検索などで使用）
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ToLatLng POIの位置情報をLatLng型に変換
func (p *POI) ToLatLng() LatLng {
	if p.Location != nil && len(p.Location.Coordinates) >= 2 {
		return LatLng{
			Lat: p.Location.Coordinates[1], // latitude
			Lng: p.Location.Coordinates[0], // longitude
		}
	}
	return LatLng{}
}

// Geometry PostGIS GEOMETRY型に対応する構造体
type Geometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"` // [longitude, latitude]
}
```

-----

#### `internal/domain/strategy/` - ルート組み合わせ戦略

テーマごとの「戦略」を定義します。

```go
// internal/domain/strategy/strategy_interface.go
package strategy

import "your_project/internal/domain/model"

// StrategyInterface は、POI候補リストからテーマに合った組み合わせを見つける戦略のインターフェース
type StrategyInterface interface {
	FindCombinations(candidates []*model.POI) [][]*model.POI
}
```

```go
// internal/domain/strategy/gourmet_strategy.go
package strategy

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/helper" // ヘルパー関数を使用
)

// GourmetStrategy はカフェやベーカリーを巡るルートを提案する
type GourmetStrategy struct{}

func NewGourmetStrategy() StrategyInterface {
	return &GourmetStrategy{}
}

func (s *GourmetStrategy) FindCombinations(candidates []*model.POI) [][]*model.POI {
	gourmetSpots := helper.FilterByCategory(candidates, []string{"cafe", "bakery"})
	if len(gourmetSpots) < 3 {
		return nil
	}

	spotA := helper.FindHighestRated(gourmetSpots)
	others := helper.RemovePOI(gourmetSpots, spotA)
	helper.SortByDistance(spotA, others)

	combination := []*model.POI{spotA, others[0], others[1]}
	return [][]*model.POI{combination}
}
```

-----

#### `internal/domain/service/` - ドメインサービスとヘルパー関数

ルート提案のコアロジックと、それに必要な全てのヘルパー関数をここに実装します。

```go
// internal/domain/service/route_suggestion_service.go
package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/domain/strategy"
	"Team8-App/internal/domain/helper" // ヘルパー関数を使用
	"Team8-App/internal/infrastructure/maps" // Google Maps実装を直接使用
)

// RouteSuggestionService はテーマに応じたルート提案のオーケストレーションを行う
type RouteSuggestionService interface {
	SuggestRoutesForTheme(ctx context.Context, theme string, userLocation model.LatLng) ([]*model.SuggestedRoute, error)
}

type routeSuggestionService struct {
	directionsProvider *maps.GoogleDirectionsProvider
	strategies         map[string]strategy.StrategyInterface
	poiRepo            repository.POIsRepository
}

func NewRouteSuggestionService(dp *maps.GoogleDirectionsProvider, repo repository.POIsRepository) RouteSuggestionService {
	return &routeSuggestionService{
		directionsProvider: dp,
		poiRepo:            repo,
		strategies: map[string]strategy.StrategyInterface{
			"gourmet": strategy.NewGourmetStrategy(),
			// ... 他の戦略も同様に初期化 ...
		},
	}
}

// SuggestRoutesForTheme はテーマに基づき、POI取得からルート提案までを一貫して行う
func (s *routeSuggestionService) SuggestRoutesForTheme(ctx context.Context, theme string, userLocation model.LatLng) ([]*model.SuggestedRoute, error) {
	// Step 1: テーマに合ったPOIカテゴリを決定し、リポジトリから候補を取得する
	targetCategories := mapThemeToCategories(theme)
	if len(targetCategories) == 0 {
		return nil, errors.New("テーマに該当するカテゴリがありません: " + theme)
	}
	candidates, err := s.poiRepo.FindNearbyByCategories(ctx, userLocation, targetCategories, 1500, 30)
	if err != nil {
		return nil, fmt.Errorf("POI候補の取得に失敗しました: %w", err)
	}
	if len(candidates) < 3 {
		return nil, errors.New("周辺に見つかったスポットが3件未満です")
	}

	// Step 2: 戦略を選択し、組み合わせを取得
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	combinations := selectedStrategy.FindCombinations(candidates)
	if len(combinations) == 0 {
		return nil, errors.New("このテーマに合うルートの組み合わせが見つかりませんでした")
	}

	// Step 3: 組み合わせからルート構築処理を共通メソッドに委譲
	suggestedRoutes := s.buildRoutesFromCombinations(ctx, theme, combinations)

	return suggestedRoutes, nil
}

// buildRoutesFromCombinations は、複数の組み合わせから並行でルートを構築する
func (s *routeSuggestionService) buildRoutesFromCombinations(ctx context.Context, theme string, combinations [][]*model.POI) []*model.SuggestedRoute {
	var suggestedRoutes []*model.SuggestedRoute
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, comb := range combinations {
		wg.Add(1)
		go func(combination []*model.POI) {
			defer wg.Done()
			routeName := generateRouteName(theme, combination)
			route, err := s.optimizeAndBuildRoute(ctx, routeName, combination)
			if err == nil {
				mu.Lock()
				suggestedRoutes = append(suggestedRoutes, route)
				mu.Unlock()
			}
		}(comb)
	}
	wg.Wait()
	return suggestedRoutes
}

// optimizeAndBuildRoute は3つのスポットを巡る最短ルートを決定する
func (s *routeSuggestionService) optimizeAndBuildRoute(ctx context.Context, name string, combination []*model.POI) (*model.SuggestedRoute, error) {
	if len(combination) != 3 {
		return nil, errors.New("組み合わせは3つのスポットである必要があります")
	}
	permutations := helper.GeneratePermutations(combination)

	var bestRoute *model.SuggestedRoute
	var bestPermutation []*model.POI

	for _, p := range permutations {
		// DirectionsProvider を呼び出して実際のルート情報を取得
		details, err := s.directionsProvider.GetWalkingRoute(ctx, p[0].Location, p[1].Location, p[2].Location)
		if err != nil {
			continue
		}

		if bestRoute == nil || details.TotalDuration < bestRoute.TotalDuration {
			bestRoute = &model.SuggestedRoute{
				Name:          name,
				TotalDuration: details.TotalDuration,
				Polyline:      details.Polyline,
			}
			bestPermutation = p
		}
	}

	if bestRoute == nil {
		return nil, errors.New("外部APIから有効なルートが見つかりませんでした")
	}

	bestRoute.Spots = bestPermutation // 最適な順序のスポットをセット
	return bestRoute, nil
}

// mapThemeToCategories はテーマ名と検索対象のPOIカテゴリをマッピングする
func mapThemeToCategories(theme string) []string {
	switch theme {
	case "gourmet":
		return []string{"cafe", "bakery"}
	case "nature":
		return []string{"park", "tourist_attraction"}
	case "history":
		return []string{"tourist_attraction", "museum", "book_store"}
	case "art":
		return []string{"art_gallery", "museum"}
	case "shopping":
		return []string{"store", "home_goods_store", "book_store", "florist"}
	case "date":
		return []string{"park", "cafe", "florist", "art_gallery", "museum", "book_store"}
	case "magellan":
		return []string{"cafe", "park", "tourist_attraction", "art_gallery", "book_store", "bakery", "store", "home_goods_store", "museum", "florist", "library"}
	default:
		return []string{}
	}
}

// generateRouteName はテーマとスポット名からルート名を生成する
func generateRouteName(theme string, comb []*model.POI) string {
	// 実際のアプリではもっと凝った名前にする
	return fmt.Sprintf("【%s】%sと%sを巡る旅", theme, comb[0].Name, comb[1].Name)
}
```

#### `internal/infrastructure/maps/` - Google Maps API の実装

Google Maps Directions APIを使用した経路検索の具体的な実装です。

```go
// internal/infrastructure/maps/google_directions_provider.go
package maps

// GoogleDirectionsProvider はGoogle Maps Directions APIを使用した経路検索の実装
type GoogleDirectionsProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewGoogleDirectionsProvider は新しいプロバイダを生成する
func NewGoogleDirectionsProvider(apiKey string) *GoogleDirectionsProvider {
	return &GoogleDirectionsProvider{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetWalkingRoute はGoogle Maps Directions APIを呼び出して徒歩ルート情報を取得する
func (g *GoogleDirectionsProvider) GetWalkingRoute(ctx context.Context, origin model.LatLng, waypoints ...model.LatLng) (*model.RouteDetails, error) {
	// 1. APIリクエストURLを構築
	reqURL, err := g.buildURL(origin, waypoints...)
	if err != nil {
		return nil, fmt.Errorf("URLの構築に失敗: %w", err)
	}

	// 2. HTTPリクエストを作成・実行
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("リクエストの作成に失敗: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("APIリクエストに失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("APIからエラーステータスが返されました: %s", resp.Status)
	}

	// 3. JSONレスポンスをパース
	var apiResp googleRouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("JSONのパースに失敗: %w", err)
	}

	if len(apiResp.Routes) == 0 {
		return nil, errors.New("APIから有効なルートが返されませんでした")
	}

	// 4. ドメインモデルに変換して返す
	firstRoute := apiResp.Routes[0]
	var totalDurationSec int
	for _, leg := range firstRoute.Legs {
		totalDurationSec += leg.Duration.Value
	}

	return &model.RouteDetails{
		TotalDuration: time.Duration(totalDurationSec) * time.Second,
		Polyline:      firstRoute.OverviewPolyline.Points,
	}, nil
}

// GetWalkingRouteFromPOIs はPOIから位置情報を取得して徒歩ルート情報を取得する便利メソッド
func (g *GoogleDirectionsProvider) GetWalkingRouteFromPOIs(ctx context.Context, origin *model.POI, waypoints ...*model.POI) (*model.RouteDetails, error) {
	originLatLng := origin.ToLatLng()
	waypointLatLngs := make([]model.LatLng, len(waypoints))
	for i, wp := range waypoints {
		waypointLatLngs[i] = wp.ToLatLng()
	}
	return g.GetWalkingRoute(ctx, originLatLng, waypointLatLngs...)
}

func (g *GoogleDirectionsProvider) buildURL(origin model.LatLng, waypoints ...model.LatLng) (string, error) {
	baseURL := "https://maps.googleapis.com/maps/api/directions/json"
	params := url.Values{}
	params.Set("origin", fmt.Sprintf("%f,%f", origin.Lat, origin.Lng))
	// 最後の地点がdestinationになる
	destination := waypoints[len(waypoints)-1]
	params.Set("destination", fmt.Sprintf("%f,%f", destination.Lat, destination.Lng))

	// 経由地を設定
	if len(waypoints) > 1 {
		viaPoints := make([]string, 0, len(waypoints)-1)
		for _, wp := range waypoints[:len(waypoints)-1] {
			viaPoints = append(viaPoints, fmt.Sprintf("%f,%f", wp.Lat, wp.Lng))
		}
		params.Set("waypoints", strings.Join(viaPoints, "|"))
	}

	params.Set("mode", "walking")
	params.Set("language", "ja")
	params.Set("key", g.apiKey)

	return fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil
}

// --- Google Maps APIのレスポンスをパースするための構造体 ---

type googleRouteResponse struct {
	Routes []route `json:"routes"`
}
type route struct {
	Legs             []leg            `json:"legs"`
	OverviewPolyline overviewPolyline `json:"overview_polyline"`
}
type leg struct {
	Duration duration `json:"duration"`
}
type duration struct {
	Value int `json:"value"` // seconds
}
type overviewPolyline struct {
	Points string `json:"points"`
}
```

#### `internal/domain/helper/` - ヘルパー関数群

POI 処理に関する汎用的なヘルパー関数を一箇所に集約します。

```go
// internal/domain/helper/poi_helper.go
package helper

import (
    "math"
    "sort"
    "Team8-App/internal/domain/model"
)

const earthRadiusKm = 6371.0

// HaversineDistance は2地点間の距離を計算する (km)
func HaversineDistance(p1, p2 model.LatLng) float64 {
    lat1 := p1.Lat * math.Pi / 180; lng1 := p1.Lng * math.Pi / 180
    lat2 := p2.Lat * math.Pi / 180; lng2 := p2.Lng * math.Pi / 180
    dLat := lat2 - lat1; dLng := lng2 - lng1
    a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    return earthRadiusKm * c
}

// FilterByCategory は指定されたカテゴリのPOIのみを抽出する
func FilterByCategory(pois []*model.POI, categories []string) []*model.POI {
    var filtered []*model.POI
    catSet := make(map[string]struct{})
    for _, c := range categories {
        catSet[c] = struct{}{}
    }
    for _, p := range pois {
        for _, cat := range p.Categories {
            if _, ok := catSet[cat]; ok {
                filtered = append(filtered, p)
                break
            }
        }
    }
    return filtered
}

// FindHighestRated は最も評価の高いPOIを見つける
func FindHighestRated(pois []*model.POI) *model.POI {
    if len(pois) == 0 {
        return nil
    }
    highest := pois[0]
    for _, p := range pois {
        if p.Rating > highest.Rating {
            highest = p
        }
    }
    return highest
}

// SortByDistance は基準地点からの距離でPOIスライスをソートする
func SortByDistance(origin *model.POI, targets []*model.POI) {
    sort.Slice(targets, func(i, j int) bool {
        distI := HaversineDistance(origin.Location, targets[i].Location)
        distJ := HaversineDistance(origin.Location, targets[j].Location)
        return distI < distJ
    })
}

// RemovePOI はスライスから特定のPOIを削除する
func RemovePOI(pois []*model.POI, target *model.POI) []*model.POI {
    var result []*model.POI
    for _, p := range pois {
        if p.ID != target.ID {
            result = append(result, p)
        }
    }
    return result
}

// GeneratePermutations はPOIスライスの全ての順列を生成する
func GeneratePermutations(pois []*model.POI) [][]*model.POI {
    var result [][]*model.POI
    var helper func([]*model.POI, int)
    helper = func(arr []*model.POI, n int) {
        if n == 1 {
            tmp := make([]*model.POI, len(arr))
            copy(tmp, arr)
            result = append(result, tmp)
        } else {
            for i := 0; i < n; i++ {
                helper(arr, n-1)
                if n%2 == 1 {
                    arr[0], arr[n-1] = arr[n-1], arr[0]
                } else {
                    arr[i], arr[n-1] = arr[n-1], arr[i]
                }
            }
        }
    }
    helper(pois, len(pois))
    return result
}
```
