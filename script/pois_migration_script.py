import os
import time
import math
import json
import requests
import pygeohash as pgh
from dotenv import load_dotenv
from supabase import create_client, Client
from shapely.geometry import box, Point

# .envファイルから環境変数を読み込む
load_dotenv()

# --- 設定項目 ---
SUPABASE_URL = os.environ.get("SUPABASE_URL")
SUPABASE_KEY = os.environ.get("SUPABASE_ANON_KEY")
GOOGLE_MAPS_API_KEY = os.environ.get("GOOGLE_MAPS_API_KEY")

GEOHASH_PRECISION = 6 # グリッドの精度
MIN_RATING = 3.5
MAX_RETRIES = 3  # API失敗時の最大リトライ回数
CHUNK_SIZE = 10  # 一度に処理するGeohashの数
PROGRESS_FILE = 'pois_migration_progress.json'  # 進捗保存ファイル

# 散歩が楽しくなるスポットのカテゴリ
POI_TYPES = [
    'cafe', 'park', 'tourist_attraction', 'art_gallery', 'book_store',
    'bakery', 'store', 'home_goods_store', 'museum',
    'shrine', 'temple', 'florist', 'library', 'riverside_park'
]

# 河川敷判定用キーワード
RIVERSIDE_KEYWORDS = [
    '河川敷', '川原', '河原', '河川', '堤防', '川辺', '水辺', 
    'リバーサイド', 'riverside', '川沿い', '河岸'
]

# --- エリア定義 (4人分担用) ---

# 【KABUさんの担当エリア】京都中心部＋嵐山 - 細分化版 (各エリア約30分処理)

# KABU_1: 四条烏丸エリア中心部
BOUNDING_BOXES_KABU_1 = [
    {'name': 'A1-1: Shijo_Karasuma_Core', 'min_lat': 35.002, 'max_lat': 35.005, 'min_lon': 135.758, 'max_lon': 135.765},
    {'name': 'A1-2: Karasuma_Gojo', 'min_lat': 35.000, 'max_lat': 35.003, 'min_lon': 135.760, 'max_lon': 135.768},
]

# KABU_2: 河原町エリア
BOUNDING_BOXES_KABU_2 = [
    {'name': 'A2-1: Kawaramachi_Shijo', 'min_lat': 35.003, 'max_lat': 35.006, 'min_lon': 135.765, 'max_lon': 135.772},
    {'name': 'A2-2: Kawaramachi_Sanjo', 'min_lat': 35.005, 'max_lat': 35.008, 'min_lon': 135.767, 'max_lon': 135.775},
]

# KABU_3: 祇園エリア西部
BOUNDING_BOXES_KABU_3 = [
    {'name': 'A3-1: Gion_West', 'min_lat': 34.995, 'max_lat': 35.000, 'min_lon': 135.774, 'max_lon': 135.781},
    {'name': 'A3-2: Gion_Shirakawa', 'min_lat': 35.000, 'max_lat': 35.004, 'min_lon': 135.774, 'max_lon': 135.780},
]

# KABU_4: 祇園・東山エリア東部
BOUNDING_BOXES_KABU_4 = [
    {'name': 'A4-1: Higashiyama_South', 'min_lat': 34.992, 'max_lat': 34.998, 'min_lon': 135.780, 'max_lon': 135.788},
    {'name': 'A4-2: Higashiyama_North', 'min_lat': 34.998, 'max_lat': 35.004, 'min_lon': 135.780, 'max_lon': 135.788},
]

# KABU_5: 東山山麓エリア
BOUNDING_BOXES_KABU_5 = [
    {'name': 'A5-1: Maruyama_Yasaka', 'min_lat': 35.003, 'max_lat': 35.008, 'min_lon': 135.780, 'max_lon': 135.788},
    {'name': 'A5-2: Chionin_Shoren', 'min_lat': 35.006, 'max_lat': 35.008, 'min_lon': 135.774, 'max_lon': 135.782},
]

# KABU_6: 嵐山エリア西部（竹林・天龍寺）
BOUNDING_BOXES_KABU_6 = [
    {'name': 'A6-1: Arashiyama_Bamboo', 'min_lat': 35.015, 'max_lat': 35.018, 'min_lon': 135.665, 'max_lon': 135.672},
    {'name': 'A6-2: Tenryuji_Area', 'min_lat': 35.013, 'max_lat': 35.017, 'min_lon': 135.672, 'max_lon': 135.678},
]

