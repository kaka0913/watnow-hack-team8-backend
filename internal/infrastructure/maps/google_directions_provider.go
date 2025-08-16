package maps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Team8-App/internal/domain/model"
)

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
	Routes       []route `json:"routes"`
	Status       string  `json:"status"`
	ErrorMessage string  `json:"error_message,omitempty"`
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
