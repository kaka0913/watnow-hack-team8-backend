package handler

import (
	"net/http"
	"strconv"
	"strings"

	"Team8-App/internal/service"
	"Team8-App/model"

	"github.com/gin-gonic/gin"
)

// WalksHandler 散歩記録に関するHTTPハンドラー
type WalksHandler struct {
	walksService service.WalksService
}

// NewWalksHandler WalksHandlerの新しいインスタンスを作成
func NewWalksHandler(walksService service.WalksService) *WalksHandler {
	return &WalksHandler{
		walksService: walksService,
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

	// サービス層で処理
	response, err := h.walksService.CreateWalk(c.Request.Context(), &req)
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

// GetWalksByBoundingBox GET /walks - 境界ボックス内の散歩記録一覧を取得
func (h *WalksHandler) GetWalksByBoundingBox(c *gin.Context) {
	// クエリパラメータの解析
	bbox := c.Query("bbox")
	if bbox == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "missing_parameter",
			"message": "bbox parameter is required (format: min_lng,min_lat,max_lng,max_lat)",
		})
		return
	}

	// bbox の解析
	coords := strings.Split(bbox, ",")
	if len(coords) != 4 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameter",
			"message": "bbox must contain 4 coordinates: min_lng,min_lat,max_lng,max_lat",
		})
		return
	}

	minLng, err := strconv.ParseFloat(coords[0], 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameter",
			"message": "Invalid min_lng value",
		})
		return
	}

	minLat, err := strconv.ParseFloat(coords[1], 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameter",
			"message": "Invalid min_lat value",
		})
		return
	}

	maxLng, err := strconv.ParseFloat(coords[2], 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameter",
			"message": "Invalid max_lng value",
		})
		return
	}

	maxLat, err := strconv.ParseFloat(coords[3], 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_parameter",
			"message": "Invalid max_lat value",
		})
		return
	}

	// サービス層で処理
	walks, err := h.walksService.GetWalksByBoundingBox(c.Request.Context(), minLng, minLat, maxLng, maxLat)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get walks: " + err.Error(),
		})
		return
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

	// サービス層で処理
	walkDetail, err := h.walksService.GetWalkDetail(c.Request.Context(), walkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to get walk detail: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, walkDetail)
}
