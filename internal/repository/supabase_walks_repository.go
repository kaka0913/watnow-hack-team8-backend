package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"Team8-App/internal/database"
	"Team8-App/model"
)

type SupabaseWalksRepository struct {
	client *database.SupabaseClient
}

func NewSupabaseWalksRepository(client *database.SupabaseClient) WalksRepository {
	return &SupabaseWalksRepository{
		client: client,
	}
}

func (r *SupabaseWalksRepository) Create(ctx context.Context, walk *model.Walk) error {
	// Walk を DB 保存用の形式に変換（地理情報を含む）
	walkDB := WalkToWalkDB(walk)

	data, err := json.Marshal(walkDB)
	if err != nil {
		return fmt.Errorf("散歩データのJSONマーシャル失敗: %w", err)
	}

	_, _, err = r.client.GetClient().From("walks").Insert(string(data), false, "", "", "").Execute()
	if err != nil {
		return fmt.Errorf("散歩データの作成失敗: %w", err)
	}

	return nil
}

func (r *SupabaseWalksRepository) GetByID(ctx context.Context, id string) (*model.Walk, error) {
	var walks []model.Walk
	data, count, err := r.client.GetClient().From("walks").Select("*", "exact", false).Eq("id", id).Execute()
	if err != nil {
		return nil, fmt.Errorf("散歩データの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &walks); err != nil {
		return nil, fmt.Errorf("散歩データのJSONアンマーシャル失敗: %w", err)
	}

	if len(walks) == 0 {
		return nil, fmt.Errorf("散歩ID %s が見つかりません", id)
	}

	return &walks[0], nil
}

func (r *SupabaseWalksRepository) GetWalksByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.WalkSummary, error) {
	// 境界ボックスのPolygonを作成（WKT形式）
	bbox := fmt.Sprintf("POLYGON((%f %f,%f %f,%f %f,%f %f,%f %f))",
		minLng, minLat, // 左下
		maxLng, minLat, // 右下
		maxLng, maxLat, // 右上
		minLng, maxLat, // 左上
		minLng, minLat) // 閉じる

	// PostGIS ST_Intersects関数を使用して境界ボックス内のwalksを検索
	var walks []model.Walk
	data, count, err := r.client.GetClient().From("walks").
		Select("id,title,area,description,duration_minutes,distance_meters,tags,route_polyline,created_at,start_location,end_location", "exact", false).
		Filter("route_bounds", "st_intersects", fmt.Sprintf("ST_GeomFromText('%s', 4326)", bbox)).
		Execute()

	if err != nil {
		return nil, fmt.Errorf("境界ボックス検索エラー: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &walks); err != nil {
		return nil, fmt.Errorf("散歩データのJSONアンマーシャル失敗: %w", err)
	}

	// Walk から WalkSummary に変換
	var summaries []model.WalkSummary
	for _, walk := range walks {
		summary := model.WalkSummary{
			ID:              walk.ID,
			Title:           walk.Title,
			AreaName:        walk.Area,
			Date:            walk.CreatedAt.Format("2006年1月2日"),
			Summary:         walk.Description,
			DurationMinutes: walk.DurationMinutes,
			DistanceMeters:  walk.DistanceMeters,
			Tags:            walk.Tags,
			StartLocation:   walk.StartLocation,
			EndLocation:     walk.EndLocation,
			RoutePolyline:   walk.RoutePolyline,
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (r *SupabaseWalksRepository) GetWalkDetail(ctx context.Context, id string) (*model.WalkDetail, error) {
	walk, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Walk から WalkDetail に変換
	detail := &model.WalkDetail{
		ID:              walk.ID,
		Title:           walk.Title,
		AreaName:        walk.Area,
		Date:            walk.CreatedAt.Format("2006年1月2日"),
		Description:     walk.Description,
		Theme:           walk.Theme,
		DurationMinutes: walk.DurationMinutes,
		DistanceMeters:  walk.DistanceMeters,
		RoutePolyline:   walk.RoutePolyline,
		Tags:            walk.Tags,
		// NavigationStepsは別途POIデータから生成する必要があります
		NavigationSteps: []model.NavigationStep{},
	}

	return detail, nil
}

func (r *SupabaseWalksRepository) Delete(ctx context.Context, id string) error {
	_, _, err := r.client.GetClient().From("walks").Delete("", "").Eq("id", id).Execute()
	if err != nil {
		return fmt.Errorf("散歩データの削除失敗: %w", err)
	}

	return nil
}

func (r *SupabaseWalksRepository) GetAll(ctx context.Context) ([]model.Walk, error) {
	var walks []model.Walk
	data, count, err := r.client.GetClient().From("walks").Select("*", "exact", false).Execute()
	if err != nil {
		return nil, fmt.Errorf("全散歩データの取得失敗: %w", err)
	}
	_ = count

	if err := json.Unmarshal([]byte(data), &walks); err != nil {
		return nil, fmt.Errorf("散歩データのJSONアンマーシャル失敗: %w", err)
	}

	return walks, nil
}
