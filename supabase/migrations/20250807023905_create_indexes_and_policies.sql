-- インデックス作成
-- api.mdの仕様に基づく効率的なクエリのためのインデックス

-- === WALKS テーブルのインデックス ===

-- 作成日時でのソート用
CREATE INDEX IF NOT EXISTS idx_walks_created_at ON walks (created_at DESC);

-- エリアでの検索用
CREATE INDEX IF NOT EXISTS idx_walks_area ON walks (area);

-- テーマでの検索用
CREATE INDEX IF NOT EXISTS idx_walks_theme ON walks (theme);

-- タグでの検索用（JSONB GINインデックス）
CREATE INDEX IF NOT EXISTS idx_walks_tags ON walks USING GIN (tags);

-- POI IDsでの検索用（JSONB GINインデックス）
CREATE INDEX IF NOT EXISTS idx_walks_poi_ids ON walks USING GIN (poi_ids);

-- === GRID_CELLS テーブルのインデックス ===

-- 空間インデックス（GiSTインデックス）- 地理的検索用
CREATE INDEX IF NOT EXISTS idx_grid_cells_geometry ON grid_cells USING GIST (geometry);

-- === POIS テーブルのインデックス ===

-- 外部キー（grid_cell_id）のインデックス
CREATE INDEX IF NOT EXISTS idx_pois_grid_cell_id ON pois (grid_cell_id);

-- 位置情報の空間インデックス（GiSTインデックス）
CREATE INDEX IF NOT EXISTS idx_pois_location ON pois USING GIST (location);

-- カテゴリでの検索用（JSONB GINインデックス）
CREATE INDEX IF NOT EXISTS idx_pois_categories ON pois USING GIN (categories);

-- 評価値でのフィルタリング用
CREATE INDEX IF NOT EXISTS idx_pois_rate ON pois (rate DESC);

-- 名前での検索用（部分一致検索対応）
-- 注意: 現在 'english' テキスト検索設定を使用していますが、日本語POI名の検索には最適ではありません。
-- 可能であれば 'japanese' 設定を利用するか、ハイブリッドアプローチ（両方のインデックス作成）を推奨します。
-- 例: CREATE INDEX ... ON pois USING GIN (to_tsvector('japanese', name));
-- 今後の改善を検討してください。
CREATE INDEX IF NOT EXISTS idx_pois_name ON pois USING GIN (to_tsvector('english', name));

-- === Row Level Security (RLS) ポリシー ===

-- walks テーブルのRLS有効化
ALTER TABLE walks ENABLE ROW LEVEL SECURITY;

-- pois テーブルのRLS有効化
ALTER TABLE pois ENABLE ROW LEVEL SECURITY;

-- grid_cells テーブルのRLS有効化
ALTER TABLE grid_cells ENABLE ROW LEVEL SECURITY;

-- === WALKS テーブルのポリシー ===

-- 全ユーザーが散歩記録を読み取り可能
CREATE POLICY "Everyone can read walks" ON walks
    FOR SELECT USING (true);

-- 認証済みユーザーが散歩記録を作成可能
CREATE POLICY "Authenticated users can insert walks" ON walks
    FOR INSERT WITH CHECK (auth.role() = 'authenticated');

-- 作成者のみが散歩記録を更新可能（将来的にuser_idカラム追加時用）
-- CREATE POLICY "Users can update own walks" ON walks
--     FOR UPDATE USING (auth.uid() = user_id);

-- 作成者のみが散歩記録を削除可能（将来的にuser_idカラム追加時用）
-- CREATE POLICY "Users can delete own walks" ON walks
--     FOR DELETE USING (auth.uid() = user_id);

-- === POIS テーブルのポリシー ===

-- 全ユーザーがPOI情報を読み取り可能
CREATE POLICY "Everyone can read pois" ON pois
    FOR SELECT USING (true);

-- 管理者のみがPOI情報を作成・更新・削除可能
CREATE POLICY "Admin users can manage pois" ON pois
    FOR ALL USING (auth.jwt() ->> 'role' = 'admin');

-- === GRID_CELLS テーブルのポリシー ===

-- 全ユーザーがグリッドセル情報を読み取り可能
CREATE POLICY "Everyone can read grid_cells" ON grid_cells
    FOR SELECT USING (true);

-- 管理者のみがグリッドセル情報を管理可能
CREATE POLICY "Admin users can manage grid_cells" ON grid_cells
    FOR ALL USING (auth.jwt() ->> 'role' = 'admin');

-- === 便利な関数の作成 ===

-- 指定した座標周辺のPOIを検索する関数
CREATE OR REPLACE FUNCTION get_nearby_pois(
    center_lat DOUBLE PRECISION,
    center_lng DOUBLE PRECISION,
    radius_meters INTEGER DEFAULT 1000
)
RETURNS SETOF pois AS $$
BEGIN
    RETURN QUERY
    SELECT p.*
    FROM pois p
    WHERE ST_DWithin(
        p.location::geography,
        ST_MakePoint(center_lng, center_lat)::geography,
        radius_meters
    )
    ORDER BY ST_Distance(
        p.location::geography,
        ST_MakePoint(center_lng, center_lat)::geography
    );
END;
$$ LANGUAGE plpgsql;

-- 指定した境界ボックス内の散歩記録を取得する関数
CREATE OR REPLACE FUNCTION get_walks_in_bbox(
    min_lng DOUBLE PRECISION,
    min_lat DOUBLE PRECISION,
    max_lng DOUBLE PRECISION,
    max_lat DOUBLE PRECISION
)
RETURNS SETOF walks AS $$
BEGIN
    -- 簡易的な実装（将来的にはwalksテーブルに地理情報カラム追加予定）
    RETURN QUERY
    SELECT w.*
    FROM walks w
    ORDER BY w.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- コメント追加
COMMENT ON FUNCTION get_nearby_pois IS '指定した座標周辺のPOIを距離順で検索';
COMMENT ON FUNCTION get_walks_in_bbox IS '指定した境界ボックス内の散歩記録を取得';