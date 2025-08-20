#!/usr/bin/env python3
"""
Grid Cellの重複チェックと修正スクリプト
"""

import os
from supabase import create_client, Client
from dotenv import load_dotenv

# 設定
load_dotenv()

SUPABASE_URL = os.environ.get("SUPABASE_URL")
SUPABASE_ANON_KEY = os.environ.get("SUPABASE_ANON_KEY")

def main():
    """メイン処理"""
    print("--- 🔍 Grid Cell重複チェック開始 ---")
    
    if not all([SUPABASE_URL, SUPABASE_ANON_KEY]):
        print("❌ エラー: 環境変数 SUPABASE_URL と SUPABASE_ANON_KEY が設定されていません。")
        return
    
    supabase: Client = create_client(SUPABASE_URL, SUPABASE_ANON_KEY)
    print("✅ Supabaseクライアントの初期化完了")
    
    # 1. 重複するgeohashを検索
    print("\n🔎 重複するgeohashを検索中...")
    
    try:
        # すべてのgrid_cellsを取得してgeohashでグループ化
        result = supabase.table('grid_cells').select('id, geohash, created_at').execute()
        
        if not result.data:
            print("❌ Grid Cellデータが見つかりません。")
            return
            
        # geohashごとにグループ化
        geohash_groups = {}
        for cell in result.data:
            geohash = cell['geohash']
            if geohash not in geohash_groups:
                geohash_groups[geohash] = []
            geohash_groups[geohash].append(cell)
        
        # 重複を検出
        duplicates = {}
        for geohash, cells in geohash_groups.items():
            if len(cells) > 1:
                # 作成日時順でソート（古い順）
                sorted_cells = sorted(cells, key=lambda x: x['created_at'])
                duplicates[geohash] = {
                    'keep': sorted_cells[0],  # 最初に作成された方を保持
                    'remove': sorted_cells[1:]  # 後から作成された方を削除
                }
        
        if not duplicates:
            print("✅ 重複するGrid Cellは見つかりませんでした。")
            return
        
        print(f"⚠️  {len(duplicates)}個の重複するgeohashが見つかりました:")
        
        total_keep = 0
        total_remove = 0
        total_pois_to_update = 0
        
        for geohash, dup_data in duplicates.items():
            keep_cell = dup_data['keep']
            remove_cells = dup_data['remove']
            
            print(f"\n📍 Geohash: {geohash}")
            print(f"   保持: ID {keep_cell['id']} (作成日時: {keep_cell['created_at']})")
            
            for remove_cell in remove_cells:
                print(f"   削除予定: ID {remove_cell['id']} (作成日時: {remove_cell['created_at']})")
                
                # このgrid_cellを参照しているPOIを確認
                poi_result = supabase.table('pois').select('id, name').eq('grid_cell_id', remove_cell['id']).execute()
                if poi_result.data:
                    print(f"      → {len(poi_result.data)}個のPOIが参照中:")
                    for poi in poi_result.data:
                        print(f"         - {poi['name']} (ID: {poi['id']})")
                    total_pois_to_update += len(poi_result.data)
                else:
                    print(f"      → 参照するPOIなし")
            
            total_keep += 1
            total_remove += len(remove_cells)
        
        print(f"\n📊 統計:")
        print(f"   保持するGrid Cell: {total_keep}個")
        print(f"   削除するGrid Cell: {total_remove}個") 
        print(f"   更新が必要なPOI: {total_pois_to_update}個")
        
        # 2. 修正実行の確認
        print(f"\n❓ 修正を実行しますか？")
        print(f"   1. POIのgrid_cell_idを古い方のIDに更新")
        print(f"   2. 重複するGrid Cellを削除")
        
        confirm = input("実行する場合は 'yes' を入力してください: ")
        
        if confirm.lower() != 'yes':
            print("⏹️  修正をキャンセルしました。")
            return
        
        # 3. 修正実行
        print("\n🔧 修正実行中...")
        
        updated_pois = 0
        deleted_cells = 0
        
        for geohash, dup_data in duplicates.items():
            keep_cell = dup_data['keep']
            remove_cells = dup_data['remove']
            
            for remove_cell in remove_cells:
                # このgrid_cellを参照しているPOIを更新
                poi_result = supabase.table('pois').select('id').eq('grid_cell_id', remove_cell['id']).execute()
                
                if poi_result.data:
                    for poi in poi_result.data:
                        update_result = supabase.table('pois').update({
                            'grid_cell_id': keep_cell['id']
                        }).eq('id', poi['id']).execute()
                        
                        if update_result.data:
                            updated_pois += 1
                            print(f"   ✅ POI ID {poi['id']} のgrid_cell_idを {remove_cell['id']} → {keep_cell['id']} に更新")
                        else:
                            print(f"   ❌ POI ID {poi['id']} の更新に失敗")
                
                # 重複するgrid_cellを削除
                delete_result = supabase.table('grid_cells').delete().eq('id', remove_cell['id']).execute()
                
                if delete_result.data is not None:  # 削除成功（空配列でも成功）
                    deleted_cells += 1
                    print(f"   🗑️  Grid Cell ID {remove_cell['id']} ({geohash}) を削除")
                else:
                    print(f"   ❌ Grid Cell ID {remove_cell['id']} の削除に失敗")
        
        print(f"\n✅ 修正完了:")
        print(f"   更新されたPOI: {updated_pois}個")
        print(f"   削除されたGrid Cell: {deleted_cells}個")
        
    except Exception as e:
        print(f"❌ エラーが発生しました: {e}")

if __name__ == "__main__":
    main()
