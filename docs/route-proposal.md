# ルート提案システム要件書

## 🎯 システム概要

「スタート地点から 3 つのスポットを巡る 4 箇所の最適化された散歩ルート」を提案するシステムです。

本システムは、ユーザーの現在地から出発し、テーマに応じた魅力的なスポットを 3 箇所経由して、最適化されたルートを提案します。目的地の有無に関わらず、柔軟で楽しい散歩体験を提案することを目的としています。

## 🎨 テーマ・シナリオ対応

### 🍽️ グルメテーマ (`gourmet`)

#### シナリオ一覧

| シナリオ ID     | 日本語名       | 対象 POI カテゴリ                     | 説明                                         |
| --------------- | -------------- | ------------------------------------- | -------------------------------------------- |
| `cafe_hopping`  | カフェ巡り     | `cafe`, `restaurant`                  | 評価の高いカフェやレストランを中心とした巡回 |
| `bakery_tour`   | ベーカリー巡り | `bakery`, `cafe`, `store`             | ベーカリーを優先的に選択した甘い香りの散歩   |
| `local_gourmet` | 地元グルメ     | `restaurant`, `food`, `meal_takeaway` | 地域の食文化を味わうレストラン・食事処巡り   |
| `sweet_journey` | スイーツ巡り   | `bakery`, `cafe`, `store`             | ベーカリーとカフェを組み合わせたスイーツ体験 |

### 🌿 自然テーマ (`nature`)

#### シナリオ一覧

| シナリオ ID     | 日本語名   | 対象 POI カテゴリ                                | 説明                                             |
| --------------- | ---------- | ------------------------------------------------ | ------------------------------------------------ |
| `park_tour`     | 公園巡り   | `park`, `tourist_attraction`, `establishment`    | 公園を中心とした自然散策ルート                   |
| `riverside`     | 河川敷散歩 | `park`, `natural_feature`, `point_of_interest`   | 河川敷や水辺エリアでの一直線散歩                 |
| `temple_nature` | 寺社と自然 | `place_of_worship`, `park`, `tourist_attraction` | 寺社仏閣と自然スポットを組み合わせた心安らぐ散歩 |

### 🏛️ 歴史テーマ (`history`)

#### シナリオ一覧

| シナリオ ID     | 日本語名       | 対象 POI カテゴリ                                           | 説明                                         |
| --------------- | -------------- | ----------------------------------------------------------- | -------------------------------------------- |
| `temple_shrine` | 寺社仏閣巡り   | `place_of_worship`, `tourist_attraction`                    | 神社仏閣を中心とした歴史散策                 |
| `museum_tour`   | 博物館巡り     | `museum`, `art_gallery`, `tourist_attraction`               | 博物館・美術館を中心とした文化的散歩         |
| `old_town`      | 古い街並み散策 | `tourist_attraction`, `establishment`, `book_store`         | 歴史ある街並みや書店などを巡る文化散歩       |
| `cultural_walk` | 文化的散歩     | `tourist_attraction`, `museum`, `book_store`, `art_gallery` | 文化的スポットをバランス良く組み合わせた散歩 |

## 📊 POI カテゴリ詳細

### 現在対応中の POI カテゴリ

| カテゴリ             | 説明               | 使用テーマ |
| -------------------- | ------------------ | ---------- |
| `cafe`               | カフェ・喫茶店     | グルメ     |
| `restaurant`         | レストラン・食堂   | グルメ     |
| `bakery`             | ベーカリー・パン屋 | グルメ     |
| `food`               | 食品関連施設       | グルメ     |
| `meal_takeaway`      | テイクアウト専門店 | グルメ     |
| `store`              | 商店・販売店       | グルメ     |
| `park`               | 公園・緑地         | 自然       |
| `tourist_attraction` | 観光地・名所       | 自然、歴史 |
| `establishment`      | 施設・建物         | 自然、歴史 |
| `natural_feature`    | 自然の地形・特徴   | 自然       |
| `point_of_interest`  | 興味深いスポット   | 自然       |
| `place_of_worship`   | 宗教施設・寺社     | 自然、歴史 |
| `museum`             | 博物館             | 歴史       |
| `art_gallery`        | 美術館・ギャラリー | 歴史       |
| `book_store`         | 書店               | 歴史       |

## 🚀 将来的なテーマ拡張案

### 未実装テーマ案

元々の構想から、以下のテーマが今後追加可能です：

| テーマ案   | 説明                         | 想定 POI カテゴリ                                    |
| ---------- | ---------------------------- | ---------------------------------------------------- |
| `art`      | アートギャラリー             | `art_gallery`, `museum`, `tourist_attraction`        |
| `shopping` | ショッピング雑貨             | `store`, `shopping_mall`, `clothing_store`           |
| `date`     | デートコース                 | `restaurant`, `cafe`, `park`, `tourist_attraction`   |
| `health`   | 健康意識・長距離ウォーキング | `park`, `gym`, `health`, `natural_feature`           |
| `horror`  | おばけ・肝試しコース         | `place_of_worship`, `cemetery`, `tourist_attraction` |

### シナリオ拡張例

- **アートテーマ**: `gallery_hopping` (ギャラリー巡り), `street_art` (ストリートアート探索)
- **ショッピングテーマ**: `vintage_shopping` (古着・雑貨巡り), `local_market` (地元市場探索)
- **デートテーマ**: `romantic_walk` (ロマンチック散歩), `confession_course` (告白大作戦 ❤️)
- **健康テーマ**: `long_distance` (マゼランコース), `morning_jog` (朝のジョギング)
- **ミステリーテーマ**: `ghost_tour` (心霊スポット巡り), `temple_mystery` (神秘的寺社巡り)

## 🔧 技術実装状況

### 定数管理

- テーマ・シナリオは `internal/domain/model/constants.go` で一元管理
- 日本語名のマッピングも含めて型安全に管理

### ストラテジーパターン

- 各テーマごとに専用のストラテジークラスで実装
- シナリオごとに異なる POI 選択ロジックを提供
- 目的地の有無に応じた柔軟なルート生成