# KABU_7: 嵐山エリア中央（渡月橋周辺）
BOUNDING_BOXES_KABU_7 = [
    {'name': 'A7-1: Togetsukyo_Bridge', 'min_lat': 35.010, 'max_lat': 35.014, 'min_lon': 135.675, 'max_lon': 135.682},
    {'name': 'A7-2: Arashiyama_Station', 'min_lat': 35.012, 'max_lat': 35.016, 'min_lon': 135.678, 'max_lon': 135.685},
]

# KABU_8: 嵯峨野エリア北部
BOUNDING_BOXES_KABU_8 = [
    {'name': 'A8-1: Sagano_North', 'min_lat': 35.017, 'max_lat': 35.020, 'min_lon': 135.668, 'max_lon': 135.675},
    {'name': 'A8-2: Adashino_Nenbutsuji', 'min_lat': 35.018, 'max_lat': 35.020, 'min_lon': 135.675, 'max_lon': 135.682},
]

# ★★★ 実行時に使用するエリアを選択してください ★★★
# 処理順序の推奨：KABU_1 → KABU_2 → KABU_3 → KABU_4 → KABU_5 → KABU_6 → KABU_7 → KABU_8

# 統合エリア（全て実行する場合）
BOUNDING_BOXES_KABU_ALL = (
    BOUNDING_BOXES_KABU_1 + BOUNDING_BOXES_KABU_2 + BOUNDING_BOXES_KABU_3 + BOUNDING_BOXES_KABU_4 +
    BOUNDING_BOXES_KABU_5 + BOUNDING_BOXES_KABU_6 + BOUNDING_BOXES_KABU_7 + BOUNDING_BOXES_KABU_8
)

# 【KIMさんの担当エリア】京都南部・北部＋滋賀西部 (約320 Geohash)
BOUNDING_BOXES_KIM = [
    {'name': 'B: Kyoto_Gojo_Station', 'min_lat': 34.983, 'max_lat': 35.000, 'min_lon': 135.755, 'max_lon': 135.768},
    {'name': 'B: Fushimi_Inari', 'min_lat': 34.965, 'max_lat': 34.975, 'min_lon': 135.770, 'max_lon': 135.780},
    {'name': 'B: Kinkakuji_Kitano', 'min_lat': 35.025, 'max_lat': 35.039, 'min_lon': 135.725, 'max_lon': 135.745},
    {'name': 'B: Shiga_Otsu', 'min_lat': 35.000, 'max_lat': 35.025, 'min_lon': 135.850, 'max_lon': 135.880},
]

# 【KOSUKEさんの担当エリア】大阪キタ＋ミナミ (約350 Geohash)
BOUNDING_BOXES_KOSUKE = [
    {'name': 'C: Osaka_Kita_Umeda', 'min_lat': 34.695, 'max_lat': 34.710, 'min_lon': 135.490, 'max_lon': 135.510},
    {'name': 'C: Osaka_Minami_Namba', 'min_lat': 34.660, 'max_lat': 34.675, 'min_lon': 135.495, 'max_lon': 135.515},
]

# 【RIHOさんの担当エリア】大阪その他＋滋賀東部 (約250 Geohash)
BOUNDING_BOXES_RIHO = [
    {'name': 'D: Osaka_Tennoji_Shinsekai', 'min_lat': 34.645, 'max_lat': 34.658, 'min_lon': 135.505, 'max_lon': 135.520},
    {'name': 'D: Osaka_Castle', 'min_lat': 34.682, 'max_lat': 34.692, 'min_lon': 135.520, 'max_lon': 135.535},
    {'name': 'D: Shiga_Kusatsu', 'min_lat': 35.010, 'max_lat': 35.035, 'min_lon': 135.940, 'max_lon': 135.970},
]

# ===================================================================
# KABUさん向け使用方法:
# 1. 上記のTARGET_BOUNDING_BOXESで1つずつエリアを選択
# 2. 推奨順序: KABU_1 → KABU_2 → KABU_3 → KABU_4 → KABU_5 → KABU_6 → KABU_7 → KABU_8
# 3. 各エリアは約30分で完了予定（ネットワーク状況により変動）
# 4. 進捗ファイルが自動保存されるため、中断しても続きから実行可能
# ===================================================================


