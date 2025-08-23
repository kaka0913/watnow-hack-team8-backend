package usecase

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/domain/service"
	repoImpl "Team8-App/internal/repository"
	"context"
	"fmt"
	"log"
	"time"
)

type RouteRecalculateUseCase interface {
	// RecalculateRoute は元の提案を基にルートを再計算し、物語も更新する
	RecalculateRoute(ctx context.Context, req *model.RouteRecalculateRequest) (*model.RouteRecalculateResponse, error)

	// GetRecalculatedRoute は指定されたproposal_idの再計算されたルート提案をFirestoreから取得する
	GetRecalculatedRoute(ctx context.Context, proposalID string) (*model.RouteProposal, error)
}

// routeRecalculateUseCaseImpl はRouteRecalculateUseCaseの実装
type routeRecalculateUseCaseImpl struct {
	routeRecalculateService   service.RouteRecalculateService
	firestoreRepo             *repoImpl.FirestoreRouteProposalRepository
	storyGenerationRepository repository.StoryGenerationRepository
}

// NewRouteRecalculateUseCase は新しいRouteRecalculateUseCaseインスタンスを作成
func NewRouteRecalculateUseCase(
	recalcService service.RouteRecalculateService,
	firestoreRepo *repoImpl.FirestoreRouteProposalRepository,
	storyRepo repository.StoryGenerationRepository,
) RouteRecalculateUseCase {
	return &routeRecalculateUseCaseImpl{
		routeRecalculateService:   recalcService,
		firestoreRepo:             firestoreRepo,
		storyGenerationRepository: storyRepo,
	}
}

// RecalculateRoute はルート再計算の主要処理を行う
func (u *routeRecalculateUseCaseImpl) RecalculateRoute(ctx context.Context, req *model.RouteRecalculateRequest) (*model.RouteRecalculateResponse, error) {
	log.Printf("🚀 ルート再計算UseCase開始 (ProposalID: %s)", req.ProposalID)

	// Step 1: 冒険のコンテキストを正確に復元する
	originalProposal, err := u.restoreAdventureContext(ctx, req.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("冒険コンテキスト復元に失敗: %w", err)
	}

	// Step 2: ドメインサービスでルート再計算を実行
	response, err := u.routeRecalculateService.RecalculateRoute(ctx, req, originalProposal)
	if err != nil {
		return nil, fmt.Errorf("ルート再計算に失敗: %w", err)
	}

	// Step 3: 物語を"文脈を維持して"更新する
	updatedTitle, updatedStory, err := u.generateUpdatedStory(ctx, originalProposal, req, response.UpdatedRoute)
	if err != nil {
		log.Printf("⚠️ 物語生成に失敗、元の物語を使用: %v", err)
		updatedStory = originalProposal.GeneratedStory + " 新たな発見が散歩を豊かにしています。"
	}

	// Step 4: レスポンスに物語を設定
	response.UpdatedRoute.GeneratedStory = updatedStory
	response.UpdatedRoute.Title = updatedTitle

	// Step 5: 再計算されたルート提案を元の提案に上書き保存（TTLなし）
	err = u.overwriteRouteProposalInFirestore(ctx, response.UpdatedRoute, originalProposal.Theme, req.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("firestore上書き保存に失敗: %w", err)
	}

	// Step 6: 上書き保存完了をログに記録
	log.Printf("✅ ルート再計算UseCase完了 (上書き保存ProposalID: %s)", req.ProposalID)

	return response, nil
}

// restoreAdventureContext はFirestoreから元の提案を取得してコンテキストを復元
func (u *routeRecalculateUseCaseImpl) restoreAdventureContext(ctx context.Context, proposalID string) (*model.RouteProposal, error) {
	log.Printf("📚 元の提案コンテキスト復元中 (ID: %s)", proposalID)

	originalProposal, err := u.firestoreRepo.GetRouteProposal(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("元の提案が見つからないか有効期限切れです: %w", err)
	}

	log.Printf("✅ 元の提案復元完了: %s (テーマ: %s)", originalProposal.Title, originalProposal.Theme)
	return originalProposal, nil
}

