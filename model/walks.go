package model

import (
	"time"
)

type Walk struct {
	ID              string    `json:"id" db:"id"`                             // ユニークな散歩ID
	Title           string    `json:"title" db:"title"`                       // 物語のタイトル
	Area            string    `json:"area" db:"area"`                         // エリア名
	Description     string    `json:"description" db:"description"`           // 物語の本文
	Theme           string    `json:"theme" db:"theme"`                       // テーマ
	POIIds          []string  `json:"poi_ids" db:"poi_ids"`                   // 訪問したPOIのID配列
	Tags            []string  `json:"tags" db:"tags"`                         // タグ
	DurationMinutes int       `json:"duration_minutes" db:"duration_minutes"` // 実績時間
	DistanceMeters  int       `json:"distance_meters" db:"distance_meters"`   // 実績距離
	RoutePolyline   string    `json:"route_polyline" db:"route_polyline"`     // ルートの軌跡
	Impressions     string    `json:"impressions" db:"impressions"`           // 感想
	CreatedAt       time.Time `json:"created_at" db:"created_at"`             // 投稿日時
}

type CreateWalkRequest struct {
	Title              string       `json:"title" validate:"required"`
	Description        string       `json:"description" validate:"required"`
	Mode               string       `json:"mode" validate:"required,oneof=destination time_based"`
	Theme              string       `json:"theme" validate:"required"`
	ActualDurationMins int          `json:"actual_duration_minutes" validate:"min=1"`
	ActualDistanceMs   int          `json:"actual_distance_meters" validate:"min=1"`
	RoutePolyline      string       `json:"route_polyline" validate:"required"`
	StartLocation      *Location    `json:"start_location" validate:"required"`
	VisitedPOIs        []VisitedPOI `json:"visited_pois"`
	Impressions        string       `json:"impressions"`
}

type CreateWalkResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	WalkID  string `json:"walk_id"`
}

type VisitedPOI struct {
	Name      string  `json:"name" validate:"required"`
	POIId     string  `json:"poi_id" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" validate:"required,min=-180,max=180"`
}

type WalkSummary struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	AreaName        string    `json:"area_name"`
	Date            string    `json:"date"`
	Summary         string    `json:"summary"`
	DurationMinutes int       `json:"duration_minutes"`
	DistanceMeters  int       `json:"distance_meters"`
	Tags            []string  `json:"tags"`
	EndLocation     *Location `json:"end_location"`
	RoutePolyline   string    `json:"route_polyline"`
}

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

type GetWalksResponse struct {
	Walks []WalkSummary `json:"walks"`
}
