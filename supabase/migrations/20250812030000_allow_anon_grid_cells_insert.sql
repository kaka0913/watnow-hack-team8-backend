-- 匿名ユーザーがgrid_cellsテーブルに挿入できるようにポリシーを追加
-- POI移行スクリプト実行のため

-- 匿名ユーザーもgrid_cellsにINSERT可能にする
CREATE POLICY "Allow anon insert grid_cells" ON grid_cells
    FOR INSERT WITH CHECK (true);

-- 匿名ユーザーもgrid_cellsをUPSERT可能にする
CREATE POLICY "Allow anon upsert grid_cells" ON grid_cells
    FOR UPDATE USING (true);

-- コメント追加
COMMENT ON POLICY "Allow anon insert grid_cells" ON grid_cells IS 'POI移行スクリプト用: 匿名ユーザーのgrid_cells挿入を許可';
COMMENT ON POLICY "Allow anon upsert grid_cells" ON grid_cells IS 'POI移行スクリプト用: 匿名ユーザーのgrid_cells更新を許可';
