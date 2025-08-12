package model

type GridCell struct {
	ID       int       `json:"id" db:"id"`             // グリッドセルID
	Geometry *Geometry `json:"geometry" db:"geometry"` // メッシュ領域（PostGIS GEOMETRY型）
	Geohash  string    `json:"geohash" db:"geohash"`   // Geohash識別子
}

// GridCellDocument Firestoreのグリッドセルドキュメント
type GridCellDocument struct {
	ID   string      `json:"id"`   // ドキュメントID（例: "osaka_grid_123"）
	POIs []POIObject `json:"pois"` // そのグリッド内のPOI配列
}
