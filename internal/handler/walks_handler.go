package handler

import (
	"net/http"

	"Team8-App/internal/usecase"
	"Team8-App/internal/domain/model"
	"Team8-App/internal/repository"

	"github.com/gin-gonic/gin"
)

// WalksHandler 散歩記録に関するHTTPハンドラー
type WalksHandler struct {
	walksUsecase usecase.WalksUsecase
	firestoreRepo *repository.FirestoreRouteProposalRepository
}

// NewWalksHandler WalksHandlerの新しいインスタンスを作成
func NewWalksHandler(walksUsecase usecase.WalksUsecase, firestoreRepo *repository.FirestoreRouteProposalRepository) *WalksHandler {
	return &WalksHandler{
		walksUsecase:  walksUsecase,
		firestoreRepo: firestoreRepo,
	}
}

// CreateWalk POST /walks - 散歩記録の作成
func (h *WalksHandler) CreateWalk(c *gin.Context) {
	var req model.CreateWalkRequest

	// リクエストボディの解析（Ginが自動でContent-Type確認）
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "Invalid JSON format: " + err.Error(),
		})
		return
	}

	// ユースケース層で処理
	response, err := h.walksUsecase.CreateWalk(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create walk: " + err.Error(),
		})
		return
	}

	// 成功レスポンス
	c.JSON(http.StatusCreated, response)
}

// GetWalks GET /walks - Firestoreから全てのルート提案を取得
func (h *WalksHandler) GetWalks(c *gin.Context) {
	// Firestoreから全てのルート提案を取得
	routeProposals, err := h.firestoreRepo.GetAllRouteProposals(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get route proposals: " + err.Error(),
		})
		return
	}

	walks := make([]model.WalkSummary, len(routeProposals))
	for i, proposal := range routeProposals {
		walks[i] = model.WalkSummary{
			ID:              proposal.ProposalID,
			Title:           proposal.Title,
			AreaName:        "京都市", // デフォルト値（必要に応じて動的に設定）
			Date:            "", // 日付情報がない場合は空文字
			Summary:         proposal.GeneratedStory, // 生成された物語を要約として使用
			DurationMinutes: proposal.EstimatedDurationMinutes,
			DistanceMeters:  proposal.EstimatedDistanceMeters,
			Tags:            []string{proposal.Theme}, // テーマをタグとして使用
			StartLocation:   nil, // 開始位置は不明
			EndLocation:     nil, // 終了位置は不明
			RoutePolyline:   proposal.RoutePolyline,
		}
	}

	// レスポンスの作成
	response := model.GetWalksResponse{
		Walks: walks,
	}

	c.JSON(http.StatusOK, response)
}

// GetWalkDetail GET /walks/:id - 散歩記録の詳細を取得
func (h *WalksHandler) GetWalkDetail(c *gin.Context) {
	// パスパラメータから ID を取得
	walkID := c.Param("id")
	if walkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_parameter",
			"message": "Walk ID is required",
		})
		return
	}

	// ユースケース層で処理
	walkDetail, err := h.walksUsecase.GetWalkDetail(c.Request.Context(), walkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get walk detail: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, walkDetail)
}
