#!/usr/bin/env python3
"""
最近追加されたGrid CellとPOIの詳細確認スクリプト
"""

import os
from supabase import create_client, Client
from dotenv import load_dotenv
from datetime import datetime, timedelta

# 設定
load_dotenv()

SUPABASE_URL = os.environ.get("SUPABASE_URL")
SUPABASE_ANON_KEY = os.environ.get("SUPABASE_ANON_KEY")

def main():
    """メイン処理"""
    print("--- 📊 最近追加されたGrid CellとPOIの確認 ---")
    
    if not all([SUPABASE_URL, SUPABASE_ANON_KEY]):
        print("❌ エラー: 環境変数 SUPABASE_URL と SUPABASE_ANON_KEY が設定されていません。")
        return
    
    supabase: Client = create_client(SUPABASE_URL, SUPABASE_ANON_KEY)
    print("✅ Supabaseクライアントの初期化完了")
    
    try:
        # 今日の日付を取得
        today = datetime.now().strftime('%Y-%m-%d')
        
        # 1. 今日追加されたGrid Cellを確認
        print(f"\n🔍 {today}に追加されたGrid Cellを確認中...")
        
        grid_result = supabase.table('grid_cells').select('id, geohash, created_at').gte('created_at', today).order('created_at', desc=False).execute()
        
        if grid_result.data:
            print(f"✅ {len(grid_result.data)}個のGrid Cellが今日追加されました:")
            
            # ID範囲を確認
            ids = [cell['id'] for cell in grid_result.data]
            print(f"   ID範囲: {min(ids)} - {max(ids)}")
            
            # 最初の5個と最後の5個を表示
            print(f"\n   最初の5個:")
            for i, cell in enumerate(grid_result.data[:5]):
                print(f"     [{i+1}] ID: {cell['id']}, Geohash: {cell['geohash']}, 作成: {cell['created_at']}")
            
            if len(grid_result.data) > 5:
                print(f"   ...")
                print(f"   最後の5個:")
                for i, cell in enumerate(grid_result.data[-5:]):
                    idx = len(grid_result.data) - 5 + i + 1
                    print(f"     [{idx}] ID: {cell['id']}, Geohash: {cell['geohash']}, 作成: {cell['created_at']}")
        else:
            print("⚠️ 今日追加されたGrid Cellは見つかりませんでした。")
        
        # 2. 今日追加されたPOI（horror_spotカテゴリ）を確認
        print(f"\n🔍 {today}に追加されたホラーPOIを確認中...")
        
        poi_result = supabase.table('pois').select('id, name, categories, grid_cell_id, created_at').gte('created_at', today).execute()
        
        if poi_result.data:
            # horror_spotカテゴリを含むPOIをフィルタ
            horror_pois = [poi for poi in poi_result.data if 'horror_spot' in (poi.get('categories') or '')]
            
            print(f"✅ {len(horror_pois)}個のホラーPOIが今日追加されました:")
            
            if horror_pois:
                # ID範囲を確認
                poi_ids = [poi['id'] for poi in horror_pois]
                grid_cell_ids = [poi['grid_cell_id'] for poi in horror_pois if poi['grid_cell_id']]
                
                print(f"   POI ID範囲: {min(poi_ids)} - {max(poi_ids)}")
                if grid_cell_ids:
                    print(f"   参照Grid Cell ID範囲: {min(grid_cell_ids)} - {max(grid_cell_ids)}")
                
                # 最初の5個を表示
                print(f"\n   最初の5個:")
                for i, poi in enumerate(horror_pois[:5]):
                    print(f"     [{i+1}] POI ID: {poi['id']}, 名前: {poi['name']}, Grid Cell ID: {poi['grid_cell_id']}")
                
                if len(horror_pois) > 5:
                    print(f"   ...")
                    print(f"   最後の5個:")
                    for i, poi in enumerate(horror_pois[-5:]):
                        idx = len(horror_pois) - 5 + i + 1
                        print(f"     [{idx}] POI ID: {poi['id']}, 名前: {poi['name']}, Grid Cell ID: {poi['grid_cell_id']}")
        else:
            print("⚠️ 今日追加されたPOIは見つかりませんでした。")
        
        # 3. Grid CellとPOIの整合性確認
        print(f"\n🔍 Grid CellとPOIの整合性を確認中...")
        
        if grid_result.data and poi_result.data:
            horror_pois = [poi for poi in poi_result.data if 'horror_spot' in (poi.get('categories') or '')]
            
            # 新しく作成されたGrid Cell IDのセット
            new_grid_ids = set(cell['id'] for cell in grid_result.data)
            
            # ホラーPOIが参照するGrid Cell IDのセット
            horror_grid_ids = set(poi['grid_cell_id'] for poi in horror_pois if poi['grid_cell_id'])
            
            # 整合性チェック
            missing_in_new = horror_grid_ids - new_grid_ids
            unused_new = new_grid_ids - horror_grid_ids
            
            print(f"   新規Grid Cell数: {len(new_grid_ids)}")
            print(f"   ホラーPOIが参照するGrid Cell数: {len(horror_grid_ids)}")
            print(f"   新規作成されていないが参照されているGrid Cell: {len(missing_in_new)}")
            print(f"   新規作成されたが参照されていないGrid Cell: {len(unused_new)}")
            
            if missing_in_new:
                print(f"   ⚠️ 参照されているが新規作成されていないGrid Cell ID: {sorted(missing_in_new)[:10]}...")
            
            if unused_new:
                print(f"   ⚠️ 新規作成されたが参照されていないGrid Cell ID: {sorted(unused_new)[:10]}...")
        
        # 4. 統計情報
        print(f"\n📊 統計情報:")
        
        # 全Grid Cell数
        total_grid_result = supabase.table('grid_cells').select('id', count='exact').execute()
        print(f"   総Grid Cell数: {total_grid_result.count}")
        
        # 全POI数
        total_poi_result = supabase.table('pois').select('id', count='exact').execute()
        print(f"   総POI数: {total_poi_result.count}")
        
        # ホラーPOI数
        horror_poi_result = supabase.table('pois').select('id', count='exact').like('categories', '%horror_spot%').execute()
        print(f"   ホラーPOI数: {horror_poi_result.count}")
        
    except Exception as e:
        print(f"❌ エラーが発生しました: {e}")

if __name__ == "__main__":
    main()
