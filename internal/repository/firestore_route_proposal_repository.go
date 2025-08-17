package repository

import (
	"Team8-App/internal/domain/model"
	"context"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"
)

// FirestoreRouteProposalRepository Firestoreを使用したルート提案キャッシュリポジトリ
type FirestoreRouteProposalRepository struct {
	client *firestore.Client
}

// NewFirestoreRouteProposalRepository 新しいFirestoreRouteProposalRepositoryインスタンスを作成
func NewFirestoreRouteProposalRepository(client *firestore.Client) *FirestoreRouteProposalRepository {
	return &FirestoreRouteProposalRepository{
		client: client,
	}
}

// SaveRouteProposals は複数のルート提案をFirestoreに保存し、proposal_idを生成して返す
func (r *FirestoreRouteProposalRepository) SaveRouteProposals(ctx context.Context, proposals []*model.SuggestedRoute, theme string, ttlHours int) ([]*model.RouteProposal, error) {
	var result []*model.RouteProposal

	collection := r.client.Collection("routeProposals")

	for i, suggestedRoute := range proposals {
		// 一時IDを生成
		proposalID := fmt.Sprintf("temp_prop_%s", uuid.New().String())

		// SuggestedRouteをRouteProposalに変換
		routeProposal := &model.RouteProposal{
			ProposalID:               proposalID,
			Title:                    suggestedRoute.Name,
			EstimatedDurationMinutes: int(suggestedRoute.TotalDuration.Minutes()),
			EstimatedDistanceMeters:  0, // SuggestedRouteには距離情報がないため0とする
			Theme:                    theme,
			DisplayHighlights:        r.extractHighlights(suggestedRoute),
			NavigationSteps:          r.convertToNavigationSteps(suggestedRoute),
			RoutePolyline:            suggestedRoute.Polyline,
			GeneratedStory:           r.generateStoryFromRoute(suggestedRoute, i),
		}

		// Firestore用の構造体に変換
		firestoreData := routeProposal.ToFirestoreRouteProposal(ttlHours)

		// Firestoreに保存
		_, err := collection.Doc(proposalID).Set(ctx, firestoreData)
		if err != nil {
			log.Printf("❌ Failed to save route proposal %s: %v", proposalID, err)
			return nil, fmt.Errorf("ルート提案の保存に失敗しました: %w", err)
		}

		log.Printf("✅ Route proposal saved: %s (expires in %d hours)", proposalID, ttlHours)
		result = append(result, routeProposal)
	}

	return result, nil
}

// GetRouteProposal は指定されたproposal_idのルート提案をFirestoreから取得する
func (r *FirestoreRouteProposalRepository) GetRouteProposal(ctx context.Context, proposalID string) (*model.RouteProposal, error) {
	doc, err := r.client.Collection("routeProposals").Doc(proposalID).Get(ctx)
	if err != nil {
		// Firestoreのエラータイプをチェック
		if status := err.Error(); strings.Contains(status, "NotFound") || strings.Contains(status, "not found") {
			return nil, fmt.Errorf("ルート提案が見つかりません（有効期限切れまたは無効なID）: %s", proposalID)
		}
		return nil, fmt.Errorf("ルート提案の取得に失敗しました: %w", err)
	}

	var firestoreData model.FirestoreRouteProposal
	if err := doc.DataTo(&firestoreData); err != nil {
		return nil, fmt.Errorf("データの変換に失敗しました: %w", err)
	}

	// RouteProposalに変換
	routeProposal := firestoreData.ToRouteProposal(proposalID)

	log.Printf("✅ Route proposal retrieved: %s", proposalID)
	return routeProposal, nil
}

// extractHighlights はSuggestedRouteからハイライト情報を抽出する
func (r *FirestoreRouteProposalRepository) extractHighlights(route *model.SuggestedRoute) []string {
	var highlights []string
	for _, spot := range route.Spots {
		if spot != nil && spot.Name != "" {
			highlights = append(highlights, spot.Name)
		}
	}
	return highlights
}

// convertToNavigationSteps はSuggestedRouteをNavigationStepsに変換する
func (r *FirestoreRouteProposalRepository) convertToNavigationSteps(route *model.SuggestedRoute) []model.NavigationStep {
	var steps []model.NavigationStep

	for i, spot := range route.Spots {
		if spot == nil {
			continue
		}

		// POIのステップを追加
		step := model.NavigationStep{
			Type:                 "poi",
			Name:                 spot.Name,
			POIId:                spot.ID,
			Description:          fmt.Sprintf("%sに立ち寄る", spot.Name),
			Latitude:             spot.Location.Coordinates[1], // GeoJSONでは [lng, lat]
			Longitude:            spot.Location.Coordinates[0],
			DistanceToNextMeters: 0, // 実際の距離計算は後で実装可能
		}

		// 次のスポットがある場合、簡易的な距離を設定
		if i < len(route.Spots)-1 && route.Spots[i+1] != nil {
			step.DistanceToNextMeters = 200 // 仮の値、実際は計算が必要
		}

		steps = append(steps, step)
	}

	return steps
}

// generateStoryFromRoute はルートから簡単な物語を生成する（将来的にGemini APIを使用）
func (r *FirestoreRouteProposalRepository) generateStoryFromRoute(route *model.SuggestedRoute, index int) string {
	if len(route.Spots) == 0 {
		return fmt.Sprintf("素敵な散歩ルート%d", index+1)
	}

	// 簡易的な物語生成（将来的にGemini APIで置き換え）
	firstSpot := ""
	lastSpot := ""

	for _, spot := range route.Spots {
		if spot != nil {
			if firstSpot == "" {
				firstSpot = spot.Name
			}
			lastSpot = spot.Name
		}
	}

	if firstSpot == lastSpot {
		return fmt.Sprintf("%sを中心とした魅力的な散歩をお楽しみください。", firstSpot)
	}

	return fmt.Sprintf("%sから始まり、%sで終わる素晴らしい散歩の物語。", firstSpot, lastSpot)
}
