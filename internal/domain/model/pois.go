package model

// LatLng 緯度経度を表す基本的な型（経路検索などで使用）
type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

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

// GetURL URLが存在する場合は値を、存在しない場合は空文字列を返す
func (p *POI) GetURL() string {
	if p.URL != nil {
		return *p.URL
	}
	return ""
}

// SetURL URLを設定する（nilの場合はnilのまま保持）
func (p *POI) SetURL(url string) {
	if url != "" {
		p.URL = &url
	}
}

// HasURL URLが設定されているかチェック
func (p *POI) HasURL() bool {
	return p.URL != nil && *p.URL != ""
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
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Location *Location `json:"location"` // Firestoreではジオポイント
	Category string    `json:"category"` // カテゴリ（単一文字列）
	Rating   float64   `json:"rating"`
	URL      *string   `json:"url,omitempty"` // URL（NULLABLE）
}
