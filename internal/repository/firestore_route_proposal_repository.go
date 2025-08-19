package repository

import (
	"Team8-App/internal/domain/model"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

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

// SaveRouteProposalsWithStory は複数のルート提案をFirestoreに並行保存し、proposal_idを生成して返す
func (r *FirestoreRouteProposalRepository) SaveRouteProposalsWithStory(ctx context.Context, proposals []*model.SuggestedRoute, theme string, ttlHours int, titles []string, stories []string) ([]*model.RouteProposal, error) {
	if len(proposals) != len(titles) || len(proposals) != len(stories) {
		return nil, fmt.Errorf("提案数とタイトル数・物語数が一致しません")
	}

	collection := r.client.Collection("routeProposals")

	// 並行保存用の構造体
	type saveResult struct {
		index         int
		routeProposal *model.RouteProposal
		err           error
	}

	resultChan := make(chan saveResult, len(proposals))
	var wg sync.WaitGroup

	// 各ルート提案を並行でFirestoreに保存
	for i, suggestedRoute := range proposals {
		wg.Add(1)
		go func(idx int, route *model.SuggestedRoute) {
			defer wg.Done()

			// 一時IDを生成
			proposalID := fmt.Sprintf("temp_prop_%s", uuid.New().String())

			// SuggestedRouteをRouteProposalに変換
			routeProposal := &model.RouteProposal{
				ProposalID:               proposalID,
				Title:                    titles[idx],
				EstimatedDurationMinutes: int(route.TotalDuration.Minutes()),
				EstimatedDistanceMeters:  0, // SuggestedRouteには距離情報がないため0とする
				Theme:                    theme,
				DisplayHighlights:        r.extractHighlights(route),
				NavigationSteps:          r.convertToNavigationSteps(route),
				RoutePolyline:            route.Polyline,
				GeneratedStory:           stories[idx],
			}

			// Firestore用の構造体に変換
			firestoreData := routeProposal.ToFirestoreRouteProposal(ttlHours)

			// Firestoreに保存
			_, err := collection.Doc(proposalID).Set(ctx, firestoreData)
			if err != nil {
				log.Printf("❌ Failed to save route proposal %s: %v", proposalID, err)
				resultChan <- saveResult{
					index:         idx,
					routeProposal: nil,
					err:           fmt.Errorf("ルート提案%d の保存に失敗しました: %w", idx+1, err),
				}
				return
			}

			log.Printf("✅ Route proposal saved: %s (expires in %d hours)", proposalID, ttlHours)
			resultChan <- saveResult{
				index:         idx,
				routeProposal: routeProposal,
				err:           nil,
			}
		}(i, suggestedRoute)
	}

	// 別のgoroutineでwaitしてチャンネルを閉じる
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 結果を収集
	result := make([]*model.RouteProposal, len(proposals))
	var saveErrors []error

	for saveRes := range resultChan {
		if saveRes.err != nil {
			saveErrors = append(saveErrors, saveRes.err)
		} else {
			result[saveRes.index] = saveRes.routeProposal
		}
	}

	// エラーが成功数より多かった場合、エラーメッセージをまとめて返す
	if len(saveErrors) > len(resultChan) {
		var errorMessages []string
		for _, err := range saveErrors {
			errorMessages = append(errorMessages, err.Error())
		}
		return nil, fmt.Errorf("大部分のルート提案の保存に失敗しました: %s", strings.Join(errorMessages, "; "))
	}

	// 成功した提案のみを返す（nilを除外）
	var successResults []*model.RouteProposal
	for _, proposal := range result {
		if proposal != nil {
			successResults = append(successResults, proposal)
		}
	}

	log.Printf("🎉 全ルート提案の並行保存完了 (%d件)", len(successResults))
	return successResults, nil
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
