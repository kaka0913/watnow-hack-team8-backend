# POI取得スクリプト集

このフォルダには、京都市内のPOI（Point of Interest）をGoogle Places API (New) から取得してSupabaseに保存するためのスクリプトが含まれています。

## 📁 ファイル構成

### 🎯 メインスクリプト
- **`kyoto_plan3_migration.py`** - 京都市内全域のPOI取得メインスクリプト（Google Places API New対応、テスト機能内蔵）

### ⚙️ 設定ファイル
- **`plan3_execution_config.json`** - プラン3の実行設定（323個Geohash、約4,845件POI予定）

## 🚀 クイックスタート

### 1. 前提条件
- Python 3.8以上
- Google Places API (New) キー
- Supabase プロジェクト

### 2. 環境設定
```bash
# 必要なライブラリをインストール
pip install python-dotenv supabase requests pygeohash

# .envファイルを作成して以下を設定
SUPABASE_URL=your_supabase_url
SUPABASE_ANON_KEY=your_supabase_anon_key
GOOGLE_PLACES_API_KEY=your_google_places_api_key
```

### 3. テスト実行（オプション）
```bash
# メインスクリプトでドライラン（実際には実行しない）
python kyoto_plan3_migration.py --dry-run
```

### 4. メイン実行
```bash
python kyoto_plan3_migration.py
```

## 📊 プラン3: 無料枠最大化

### 対象エリア
- **範囲**: 京都市内全域（緯度34.955〜35.055、経度135.645〜135.825）
- **総Geohash数**: 323個
- **予想POI数**: 約4,845件
- **予想料金**: $173（Google Cloud無料枠$200内）
- **実行時間**: 約4.5時間（9チャンク × 30分）

### カバーエリア詳細
- 京都駅〜五条駅周辺
- 四条烏丸〜四条河原町
- 祇園〜東山（清水寺〜銀閣寺）
- 嵐山・嵯峨野
- 金閣寺・北野天満宮周辺
- 下鴨神社・出町柳周辺
- 京都御所・二条城周辺
- 伏見稲荷・宇治周辺

### POIカテゴリ（14種類）
- レストラン、カフェ、観光名所、美術館・博物館
- 公園、ショッピングモール、コンビニエンスストア
- ガソリンスタンド、病院、薬局、銀行、ATM
- 宿泊施設、交通機関

## 🔧 技術仕様

### API
- **Google Places API (New)**: `https://places.googleapis.com/v1/places:searchNearby`
- **HTTP Method**: POST
- **認証**: X-Goog-Api-Key ヘッダー

### データベース
- **Supabase**: PostgreSQL + PostGIS
- **テーブル**: `pois`, `grid_cells`
- **認証**: Row Level Security (RLS) 対応

### エリア分割
- **Geohash精度**: 6（約1.2km × 0.6km）
- **Step値**: 0.0035（高密度カバー）
- **チャンクサイズ**: 40個/チャンク（約30分/チャンク）

## 📈 進捗管理

### 自動保存機能
- **進捗ファイル**: `plan3_progress.json`
- **リトライ機能**: 最大3回、指数バックオフ
- **チャンク単位**: 40個ずつ処理、30秒間隔

### 実行再開
前回の途中から実行を再開できます：
```bash
# 進捗ファイルが存在する場合、続きから実行するか選択可能
python kyoto_plan3_migration.py
```

## 🛠️ トラブルシューティング

### API接続テストの実行
```bash
# API接続のみテスト
python kyoto_plan3_migration.py --test-only

# 設定確認（実際の実行はしない）
python kyoto_plan3_migration.py --dry-run
```

### Google Places API エラー
1. Google Cloud ConsoleでPlaces API (New)が有効か確認
2. APIキーの制限設定を確認
3. 請求先アカウントが設定されているか確認

詳細は `../docs/google-places-api-setup.md` を参照

### Supabase接続エラー
1. `.env`ファイルの設定を確認
2. RLSポリシーでanonymous挿入が許可されているか確認
3. ネットワーク接続を確認

## 📝 実行ログ例

```
=== 京都POI取得・移行スクリプト（プラン3：無料枠最大化版） ===

総Geohash数: 323個
予想POI数: 4845件
予想料金: $173

=== チャンク 1/9 開始 ===
対象Geohash: 40個

[1/40] 処理中: xn0qye
  Geohash xn0qye (緯度: 34.95575, 経度: 135.64270) のPOI取得開始
    Grid Cell xn0qye 保存成功
  Geohash xn0qye: 15件のPOI取得完了
  POI保存: 15件
```

## 🏗️ 開発履歴

### v3.1 (2025/08/12)
- テストスクリプト統合（--test-only, --dry-run オプション追加）
- スクリプトフォルダ整理（不要ファイル削除）
- ドキュメント最新化

### v3.0 (2025/08/12)
- Google Places API (New) 対応
- 京都市内全域カバー（323 Geohash）
- 無料枠最大化設計
- 進捗管理・リトライ機能強化

### v2.0 (2025/08/11)
- KABU_1〜8エリア分割実行
- 新旧API切り替え対応
- RLSポリシー修正

### v1.0 (2025/08/10)
- 初期版（河原町カフェテスト）
- 基本POI取得機能実装
