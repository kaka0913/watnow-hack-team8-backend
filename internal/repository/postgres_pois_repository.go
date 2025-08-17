package repository

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/infrastructure/database"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type PostgresPOIsRepository struct {
	client *database.PostgreSQLClient
}

func NewPostgresPOIsRepository(client *database.PostgreSQLClient) repository.POIsRepository {
	return &PostgresPOIsRepository{
		client: client,
	}
}

// filterSmokingAreas は喫煙所を除外してPOIリストをフィルタリングする
func (r *PostgresPOIsRepository) filterSmokingAreas(pois []*model.POI) []*model.POI {
	var filtered []*model.POI
	for _, poi := range pois {
		if poi != nil && poi.Name != "喫煙所" {
			filtered = append(filtered, poi)
		}
	}
	return filtered
}

// POIResult PostGIS関数の結果を受け取るための構造体
type POIResult struct {
	ID            string
	Name          string
	Location      string
	Categories    string
	GridCellID    int
	Rate          float64
	URL           sql.NullString
	DistanceMeters float64
}

// ToPOI POIResultをmodel.POIに変換
func (pr *POIResult) ToPOI() (*model.POI, error) {
	var location model.Geometry
	if err := json.Unmarshal([]byte(pr.Location), &location); err != nil {
		return nil, fmt.Errorf("location JSONBパースエラー: %w", err)
	}

	var categories []string
	if err := json.Unmarshal([]byte(pr.Categories), &categories); err != nil {
		return nil, fmt.Errorf("categories JSONBパースエラー: %w", err)
	}

	poi := &model.POI{
		ID:         pr.ID,
		Name:       pr.Name,
		Location:   &location,
		Categories: categories,
		GridCellID: pr.GridCellID,
		Rate:       pr.Rate,
	}

	if pr.URL.Valid {
		poi.URL = &pr.URL.String
	}

	return poi, nil
}

func (r *PostgresPOIsRepository) GetByID(ctx context.Context, id string) (*model.POI, error) {
	query := `SELECT id, name, location, categories, grid_cell_id, rate, url FROM pois WHERE id = $1`
	
	row := r.client.DB.QueryRowContext(ctx, query, id)
	
	var result POIResult
	err := row.Scan(&result.ID, &result.Name, &result.Location, &result.Categories, 
		&result.GridCellID, &result.Rate, &result.URL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("POI ID %s が見つかりません", id)
		}
		return nil, fmt.Errorf("POIデータの取得失敗: %w", err)
	}

	return result.ToPOI()
}

func (r *PostgresPOIsRepository) GetByGridCellID(ctx context.Context, gridCellID int) ([]model.POI, error) {
	query := `SELECT id, name, location, categories, grid_cell_id, rate, url FROM pois WHERE grid_cell_id = $1`
	
	rows, err := r.client.DB.QueryContext(ctx, query, gridCellID)
	if err != nil {
		return nil, fmt.Errorf("グリッドセル %d のPOIデータ取得失敗: %w", gridCellID, err)
	}
	defer rows.Close()

	var pois []model.POI
	for rows.Next() {
		var result POIResult
		err := rows.Scan(&result.ID, &result.Name, &result.Location, &result.Categories,
			&result.GridCellID, &result.Rate, &result.URL)
		if err != nil {
			return nil, fmt.Errorf("POIデータスキャンエラー: %w", err)
		}

		poi, err := result.ToPOI()
		if err != nil {
			return nil, err
		}
		pois = append(pois, *poi)
	}

	return pois, nil
}

