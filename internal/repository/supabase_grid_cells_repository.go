package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"Team8-App/internal/database"
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
)

type SupabaseGridCellsRepository struct {
	client *database.SupabaseClient
}

func NewSupabaseGridCellsRepository(client *database.SupabaseClient) repository.GridCellsRepository {
	return &SupabaseGridCellsRepository{
		client: client,
	}
}

func (r *SupabaseGridCellsRepository) GetByID(ctx context.Context, id int) (*model.GridCell, error) {
	var gridCells []model.GridCell
	data, count, err := r.client.GetClient().From("grid_cells").Select("*", "exact", false).Eq("id", strconv.Itoa(id)).Execute()
	if err != nil {
		return nil, fmt.Errorf("グリッドセルデータの取得失敗: %w", err)
	}
	_ = count // countは使わないが、構文エラーを避けるため

	if err := json.Unmarshal([]byte(data), &gridCells); err != nil {
		return nil, fmt.Errorf("グリッドセルデータのJSONアンマーシャル失敗: %w", err)
	}

	if len(gridCells) == 0 {
		return nil, fmt.Errorf("グリッドセル ID %d が見つかりません", id)
	}

	return &gridCells[0], nil
}

func (r *SupabaseGridCellsRepository) GetContainingPoint(ctx context.Context, lat, lng float64) (*model.GridCell, error) {
	// PostGIS ST_Contains関数を使用した空間検索
	// ここでは簡易的な実装として、すべてのグリッドセルを取得
	var gridCells []model.GridCell
	data, count, err := r.client.GetClient().From("grid_cells").Select("*", "exact", false).Execute()
	if err != nil {
		return nil, fmt.Errorf("指定座標を含むグリッドセルの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &gridCells); err != nil {
		return nil, fmt.Errorf("グリッドセルデータのJSONアンマーシャル失敗: %w", err)
	}

	// TODO: 実際にはPostGISのST_Contains関数を使用して効率的に検索
	// 現在は簡易的な実装として最初のグリッドセルを返す
	if len(gridCells) > 0 {
		return &gridCells[0], nil
	}

	return nil, fmt.Errorf("指定座標 (%.6f, %.6f) を含むグリッドセルが見つかりません", lat, lng)
}

// GetByBoundingBox 指定された境界ボックス内のグリッドセル一覧を取得
func (r *SupabaseGridCellsRepository) GetByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.GridCell, error) {
	// PostGIS ST_Intersects関数を使用した空間検索
	var gridCells []model.GridCell
	data, count, err := r.client.GetClient().From("grid_cells").Select("*", "exact", false).Execute()
	if err != nil {
		return nil, fmt.Errorf("境界ボックス内グリッドセルの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &gridCells); err != nil {
		return nil, fmt.Errorf("グリッドセルデータのJSONアンマーシャル失敗: %w", err)
	}

	// TODO: 実際にはPostGISのST_Intersects関数を使用して効率的に検索
	// 現在は簡易的な実装としてすべてのグリッドセルを返す
	return gridCells, nil
}

func (r *SupabaseGridCellsRepository) Create(ctx context.Context, gridCell *model.GridCell) error {
	data, err := json.Marshal(gridCell)
	if err != nil {
		return fmt.Errorf("グリッドセルデータのJSONマーシャル失敗: %w", err)
	}

	_, _, err = r.client.GetClient().From("grid_cells").Insert(string(data), false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("グリッドセルデータの作成失敗: %w", err)
	}

	return nil
}

func (r *SupabaseGridCellsRepository) Update(ctx context.Context, gridCell *model.GridCell) error {
	data, err := json.Marshal(gridCell)
	if err != nil {
		return fmt.Errorf("グリッドセルデータのJSONマーシャル失敗: %w", err)
	}

	_, _, err = r.client.GetClient().From("grid_cells").Update(string(data), "", "").Eq("id", strconv.Itoa(gridCell.ID)).Execute()
	if err != nil {
		return fmt.Errorf("グリッドセルデータの更新失敗: %w", err)
	}

	return nil
}

func (r *SupabaseGridCellsRepository) Delete(ctx context.Context, id int) error {
	_, _, err := r.client.GetClient().From("grid_cells").Delete("", "").Eq("id", strconv.Itoa(id)).Execute()
	if err != nil {
		return fmt.Errorf("グリッドセルデータの削除失敗: %w", err)
	}

	return nil
}

func (r *SupabaseGridCellsRepository) GetAll(ctx context.Context) ([]model.GridCell, error) {
	var gridCells []model.GridCell
	data, count, err := r.client.GetClient().From("grid_cells").Select("*", "exact", false).Execute()
	if err != nil {
		return nil, fmt.Errorf("全グリッドセルデータの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &gridCells); err != nil {
		return nil, fmt.Errorf("グリッドセルデータのJSONアンマーシャル失敗: %w", err)
	}

	return gridCells, nil
}
