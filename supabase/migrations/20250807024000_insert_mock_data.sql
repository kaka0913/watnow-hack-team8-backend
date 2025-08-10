-- モックデータの挿入
-- テスト・デモ用の最小限のサンプルデータ（すべて大阪・梅田エリア）

-- ==========================================
-- 1. グリッドセルのモックデータ
-- ==========================================

-- 大阪（梅田エリア）のグリッドセル
-- grid_cell_id: 1 (JR大阪駅・グランフロント周辺)
INSERT INTO grid_cells (id, geometry) VALUES
(1, ST_GeomFromText('POLYGON((135.493 34.700, 135.500 34.700, 135.500 34.707, 135.493 34.707, 135.493 34.700))', 4326)),
-- grid_cell_id: 2 (阪急・茶屋町周辺)
(2, ST_GeomFromText('POLYGON((135.498 34.703, 135.503 34.703, 135.503 34.708, 135.498 34.708, 135.498 34.703))', 4326)),
-- grid_cell_id: 3 (西梅田・お初天神周辺)
(3, ST_GeomFromText('POLYGON((135.490 34.697, 135.501 34.697, 135.501 34.701, 135.490 34.701, 135.490 34.697))', 4326));

-- ==========================================
-- 2. POI（Point of Interest）のモックデータ
-- ==========================================

-- JR大阪駅・グランフロント周辺のPOI (grid_cell_id: 1)
INSERT INTO pois (id, name, location, categories, grid_cell_id, rate) VALUES
('ChIJi8qkAQsOAGARAVf', 'グランフロント大阪', ST_GeomFromText('POINT(135.4969 34.7058)', 4326), '["shopping", "restaurant", "architecture"]', 1, 4.3),
('ChIJP5qoJwgOAGARVD0', 'JR大阪駅', ST_GeomFromText('POINT(135.4959 34.7024)', 4326), '["transportation", "landmark", "architecture"]', 1, 4.2),
('ChIJ8esLKQkOAGARfM9', 'ルクア大阪', ST_GeomFromText('POINT(135.4954 34.7040)', 4326), '["shopping", "food", "fashion"]', 1, 4.2);

-- 阪急・茶屋町周辺のPOI (grid_cell_id: 2)
INSERT INTO pois (id, name, location, categories, grid_cell_id, rate) VALUES
('ChIJ_eM8LwgOAGAR3sK', '阪急うめだ本店', ST_GeomFromText('POINT(135.4984 34.7032)', 4326), '["department_store", "shopping", "food"]', 2, 4.3),
('ChIJ4-gSKAgOAGARy0z', 'HEP FIVE', ST_GeomFromText('POINT(135.4996 34.7039)', 4326), '["shopping", "entertainment", "ferris_wheel"]', 2, 4.0),
('ChIJl-gSKAgOAGARe5B', 'NU茶屋町', ST_GeomFromText('POINT(135.4993 34.7051)', 4326), '["shopping", "fashion", "cafe"]', 2, 4.1);

-- 西梅田・お初天神周辺のPOI (grid_cell_id: 3)
INSERT INTO pois (id, name, location, categories, grid_cell_id, rate) VALUES
('ChIJN5-eLwgOAGARcAI', '梅田スカイビル・空中庭園展望台', ST_GeomFromText('POINT(135.4903 34.6987)', 4326), '["landmark", "observation_deck", "architecture"]', 3, 4.4),
('ChIJYdo7sAkOAGAR7lI', 'ハービスPLAZA ENT', ST_GeomFromText('POINT(135.4947 34.6984)', 4326), '["shopping", "theater", "luxury"]', 3, 4.0),
('ChIJ2eMzMAkOAGARcpC', 'お初天神（露天神社）', ST_GeomFromText('POINT(135.5003 34.7003)', 4326), '["shrine", "history", "culture"]', 3, 4.2);


-- ==========================================
-- 3. 散歩記録のモックデータ
-- ==========================================

