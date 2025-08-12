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

// RouteDetails はDirectionsProviderから返される生のルート情報
// SuggestedRouteを構築するために使われる
type RouteDetails struct {
	TotalDuration time.Duration
	Polyline      string
}
```

-----

#### `internal/domain/external/` - 外部サービス契約

外部の経路検索サービスとの「契約（インターフェース）」を定義します。

```go
// internal/domain/external/directions_provider.go
package external

import (
	"context"
	"your_project/internal/domain/model"
)

// DirectionsProvider は経路検索サービスとの契約を定義するインターフェース
type DirectionsProvider interface {
	// GetWalkingRoute は指定した地点間の徒歩ルート情報を取得する
	GetWalkingRoute(ctx context.Context, origin model.LatLng, waypoints ...model.LatLng) (*model.RouteDetails, error)
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
	"your_project/internal/domain/model"
	"your_project/internal/domain/service/helper" // ヘルパー関数を別パッケージにした場合
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
	"math"
	"sort"
	"time"

	"your_project/internal/domain/external"
	"your_project/internal/domain/model"
	"your_project/internal/domain/strategy"
)

// RouteSuggestionService はテーマに応じたルート提案のオーケストレーションを行う
type RouteSuggestionService interface {
	SuggestRoutesForTheme(ctx context.Context, theme string, candidates []*model.POI) ([]*model.SuggestedRoute, error)
}

type routeSuggestionService struct {
	directionsProvider external.DirectionsProvider
	strategies         map[string]strategy.StrategyInterface
}

func NewRouteSuggestionService(dp external.DirectionsProvider) RouteSuggestionService {
	return &routeSuggestionService{
		directionsProvider: dp,
		strategies: map[string]strategy.StrategyInterface{
			"gourmet": strategy.NewGourmetStrategy(),
			// ... 他の戦略も同様に初期化 ...
		},
	}
}

func (s *routeSuggestionService) SuggestRoutesForTheme(ctx context.Context, theme string, candidates []*model.POI) ([]*model.SuggestedRoute, error) {
	selectedStrategy, ok := s.strategies[theme]
	if !ok {
		return nil, errors.New("対応していないテーマです: " + theme)
	}

	combinations := selectedStrategy.FindCombinations(candidates)
	if len(combinations) == 0 {
		return nil, errors.New("このテーマに合うルートの組み合わせが見つかりませんでした")
	}

	var suggestedRoutes []*model.SuggestedRoute
	for _, comb := range combinations {
		routeName := generateRouteName(theme, comb)
		route, err := s.optimizeAndBuildRoute(ctx, routeName, comb)
		if err == nil {
			suggestedRoutes = append(suggestedRoutes, route)
		}
	}
	return suggestedRoutes, nil
}

// optimizeAndBuildRoute は3つのスポットを巡る最短ルートを決定する
func (s *routeSuggestionService) optimizeAndBuildRoute(ctx context.Context, name string, combination []*model.POI) (*model.SuggestedRoute, error) {
	if len(combination) != 3 {
		return nil, errors.New("組み合わせは3つのスポットである必要があります")
	}
	permutations := generatePermutations(combination)

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

// --- ヘルパー関数群 ---
// generatePermutations はPOIスライスの全ての順列を生成する
func generatePermutations(pois []*model.POI) [][]*model.POI {
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

// generateRouteName はテーマとスポット名からルート名を生成する
func generateRouteName(theme string, comb []*model.POI) string {
	// 実際のアプリではもっと凝った名前にする
	return fmt.Sprintf("【%s】%sと%sを巡る旅", theme, comb[0].Name, comb[1].Name)
}

// (strategyパッケージから移動・または共通ヘルパーパッケージに配置)
func FilterByCategory(pois []*model.POI, categories []string) []*model.POI {
	var filtered []*model.POI
	catSet := make(map[string]struct{})
	for _, c := range categories {
		catSet[c] = struct{}{}
	}
	for _, p := range pois {
		if _, ok := catSet[p.Category]; ok {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

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

func SortByDistance(origin *model.POI, targets []*model.POI) {
	sort.Slice(targets, func(i, j int) bool {
		distI := haversineDistance(origin.Location, targets[i].Location)
		distJ := haversineDistance(origin.Location, targets[j].Location)
		return distI < distJ
	})
}

func RemovePOI(pois []*model.POI, target *model.POI) []*model.POI {
	var result []*model.POI
	for _, p := range pois {
		if p.ID != target.ID {
			result = append(result, p)
		}
	}
	return result
}

```

#### `internal/infrastructure/maps/` - Google Map API との接続

ルート提案の取得


```go
// internal/infrastructure/maps/google_directions_provider.go
package maps

// googleDirectionsProvider はDirectionsProviderインターフェースのGoogle Maps実装
type googleDirectionsProvider struct {
	apiKey     string
	httpClient *http.Client
}

// NewGoogleDirectionsProvider は新しいプロバイダを生成する
func NewGoogleDirectionsProvider(apiKey string) external.DirectionsProvider {
	return &googleDirectionsProvider{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetWalkingRoute はGoogle Maps Directions APIを呼び出して徒歩ルート情報を取得する
func (g *googleDirectionsProvider) GetWalkingRoute(ctx context.Context, origin model.LatLng, waypoints ...model.LatLng) (*model.RouteDetails, error) {
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

func (g *googleDirectionsProvider) buildURL(origin model.LatLng, waypoints ...model.LatLng) (string, error) {
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