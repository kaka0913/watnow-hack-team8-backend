-- テスト用grid_cell作成とRLSポリシー設定

-- grid_cellsテーブルに匿名INSERT許可
CREATE POLICY "Allow anonymous insert on grid_cells"
    ON grid_cells
    FOR INSERT
    TO anon
    WITH CHECK (true);

-- テスト用のgrid_cellを作成（河原町エリア）
INSERT INTO grid_cells (id, geometry) VALUES
(100, ST_GeomFromText('POLYGON((135.760 35.000, 135.780 35.000, 135.780 35.010, 135.760 35.010, 135.760 35.000))', 4326))
ON CONFLICT (id) DO NOTHING;

-- 作成したgrid_cellのIDがsequenceに影響しないよう調整
SELECT setval('grid_cells_id_seq', GREATEST(100, (SELECT COALESCE(MAX(id), 0) FROM grid_cells)));