# ★★★ 自分の担当に合わせて、以下の1行のコメントを外してください ★★★

# KABUさん用（細分化エリア - 1つずつ実行推奨）
TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_1  # 四条烏丸エリア中心部（約30分）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_2  # 河原町エリア（約30分）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_3  # 祇園エリア西部（約30分）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_4  # 祇園・東山エリア東部（約30分）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_5  # 東山山麓エリア（約30分）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_6  # 嵐山エリア西部（約30分）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_7  # 嵐山エリア中央（約30分）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_8  # 嵯峨野エリア北部（約30分）

# 全エリアを一度に実行する場合（非推奨：時間がかかります）
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU_ALL

# 他のメンバー用
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KIM
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KOSUKE
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_RIHO


# --- スクリプト本体 ---

def save_progress(processed_geohashes, total_geohashes):
    """進捗をファイルに保存"""
    progress_data = {
        'processed_geohashes': list(processed_geohashes),
        'total_count': len(total_geohashes),
        'processed_count': len(processed_geohashes),
        'timestamp': time.time()
    }
    try:
        with open(PROGRESS_FILE, 'w', encoding='utf-8') as f:
            json.dump(progress_data, f, ensure_ascii=False, indent=2)
        print(f"進捗を保存しました ({len(processed_geohashes)}/{len(total_geohashes)})")
    except Exception as e:
        print(f"進捗保存エラー: {e}")

def load_progress():
    """進捗ファイルから処理済みGeohashを読み込み"""
    try:
        if os.path.exists(PROGRESS_FILE):
            with open(PROGRESS_FILE, 'r', encoding='utf-8') as f:
                progress_data = json.load(f)
            processed = set(progress_data.get('processed_geohashes', []))
            print(f"前回の進捗を読み込みました: {len(processed)}件処理済み")
            return processed
    except Exception as e:
        print(f"進捗読み込みエラー: {e}")
    return set()

def clear_progress():
    """進捗ファイルを削除"""
    try:
        if os.path.exists(PROGRESS_FILE):
            os.remove(PROGRESS_FILE)
            print("進捗ファイルをクリアしました")
    except Exception as e:
        print(f"進捗ファイル削除エラー: {e}")

def get_geohashes_in_all_boxes(bounding_boxes, precision):
    """指定した複数の矩形範囲内のGeohashをすべてリストアップする"""
    all_geohashes = set()
    lat_step = 0.005 # 精度6に合わせてステップを調整
    lon_step = 0.005

    for bbox in bounding_boxes:
        print(f"  - エリア '{bbox['name']}' のGeohashを計算中...")
        lat = bbox['min_lat']
        while lat < bbox['max_lat']:
            lon = bbox['min_lon']
            while lon < bbox['max_lon']:
                all_geohashes.add(pgh.encode(lat, lon, precision))
                lon += lon_step
            lat += lat_step
    return list(all_geohashes)

def is_riverside_park(place_name, place_types):
    """公園が河川敷かどうかを判定する"""
    # 名前に河川敷関連キーワードが含まれているかチェック
    name_lower = place_name.lower()
    for keyword in RIVERSIDE_KEYWORDS:
        if keyword.lower() in name_lower:
            return True
    return False

