-- POI（Point of Interest）テーブルの作成
-- 興味のあるスポット情報を管理

CREATE TABLE IF NOT EXISTS pois (
    -- 主キー: ユニークなスポットID（Google Place ID等）
    id VARCHAR(255) PRIMARY KEY,
    
    -- スポット名
    name VARCHAR(500) NOT NULL,
    
    -- 位置情報（PostGIS GEOMETRY型 - POINT）
    -- SRID 4326 = WGS84座標系（GPS座標）
    location GEOMETRY(POINT, 4326) NOT NULL,
    
    -- カテゴリ（JSONB配列）
    -- 例: ["cafe", "park", "art_gallery"]
    categories JSONB DEFAULT '[]'::jsonb NOT NULL,
    
    -- グリッドセルID（外部キー）
    grid_cell_id INTEGER NOT NULL REFERENCES grid_cells(id) ON DELETE CASCADE,
    
    -- 評価値（1.0〜5.0）
    rate DOUBLE PRECISION DEFAULT 0.0 CHECK (rate >= 0.0 AND rate <= 5.0),
    
    -- 作成日時
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    
    -- 更新日時
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 更新日時自動更新トリガー
CREATE TRIGGER update_pois_updated_at 
    BEFORE UPDATE ON pois 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- コメント追加
COMMENT ON TABLE pois IS 'POI（興味のあるスポット）テーブル';
COMMENT ON COLUMN pois.id IS 'ユニークなスポットID（Google Place ID等）';
COMMENT ON COLUMN pois.name IS 'スポット名';
COMMENT ON COLUMN pois.location IS '位置情報（PostGIS POINT）';
COMMENT ON COLUMN pois.categories IS 'カテゴリ配列（JSON）';
COMMENT ON COLUMN pois.grid_cell_id IS 'グリッドセルID（外部キー）';
COMMENT ON COLUMN pois.rate IS '評価値（1.0〜5.0）';
COMMENT ON COLUMN pois.created_at IS '作成日時';
COMMENT ON COLUMN pois.updated_at IS '更新日時';