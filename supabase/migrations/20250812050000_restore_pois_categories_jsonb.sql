-- POIテーブルのcategoryカラム（VARCHAR単体）をcategoriesカラム（JSONB配列）に戻す
-- 複数カテゴリ対応のため

-- 1. 新しいcategoriesカラムを追加（JSONB配列）
ALTER TABLE pois ADD COLUMN categories JSONB DEFAULT '[]'::jsonb;

-- 2. 既存データの移行（categoryの値をcategoriesの配列に変換）
UPDATE pois 
SET categories = jsonb_build_array(category)
WHERE categories = '[]'::jsonb AND category IS NOT NULL;

-- 3. categoriesカラムをNOT NULLに設定
ALTER TABLE pois ALTER COLUMN categories SET NOT NULL;

-- 4. 古いcategoryカラムとそのインデックスを削除
DROP INDEX IF EXISTS idx_pois_category;
ALTER TABLE pois DROP COLUMN category;

-- 5. categoriesカラムにインデックスを追加（GINインデックスでJSONB検索を高速化）
CREATE INDEX IF NOT EXISTS idx_pois_categories ON pois USING GIN (categories);

-- 6. コメントを更新
COMMENT ON COLUMN pois.categories IS 'カテゴリ配列（JSONB） - 複数カテゴリ対応';
COMMENT ON TABLE pois IS 'POI（興味のあるスポット）テーブル - 複数カテゴリ対応（categories JSONB配列）';
