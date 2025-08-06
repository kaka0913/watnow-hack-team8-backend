-- グリッドセルテーブルの作成
-- 地図をメッシュ状に分割したグリッドセル情報を管理

CREATE TABLE IF NOT EXISTS grid_cells (
    -- 主キー: グリッドセルID
    id SERIAL PRIMARY KEY,
    
    -- メッシュ領域（PostGIS GEOMETRY型 - POLYGON）
    geometry GEOMETRY(POLYGON, 4326) NOT NULL,
    
    -- 作成日時
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    
    -- 更新日時
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- 更新日時自動更新トリガー
CREATE TRIGGER update_grid_cells_updated_at 
    BEFORE UPDATE ON grid_cells 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- コメント追加
COMMENT ON TABLE grid_cells IS 'グリッドセルテーブル - 地図のメッシュ分割情報';
COMMENT ON COLUMN grid_cells.id IS 'グリッドセルID';
COMMENT ON COLUMN grid_cells.geometry IS 'メッシュ領域（PostGIS POLYGON）';
COMMENT ON COLUMN grid_cells.created_at IS '作成日時';
COMMENT ON COLUMN grid_cells.updated_at IS '更新日時';