-- 梅田散歩記録 1: ショッピングと最新グルメ
INSERT INTO walks (
    id, title, area, description, theme, poi_ids, tags,
    duration_minutes, distance_meters, route_polyline, impressions
) VALUES
(
    'a7b8c9d0-e1f2-3456-7890-abcdef123456',
    '梅田の三大商業施設を巡るショッピングクルーズ',
    '大阪・梅田エリア',
    'JR大阪駅直結のルクア、グランフロント、そして阪急うめだ本店を巡り、最新のファッション、コスメ、グルメを一日で満喫する散歩です。大阪の「今」を感じることができます。',
    'shopping',
    '["ChIJ8esLKQkOAGARfM9", "ChIJi8qkAQsOAGARAVf", "ChIJ_eM8LwgOAGAR3sK"]',
    '["ショッピング", "グルメ", "デパ地下", "最新スポット"]',
    90,
    2500,
    'a~aaGznvoW`@qBj@wBdB_G`BaF~@qCdA_ChBsEh@uA`@cAj@qB',
    '一日中いても飽きないくらいお店が充実していました。特にデパ地下のスイーツ巡りは最高！ウィンドウショッピングだけでも楽しいです。'
);

-- 梅田散歩記録 2: エンタメとカルチャー
INSERT INTO walks (
    id, title, area, description, theme, poi_ids, tags,
    duration_minutes, distance_meters, route_polyline, impressions
) VALUES
(
    'b8c9d0e1-f2a3-4567-8901-bcdef2345678',
    '若者文化と恋の物語に触れる茶屋町散歩',
    '大阪・梅田エリア',
    '赤い観覧車が目印のHEP FIVEから始まり、おしゃれなショップが並ぶNU茶屋町を散策。最後は恋人の聖地としても知られるお初天神へ。梅田のポップカルチャーと歴史の両面を楽しめます。',
    'culture',
    '["ChIJ4-gSKAgOAGARy0z", "ChIJl-gSKAgOAGARe5B", "ChIJ2eMzMAkOAGARcpC"]',
    '["エンタメ", "若者文化", "神社", "恋愛成就", "カフェ巡り"]',
    75,
    2100,
    's}caGnoooW`AyApCgFxBqDn@iAn@gA`@c@x@q@x@u@r@Wf@Mr@IdACl@D',
    'HEP FIVEの観覧車からの夜景がロマンチックでした。お初天神は都会の真ん中にあるのにとても静かで、心が落ち着く不思議な空間でした。'
);

-- 梅田散歩記録 3: 建築美と絶景
INSERT INTO walks (
    id, title, area, description, theme, poi_ids, tags,
    duration_minutes, distance_meters, route_polyline, impressions
) VALUES
(
    'c9d0e1f2-a3b4-5678-9012-cdef34567890',
    '梅田の摩天楼と芸術を巡る建築散歩',
    '大阪・梅田エリア',
    '世界的に有名な梅田スカイビルからスタートし、そのユニークな建築美を堪能。その後、西梅田のラグジュアリーな空間ハービスENTを通り、JR大阪駅の巨大な屋根にも注目。近代建築の迫力を感じるルートです。',
    'architecture',
    '["ChIJN5-eLwgOAGARcAI", "ChIJYdo7sAkOAGAR7lI", "ChIJP5qoJwgOAGARVD0"]',
    '["建築", "絶景", "展望台", "アート", "都市景観"]',
    60,
    1900,
    'o}baGnhuoW`@c@vAiB`@y@v@uBt@uCl@aD`@kCXoBf@sF',
    '空中庭園からの360度のパノラマビューは圧巻の一言です。夕暮れから夜景に変わる瞬間は本当に感動的でした。建築好きにはたまらない散歩でした。'
);


-- ==========================================
-- データ確認用コメント
-- ==========================================

-- 以下のクエリでデータを確認できます：
-- SELECT COUNT(*) FROM grid_cells; -- 結果: 3件
-- SELECT COUNT(*) FROM pois; -- 結果: 9件
-- SELECT COUNT(*) FROM walks; -- 結果: 3件

-- POIと所属グリッドセルの確認：
-- SELECT p.name, p.categories, gc.id as grid_id
-- FROM pois p
-- JOIN grid_cells gc ON p.grid_cell_id = gc.id
-- ORDER BY gc.id, p.name;

-- 散歩記録の基本情報確認：
-- SELECT title, area, theme, duration_minutes, array_length(poi_ids, 1) as poi_count
-- FROM walks
-- ORDER BY created_at;