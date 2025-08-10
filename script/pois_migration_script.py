import os
import time
import requests
import argparse
import pygeohash as pgh
from dotenv import load_dotenv
from supabase import create_client, Client
from shapely.geometry import Point

# .envファイルから環境変数を読み込む
load_dotenv()

# --- 設定項目 ---
SUPABASE_URL = os.environ.get("SUPABASE_URL")
SUPABASE_KEY = os.environ.get("SUPABASE_ANON_KEY")
GOOGLE_API_KEY = os.environ.get("GOOGLE_API_KEY")

GEOHASH_PRECISION = 6 # グリッドの精度
MIN_RATING = 3.5

# 散歩が楽しくなるスポットのカテゴリ
POI_TYPES = [
    'cafe', 'park', 'tourist_attraction', 'art_gallery', 'book_store',
    'bakery', 'store', 'shopping_mall', 'home_goods_store', 'museum',
    'shrine', 'temple'
]

# --- エリア定義 (4人分担用) ---

# 【KABUさんの担当エリア】京都中心部＋嵐山 (約330 Geohash)
BOUNDING_BOXES_KABU = [
    {'name': 'A: Shijo_Karasuma_Kawaramachi', 'min_lat': 35.000, 'max_lat': 35.008, 'min_lon': 135.758, 'max_lon': 135.775},
    {'name': 'A: Gion_Higashiyama', 'min_lat': 34.992, 'max_lat': 35.008, 'min_lon': 135.774, 'max_lon': 135.788},
    {'name': 'A: Arashiyama_Sagano', 'min_lat': 35.008, 'max_lat': 35.020, 'min_lon': 135.665, 'max_lon': 135.685},
]

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


# ★★★ 自分の担当に合わせて、以下の1行のコメントを外してください ★★★
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KABU
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KIM
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_KOSUKE
# TARGET_BOUNDING_BOXES = BOUNDING_BOXES_RIHO


# --- スクリプト本体 ---

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

def fetch_places(lat, lon, radius, poi_type):
    """Google Places APIからスポット情報を取得する"""
    all_places = []
    url = "https://maps.googleapis.com/maps/api/place/nearbysearch/json"
    params = {
        'location': f"{lat},{lon}",
        'radius': radius,
        'type': poi_type,
        'key': GOOGLE_API_KEY,
        'language': 'ja'
    }
    
    while True:
        try:
            response = requests.get(url, params=params)
            response.raise_for_status()
            results = response.json()
            all_places.extend(results.get('results', []))
            
            if 'next_page_token' in results and results['next_page_token']:
                params['pagetoken'] = results['next_page_token']
                time.sleep(2) # Googleの推奨に従い、次のページ取得前に待機
            else:
                break
        except requests.exceptions.RequestException as e:
            print(f"  - APIリクエストエラー: {e}")
            break
            
    return all_places

def populate_pois(supabase: Client, geohash_chunk):
    """各Geohashに対応するPOIをDBに保存する"""
    print(f"{len(geohash_chunk)}個のGeohashの処理を開始します...")
    
    for i, geohash in enumerate(geohash_chunk):
        print(f"  - Geohash {i+1}/{len(geohash_chunk)}: {geohash} を処理中...")
        lat, lon = pgh.decode(geohash)
        (lat_err, lon_err) = pgh.decode_exactly(geohash)[2:4]
        radius = ((lat_err * 111000)**2 + (lon_err * 111000)**2)**0.5

        found_place_ids = set()
        pois_to_insert = []
        
        for poi_type in POI_TYPES:
            places = fetch_places(lat, lon, radius, poi_type)
            for place in places:
                place_id = place.get('place_id')
                rating = place.get('rating', 0)

                if place_id and place_id not in found_place_ids and rating >= MIN_RATING:
                    found_place_ids.add(place_id)
                    loc = place['geometry']['location']
                    point = Point(loc['lng'], loc['lat'])
                    
                    pois_to_insert.append({
                        'id': place_id,
                        'name': place['name'],
                        'location': f"SRID=4326;{point.wkt}",
                        'categories': place.get('types', []),
                        'rate': rating
                    })

        if pois_to_insert:
            try:
                supabase.table('pois').upsert(pois_to_insert, on_conflict='id').execute()
                print(f"    -> {len(pois_to_insert)}件のPOIをDBに保存しました。")
            except Exception as e:
                print(f"    -> POI挿入エラー: {e}")
        time.sleep(1) # APIの過剰な連続リクエストを防ぐためのウェイト

def main():
    """メイン処理"""
    if 'TARGET_BOUNDING_BOXES' not in globals():
        print("\n★★★ エラー ★★★")
        print("スクリプト上部の「自分の担当に合わせて、以下の1行のコメントを外してください」の部分で、")
        print("実行したいエリアの行のコメントアウトを解除してください。 (例: TARGET_BOUNDING_BOXES = BOUNDING_BOXES_A)")
        return

    if not all([SUPABASE_URL, SUPABASE_KEY, GOOGLE_API_KEY]):
        print("エラー: .envファイルを正しく設定してください。")
        return

    supabase: Client = create_client(SUPABASE_URL, SUPABASE_KEY)
    
    print("全対象エリアのGeohashを計算中...")
    all_geohashes = get_geohashes_in_all_boxes(TARGET_BOUNDING_BOXES, GEOHASH_PRECISION)
    print(f"合計 {len(all_geohashes)}個のGeohashが見つかりました。")
    
    populate_pois(supabase, all_geohashes)

    print(f"担当エリアの処理が完了しました。")


if __name__ == "__main__":
    main()
