package repository

import (
	"Team8-App/internal/domain/model"
	"context"
)

// StoryGenerationRepository は物語とタイトル生成の責務を持つリポジトリインターフェース
type StoryGenerationRepository interface {
	// GenerateStoryWithTitle は物語とタイトルを同時に生成する
	GenerateStoryWithTitle(ctx context.Context, route *model.SuggestedRoute, theme string, realtimeContext *model.RealtimeContext) (title, story string, err error)
}
