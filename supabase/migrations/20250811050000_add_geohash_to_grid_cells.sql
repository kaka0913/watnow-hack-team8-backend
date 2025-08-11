-- grid_cellsテーブルにgeohashカラムを追加

-- 1. geohashカラムを追加
ALTER TABLE grid_cells ADD COLUMN geohash VARCHAR(12);

-- 2. geohashカラムにインデックスを追加（検索の高速化）
CREATE INDEX IF NOT EXISTS idx_grid_cells_geohash ON grid_cells(geohash);

-- 3. geohashカラムを一意制約に設定
ALTER TABLE grid_cells ADD CONSTRAINT unique_grid_cells_geohash UNIQUE (geohash);

-- 4. コメントを追加
COMMENT ON COLUMN grid_cells.geohash IS 'グリッドセルのGeohash識別子';

-- 5. テーブルコメントを更新
COMMENT ON TABLE grid_cells IS 'グリッドセルテーブル - 地図のメッシュ分割情報（geohashカラム追加済み）';
