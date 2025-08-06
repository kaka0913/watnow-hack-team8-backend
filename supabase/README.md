# Team8-App Supabase Database

Team8散歩アプリのSupabaseデータベース管理ディレクトリです。

## 📊 デプロイメント状況

✅ **本番環境デプロイ済み** (2025年8月7日)
- プロジェクト: **BeFree**
- リージョン: Northeast Asia (Tokyo)

## 📁 ファイル構成

```
supabase/
├── migrations/                                    # 実行済みマイグレーション
│   ├── 20250807023901_enable_postgis.sql         # PostGIS拡張有効化 ✅
│   ├── 20250807023902_create_walks_table.sql     # 散歩記録テーブル ✅
│   ├── 20250807023903_create_grid_cells_table.sql # グリッドセルテーブル ✅
│   ├── 20250807023904_create_pois_table.sql      # POIテーブル ✅
│   └── 20250807023905_create_indexes_and_policies.sql # インデックス・RLS ✅
├── config.toml                                   # Supabase設定ファイル
└── README.md                                     # このファイル
```

## データベース構造

### テーブル構成

1. **walks** - 散歩記録
   - ユーザーが完了した散歩の物語と詳細データ
   - UUID主キー、タイトル、エリア、物語本文、テーマ等

2. **pois** - 興味のあるスポット
   - Google Place ID等のユニークID
   - PostGIS POINT型での位置情報
   - カテゴリ、評価値

3. **grid_cells** - グリッドセル
   - 地図のメッシュ分割情報
   - PostGIS POLYGON型での領域情報

### 🚀 実装済み機能

- **PostGIS地理空間データ**: 位置情報、距離計算、空間検索
- **JSONB配列**: タグ、カテゴリ、POI IDの効率的な格納・検索
- **空間インデックス**: GiSTインデックスによる高速地理検索（GiST）
- **全文検索**: POI名の高速検索（GINインデックス）
- **RLS (Row Level Security)**: 適切なアクセス権限制御
- **自動更新トリガー**: updated_at カラムの自動更新
- **便利関数**: 周辺POI検索、境界ボックス検索

## 🔧 開発者向けコマンド

### 現在の環境に接続

```bash
# 既存のBeFreeプロジェクトに接続
supabase link --project-ref <your-project-ref>
```

### 新しいマイグレーション作成

```bash
# 新しいマイグレーションファイルを作成
supabase migration new add_new_feature

# 作成されたファイルを編集
# supabase/migrations/[timestamp]_add_new_feature.sql
```

### マイグレーション実行

```bash
# リモートデータベースに適用
supabase db push

# ローカル開発環境での確認（要Docker）
supabase start
supabase db reset
```

## 📋 作成済みデータベースオブジェクト

### テーブル
- ✅ `walks` - 散歩記録（UUID主キー、JSONB配列、トリガー）
- ✅ `pois` - POI情報（PostGIS POINT、JSONB、外部キー）
- ✅ `grid_cells` - グリッドセル（PostGIS POLYGON）

### インデックス
- ✅ 地理空間検索用GiSTインデックス（`location`, `geometry`）
- ✅ JSONB検索用GINインデックス（`tags`, `categories`, `poi_ids`）
- ✅ 一般検索用B-Treeインデックス（`created_at`, `area`, `theme`, `rate`）
- ✅ 全文検索用GINインデックス（`name`）

### 関数
- ✅ `get_nearby_pois(lat, lng, radius)` - 周辺POI検索
- ✅ `get_walks_in_bbox(min_lng, min_lat, max_lng, max_lat)` - 境界ボックス検索
- ✅ `update_updated_at_column()` - 自動更新トリガー関数

### RLSポリシー
- ✅ **walks**: 全ユーザー読み取り可、認証ユーザー作成可
- ✅ **pois**: 全ユーザー読み取り可、管理者のみ管理可
- ✅ **grid_cells**: 全ユーザー読み取り可、管理者のみ管理可

## ⚠️ 重要な注意事項

1. **実行済みマイグレーション**: 全てのスキーマは既にデプロイ済みです
2. **PostGIS設定**: SRID 4326 (WGS84) 座標系を使用
3. **権限管理**: JWT roleフィールドでadmin権限を判定
4. **日本語検索**: English text search configurationを使用（Japanese未対応）
5. **Docker必須**: `supabase db diff` 等のローカルコマンドにはDockerが必要
