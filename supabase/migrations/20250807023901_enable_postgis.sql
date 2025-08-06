-- PostGIS拡張を有効化
-- Supabaseでは地理空間データを扱うためにPostGIS拡張が必要

-- PostGIS拡張を有効化
CREATE EXTENSION IF NOT EXISTS postgis;

-- PostGIS拡張の追加機能を有効化（距離計算、ルーティング等で使用）
CREATE EXTENSION IF NOT EXISTS postgis_topology;

-- PostGIS Tigerジオコーダー（住所から座標変換用、必要に応じて）
-- CREATE EXTENSION IF NOT EXISTS postgis_tiger_geocoder;

-- PostGIS SFCGALサポート（3D計算用、必要に応じて）
-- CREATE EXTENSION IF NOT EXISTS postgis_sfcgal;

-- UUIDサポート
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 拡張機能の確認
SELECT name, default_version, installed_version 
FROM pg_available_extensions 
WHERE name IN ('postgis', 'postgis_topology', 'uuid-ossp');