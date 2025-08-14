package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/infrastructure/database"
)

type SupabasePOIsRepository struct {
	client *database.SupabaseClient
}

func NewSupabasePOIsRepository(client *database.SupabaseClient) repository.POIsRepository {
	return &SupabasePOIsRepository{
		client: client,
	}
}

func (r *SupabasePOIsRepository) GetByID(ctx context.Context, id string) (*model.POI, error) {
	var pois []model.POI
	data, count, err := r.client.GetClient().From("pois").Select("*", "exact", false).Eq("id", id).Execute()
	if err != nil {
		return nil, fmt.Errorf("POIデータの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &pois); err != nil {
		return nil, fmt.Errorf("POIデータのJSONアンマーシャル失敗: %w", err)
	}

	if len(pois) == 0 {
		return nil, fmt.Errorf("POI ID %s が見つかりません", id)
	}

	return &pois[0], nil
}

func (r *SupabasePOIsRepository) GetByGridCellID(ctx context.Context, gridCellID int) ([]model.POI, error) {
	var pois []model.POI
	data, count, err := r.client.GetClient().From("pois").Select("*", "exact", false).Eq("grid_cell_id", strconv.Itoa(gridCellID)).Execute()
	if err != nil {
		return nil, fmt.Errorf("グリッドセル %d のPOIデータ取得失敗: %w", gridCellID, err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &pois); err != nil {
		return nil, fmt.Errorf("POIデータのJSONアンマーシャル失敗: %w", err)
	}

	return pois, nil
}

func (r *SupabasePOIsRepository) GetNearbyPOIs(ctx context.Context, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	// PostGIS ST_DWithin関数を使用した地理的検索
	// 簡易的な実装として、ここでは全POIを取得してフィルタリング
	var pois []model.POI
	data, count, err := r.client.GetClient().From("pois").Select("*", "exact", false).Execute()
	if err != nil {
		return nil, fmt.Errorf("周辺POIデータの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &pois); err != nil {
		return nil, fmt.Errorf("POIデータのJSONアンマーシャル失敗: %w", err)
	}

	// TODO: 実際にはPostGISのST_DWithin関数を使用して効率的に検索
	// 現在は簡易的な実装
	var nearbyPOIs []model.POI
	for _, poi := range pois {
		// 距離計算はここでは省略
		nearbyPOIs = append(nearbyPOIs, poi)
	}

	return nearbyPOIs, nil
}

func (r *SupabasePOIsRepository) GetByCategories(ctx context.Context, categories []string, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	// 複数カテゴリのORクエリを作成
	var pois []model.POI
	data, count, err := r.client.GetClient().From("pois").
		Select("*", "exact", false).
		In("category", categories).
		Execute()

	if err != nil {
		return nil, fmt.Errorf("カテゴリ別POIデータの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &pois); err != nil {
		return nil, fmt.Errorf("POIデータのJSONアンマーシャル失敗: %w", err)
	}

	return pois, nil
}

// GetByCategory 単一カテゴリでPOIを取得（新しいメソッド）
func (r *SupabasePOIsRepository) GetByCategory(ctx context.Context, category string, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	var pois []model.POI
	data, count, err := r.client.GetClient().From("pois").
		Select("*", "exact", false).
		Eq("category", category).
		Execute()

	if err != nil {
		return nil, fmt.Errorf("カテゴリ別POIデータの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &pois); err != nil {
		return nil, fmt.Errorf("POIデータのJSONアンマーシャル失敗: %w", err)
	}

	return pois, nil
}

func (r *SupabasePOIsRepository) GetByRatingRange(ctx context.Context, minRating float64, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	var pois []model.POI
	data, count, err := r.client.GetClient().From("pois").
		Select("*", "exact", false).
		Gte("rate", fmt.Sprintf("%.2f", minRating)).
		Execute()

	if err != nil {
		return nil, fmt.Errorf("評価値別POIデータの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &pois); err != nil {
		return nil, fmt.Errorf("POIデータのJSONアンマーシャル失敗: %w", err)
	}

	return pois, nil
}

func (r *SupabasePOIsRepository) Create(ctx context.Context, poi *model.POI) error {
	data, err := json.Marshal(poi)
	if err != nil {
		return fmt.Errorf("POIデータのJSONマーシャル失敗: %w", err)
	}

	_, _, err = r.client.GetClient().From("pois").Insert(string(data), false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("POIデータの作成失敗: %w", err)
	}

	return nil
}

func (r *SupabasePOIsRepository) Update(ctx context.Context, poi *model.POI) error {
	data, err := json.Marshal(poi)
	if err != nil {
		return fmt.Errorf("POIデータのJSONマーシャル失敗: %w", err)
	}

	_, _, err = r.client.GetClient().From("pois").Update(string(data), "", "").Eq("id", poi.ID).Execute()
	if err != nil {
		return fmt.Errorf("POIデータの更新失敗: %w", err)
	}

	return nil
}

func (r *SupabasePOIsRepository) Delete(ctx context.Context, id string) error {
	_, _, err := r.client.GetClient().From("pois").Delete("", "").Eq("id", id).Execute()
	if err != nil {
		return fmt.Errorf("POIデータの削除失敗: %w", err)
	}

	return nil
}

func (r *SupabasePOIsRepository) BulkCreate(ctx context.Context, pois []model.POI) error {
	data, err := json.Marshal(pois)
	if err != nil {
		return fmt.Errorf("POI一括データのJSONマーシャル失敗: %w", err)
	}

	_, _, err = r.client.GetClient().From("pois").Insert(string(data), false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("POI一括データの作成失敗: %w", err)
	}

	return nil
}

// FindNearbyByCategories ルート提案用のメソッド：カテゴリと位置に基づいてPOIを検索
func (r *SupabasePOIsRepository) FindNearbyByCategories(ctx context.Context, location model.LatLng, categories []string, radiusMeters int, limit int) ([]*model.POI, error) {
	var pois []model.POI
	data, count, err := r.client.GetClient().From("pois").
		Select("*", "exact", false).
		In("category", categories).
		Limit(limit, "").
		Execute()

	if err != nil {
		return nil, fmt.Errorf("周辺カテゴリ別POIデータの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &pois); err != nil {
		return nil, fmt.Errorf("POIデータのJSONアンマーシャル失敗: %w", err)
	}

	// ポインタスライスに変換
	var result []*model.POI
	for i := range pois {
		result = append(result, &pois[i])
	}

	// TODO: 実際にはPostGISのST_DWithin関数を使用して位置による絞り込みを行う
	// 現在は簡易的な実装（位置フィルタリングなし）

	return result, nil
}
