#!/usr/bin/env python3
"""
京都POI取得・移行スクリプト（プラン3：無料枠最大化版）
323個のGeohash、約4,845件POI、$173で実行
"""

import requests
import time
import json
import os
import sys
import argparse
from datetime import datetime, timedelta
import pygeohash as pgh
from supabase import create_client
from dotenv import load_dotenv

# 環境変数を読み込み
load_dotenv()

# 設定
GOOGLE_MAPS_API_KEY = os.environ.get('GOOGLE_MAPS_API_KEY')
SUPABASE_URL = os.environ.get('SUPABASE_URL')
SUPABASE_ANON_KEY = os.environ.get('SUPABASE_ANON_KEY')

# Supabase クライアント
supabase = create_client(SUPABASE_URL, SUPABASE_ANON_KEY)

# APIパラメータ
CHUNK_SIZE = 40  # 同時処理するGeohash数
MAX_RETRIES = 3
RETRY_DELAY = 2  # 秒
RADIUS = 1000  # メートル

# POIタイプのマッピング（新API対応、日本語対応）
POI_TYPES = {
    'restaurant': 'レストラン',
    'cafe': 'カフェ',
    'tourist_attraction': '観光名所',
    'museum': '美術館・博物館',
    'park': '公園',
    'shopping_mall': 'ショッピングモール',
    'convenience_store': 'コンビニエンスストア',
    'gas_station': 'ガソリンスタンド',
    'hospital': '病院',
    'pharmacy': '薬局',
    'bank': '銀行',
    'atm': 'ATM',
    'lodging': '宿泊施設',
    'transit_station': '交通機関'
}

def load_plan3_config():
    """プラン3の設定を読み込み"""
    with open('/Users/kaka/dev/Go/Team8-App/script/plan3_execution_config.json', 'r', encoding='utf-8') as f:
        return json.load(f)

