package model

// POI Point of Interest（興味のあるスポット）を表すモデル
type POI struct {
	ID         string    `json:"id" db:"id"`                     // ユニークなスポットID
	Name       string    `json:"name" db:"name"`                 // スポット名
	Location   *Geometry `json:"location" db:"location"`         // 位置情報（PostGIS GEOMETRY型）
	Categories []string  `json:"categories" db:"categories"`     // カテゴリ
	GridCellID int       `json:"grid_cell_id" db:"grid_cell_id"` // グリッドセルID
	Rate       float64   `json:"rate" db:"rate"`                 // 評価値
}

// Geometry PostGIS GEOMETRY型に対応する構造体
type Geometry struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"` // [longitude, latitude]
}

type Location struct {
	Latitude  float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" validate:"required,min=-180,max=180"`
}

// ToGeometry Location を PostGIS GEOMETRY 型に変換
func (l *Location) ToGeometry() *Geometry {
	return &Geometry{
		Type:        "Point",
		Coordinates: []float64{l.Longitude, l.Latitude},
	}
}

// FromGeometry PostGIS GEOMETRY 型から Location に変換
func (l *Location) FromGeometry(g *Geometry) {
	if g != nil && len(g.Coordinates) >= 2 {
		l.Longitude = g.Coordinates[0]
		l.Latitude = g.Coordinates[1]
	}
}

// POIObject Firestoreのグリッドセル内のPOI情報
type POIObject struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Location   *Location `json:"location"` // Firestoreではジオポイント
	Categories []string  `json:"categories"`
	Rating     float64   `json:"rating"`
}
