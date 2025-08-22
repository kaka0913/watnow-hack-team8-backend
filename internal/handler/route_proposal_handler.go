package handler

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/usecase"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RouteProposalHandler はルート提案APIのハンドラー
type RouteProposalHandler struct {
	proposalUseCase     usecase.RouteProposalUseCase
	recalculateUseCase  usecase.RouteRecalculateUseCase
}

// NewRouteProposalHandler は新しいRouteProposalHandlerインスタンスを作成
func NewRouteProposalHandler(proposalUseCase usecase.RouteProposalUseCase, recalculateUseCase usecase.RouteRecalculateUseCase) *RouteProposalHandler {
	return &RouteProposalHandler{
		proposalUseCase:    proposalUseCase,
		recalculateUseCase: recalculateUseCase,
	}
}

// PostRouteProposals はルート提案を生成するエンドポイント
// POST /routes/proposals
func (h *RouteProposalHandler) PostRouteProposals(c *gin.Context) {
	var req model.RouteProposalRequest

	// リクエストボディのバインド
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "リクエストの形式が正しくありません",
			"details": err.Error(),
		})
		return
	}

	// バリデーション
	if err := h.validateRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "バリデーションエラー",
			"details": err.Error(),
		})
		return
	}

	// UseCase呼び出し
	response, err := h.proposalUseCase.GenerateProposals(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "ルート提案の生成に失敗しました",
			"details": err.Error(),
		})
		return
	}

	// 成功レスポンス
	c.JSON(http.StatusOK, response)
}

// validateRequest はリクエストの詳細バリデーションを行う
func (h *RouteProposalHandler) validateRequest(req *model.RouteProposalRequest) error {
	// StartLocationは必須
	if req.StartLocation == nil {
		return &ValidationError{Field: "start_location", Message: "開始地点は必須です"}
	}

	// 緯度経度の範囲チェック
	if req.StartLocation.Latitude < -90 || req.StartLocation.Latitude > 90 {
		return &ValidationError{Field: "start_location.latitude", Message: "緯度は-90から90の範囲で指定してください"}
	}
	if req.StartLocation.Longitude < -180 || req.StartLocation.Longitude > 180 {
		return &ValidationError{Field: "start_location.longitude", Message: "経度は-180から180の範囲で指定してください"}
	}

	// 目的地が指定されている場合の緯度経度チェック
	if req.DestinationLocation != nil {
		if req.DestinationLocation.Latitude < -90 || req.DestinationLocation.Latitude > 90 {
			return &ValidationError{Field: "destination_location.latitude", Message: "緯度は-90から90の範囲で指定してください"}
		}
		if req.DestinationLocation.Longitude < -180 || req.DestinationLocation.Longitude > 180 {
			return &ValidationError{Field: "destination_location.longitude", Message: "経度は-180から180の範囲で指定してください"}
		}
	}

	// モードのチェック
	if req.Mode != "destination" && req.Mode != "time_based" {
		return &ValidationError{Field: "mode", Message: "modeは'destination'または'time_based'を指定してください"}
	}

	// time_basedモードの場合、TimeMinutesが必須
	if req.Mode == "time_based" && req.TimeMinutes <= 0 {
		return &ValidationError{Field: "time_minutes", Message: "time_basedモードでは正の整数のtime_minutesが必要です"}
	}

	// テーマのチェック
	if req.Theme == "" {
		return &ValidationError{Field: "theme", Message: "テーマは必須です"}
	}

	return nil
}

// ValidationError はバリデーションエラーを表す
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// GetRouteProposal は特定のルート提案を取得するエンドポイント
// GET /routes/proposals/:id
func (h *RouteProposalHandler) GetRouteProposal(c *gin.Context) {
	proposalID := c.Param("id")
	if proposalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "proposal_idが指定されていません",
		})
		return
	}

	// UseCaseから取得
	proposal, err := h.proposalUseCase.GetRouteProposal(c.Request.Context(), proposalID)
	if err != nil {
		// エラーメッセージから404か500かを判定
		if strings.Contains(err.Error(), "見つかりません") || strings.Contains(err.Error(), "有効期限切れ") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "ルート提案が見つかりません",
				"details": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "ルート提案の取得に失敗しました",
				"details": err.Error(),
			})
		}
		return
	}

	// 成功レスポンス
	c.JSON(http.StatusOK, proposal)
}
