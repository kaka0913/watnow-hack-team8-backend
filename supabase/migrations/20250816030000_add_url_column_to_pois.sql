-- Migration: Add URL column to pois table
-- Date: 2025-08-16
-- Description: 既存のPOIテーブルにURLカラムを追加するマイグレーション

-- 1. URLカラムを追加（既存データがあるためNULLABLE）
ALTER TABLE pois 
ADD COLUMN url VARCHAR;

-- 2. URLカラムにコメントを追加
COMMENT ON COLUMN pois.url IS 'スポットの詳細情報URL';

-- 3. URLカラムにインデックスを作成（検索性能向上・重複チェックのため）
CREATE INDEX idx_pois_url ON pois(url);

-- 4. URL重複チェック用の一意制約を追加（必要に応じて）
-- CREATE UNIQUE INDEX idx_pois_url_unique ON pois(url) WHERE url IS NOT NULL;

-- 5. 既存データ確認用クエリ
-- SELECT 
--   COUNT(*) as total_records,
--   COUNT(url) as records_with_url,
--   COUNT(*) - COUNT(url) as records_without_url
-- FROM pois;

-- 6. テーブル構造確認用クエリ
-- SELECT column_name, data_type, is_nullable, column_default 
-- FROM information_schema.columns 
-- WHERE table_name = 'pois' 
-- ORDER BY ordinal_position;
