-- POIテーブルのcategoriesカラム（JSONB配列）をcategoryカラム（VARCHAR単体）に変更

-- 1. 新しいcategoryカラムを追加
ALTER TABLE pois ADD COLUMN category VARCHAR(100);

-- 2. 既存データの移行（categoriesの最初の要素をcategoryに設定）
UPDATE pois 
SET category = CASE 
    WHEN jsonb_array_length(categories) > 0 
    THEN categories->>0 
    ELSE 'other' 
END
WHERE category IS NULL;

-- 3. categoryカラムをNOT NULLに設定
ALTER TABLE pois ALTER COLUMN category SET NOT NULL;

-- 4. categoryカラムにデフォルト値を設定
ALTER TABLE pois ALTER COLUMN category SET DEFAULT 'other';

-- 5. 古いcategoriesカラムを削除
ALTER TABLE pois DROP COLUMN categories;

-- 6. コメントを更新
COMMENT ON COLUMN pois.category IS 'カテゴリ（単一文字列）';

-- 7. インデックスを追加（カテゴリ検索の高速化）
CREATE INDEX IF NOT EXISTS idx_pois_category ON pois(category);

-- 変更履歴のコメント
COMMENT ON TABLE pois IS 'POI（興味のあるスポット）テーブル - categories(JSONB)からcategory(VARCHAR)に変更';