def ensure_grid_cells_exist(supabase: Client, geohashes):
    """DBにgrid_cellが存在することを確認し、なければ作成する。geohash->idの辞書を返す。"""
    print("グリッドセルの準備を開始します...")
    
    # 既存のgrid_cellsを確認
    try:
        existing_response = supabase.table('grid_cells').select('id, geohash').execute()
        print(f"🔍 既存のgrid_cells: {len(existing_response.data)}件")
    except Exception as e:
        print(f"❌ grid_cellsテーブルへのアクセスエラー: {e}")
        return None
    
    cells_to_insert = []
    for geohash in geohashes:
        bbox = pgh.get_bounding_box(geohash)
        cell_polygon = box(bbox.min_lon, bbox.min_lat, bbox.max_lon, bbox.max_lat)
        cells_to_insert.append({
            'geohash': geohash,
            'geometry': f"SRID=4326;{cell_polygon.wkt}"
        })
    
    try:
        # geohashが重複した場合は無視する (upsert)
        supabase.table('grid_cells').upsert(cells_to_insert, on_conflict='geohash').execute()
        print("✅ グリッドセルの準備が完了しました。")
    except Exception as e:
        print(f"❌ グリッドセル準備エラー: {e}")
        return None

    # 作成/確認したセルの情報をDBから取得して辞書を作成
    try:
        response = supabase.table('grid_cells').select('id, geohash').in_('geohash', geohashes).execute()
        geohash_to_id_map = {cell['geohash']: cell['id'] for cell in response.data}
        
        # 全てのgeohashに対してgrid_cell_idが取得できたか確認
        missing_geohashes = [gh for gh in geohashes if gh not in geohash_to_id_map]
        if missing_geohashes:
            print(f"⚠️ 警告: {len(missing_geohashes)}個のgeohashでgrid_cell_idが取得できませんでした")
            print(f"   例: {missing_geohashes[:3]}")
        
        print(f"📦 geohash->grid_cell_id マッピング: {len(geohash_to_id_map)}件")
        return geohash_to_id_map
    except Exception as e:
        print(f"❌ grid_cell取得エラー: {e}")
        return None

def fetch_places(lat, lon, radius, poi_type):
    """Google Places API (New)からスポット情報を取得する"""
    all_places = []
    url = "https://places.googleapis.com/v1/places:searchNearby"
    
    # POI_TYPEをPlaces API (New)の形式に変換
    type_mapping = {
        'cafe': 'cafe',
        'park': 'park',
        'tourist_attraction': 'tourist_attraction',
        'art_gallery': 'art_gallery',
        'book_store': 'book_store',
        'bakery': 'bakery',
        'store': 'store',
        'home_goods_store': 'home_goods_store',
        'museum': 'museum',
        'shrine': 'hindu_temple',  # shrineに最も近いタイプ
        'temple': 'hindu_temple',
        'florist': 'florist',
        'library': 'library',
        'riverside_park': 'park'  # riverside_parkは後でフィルタリング
    }
    
    included_type = type_mapping.get(poi_type, poi_type)
    
    headers = {
        'Content-Type': 'application/json',
        'X-Goog-Api-Key': GOOGLE_MAPS_API_KEY,
        'X-Goog-FieldMask': 'places.id,places.displayName,places.location,places.rating,places.types,places.primaryType'
    }
    
    data = {
        'locationRestriction': {
            'circle': {
                'center': {
                    'latitude': lat,
                    'longitude': lon
                },
                'radius': radius
            }
        },
        'includedTypes': [included_type],
        'maxResultCount': 20,
        'languageCode': 'ja'
    }
    
    retry_count = 0
    while retry_count <= MAX_RETRIES:
        try:
            response = requests.post(url, headers=headers, json=data)
            response.raise_for_status()
            results = response.json()
            
            # 新しいAPIの結果を古い形式に変換
            places = results.get('places', [])
            for place in places:
                converted_place = {
                    'place_id': place.get('id', ''),
                    'name': place.get('displayName', {}).get('text', ''),
                    'rating': place.get('rating', 0),
                    'types': place.get('types', []),
                    'geometry': {
                        'location': {
                            'lat': place.get('location', {}).get('latitude', 0),
                            'lng': place.get('location', {}).get('longitude', 0)
                        }
                    }
                }
                all_places.append(converted_place)
            
            break  # 新しいAPIはページングが異なるため、一度で取得
                
        except requests.exceptions.RequestException as e:
            retry_count += 1
            if retry_count <= MAX_RETRIES:
                wait_time = 2 ** retry_count  # 指数バックオフ
                print(f"    -> APIエラー (試行{retry_count}/{MAX_RETRIES}): {e}")
                print(f"    -> {wait_time}秒後にリトライします...")
                time.sleep(wait_time)
            else:
                print(f"    -> 最大リトライ回数に達しました: {e}")
                break
        except Exception as e:
            print(f"    -> API警告: {str(e)}")
            break
            
    return all_places

