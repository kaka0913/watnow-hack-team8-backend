package handler

import (
	"Team8-App/internal/domain/model"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// PostRouteRecalculate はルート再計算を行うエンドポイント
// POST /routes/recalculate
func (h *RouteProposalHandler) PostRouteRecalculate(c *gin.Context) {
	var req model.RouteRecalculateRequest

	// リクエストボディのバインド
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "リクエストの形式が正しくありません",
			"details": err.Error(),
		})
		return
	}

	// バリデーション
	if err := h.validateRecalculateRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "バリデーションエラー",
			"details": err.Error(),
		})
		return
	}

	// UseCase呼び出し
	response, err := h.recalculateUseCase.RecalculateRoute(c.Request.Context(), &req)
	if err != nil {
		// エラーメッセージから404か500かを判定
		if strings.Contains(err.Error(), "見つかりません") || strings.Contains(err.Error(), "有効期限切れ") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "元のルート提案が見つかりません",
				"details": err.Error(),
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "ルート再計算に失敗しました",
				"details": err.Error(),
			})
		}
		return
	}

	// 成功レスポンス
	c.JSON(http.StatusOK, response)
}

// validateRecalculateRequest はルート再計算リクエストのバリデーションを行う
func (h *RouteProposalHandler) validateRecalculateRequest(req *model.RouteRecalculateRequest) error {
	// ProposalIDは必須
	if req.ProposalID == "" {
		return &ValidationError{Field: "proposal_id", Message: "元の提案IDは必須です"}
	}

	// CurrentLocationは必須
	if req.CurrentLocation == nil {
		return &ValidationError{Field: "current_location", Message: "現在地は必須です"}
	}

	// 緯度経度の範囲チェック
	if req.CurrentLocation.Latitude < -90 || req.CurrentLocation.Latitude > 90 {
		return &ValidationError{Field: "current_location.latitude", Message: "緯度は-90から90の範囲で指定してください"}
	}
	if req.CurrentLocation.Longitude < -180 || req.CurrentLocation.Longitude > 180 {
		return &ValidationError{Field: "current_location.longitude", Message: "経度は-180から180の範囲で指定してください"}
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

	// VisitedPOIsは必須
	if req.VisitedPOIs == nil {
		return &ValidationError{Field: "visited_pois", Message: "訪問済みPOI情報は必須です"}
	}

	// 訪問済みPOIリストの検証
	if len(req.VisitedPOIs.PreviousPOIs) == 0 {
		return &ValidationError{Field: "visited_pois.previous_pois", Message: "訪問済みPOIが1つも指定されていません"}
	}

	for i, poi := range req.VisitedPOIs.PreviousPOIs {
		if poi.Name == "" {
			return &ValidationError{Field: fmt.Sprintf("visited_pois.previous_pois[%d].name", i), Message: "POI名は必須です"}
		}
		if poi.POIId == "" {
			return &ValidationError{Field: fmt.Sprintf("visited_pois.previous_pois[%d].poi_id", i), Message: "POI IDは必須です"}
		}
	}

	return nil
}
