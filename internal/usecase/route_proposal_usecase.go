package usecase

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"Team8-App/internal/domain/service"
	repoImpl "Team8-App/internal/repository"
	"context"
	"fmt"
	"log"
	"sync"
)

type RouteProposalUseCase interface {
	// GenerateProposals はリクエストに基づいてルート提案を生成し、Firestoreに保存してレスポンスを返す
	GenerateProposals(ctx context.Context, req *model.RouteProposalRequest) (*model.RouteProposalResponse, error)
	
	// GetRouteProposal は指定されたproposal_idのルート提案をFirestoreから取得する
	GetRouteProposal(ctx context.Context, proposalID string) (*model.RouteProposal, error)
}

// routeProposalUseCaseImpl はRouteProposalUseCaseの実装
type routeProposalUseCaseImpl struct {
	routeSuggestionService    service.RouteSuggestionService
	firestoreRepo             *repoImpl.FirestoreRouteProposalRepository
	storyGenerationRepository repository.StoryGenerationRepository
}

// NewRouteProposalUseCase は新しいRouteProposalUseCaseインスタンスを作成
func NewRouteProposalUseCase(
	routeService service.RouteSuggestionService,
	firestoreRepo *repoImpl.FirestoreRouteProposalRepository,
	storyRepo repository.StoryGenerationRepository,
) RouteProposalUseCase {
	return &routeProposalUseCaseImpl{
		routeSuggestionService:    routeService,
		firestoreRepo:             firestoreRepo,
		storyGenerationRepository: storyRepo,
	}
}

// GenerateProposals はリクエストに基づいてルート提案を生成し、Firestoreに保存してレスポンスを返す
func (u *routeProposalUseCaseImpl) GenerateProposals(ctx context.Context, req *model.RouteProposalRequest) (*model.RouteProposalResponse, error) {
	log.Printf("🚀 ルート提案生成開始 (テーマ: %s, モード: %s)", req.Theme, req.Mode)

	// Step 1: ルート候補を生成
	suggestionReq := &model.SuggestionRequest{
		StartLocation:       req.StartLocation,
		DestinationLocation: req.DestinationLocation,
		Mode:                req.Mode,
		TimeMinutes:         req.TimeMinutes,
		Theme:               req.Theme,
		Scenarios:           []string{}, // デフォルトシナリオを使用している
		RealtimeContext:     req.RealtimeContext,
	}

	suggestedRoutes, err := u.routeSuggestionService.SuggestRoutes(ctx, suggestionReq)
	if err != nil {
		return nil, fmt.Errorf("ルート生成に失敗: %w", err)
	}

	if len(suggestedRoutes) == 0 {
		return &model.RouteProposalResponse{
			Proposals: []model.RouteProposal{},
		}, nil
	}

	log.Printf("✅ %d件のルート候補を生成", len(suggestedRoutes))

	// Step 2: 各ルートに対してタイトルと物語を並行生成
	log.Printf("🤖 Gemini APIでタイトル・物語を並行生成中...")
	
	type storyResult struct {
		index int
		title string
		story string
		err   error
	}

	resultChan := make(chan storyResult, len(suggestedRoutes))
	var wg sync.WaitGroup

	// 各ルートに対して並行でタイトル・物語生成
	for i, route := range suggestedRoutes {
		wg.Add(1)
		go func(idx int, r *model.SuggestedRoute) {
			defer wg.Done()
			title, story, err := u.storyGenerationRepository.GenerateStoryWithTitle(ctx, r, req.Theme, req.RealtimeContext)
			resultChan <- storyResult{
				index: idx,
				title: title,
				story: story,
				err:   err,
			}
		}(i, route)
	}

	// 別のgoroutineでwaitしてチャンネルを閉じる
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 結果を収集
	titles := make([]string, len(suggestedRoutes))
	stories := make([]string, len(suggestedRoutes))
	
	for result := range resultChan {
		if result.err != nil {
			log.Printf("⚠️ ルート%d のタイトル・物語生成に失敗、フォールバック使用: %v", result.index+1, result.err)
			titles[result.index] = suggestedRoutes[result.index].Name
			stories[result.index] = fmt.Sprintf("%sの素晴らしい散歩をお楽しみください。新しい発見があなたを待っています。", suggestedRoutes[result.index].Name)
		} else {
			titles[result.index] = result.title
			stories[result.index] = result.story
		}
		log.Printf("✅ ルート%d: タイトル「%s」物語生成完了", result.index+1, titles[result.index])
	}

	// Step 3: Firestoreに保存
	log.Printf("💾 Firestore保存中...")
	savedProposals, err := u.firestoreRepo.SaveRouteProposalsWithStory(ctx, suggestedRoutes, req.Theme, 2, titles, stories) // 2時間TTL
	if err != nil {
		return nil, fmt.Errorf("Firestore保存に失敗: %w", err)
	}

	log.Printf("🎉 ルート提案生成完了 (%d件)", len(savedProposals))

	return &model.RouteProposalResponse{
		Proposals: u.convertToSlice(savedProposals),
	}, nil
}

// GetRouteProposal は指定されたproposal_idのルート提案をFirestoreから取得する
func (u *routeProposalUseCaseImpl) GetRouteProposal(ctx context.Context, proposalID string) (*model.RouteProposal, error) {
	log.Printf("📖 ルート提案取得開始 (ID: %s)", proposalID)
	
	proposal, err := u.firestoreRepo.GetRouteProposal(ctx, proposalID)
	if err != nil {
		return nil, fmt.Errorf("ルート提案の取得に失敗: %w", err)
	}
	
	log.Printf("✅ ルート提案取得完了 (ID: %s)", proposalID)
	return proposal, nil
}

// convertToSlice は[]*RouteProposalを[]RouteProposalに変換
func (u *routeProposalUseCaseImpl) convertToSlice(proposals []*model.RouteProposal) []model.RouteProposal {
	result := make([]model.RouteProposal, len(proposals))
	for i, proposal := range proposals {
		if proposal != nil {
			result[i] = *proposal
		}
	}
	return result
}