def populate_pois_chunk(supabase: Client, geohash_chunk, geohash_to_id_map, processed_geohashes, all_geohashes):
    """Geohashチャンクに対応するPOIをDBに保存する"""
    print(f"{len(geohash_chunk)}個のGeohashの処理を開始します...")
    
    for i, geohash in enumerate(geohash_chunk):
        try:
            print(f"  - Geohash {i+1}/{len(geohash_chunk)}: {geohash} を処理中...")
            lat, lon = pgh.decode(geohash)
            (lat_err, lon_err) = pgh.decode_exactly(geohash)[2:4]
            radius = ((lat_err * 111000)**2 + (lon_err * 111000)**2)**0.5

            found_place_ids = set()
            pois_to_insert = []
            park_places = []  # parkの結果を保存
            
            for poi_type in POI_TYPES:
                # riverside_parkはスキップ（parkの結果を後で処理）
                if poi_type == 'riverside_park':
                    continue
                    
                places = fetch_places(lat, lon, radius, poi_type)
                
                # parkの場合は結果を保存
                if poi_type == 'park':
                    park_places = places
                
                for place in places:
                    place_id = place.get('place_id')
                    rating = place.get('rating', 0)

                    # parkの場合は3.0以上、その他は3.5以上
                    min_rating = 3.0 if poi_type == 'park' else MIN_RATING
                    
                    if place_id and place_id not in found_place_ids and rating >= min_rating:
                        place_name = place['name']
                        place_types = place.get('types', [])
                        
                        # parkタイプで河川敷の場合はスキップ（後でriverside_parkとして処理）
                        if poi_type == 'park' and is_riverside_park(place_name, place_types):
                            continue
                        
                        found_place_ids.add(place_id)
                        loc = place['geometry']['location']
                        point = Point(loc['lng'], loc['lat'])
                        
                        # メインカテゴリを決定（最初の要素または poi_type）
                        main_category = place_types[0] if place_types else poi_type
                        
                        pois_to_insert.append({
                            'id': place_id,
                            'name': place_name,
                            'location': f"SRID=4326;{point.wkt}",
                            'category': main_category,
                            'rate': rating,
                            'grid_cell_id': geohash_to_id_map.get(geohash)
                        })
            
            # parkの結果から河川敷を抽出してriverside_parkとして処理
            for place in park_places:
                place_id = place.get('place_id')
                rating = place.get('rating', 0)

                # 河川敷も公園なので3.0以上で判定
                if place_id and place_id not in found_place_ids and rating >= 3.0:
                    place_name = place['name']
                    place_types = place.get('types', [])
                    
                    # 河川敷かどうかチェック
                    if is_riverside_park(place_name, place_types):
                        found_place_ids.add(place_id)
                        loc = place['geometry']['location']
                        point = Point(loc['lng'], loc['lat'])
                        
                        # riverside_parkを優先カテゴリとして設定
                        pois_to_insert.append({
                            'id': place_id,
                            'name': place_name,
                            'location': f"SRID=4326;{point.wkt}",
                            'category': 'riverside_park',
                            'rate': rating,
                            'grid_cell_id': geohash_to_id_map.get(geohash)
                        })

            if pois_to_insert:
                try:
                    # POI保存前にgrid_cell_idの検証
                    grid_cell_id = geohash_to_id_map.get(geohash)
                    if not grid_cell_id:
                        print(f"    ❌ エラー: geohash {geohash} に対するgrid_cell_idが見つかりません")
                        continue
                    
                    # 保存直前にgrid_cellの存在を再確認
                    verify_response = supabase.table('grid_cells').select('id').eq('id', grid_cell_id).execute()
                    if not verify_response.data:
                        print(f"    ❌ 致命的エラー: grid_cell ID {grid_cell_id} が保存直前に見つかりません")
                        continue
                    
                    print(f"    💾 {len(pois_to_insert)}件のPOIをDBに保存中（grid_cell_id: {grid_cell_id}）...")
                    supabase.table('pois').upsert(pois_to_insert, on_conflict='id').execute()
                    print(f"    ✅ {len(pois_to_insert)}件のPOIをDBに保存しました。")
                except Exception as e:
                    print(f"    ❌ POI挿入エラー: {e}")
                    print(f"       Geohash: {geohash}, grid_cell_id: {geohash_to_id_map.get(geohash)}")
                    continue  # このGeohashは失敗として扱うが、次に進む
            
            # 成功したGeohashを記録
            processed_geohashes.add(geohash)
            
            # 5個処理するごとに進捗保存
            if len(processed_geohashes) % 5 == 0:
                save_progress(processed_geohashes, all_geohashes)
            
            time.sleep(1) # APIの過剰な連続リクエストを防ぐためのウェイト
            
        except Exception as e:
            print(f"    -> Geohash {geohash} の処理でエラー: {e}")
            continue  # エラーが起きても次のGeohashに進む