func (r *PostgresPOIsRepository) GetNearbyPOIs(ctx context.Context, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	// 直接SQLでPostGIS関数を使用した効率的な検索
	query := `
		SELECT 
			p.id, p.name, 
			ST_AsGeoJSON(p.location)::jsonb as location,
			p.categories, p.grid_cell_id, p.rate, p.url,
			ST_Distance(
				ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
				p.location::geography
			) as distance_meters
		FROM pois p
		WHERE ST_DWithin(
			ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
			p.location::geography,
			$3
		)
		ORDER BY distance_meters
		LIMIT 50
	`
	
	rows, err := r.client.DB.QueryContext(ctx, query, lat, lng, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("周辺POI検索失敗: %w", err)
	}
	defer rows.Close()

	var pois []model.POI
	for rows.Next() {
		var result POIResult
		err := rows.Scan(&result.ID, &result.Name, &result.Location, &result.Categories,
			&result.GridCellID, &result.Rate, &result.URL, &result.DistanceMeters)
		if err != nil {
			return nil, fmt.Errorf("POIデータスキャンエラー: %w", err)
		}

		poi, err := result.ToPOI()
		if err != nil {
			return nil, err
		}
		pois = append(pois, *poi)
	}

	return pois, nil
}

func (r *PostgresPOIsRepository) GetByCategories(ctx context.Context, categories []string, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return nil, fmt.Errorf("カテゴリJSONマーシャルエラー: %w", err)
	}

	// 直接SQLクエリで複数カテゴリ検索（他のメソッドと統一）
	query := `
		SELECT 
			p.id, p.name, 
			ST_AsGeoJSON(p.location)::jsonb as location,
			p.categories, p.grid_cell_id, p.rate, p.url,
			ST_Distance(
				ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
				p.location::geography
			) as distance_meters
		FROM pois p
		WHERE ST_DWithin(
			ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
			p.location::geography,
			$4
		)
		AND p.categories @> $3::jsonb
		ORDER BY distance_meters
		LIMIT 50
	`
	
	rows, err := r.client.DB.QueryContext(ctx, query, lat, lng, string(categoriesJSON), radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("カテゴリ別POI検索失敗: %w", err)
	}
	defer rows.Close()

	var pois []model.POI
	for rows.Next() {
		var result POIResult
		err := rows.Scan(&result.ID, &result.Name, &result.Location, &result.Categories,
			&result.GridCellID, &result.Rate, &result.URL, &result.DistanceMeters)
		if err != nil {
			return nil, fmt.Errorf("POIデータスキャンエラー: %w", err)
		}

		poi, err := result.ToPOI()
		if err != nil {
			return nil, err
		}
		pois = append(pois, *poi)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("行イテレーション中のエラー: %w", err)
	}

	// 喫煙所を除外してフィルタリング（[]model.POI から []*model.POI に変換してフィルタリング）
	var poiPtrs []*model.POI
	for i := range pois {
		poiPtrs = append(poiPtrs, &pois[i])
	}
	filtered := r.filterSmokingAreas(poiPtrs)
	
	// 結果を[]model.POIに戻す
	var finalResult []model.POI
	for _, poi := range filtered {
		finalResult = append(finalResult, *poi)
	}

	return finalResult, nil
}

func (r *PostgresPOIsRepository) GetByCategory(ctx context.Context, category string, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	query := `
		SELECT 
			p.id, p.name, 
			ST_AsGeoJSON(p.location)::jsonb as location,
			p.categories, p.grid_cell_id, p.rate, p.url,
			ST_Distance(
				ST_GeogFromText('POINT(' || $3 || ' ' || $2 || ')'),
				p.location::geography
			) as distance_meters
		FROM pois p
		WHERE ST_DWithin(
			ST_GeogFromText('POINT(' || $3 || ' ' || $2 || ')'),
			p.location::geography,
			$4
		)
		AND p.categories ? $1
		ORDER BY distance_meters
		LIMIT 50
	`
	
	rows, err := r.client.DB.QueryContext(ctx, query, category, lat, lng, radiusMeters)
	if err != nil {
		return nil, fmt.Errorf("単一カテゴリPOI検索失敗: %w", err)
	}
	defer rows.Close()

	var pois []model.POI
	for rows.Next() {
		var result POIResult
		err := rows.Scan(&result.ID, &result.Name, &result.Location, &result.Categories,
			&result.GridCellID, &result.Rate, &result.URL, &result.DistanceMeters)
		if err != nil {
			return nil, fmt.Errorf("POIデータスキャンエラー: %w", err)
		}

		poi, err := result.ToPOI()
		if err != nil {
			return nil, err
		}
		pois = append(pois, *poi)
	}

	return pois, nil
}

