package model

import "time"

type SuggestedRoute struct {
	Name          string
	Spots         []*POI
	TotalDuration time.Duration
	Polyline      string
}

type RouteDetails struct {
	TotalDuration time.Duration
	Polyline      string
}

type RouteProposalRequest struct {
	StartLocation       *Location        `json:"start_location" validate:"required"`
	DestinationLocation *Location        `json:"destination_location"` // null可（お散歩モード）
	Mode                string           `json:"mode" validate:"required,oneof=destination time_based"`
	TimeMinutes         int              `json:"time_minutes"` // modeが"time_based"の場合必須
	Theme               string           `json:"theme" validate:"required"`
	RealtimeContext     *RealtimeContext `json:"realtime_context"`
}

type RealtimeContext struct {
	Weather   string `json:"weather"`     // "sunny", "cloudy", "rainy"など
	TimeOfDay string `json:"time_of_day"` // "morning", "afternoon", "evening"
}

type RouteProposalResponse struct {
	Proposals []RouteProposal `json:"proposals"`
}

type RouteProposal struct {
	ProposalID               string           `json:"proposal_id"`                // 一時ID
	Title                    string           `json:"title"`                      // 物語のタイトル
	EstimatedDurationMinutes int              `json:"estimated_duration_minutes"` // 予想時間
	EstimatedDistanceMeters  int              `json:"estimated_distance_meters"`  // 予想距離
	Theme                    string           `json:"theme"`                      // テーマ
	DisplayHighlights        []string         `json:"display_highlights"`         // ハイライト
	NavigationSteps          []NavigationStep `json:"navigation_steps"`           // ナビゲーションステップ
	RoutePolyline            string           `json:"route_polyline"`             // ルートポリライン
	GeneratedStory           string           `json:"generated_story"`            // 生成された物語
}

type FirestoreRouteProposal struct {
	Title                    string           `firestore:"title"`
	EstimatedDurationMinutes int              `firestore:"estimated_duration_minutes"`
	EstimatedDistanceMeters  int              `firestore:"estimated_distance_meters"`
	Theme                    string           `firestore:"theme"`
	DisplayHighlights        []string         `firestore:"display_highlights"`
	NavigationSteps          []NavigationStep `firestore:"navigation_steps"`
	RoutePolyline            string           `firestore:"route_polyline"`
	GeneratedStory           string           `firestore:"generated_story"`
	ExpireAt                 time.Time        `firestore:"expireAt"`
}

func (rp *RouteProposal) ToFirestoreRouteProposal(ttlHours int) *FirestoreRouteProposal {
	return &FirestoreRouteProposal{
		Title:                    rp.Title,
		EstimatedDurationMinutes: rp.EstimatedDurationMinutes,
		EstimatedDistanceMeters:  rp.EstimatedDistanceMeters,
		Theme:                    rp.Theme,
		DisplayHighlights:        rp.DisplayHighlights,
		NavigationSteps:          rp.NavigationSteps,
		RoutePolyline:            rp.RoutePolyline,
		GeneratedStory:           rp.GeneratedStory,
		ExpireAt:                 time.Now().Add(time.Duration(ttlHours) * time.Hour),
	}
}

func (frp *FirestoreRouteProposal) ToRouteProposal(proposalID string) *RouteProposal {
	return &RouteProposal{
		ProposalID:               proposalID,
		Title:                    frp.Title,
		EstimatedDurationMinutes: frp.EstimatedDurationMinutes,
		EstimatedDistanceMeters:  frp.EstimatedDistanceMeters,
		Theme:                    frp.Theme,
		DisplayHighlights:        frp.DisplayHighlights,
		NavigationSteps:          frp.NavigationSteps,
		RoutePolyline:            frp.RoutePolyline,
		GeneratedStory:           frp.GeneratedStory,
	}
}

type NavigationStep struct {
	Type                 string  `json:"type"`                    // "navigation" or "poi"
	Name                 string  `json:"name,omitempty"`          // POIの名前（typeがpoiの場合）
	POIId                string  `json:"poi_id,omitempty"`        // POI ID（typeがpoiの場合）
	Description          string  `json:"description"`             // 説明
	Latitude             float64 `json:"latitude,omitempty"`      // 緯度（typeがpoiの場合）
	Longitude            float64 `json:"longitude,omitempty"`     // 経度（typeがpoiの場合）
	DistanceToNextMeters int     `json:"distance_to_next_meters"` // 次のステップまでの距離
}

type RouteRecalculateRequest struct {
	ProposalID           string               `json:"proposal_id" validate:"required"`           // 元の提案ID
	CurrentLocation      *Location            `json:"current_location" validate:"required"`      // ユーザーの現在地
	DestinationLocation  *Location            `json:"destination_location"`                      // 目的地（null可）
	Mode                 string               `json:"mode" validate:"required,oneof=destination time_based"` // モード
	VisitedPOIs          *VisitedPOIsContext  `json:"visited_pois" validate:"required"`          // 訪問済みPOI情報
	RealtimeContext      *RealtimeContext     `json:"realtime_context"`                          // リアルタイム情報
}

// VisitedPOIsContext は訪問済みPOI情報を格納
type VisitedPOIsContext struct {
	PreviousPOIs []PreviousPOI `json:"previous_pois" validate:"required"` // 訪問済みPOIリスト
}

// PreviousPOI は訪問済みPOIの情報
type PreviousPOI struct {
	Name  string `json:"name" validate:"required"`   // POI名
	POIId string `json:"poi_id" validate:"required"` // POI ID
}

// RouteRecalculateResponse はルート再計算のレスポンス
type RouteRecalculateResponse struct {
	UpdatedRoute *UpdatedRoute `json:"updated_route"`
}

// UpdatedRoute は再計算された新しいルート情報
type UpdatedRoute struct {
	Title                    string           `json:"title"`                        // 更新された物語タイトル
	EstimatedDurationMinutes int              `json:"estimated_duration_minutes"`   // 予想時間
	EstimatedDistanceMeters  int              `json:"estimated_distance_meters"`    // 予想距離
	Highlights               []string         `json:"highlights"`                   // 新しいハイライト
	NavigationSteps          []NavigationStep `json:"navigation_steps"`             // 更新されたナビゲーションステップ
	RoutePolyline            string           `json:"route_polyline"`               // ルートポリライン
	GeneratedStory           string           `json:"generated_story"`              // 更新された物語
}

// RouteRecalculateContext は再計算処理で使用する内部コンテキスト
type RouteRecalculateContext struct {
	OriginalProposal   *RouteProposal    // Firestoreから取得した元の提案
	RemainingPOIs      []*POI            // 未訪問のPOIリスト
	NewDiscoveryPOIs   []*POI            // 新たに発見されたPOIリスト
	UpdatedCombination []*POI            // 更新された経由地リスト
}