def save_progress(data, filename='plan3_progress.json'):
    """進捗を保存"""
    progress_file = f'/Users/kaka/dev/Go/Team8-App/script/{filename}'
    with open(progress_file, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

def load_progress(filename='plan3_progress.json'):
    """進捗を読み込み"""
    progress_file = f'/Users/kaka/dev/Go/Team8-App/script/{filename}'
    if os.path.exists(progress_file):
        with open(progress_file, 'r', encoding='utf-8') as f:
            return json.load(f)
    return None

def search_places_by_type(lat, lon, place_type, radius=RADIUS):
    """Google Places API (New) でPOIを検索"""
    url = "https://places.googleapis.com/v1/places:searchNearby"
    
    headers = {
        'Content-Type': 'application/json',
        'X-Goog-Api-Key': GOOGLE_MAPS_API_KEY,
        'X-Goog-FieldMask': 'places.id,places.displayName,places.primaryType,places.location,places.rating,places.userRatingCount,places.shortFormattedAddress,places.photos'
    }
    
    payload = {
        "includedTypes": [place_type],
        "maxResultCount": 20,
        "locationRestriction": {
            "circle": {
                "center": {
                    "latitude": lat,
                    "longitude": lon
                },
                "radius": radius
            }
        },
        "languageCode": "ja",
        "regionCode": "JP"
    }
    
    for attempt in range(MAX_RETRIES):
        try:
            response = requests.post(url, headers=headers, json=payload, timeout=30)
            response.raise_for_status()
            
            data = response.json()
            places = data.get('places', [])
            
            # 新API形式を旧API形式に変換
            converted_results = []
            for place in places:
                converted_place = {
                    'place_id': place.get('id'),
                    'name': place.get('displayName', {}).get('text', ''),
                    'geometry': {
                        'location': {
                            'lat': place.get('location', {}).get('latitude'),
                            'lng': place.get('location', {}).get('longitude')
                        }
                    },
                    'rating': place.get('rating'),
                    'user_ratings_total': place.get('userRatingCount'),
                    'vicinity': place.get('shortFormattedAddress', ''),
                    'photos': [{'photo_reference': photo.get('name', '')} for photo in place.get('photos', [])]
                }
                converted_results.append(converted_place)
            
            return converted_results
            
        except Exception as e:
            print(f"Request failed (attempt {attempt + 1}): {e}")
            if attempt < MAX_RETRIES - 1:
                time.sleep(RETRY_DELAY * (attempt + 1))
            else:
                return []
    
    return []

def get_pois_for_geohash(geohash):
    """指定されたGeohashエリアのPOIを取得"""
    lat, lon = pgh.decode(geohash)
    poi_dict = {}  # place_id をキーにしてPOIを管理
    
    print(f"  Geohash {geohash} (緯度: {lat:.5f}, 経度: {lon:.5f}) のPOI取得開始")
    
    # Grid Cellを保存してIDを取得
    grid_cell_id = save_grid_cell(geohash)
    if not grid_cell_id:
        print(f"    Grid Cell保存失敗のため、POI取得をスキップ")
        return []
    
    for place_type in POI_TYPES.keys():
        pois = search_places_by_type(lat, lon, place_type)
        for poi in pois:
            place_id = poi.get('place_id')
            if not place_id:
                continue
                
            # POIの位置情報を取得
            poi_lat = poi.get('geometry', {}).get('location', {}).get('lat')
            poi_lng = poi.get('geometry', {}).get('location', {}).get('lng')
            
            if not poi_lat or not poi_lng:
                continue
                
            if place_id in poi_dict:
                # 既存POIにカテゴリを追加
                existing_categories = set(poi_dict[place_id]['categories'])
                new_category = POI_TYPES[place_type]
                poi_dict[place_id]['categories'] = list(existing_categories.union({new_category}))
                print(f"    POI '{poi.get('name', 'Unknown')}' にカテゴリ '{new_category}' を追加")
            else:
                # 新しいPOI
                point_wkt = f"POINT({poi_lng} {poi_lat})"
                poi_data = {
                    'id': place_id,  # Google Place IDを主キーとして使用
                    'name': poi.get('name'),
                    'location': f"SRID=4326;{point_wkt}",  # PostGIS GEOMETRY形式
                    'categories': [POI_TYPES[place_type]],  # JSONB配列形式
                    'grid_cell_id': grid_cell_id,  # 外部キー
                    'rate': poi.get('rating', 0.0)  # デフォルト値0.0
                }
                poi_dict[place_id] = poi_data
        
        # API制限対策
        time.sleep(1)
    
    all_pois = list(poi_dict.values())
    print(f"  Geohash {geohash}: {len(all_pois)}件のユニークPOI取得完了")
    
    # カテゴリ統計を出力
    category_stats = {}
    for poi in all_pois:
        for category in poi['categories']:
            category_stats[category] = category_stats.get(category, 0) + 1
    if category_stats:
        stats_str = ', '.join([f"{cat}:{count}" for cat, count in category_stats.items()])
        print(f"    カテゴリ内訳: {stats_str}")
    
    return all_pois

def save_grid_cell(geohash):
    """Grid Cellをデータベースに保存"""
    try:
        # 既存チェック
        existing = supabase.table('grid_cells').select('id').eq('geohash', geohash).execute()
        if existing.data:
            grid_cell_id = existing.data[0]['id']
            print(f"    Grid Cell {geohash} は既存 (ID: {grid_cell_id})")
            return grid_cell_id
        
        # Geohashから中心座標とポリゴンを取得
        lat, lon = pgh.decode(geohash)
        lat_err, lon_err = pgh.decode_exact(geohash)[1]
        
        # ポリゴンの境界を計算
        min_lat, max_lat = lat - lat_err, lat + lat_err
        min_lon, max_lon = lon - lon_err, lon + lon_err
        
        # PostGIS POLYGON形式のWKT文字列を作成
        polygon_wkt = f"POLYGON(({min_lon} {min_lat}, {max_lon} {min_lat}, {max_lon} {max_lat}, {min_lon} {max_lat}, {min_lon} {min_lat}))"
        
        grid_cell_data = {
            'geometry': f"SRID=4326;{polygon_wkt}",
            'geohash': geohash
        }
        
        result = supabase.table('grid_cells').insert(grid_cell_data).execute()
        
        if result.data:
            grid_cell_id = result.data[0]['id']
            print(f"    Grid Cell {geohash} 保存成功 (ID: {grid_cell_id})")
            return grid_cell_id
        else:
            print(f"    Grid Cell {geohash} 保存失敗")
            return None
            
    except Exception as e:
        print(f"    Grid Cell {geohash} 保存エラー: {e}")
        return False

def save_pois_to_db(pois):
    """POIをデータベースに保存（upsert機能使用、カテゴリマージ対応）"""
    if not pois:
        return 0
    
    # POIをplace_idでグループ化して、複数カテゴリをマージ
    poi_dict = {}
    for poi in pois:
        place_id = poi['id']
        if place_id in poi_dict:
            # 既存POIにカテゴリを追加
            existing_categories = set(poi_dict[place_id]['categories'])
            new_categories = set(poi['categories'])
            poi_dict[place_id]['categories'] = list(existing_categories.union(new_categories))
        else:
            # 新しいPOI
            poi_dict[place_id] = poi.copy()
    
    saved_count = 0
    
    for poi in poi_dict.values():
        try:
            # 既存POIをチェックしてカテゴリをマージ
            existing = supabase.table('pois').select('id, categories').eq('id', poi['id']).execute()
            
            if existing.data:
                # 既存POIのカテゴリと新しいカテゴリをマージ
                existing_categories = set(existing.data[0].get('categories', []))
                new_categories = set(poi['categories'])
                merged_categories = list(existing_categories.union(new_categories))
                poi['categories'] = merged_categories
                
                # upsertで更新
                result = supabase.table('pois').upsert(
                    poi, 
                    on_conflict='id',
                    count='exact'
                ).execute()
                
                if result.data:
                    categories_str = ', '.join(poi['categories'])
                    print(f"    POI '{poi.get('name', 'Unknown')}' 更新成功 (カテゴリ: {categories_str})")
                    saved_count += 1
            else:
                # 新規挿入
                result = supabase.table('pois').insert(poi).execute()
                
                if result.data:
                    categories_str = ', '.join(poi['categories'])
                    print(f"    POI '{poi.get('name', 'Unknown')}' 新規保存成功 (カテゴリ: {categories_str})")
                    saved_count += 1
                    
        except Exception as e:
            print(f"    POI保存エラー ({poi.get('name', 'Unknown')}): {e}")
    
    return saved_count

def process_chunk(chunk_geohashes, chunk_index, total_chunks):
    """Geohashのチャンクを処理"""
    print(f"\n=== チャンク {chunk_index + 1}/{total_chunks} 開始 ===\n")
    print(f"対象Geohash: {len(chunk_geohashes)}個")
    
    chunk_start_time = datetime.now()
    total_pois_saved = 0
    
    for i, geohash in enumerate(chunk_geohashes):
        print(f"\n[{i+1}/{len(chunk_geohashes)}] 処理中: {geohash}")
        
        # POI取得（Grid Cell保存も含む）
        pois = get_pois_for_geohash(geohash)
        
        # POI保存
        saved_count = save_pois_to_db(pois)
        total_pois_saved += saved_count
        
        print(f"  POI保存: {saved_count}件")
        
        # 進捗保存
        progress = {
            'current_chunk': chunk_index,
            'current_geohash_in_chunk': i + 1,
            'total_chunks': total_chunks,
            'chunk_geohashes': len(chunk_geohashes),
            'total_pois_saved_in_chunk': total_pois_saved,
            'last_processed_geohash': geohash,
            'timestamp': datetime.now().isoformat()
        }
        save_progress(progress)
    
    chunk_duration = datetime.now() - chunk_start_time
    print(f"\n=== チャンク {chunk_index + 1} 完了 ===\n")
    print(f"処理時間: {chunk_duration}")
    print(f"POI保存合計: {total_pois_saved}件")
    
    return total_pois_saved

def test_api_connections():
    """API接続テスト"""
    print("=== API接続テスト ===")
    
    # Google Places API (New) テスト
    test_lat, test_lon = 35.0116, 135.7681  # 祇園四条駅付近
    url = "https://places.googleapis.com/v1/places:searchNearby"
    
    headers = {
        'Content-Type': 'application/json',
        'X-Goog-Api-Key': GOOGLE_MAPS_API_KEY,
        'X-Goog-FieldMask': 'places.id,places.displayName,places.rating'
    }
    
    payload = {
        "includedTypes": ["restaurant"],
        "maxResultCount": 3,
        "locationRestriction": {
            "circle": {
                "center": {
                    "latitude": test_lat,
                    "longitude": test_lon
                },
                "radius": 1000
            }
        },
        "languageCode": "ja",
        "regionCode": "JP"
    }
    
    try:
        response = requests.post(url, headers=headers, json=payload, timeout=30)
        response.raise_for_status()
        data = response.json()
        
        places = data.get('places', [])
        if places:
            print(f"✅ Google Places API (New): OK ({len(places)}件のレストランを取得)")
            sample_place = places[0]
            name = sample_place.get('displayName', {}).get('text', '名前なし')
            rating = sample_place.get('rating', 'なし')
            print(f"   サンプル: {name} (評価: {rating})")
        else:
            print(f"⚠️ Google Places API (New): データなし（APIは正常）")
    except Exception as e:
        print(f"❌ Google Places API (New): 接続エラー - {e}")
        return False
    
    # Supabase 接続テスト
    try:
        result = supabase.table('pois').select('id').limit(1).execute()
        print(f"✅ Supabase: OK")
    except Exception as e:
        print(f"❌ Supabase: 接続エラー - {e}")
        return False
    
    print("✅ 全てのAPI接続テストが成功しました！\n")
    return True

def main():
    """メイン処理"""
    # コマンドライン引数の解析
    parser = argparse.ArgumentParser(description='京都POI取得・移行スクリプト（プラン3：無料枠最大化版）')
    parser.add_argument('--dry-run', action='store_true', help='実際の実行はせず、テストのみ行う')
    parser.add_argument('--test-only', action='store_true', help='API接続テストのみ実行')
    args = parser.parse_args()
    
    print("=== 京都POI取得・移行スクリプト（プラン3：無料枠最大化版） ===\n")
    
    # API接続テスト
    if not test_api_connections():
        print("❌ API接続テストに失敗しました。設定を確認してください。")
        print("詳細は ../docs/google-places-api-setup.md を参照")
        return
    
    # テストのみの場合はここで終了
    if args.test_only:
        print("🎯 API接続テストが完了しました。")
        return
    
    # 設定読み込み
    config = load_plan3_config()
    all_geohashes = config['geohash_list']
    chunk_size = config['chunk_size']
    
    print(f"総Geohash数: {len(all_geohashes)}個")
    print(f"予想POI数: {config['estimated_pois']}件")
    print(f"予想料金: ${config['estimated_cost']:.0f}")
    
    # Dry-runモードの場合
    if args.dry_run:
        print(f"\n🧪 DRY-RUN モード（実際の実行はしません）")
        print(f"✅ 設定ファイル読み込み成功")
        print(f"✅ Geohash生成成功: {len(all_geohashes)}個")
        print(f"✅ チャンク分割: {len([all_geohashes[i:i+chunk_size] for i in range(0, len(all_geohashes), chunk_size)])}個")
        print(f"✅ API接続確認済み")
        print(f"\n🎯 実際に実行する場合:")
        print(f"   python script/kyoto_plan3_migration.py")
        return
    
    # 進捗確認
    progress = load_progress()
    start_chunk = 0
    if progress:
        print(f"\n前回の進捗を発見: チャンク{progress['current_chunk'] + 1}まで完了")
        response = input("続きから実行しますか？ (y/n): ")
        if response.lower() == 'y':
            start_chunk = progress['current_chunk'] + 1
    
    # チャンク分割
    chunks = [all_geohashes[i:i+chunk_size] for i in range(0, len(all_geohashes), chunk_size)]
    total_chunks = len(chunks)
    
    print(f"\nチャンク数: {total_chunks}個")
    print(f"開始チャンク: {start_chunk + 1}")
    
    # 実行確認
    if start_chunk == 0:
        response = input("\n新規実行を開始しますか？ (y/n): ")
        if response.lower() != 'y':
            print("実行を中止しました。")
            return
    
    # 実行開始
    overall_start_time = datetime.now()
    total_pois_overall = 0
    
    for chunk_index in range(start_chunk, total_chunks):
        chunk_pois = process_chunk(chunks[chunk_index], chunk_index, total_chunks)
        total_pois_overall += chunk_pois
        
        # 休憩（API制限対策）
        if chunk_index < total_chunks - 1:
            print(f"\n次のチャンクまで30秒休憩...")
            time.sleep(30)
    
    # 完了報告
    overall_duration = datetime.now() - overall_start_time
    print(f"\n\n=== 全処理完了 ===\n")
    print(f"総処理時間: {overall_duration}")
    print(f"総POI保存数: {total_pois_overall}件")
    print(f"総Geohash数: {len(all_geohashes)}個")
    print(f"平均POI/Geohash: {total_pois_overall/len(all_geohashes):.1f}件")
    
    # 最終進捗クリア
    save_progress({
        'status': 'completed',
        'completed_at': datetime.now().isoformat(),
        'total_geohashes': len(all_geohashes),
        'total_pois': total_pois_overall,
        'total_duration': str(overall_duration)
    })

if __name__ == "__main__":
    main()