// generateUpdatedStory は文脈を維持して物語を更新
func (u *routeRecalculateUseCaseImpl) generateUpdatedStory(ctx context.Context, originalProposal *model.RouteProposal, req *model.RouteRecalculateRequest, updatedRoute *model.UpdatedRoute) (string, string, error) {
	log.Printf("📝 物語の続きを生成中...")

	// 更新されたルート情報からSuggestedRouteを構築
	var updatedPOIs []*model.POI
	for _, step := range updatedRoute.NavigationSteps {
		if step.Type == "poi" {
			poi := &model.POI{
				ID:   step.POIId,
				Name: step.Name,
				Location: &model.Geometry{
					Type:        "Point",
					Coordinates: []float64{step.Longitude, step.Latitude},
				},
			}
			updatedPOIs = append(updatedPOIs, poi)
		}
	}

	// SuggestedRouteオブジェクトを作成
	suggestedRoute := &model.SuggestedRoute{
		Name:  updatedRoute.Title,
		Spots: updatedPOIs,
		// 他のフィールドも設定可能
	}

	// 既存のStoryGenerationRepositoryを使用して物語とタイトルを生成
	title, story, err := u.storyGenerationRepository.GenerateStoryWithTitle(ctx, suggestedRoute, originalProposal.Theme, req.RealtimeContext)
	if err != nil {
		return "", "", fmt.Errorf("物語生成に失敗: %w", err)
	}

	// タイトルも一緒に更新（後でレスポンスに設定される）
	updatedRoute.Title = title

	log.Printf("✅ 物語の続き生成完了")
	return title, story, nil
}

// GetRecalculatedRoute は指定されたproposal_idの再計算されたルート提案をFirestoreから取得する
func (u *routeRecalculateUseCaseImpl) GetRecalculatedRoute(ctx context.Context, proposalID string) (*model.RouteProposal, error) {
	log.Printf("📖 再計算ルート提案取得開始 (ID: %s)", proposalID)

	proposal, err := u.firestoreRepo.GetRouteProposal(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("再計算ルート提案の取得に失敗: %w", err)
	}

	log.Printf("✅ 再計算ルート提案取得完了 (ID: %s)", proposalID)
	return proposal, nil
}

// overwriteRouteProposalInFirestore は再計算されたルートを元の提案に上書き保存（TTLなし）
func (u *routeRecalculateUseCaseImpl) overwriteRouteProposalInFirestore(ctx context.Context, updatedRoute *model.UpdatedRoute, theme, proposalID string) error {
	log.Printf("💾 再計算ルートを元の提案に上書き保存中... (ID: %s)", proposalID)

	// UpdatedRouteからSuggestedRouteを構築
	suggestedRoute, err := u.convertToSuggestedRoute(updatedRoute)
	if err != nil {
		return fmt.Errorf("SuggestedRoute変換に失敗: %w", err)
	}

	// Firestoreに直接上書き保存（TTLなし）
	err = u.updateRouteProposalInFirestore(ctx, proposalID, suggestedRoute, theme, updatedRoute.Title, updatedRoute.GeneratedStory)
	if err != nil {
		return fmt.Errorf("再計算ルートの上書き保存に失敗: %w", err)
	}

	log.Printf("✅ 再計算ルートを上書き保存完了: %s", proposalID)
	return nil
}

// updateRouteProposalInFirestore は指定されたproposal_idのルート提案を上書き更新する（TTLなし）
func (u *routeRecalculateUseCaseImpl) updateRouteProposalInFirestore(ctx context.Context, proposalID string, suggestedRoute *model.SuggestedRoute, theme, title, story string) error {
	log.Printf("🔄 ルート提案上書き更新開始 (ID: %s)", proposalID)

	// Firestoreに直接アクセスするためのヘルパーメソッド呼び出し
	err := u.firestoreRepo.UpdateRouteProposal(ctx, proposalID, suggestedRoute, theme, title, story)
	if err != nil {
		return fmt.Errorf("firestore上書き更新に失敗: %w", err)
	}

	log.Printf("✅ ルート提案上書き更新完了 (ID: %s)", proposalID)
	return nil
}

// convertToSuggestedRoute はUpdatedRouteをSuggestedRouteに変換する
func (u *routeRecalculateUseCaseImpl) convertToSuggestedRoute(updatedRoute *model.UpdatedRoute) (*model.SuggestedRoute, error) {
	var pois []*model.POI
	for _, step := range updatedRoute.NavigationSteps {
		if step.Type == "poi" {
			poi := &model.POI{
				ID:   step.POIId,
				Name: step.Name,
				Location: &model.Geometry{
					Type:        "Point",
					Coordinates: []float64{step.Longitude, step.Latitude},
				},
			}
			pois = append(pois, poi)
		}
	}

	return &model.SuggestedRoute{
		Name:          updatedRoute.Title,
		Spots:         pois,
		TotalDuration: time.Duration(updatedRoute.EstimatedDurationMinutes) * time.Minute,
		Polyline:      updatedRoute.RoutePolyline,
	}, nil
}
