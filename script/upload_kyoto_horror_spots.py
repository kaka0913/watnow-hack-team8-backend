#!/usr/bin/env python3
"""
京都府心霊スポットデータのみをSupabaseのPOIとして登録するスクリプト（Geometry対応版）
"""

import csv
import os
import uuid
import json
import sys
from supabase import create_client, Client
from dotenv import load_dotenv
import pygeohash as pgh

# --- 設定 ---
load_dotenv()

SUPABASE_URL = os.environ.get("SUPABASE_URL")
SUPABASE_ANON_KEY = os.environ.get("SUPABASE_ANON_KEY")
CSV_FILE = 'kansai_ghost_spots/京都府_ghost_spots.csv'
GEOHASH_PRECISION = 6

# テストモード（--test-modeオプションでテスト実行）
TEST_MODE = '--test-mode' in sys.argv

# カテゴリ判定のためのキーワード辞書
CATEGORY_KEYWORDS = {
    'establishment': [
        'トンネル', '病院', 'ホテル', '旅館', 'モーテル', '廃墟', '洋館', '学校', '中学校', '高校', '大学',
        '駅', '踏切', '商業施設', '遊園地', 'トイレ', '公衆トイレ', '廃', '跡', 'アスレチック',
        '橋', 'ブリッジ', '落合橋', '赤橋', 'クレインブリッジ', '道', '峠', 'カーブ', '山中越え'
    ],
    'natural_feature': [
        '湖', '池', 'ダム', '山', '森', '川', '滝', '樹木', '杉', 'クスノキ', '海', '深泥池', '血の池',
        '妙見山', '箕面', '保津川'
    ],
    'tourist_attraction': [
        '城跡', '古戦場', '処刑場', '将軍塚', '船岡山'
    ],
    'place_of_worship': [
        '神社', '寺', '墓地', '慰霊碑', '塚', '大明神', '千日墓地', '首塚'
    ],
    'park': [
        '公園', '緑地', '御苑', '円山公園'
    ]
}

def determine_categories(name: str, description: str = '') -> list:
    """スポット名と説明文からカテゴリを判定する"""
    categories = ['horror_spot']
    text = f"{name} {description}".lower()
    
    for category, keywords in CATEGORY_KEYWORDS.items():
        for keyword in keywords:
            if keyword in text:
                if category not in categories:
                    categories.append(category)
                break
    
    return categories

def save_grid_cell(supabase: Client, geohash: str):
    """Geohashに対応するGrid Cellを保存または取得"""
    if TEST_MODE:
        print(f"    🧪 [TEST] Grid Cell {geohash} をテスト作成 (ID: 999)")
        return 999
        
    try:
        # 既存チェック
        existing_res = supabase.table('grid_cells').select('id').eq('geohash', geohash).execute()
        if existing_res.data:
            return existing_res.data[0]['id']

        # 新規作成 - 正しいAPIを使用
        bbox = pgh.get_bounding_box(geohash)
        # BoundingBoxオブジェクトからmin/max値を取得
        min_lat = bbox.min_lat
        min_lon = bbox.min_lon  
        max_lat = bbox.max_lat
        max_lon = bbox.max_lon
        polygon_wkt = f"POLYGON(({min_lon} {min_lat}, {max_lon} {min_lat}, {max_lon} {max_lat}, {min_lon} {max_lat}, {min_lon} {min_lat}))"

        grid_cell_data = {
            'geometry': f"SRID=4326;{polygon_wkt}",
            'geohash': geohash
        }
        insert_res = supabase.table('grid_cells').insert(grid_cell_data).execute()

        if insert_res.data:
            grid_cell_id = insert_res.data[0]['id']
            print(f"    ✅ Grid Cell {geohash} を新規作成しました (ID: {grid_cell_id})")
            return grid_cell_id
        else:
            print(f"    ❌ Grid Cell {geohash} の保存に失敗しました。")
            return None

    except Exception as e:
        print(f"    ❌ Grid Cell {geohash} の処理中にエラーが発生しました: {e}")
        return None

def insert_poi_with_geometry(supabase: Client, poi_data: dict, lat: float, lon: float):
    """Geometry対応のPOI挿入"""
    if TEST_MODE:
        print(f"    🧪 [TEST] POI挿入テスト: {poi_data['name']}")
        print(f"         位置: POINT({lon} {lat})")
        print(f"         カテゴリ: {poi_data['categories']}")
        return {'success': True, 'test_mode': True}
        
    try:
        # PostGIS GEOMETRY形式で直接挿入
        point_wkt = f"POINT({lon} {lat})"
        poi_data['location'] = f"SRID=4326;{point_wkt}"  # PostGIS GEOMETRY形式
        
        result = supabase.table('pois').insert(poi_data).execute()
        return result.data
    except Exception as e:
        print(f"    ❌ POI挿入に失敗: {e}")
        return None

