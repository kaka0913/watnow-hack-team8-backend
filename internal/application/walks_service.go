package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"Team8-App/internal/domain/repository"
	"Team8-App/internal/domain/model"
)

// WalksService 散歩記録に関するビジネスロジックを提供するサービス
type WalksService interface {
	// CreateWalk 散歩記録を新規作成
	CreateWalk(ctx context.Context, req *model.CreateWalkRequest) (*model.CreateWalkResponse, error)

	// GetWalksByBoundingBox 境界ボックス内の散歩記録一覧を取得
	GetWalksByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.WalkSummary, error)

	// GetWalkDetail 散歩記録の詳細を取得
	GetWalkDetail(ctx context.Context, id string) (*model.WalkDetail, error)
}

// walksServiceImpl WalksServiceの実装
type walksServiceImpl struct {
	walksRepo repository.WalksRepository
}

// NewWalksService WalksServiceの新しいインスタンスを作成
func NewWalksService(walksRepo repository.WalksRepository) WalksService {
	return &walksServiceImpl{
		walksRepo: walksRepo,
	}
}

// CreateWalk 散歩記録を作成
func (s *walksServiceImpl) CreateWalk(ctx context.Context, req *model.CreateWalkRequest) (*model.CreateWalkResponse, error) {
	// 入力バリデーション
	if err := s.validateCreateWalkRequest(req); err != nil {
		return nil, fmt.Errorf("リクエストの検証失敗: %w", err)
	}

	// UUIDを生成
	walkID := uuid.New().String()

	// 訪問したPOIのIDを抽出
	poiIDs := make([]string, len(req.VisitedPOIs))
	for i, poi := range req.VisitedPOIs {
		poiIDs[i] = poi.POIId
	}

	// エリア名を推定
	areaName := s.estimateAreaName(req.StartLocation)

	// 終了位置を推定（最後に訪問したPOIまたは開始位置）
	endLocation := req.StartLocation // デフォルトは開始位置
	if len(req.VisitedPOIs) > 0 {
		lastPOI := req.VisitedPOIs[len(req.VisitedPOIs)-1]
		endLocation = &model.Location{
			Latitude:  lastPOI.Latitude,
			Longitude: lastPOI.Longitude,
		}
	}

	// Walkモデルを作成
	walk := &model.Walk{
		ID:              walkID,
		Title:           req.Title,
		Area:            areaName,
		Description:     req.Description,
		Theme:           req.Theme,
		POIIds:          poiIDs,
		Tags:            s.extractTagsFromTheme(req.Theme), // テーマからタグを生成
		DurationMinutes: req.ActualDurationMins,
		DistanceMeters:  req.ActualDistanceMs,
		RoutePolyline:   req.RoutePolyline,
		Impressions:     req.Impressions,
		StartLocation:   req.StartLocation, // 開始位置を設定
		EndLocation:     endLocation,       // 終了位置を設定
	}

	// データベースに保存
	if err := s.walksRepo.Create(ctx, walk); err != nil {
		return nil, fmt.Errorf("散歩記録の保存失敗: %w", err)
	}

	// レスポンスを作成
	response := &model.CreateWalkResponse{
		Status: "success",
	}

	return response, nil
}

// GetWalksByBoundingBox 境界ボックス内の散歩記録一覧を取得
func (s *walksServiceImpl) GetWalksByBoundingBox(ctx context.Context, minLng, minLat, maxLng, maxLat float64) ([]model.WalkSummary, error) {
	// 境界ボックスのバリデーション
	if err := s.validateBoundingBox(minLng, minLat, maxLng, maxLat); err != nil {
		return nil, fmt.Errorf("境界ボックスの検証失敗: %w", err)
	}

	// リポジトリから散歩記録を取得（高速境界ボックス検索）
	walks, err := s.walksRepo.GetWalksByBoundingBox(ctx, minLng, minLat, maxLng, maxLat)
	if err != nil {
		return nil, fmt.Errorf("散歩記録の取得失敗: %w", err)
	}

	return walks, nil
}

// GetWalkDetail 散歩記録の詳細を取得
func (s *walksServiceImpl) GetWalkDetail(ctx context.Context, id string) (*model.WalkDetail, error) {
	// IDの形式チェック
	if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("無効なWalk ID形式: %s", id)
	}

	// リポジトリから詳細データを取得
	walkDetail, err := s.walksRepo.GetWalkDetail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("散歩記録詳細の取得失敗: %w", err)
	}

	return walkDetail, nil
}

// validateCreateWalkRequest リクエストのバリデーション
func (s *walksServiceImpl) validateCreateWalkRequest(req *model.CreateWalkRequest) error {
	if req.Title == "" {
		return fmt.Errorf("タイトルは必須です")
	}
	if req.Description == "" {
		return fmt.Errorf("説明は必須です")
	}
	if req.Theme == "" {
		return fmt.Errorf("テーマは必須です")
	}
	if req.ActualDurationMins <= 0 {
		return fmt.Errorf("実績時間は1分以上である必要があります")
	}
	if req.ActualDistanceMs <= 0 {
		return fmt.Errorf("実績距離は1メートル以上である必要があります")
	}
	if req.RoutePolyline == "" {
		return fmt.Errorf("ルートポリラインは必須です")
	}
	if req.StartLocation == nil {
		return fmt.Errorf("開始位置は必須です")
	}
	return nil
}

// validateBoundingBox 境界ボックスのバリデーション
func (s *walksServiceImpl) validateBoundingBox(minLng, minLat, maxLng, maxLat float64) error {
	if minLng >= maxLng {
		return fmt.Errorf("経度の最小値は最大値より小さい必要があります")
	}
	if minLat >= maxLat {
		return fmt.Errorf("緯度の最小値は最大値より小さい必要があります")
	}
	if minLng < -180 || maxLng > 180 {
		return fmt.Errorf("経度は-180から180の範囲内である必要があります")
	}
	if minLat < -90 || maxLat > 90 {
		return fmt.Errorf("緯度は-90から90の範囲内である必要があります")
	}
	return nil
}

// estimateAreaName 位置情報からエリア名を推定
func (s *walksServiceImpl) estimateAreaName(location *model.Location) string {
	// TODO: 位置情報からエリア名を推定するロジックを実装する
	if location == nil {
		return "未知のエリア"
	}

	// 簡易的な地域判定（実際のアプリでは逆ジオコーディングAPIを使用）
	lat, lng := location.Latitude, location.Longitude

	// 大阪エリアの判定
	if lat >= 34.6 && lat <= 34.8 && lng >= 135.4 && lng <= 135.6 {
		return "大阪・梅田エリア"
	}

	// 東京エリアの判定
	if lat >= 35.6 && lat <= 35.7 && lng >= 139.6 && lng <= 139.8 {
		return "東京都心エリア"
	}

	return "その他エリア"
}

// extractTagsFromTheme テーマからタグを生成
func (s *walksServiceImpl) extractTagsFromTheme(theme string) []string {
	// TODO: デフォルトを返すのではなくてテーマからタグを生成するロジックを実装する
	tagMap := map[string][]string{
		"gourmet":       {"グルメ", "食べ歩き", "レストラン"},
		"culture":       {"文化", "歴史", "アート"},
		"nature":        {"自然", "公園", "癒し"},
		"shopping":      {"ショッピング", "買い物", "ファッション"},
		"architecture":  {"建築", "モダン", "デザイン"},
		"entertainment": {"エンタメ", "観光", "体験"},
	}

	if tags, exists := tagMap[theme]; exists {
		return tags
	}

	return []string{theme} // デフォルトはテーマ名をそのままタグとして使用
}
