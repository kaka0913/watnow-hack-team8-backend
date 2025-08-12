-- 匿名ユーザーによるPOI作成を一時的に許可
-- POIマイグレーションスクリプト用の設定

-- 既存のポリシーを削除（存在する場合）
DROP POLICY IF EXISTS "Authenticated users can insert pois" ON pois;

-- 匿名ユーザーも含めて全ユーザーがPOIを作成可能
CREATE POLICY "Everyone can insert pois" ON pois
    FOR INSERT WITH CHECK (true);

-- コメント追加
COMMENT ON POLICY "Everyone can insert pois" ON pois IS '匿名ユーザーもINSERT可能（POIマイグレーション用）';