func (r *PostgresPOIsRepository) GetByRatingRange(ctx context.Context, minRating float64, lat, lng float64, radiusMeters int) ([]model.POI, error) {
	// 評価値フィルタリング付きの周辺POI検索（PostGIS使用）
	query := `
		SELECT 
			p.id, p.name, p.location, p.categories, p.grid_cell_id, p.rate, p.url,
			ST_Distance(
				ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
				ST_GeogFromText('POINT(' || (p.location->>'coordinates')::jsonb->>0 || ' ' || (p.location->>'coordinates')::jsonb->>1 || ')')
			) as distance_meters
		FROM pois p
		WHERE ST_DWithin(
			ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
			ST_GeogFromText('POINT(' || (p.location->>'coordinates')::jsonb->>0 || ' ' || (p.location->>'coordinates')::jsonb->>1 || ')'),
			$3
		)
		AND p.rate >= $4
		ORDER BY distance_meters
		LIMIT 50
	`
	
	rows, err := r.client.DB.QueryContext(ctx, query, lat, lng, radiusMeters, minRating)
	if err != nil {
		return nil, fmt.Errorf("評価値別POI検索失敗: %w", err)
	}
	defer rows.Close()

	var pois []model.POI
	for rows.Next() {
		var result POIResult
		err := rows.Scan(&result.ID, &result.Name, &result.Location, &result.Categories,
			&result.GridCellID, &result.Rate, &result.URL, &result.DistanceMeters)
		if err != nil {
			return nil, fmt.Errorf("POIデータスキャンエラー: %w", err)
		}

		poi, err := result.ToPOI()
		if err != nil {
			return nil, err
		}
		pois = append(pois, *poi)
	}

	return pois, nil
}

// FindNearbyByCategories ルート提案用のメソッド：カテゴリと位置に基づいてPOIを検索
func (r *PostgresPOIsRepository) FindNearbyByCategories(ctx context.Context, location model.LatLng, categories []string, radiusMeters int, limit int) ([]*model.POI, error) {
	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return nil, fmt.Errorf("カテゴリJSONマーシャルエラー: %w", err)
	}

	query := `
		SELECT 
			p.id, p.name, 
			ST_AsGeoJSON(p.location)::jsonb as location,
			p.categories, p.grid_cell_id, p.rate, p.url,
			ST_Distance(
				ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
				p.location::geography
			) as distance_meters
		FROM pois p
		WHERE ST_DWithin(
			ST_GeogFromText('POINT(' || $2 || ' ' || $1 || ')'),
			p.location::geography,
			$4
		)
		AND p.categories @> $3::jsonb
		ORDER BY distance_meters
		LIMIT $5
	`
	
	rows, err := r.client.DB.QueryContext(ctx, query, location.Lat, location.Lng, string(categoriesJSON), radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("周辺カテゴリ別POI検索失敗: %w", err)
	}
	defer rows.Close()

	var result []*model.POI
	for rows.Next() {
		var poiResult POIResult
		err := rows.Scan(&poiResult.ID, &poiResult.Name, &poiResult.Location, &poiResult.Categories,
			&poiResult.GridCellID, &poiResult.Rate, &poiResult.URL, &poiResult.DistanceMeters)
		if err != nil {
			return nil, fmt.Errorf("POIデータスキャンエラー: %w", err)
		}

		poi, err := poiResult.ToPOI()
		if err != nil {
			return nil, err
		}
		result = append(result, poi)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("行イテレーション中のエラー: %w", err)
	}

	// 喫煙所を除外してフィルタリング
	result = r.filterSmokingAreas(result)

	return result, nil
}

// FindNearbyByCategoriesIncludingHorror ホラースポットを含めてPOIを検索
func (r *PostgresPOIsRepository) FindNearbyByCategoriesIncludingHorror(ctx context.Context, location model.LatLng, categories []string, radiusMeters int, limit int) ([]*model.POI, error) {
	// ホラースポット用は同じ実装
	return r.FindNearbyByCategories(ctx, location, categories, radiusMeters, limit)
}