def main():
    """メイン処理"""
    if 'TARGET_BOUNDING_BOXES' not in globals():
        print("\n★★★ エラー ★★★")
        print("スクリプト上部の「自分の担当に合わせて、以下の1行のコメントを外してください」の部分で、")
        print("実行したいエリアの行のコメントアウトを解除してください。 (例: TARGET_BOUNDING_BOXES = BOUNDING_BOXES_A)")
        return

    if not all([SUPABASE_URL, SUPABASE_KEY, GOOGLE_MAPS_API_KEY]):
        print("エラー: .envファイルを正しく設定してください。")
        return

    supabase: Client = create_client(SUPABASE_URL, SUPABASE_KEY)
    
    # 進捗確認
    processed_geohashes = load_progress()
    restart = input("\n前回の続きから実行しますか？ (y/n): ").lower().strip()
    if restart != 'y':
        processed_geohashes.clear()
        clear_progress()
        print("最初から実行します。")
    
    print("全対象エリアのGeohashを計算中...")
    all_geohashes = get_geohashes_in_all_boxes(TARGET_BOUNDING_BOXES, GEOHASH_PRECISION)
    print(f"合計 {len(all_geohashes)}個のGeohashが見つかりました。")
    
    # 未処理のGeohashを抽出
    remaining_geohashes = [gh for gh in all_geohashes if gh not in processed_geohashes]
    print(f"未処理: {len(remaining_geohashes)}個のGeohash")
    
    if not remaining_geohashes:
        print("すべてのGeohashが処理済みです。")
        return
    
    # グリッドセルを準備し、geohash->idの対応辞書を取得
    print("\n🔧 グリッドセル準備中...")
    geohash_to_id_map = ensure_grid_cells_exist(supabase, all_geohashes)
    if not geohash_to_id_map:
        print("❌ グリッドセルの準備に失敗したため、処理を中断します。")
        return
    
    print(f"✅ grid_cellマッピング準備完了: {len(geohash_to_id_map)}件")

    # チャンク単位で処理
    try:
        for i in range(0, len(remaining_geohashes), CHUNK_SIZE):
            chunk = remaining_geohashes[i:i + CHUNK_SIZE]
            chunk_num = i // CHUNK_SIZE + 1
            total_chunks = (len(remaining_geohashes) + CHUNK_SIZE - 1) // CHUNK_SIZE
            
            print(f"\n📦 チャンク {chunk_num}/{total_chunks} を処理中...")
            populate_pois_chunk(supabase, chunk, geohash_to_id_map, processed_geohashes, all_geohashes)
            
            print(f"✅ チャンク {chunk_num} 完了。進捗: {len(processed_geohashes)}/{len(all_geohashes)}")
            
        # 最終進捗保存
        save_progress(processed_geohashes, all_geohashes)
        print(f"\n🎉 全処理が完了しました！")
        print(f"📊 処理済み: {len(processed_geohashes)}/{len(all_geohashes)}")
        
        # 完了後に進捗ファイルを削除するか確認
        cleanup = input("進捗ファイルを削除しますか？ (y/n): ").lower().strip()
        if cleanup == 'y':
            clear_progress()
            print("🗑️  進捗ファイルを削除しました。")
            
    except KeyboardInterrupt:
        print(f"\n⚠️  処理が中断されました。")
        save_progress(processed_geohashes, all_geohashes)
        print("💾 進捗は保存されました。次回は続きから実行できます。")
    except Exception as e:
        print(f"\n❌ 予期しないエラーが発生しました: {e}")
        save_progress(processed_geohashes, all_geohashes)
        print("💾 進捗は保存されました。問題を解決してから再実行してください。")


if __name__ == "__main__":
    main()