def check_duplicate(supabase: Client, name: str, url: str) -> bool:
    """重複チェック（名前またはURLで判定）"""
    if TEST_MODE:
        # テストモードでは重複なしとして扱う
        return False
        
    try:
        # 名前での重複チェック
        name_result = supabase.table('pois').select('id', count='exact').eq('name', name).execute()
        if name_result.count > 0:
            return True
            
        # URLでの重複チェック（URLが存在する場合）
        if url:
            url_result = supabase.table('pois').select('id', count='exact').eq('url', url).execute()
            if url_result.count > 0:
                return True
                
        return False
    except Exception as e:
        print(f"    ❌ 重複チェックエラー: {e}")
        return False

def main():
    """メイン処理"""
    mode_text = "🧪 [TEST MODE]" if TEST_MODE else ""
    print(f"--- 👻 京都府心霊スポット登録開始 {mode_text} ---")

    # Supabaseクライアント初期化（テストモードでは省略）
    supabase = None
    if not TEST_MODE:
        if not all([SUPABASE_URL, SUPABASE_ANON_KEY]):
            print("❌ エラー: 環境変数 SUPABASE_URL と SUPABASE_ANON_KEY が設定されていません。")
            return
        
        supabase: Client = create_client(SUPABASE_URL, SUPABASE_ANON_KEY)
        print("✅ Supabaseクライアントの初期化完了")
    else:
        print("🧪 テストモード: Supabase接続をスキップします")

    # CSVファイル確認
    if not os.path.exists(CSV_FILE):
        print(f"❌ エラー: ファイル '{CSV_FILE}' が見つかりません。")
        return

    print(f"🔍 ファイル発見: {CSV_FILE}")

    total_added = 0
    total_skipped = 0
    total_errors = 0

    try:
        with open(CSV_FILE, mode='r', encoding='utf-8-sig') as f:
            reader = csv.DictReader(f)
            
            for i, row in enumerate(reader, 1):
                spot_name = row.get('name')
                print(f"\n  [{i}] 処理中: {spot_name}")

                try:
                    lat = float(row['latitude'])
                    lon = float(row['longitude'])
                    url = row.get('url')

                    if not all([spot_name, lat, lon, url]):
                        print("    ⚠️ データ不足のためスキップ")
                        total_errors += 1
                        continue

                    # 重複チェック
                    if check_duplicate(supabase, spot_name, url):
                        print(f"    ⏭️  登録済みのためスキップします。")
                        total_skipped += 1
                        continue
                    
                    # Grid Cell取得
                    if TEST_MODE:
                        geohash = f"test_{lat:.4f}_{lon:.4f}"
                    else:
                        geohash = pgh.encode(lat, lon, precision=GEOHASH_PRECISION)
                    
                    grid_cell_id = save_grid_cell(supabase, geohash)
                    if not grid_cell_id:
                        print(f"    ❌ Grid Cell ID取得失敗のためスキップ。")
                        total_errors += 1
                        continue
                    
                    # カテゴリ判定
                    categories = determine_categories(spot_name, row.get('description', ''))
                    categories_json = json.dumps(categories, ensure_ascii=False)
                    print(f"    📋 判定されたカテゴリ: {categories}")
                    
                    # POIデータ作成
                    poi_data = {
                        'id': str(uuid.uuid4()),
                        'name': spot_name,
                        'categories': categories_json,
                        'grid_cell_id': grid_cell_id,
                        'rate': 0.0,
                        'url': url
                    }

                    # Geometry対応挿入
                    result = insert_poi_with_geometry(supabase, poi_data, lat, lon)
                    
                    if result:
                        print(f"    👍 {spot_name} をPOIとして新規登録しました。")
                        total_added += 1
                    else:
                        print(f"    ❌ {spot_name} の登録に失敗しました。")
                        total_errors += 1
                
                except (ValueError, TypeError) as e:
                    print(f"    ❌ データ形式が無効です: {e}")
                    total_errors += 1
                except Exception as e:
                    print(f"    ❌ 予期せぬエラーが発生: {e}")
                    total_errors += 1
                    
    except Exception as e:
        print(f"❌ ファイル処理中に大きなエラーが発生しました: {e}")
        return

    print("\n--- ✅ 京都府心霊スポット登録完了 ---")
    print(f"✨ 新規追加: {total_added}件")
    print(f"⏩ スキップ (重複): {total_skipped}件")
    print(f"🚫 エラー: {total_errors}件")
    print("------------------------------------------")

if __name__ == "__main__":
    main()
