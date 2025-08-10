-- 匿名ユーザーによる散歩記録作成を一時的に許可
-- テスト環境用の設定

-- 既存のポリシーを削除
DROP POLICY IF EXISTS "Authenticated users can insert walks" ON walks;

-- 匿名ユーザーも含めて全ユーザーが散歩記録を作成可能
CREATE POLICY "Everyone can insert walks" ON walks
    FOR INSERT WITH CHECK (true);

-- コメント追加
COMMENT ON POLICY "Everyone can insert walks" ON walks IS '匿名ユーザーもINSERT可能';
