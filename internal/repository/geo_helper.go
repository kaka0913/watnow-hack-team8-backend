package repository

import (
	"github.com/paulmach/orb"

	"Team8-App/internal/domain/model"
)

// GeoPoint PostGIS POINT 型の JSON 表現
type GeoPoint struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// LocationToGeoPoint model.Location を PostGIS POINT 形式に変換
func LocationToGeoPoint(location *model.Location) *GeoPoint {
	if location == nil {
		return nil
	}

	// orb.Point を作成
	point := orb.Point{location.Longitude, location.Latitude}

	return &GeoPoint{
		Type:        "Point",
		Coordinates: []float64{point.Lon(), point.Lat()},
	}
}

// GeoPointToLocation PostGIS POINT を model.Location に変換
func GeoPointToLocation(geoPoint *GeoPoint) *model.Location {
	if geoPoint == nil || len(geoPoint.Coordinates) < 2 {
		return nil
	}

	// orb.Point として解析
	point := orb.Point{geoPoint.Coordinates[0], geoPoint.Coordinates[1]}

	return &model.Location{
		Latitude:  point.Lat(),
		Longitude: point.Lon(),
	}
}

// CreateBoundingBoxPolygon 開始・終了位置からシンプルな境界ボックスを作成
func CreateBoundingBoxPolygon(startLoc, endLoc *model.Location) *model.GeoPolygon {
	if startLoc == nil || endLoc == nil {
		return nil
	}

	// orb.Point として作成
	start := orb.Point{startLoc.Longitude, startLoc.Latitude}
	end := orb.Point{endLoc.Longitude, endLoc.Latitude}

	// orb.Bound を使用して境界ボックスを作成
	bound := orb.Bound{
		Min: orb.Point{
			start.Lon(),
			start.Lat(),
		},
		Max: orb.Point{
			end.Lon(),
			end.Lat(),
		},
	}

	// 2つの点から正しい境界ボックスを拡張
	bound = bound.Extend(start).Extend(end)

	// 少し余裕を持たせる（約100m程度）
	padding := 0.001 // 約111m
	bound = bound.Pad(padding)

	// 手動でPolygon座標配列を作成
	minLng := bound.Min.Lon()
	minLat := bound.Min.Lat()
	maxLng := bound.Max.Lon()
	maxLat := bound.Max.Lat()

	coordinates := [][][]float64{
		{
			{minLng, minLat}, // 左下
			{maxLng, minLat}, // 右下
			{maxLng, maxLat}, // 右上
			{minLng, maxLat}, // 左上
			{minLng, minLat}, // 閉じる
		},
	}

	return &model.GeoPolygon{
		Type:        "Polygon",
		Coordinates: coordinates,
	}
}

// PrepareWalkForDB Walk を DB 保存用の構造体に変換
type WalkDB struct {
	ID              string            `json:"id"`
	Title           string            `json:"title"`
	Area            string            `json:"area"`
	Description     string            `json:"description"`
	Theme           string            `json:"theme"`
	POIIds          []string          `json:"poi_ids"`
	Tags            []string          `json:"tags"`
	DurationMinutes int               `json:"duration_minutes"`
	DistanceMeters  int               `json:"distance_meters"`
	RoutePolyline   string            `json:"route_polyline"`
	Impressions     string            `json:"impressions"`
	StartLocation   *GeoPoint         `json:"start_location"`
	EndLocation     *GeoPoint         `json:"end_location"`
	RouteBounds     *model.GeoPolygon `json:"route_bounds"`
}

// WalkToWalkDB model.Walk を DB 保存用に変換
func WalkToWalkDB(walk *model.Walk) *WalkDB {
	startGeo := LocationToGeoPoint(walk.StartLocation)
	endGeo := LocationToGeoPoint(walk.EndLocation)

	// 境界ボックスを作成
	// 現在は開始・終了位置から計算（将来的にはポリライン全体から計算）
	routeBounds := CreateBoundingBoxPolygon(walk.StartLocation, walk.EndLocation)

	return &WalkDB{
		ID:              walk.ID,
		Title:           walk.Title,
		Area:            walk.Area,
		Description:     walk.Description,
		Theme:           walk.Theme,
		POIIds:          walk.POIIds,
		Tags:            walk.Tags,
		DurationMinutes: walk.DurationMinutes,
		DistanceMeters:  walk.DistanceMeters,
		RoutePolyline:   walk.RoutePolyline,
		Impressions:     walk.Impressions,
		StartLocation:   startGeo,
		EndLocation:     endGeo,
		RouteBounds:     routeBounds,
	}
}
