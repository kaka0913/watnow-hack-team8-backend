-- walksテーブルに効率的な境界ボックス検索用カラムを追加
-- 開始位置、終了位置、ルート境界ボックスを追加して高速検索を実現

-- ==========================================
-- 1. 地理情報カラムを追加
-- ==========================================

-- 開始位置、終了位置、ルート境界ボックスを追加
ALTER TABLE walks 
ADD COLUMN start_location GEOMETRY(POINT, 4326),
ADD COLUMN end_location GEOMETRY(POINT, 4326),
ADD COLUMN route_bounds GEOMETRY(POLYGON, 4326);

-- カラムにコメントを追加
COMMENT ON COLUMN walks.start_location IS '散歩の開始位置（緯度・経度）';
COMMENT ON COLUMN walks.end_location IS '散歩の終了位置（緯度・経度）';
COMMENT ON COLUMN walks.route_bounds IS 'ルート全体の境界ボックス（検索最適化用）';

-- ==========================================
-- 2. インデックスの作成
-- ==========================================

-- 境界ボックス検索用のGiSTインデックスのみ作成（最も効率的）
CREATE INDEX idx_walks_route_bounds_gist ON walks USING GIST (route_bounds);

-- ==========================================
-- 3. polylineから境界ボックスを計算するヘルパー関数
-- ==========================================

-- polylineをデコードして境界ボックスを計算する関数
CREATE OR REPLACE FUNCTION calculate_route_bounds_from_polyline(polyline_str TEXT)
RETURNS GEOMETRY AS $$
DECLARE
    route_line GEOMETRY;
    bbox GEOMETRY;
BEGIN
    -- polylineが空またはNULLの場合はNULLを返す
    IF polyline_str IS NULL OR polyline_str = '' THEN
        RETURN NULL;
    END IF;
    
    -- polylineをLineStringに変換
    route_line := ST_LineFromEncodedPolyline(polyline_str);
    
    -- LineStringの境界ボックスをPolygonとして計算
    bbox := ST_Envelope(route_line);
    
    RETURN bbox;
EXCEPTION
    WHEN OTHERS THEN
        -- polylineのデコードに失敗した場合はNULLを返す
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- 4. 既存のモックデータに地理情報を設定
-- ==========================================

-- 梅田の三大商業施設を巡るショッピングクルーズ
UPDATE walks 
SET 
    start_location = ST_GeomFromText('POINT(135.4959 34.7024)', 4326),  -- JR大阪駅
    end_location = ST_GeomFromText('POINT(135.4984 34.7032)', 4326),    -- 阪急うめだ本店
    route_bounds = ST_GeomFromText('POLYGON((135.493 34.700, 135.503 34.700, 135.503 34.708, 135.493 34.708, 135.493 34.700))', 4326)
WHERE id = 'a7b8c9d0-e1f2-3456-7890-abcdef123456';

-- 若者文化と恋の物語に触れる茶屋町散歩
UPDATE walks 
SET 
    start_location = ST_GeomFromText('POINT(135.4996 34.7039)', 4326),  -- HEP FIVE
    end_location = ST_GeomFromText('POINT(135.5003 34.7003)', 4326),    -- お初天神
    route_bounds = ST_GeomFromText('POLYGON((135.498 34.700, 135.504 34.700, 135.504 34.706, 135.498 34.706, 135.498 34.700))', 4326)
WHERE id = 'b8c9d0e1-f2a3-4567-8901-bcdef2345678';

-- 梅田の摩天楼と芸術を巡る建築散歩
UPDATE walks 
SET 
    start_location = ST_GeomFromText('POINT(135.4903 34.6987)', 4326),  -- 梅田スカイビル
    end_location = ST_GeomFromText('POINT(135.4959 34.7024)', 4326),    -- JR大阪駅
    route_bounds = ST_GeomFromText('POLYGON((135.488 34.695, 135.500 34.695, 135.500 34.705, 135.488 34.705, 135.488 34.695))', 4326)
WHERE id = 'c9d0e1f2-a3b4-5678-9012-cdef34567890';

-- ==========================================
-- 5. 境界ボックス検索関数を更新
-- ==========================================

-- 境界ボックス内のwalksを効率的に取得する関数（高速版）
CREATE OR REPLACE FUNCTION get_walks_in_bbox(
    min_lng DOUBLE PRECISION,
    min_lat DOUBLE PRECISION,
    max_lng DOUBLE PRECISION,
    max_lat DOUBLE PRECISION
)
RETURNS SETOF walks AS $$
BEGIN
    -- 境界ボックスのバリデーション
    IF min_lng >= max_lng OR min_lat >= max_lat THEN
        RAISE EXCEPTION '無効な境界ボックス: min values must be less than max values';
    END IF;
    
    -- 指定した境界ボックスと重複するwalksを取得（超高速）
    RETURN QUERY
    SELECT w.*
    FROM walks w
    WHERE w.route_bounds IS NOT NULL
      AND ST_Intersects(
        w.route_bounds,
        ST_MakeEnvelope(min_lng, min_lat, max_lng, max_lat, 4326)
      )
    ORDER BY w.created_at DESC;
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- 6. 今後のデータ整合性の設定
-- ==========================================

-- 新しい散歩記録作成時に自動でboundsを計算するトリガー関数
CREATE OR REPLACE FUNCTION auto_calculate_walk_bounds()
RETURNS TRIGGER AS $$
BEGIN
    -- route_polylineが設定されている場合、自動でboundsを計算
    IF NEW.route_polyline IS NOT NULL AND NEW.route_polyline != '' THEN
        NEW.route_bounds := calculate_route_bounds_from_polyline(NEW.route_polyline);
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- トリガーを作成（INSERT/UPDATE時に自動実行）
CREATE TRIGGER trigger_auto_calculate_walk_bounds
    BEFORE INSERT OR UPDATE ON walks
    FOR EACH ROW
    EXECUTE FUNCTION auto_calculate_walk_bounds();

-- ==========================================
-- 7. パフォーマンステスト用のサンプルクエリ
-- ==========================================

-- 大阪・梅田エリア全体での検索テスト
-- SELECT title, start_location, end_location 
-- FROM get_walks_in_bbox(135.490, 34.695, 135.505, 34.710);

-- パフォーマンス確認用
-- EXPLAIN ANALYZE SELECT * FROM get_walks_in_bbox(135.490, 34.695, 135.505, 34.710);
