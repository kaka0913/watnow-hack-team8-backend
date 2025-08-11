-- 匿名ユーザーがpoisテーブルに挿入できるようにポリシーを追加
-- POI移行スクリプト実行のため

-- 匿名ユーザーもpoisにINSERT可能にする
CREATE POLICY "Allow anon insert pois" ON pois
    FOR INSERT WITH CHECK (true);

-- 匿名ユーザーもpoisをUPSERT可能にする  
CREATE POLICY "Allow anon upsert pois" ON pois
    FOR UPDATE USING (true);

-- コメント追加
COMMENT ON POLICY "Allow anon insert pois" ON pois IS 'POI移行スクリプト用: 匿名ユーザーのpois挿入を許可';
COMMENT ON POLICY "Allow anon upsert pois" ON pois IS 'POI移行スクリプト用: 匿名ユーザーのpois更新を許可';
