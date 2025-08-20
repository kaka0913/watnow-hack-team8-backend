package ai

import (
	"Team8-App/internal/domain/model"
	"Team8-App/internal/domain/repository"
	"context"
	"fmt"
	"log"
	"strings"
)

// geminiStoryRepository はGemini APIを使用してstoryGenerationRepositoryを実装
type geminiStoryRepository struct {
	client *GeminiClient
}

// NewGeminiStoryRepository は新しいGeminiStoryRepositoryインスタンスを作成
func NewGeminiStoryRepository(client *GeminiClient) repository.StoryGenerationRepository {
	return &geminiStoryRepository{
		client: client,
	}
}

// GenerateStoryWithTitle は物語とタイトルを同時に生成する
func (g *geminiStoryRepository) GenerateStoryWithTitle(ctx context.Context, route *model.SuggestedRoute, theme string, realtimeContext *model.RealtimeContext) (title, story string, err error) {
	prompt := g.buildStoryPrompt(route, theme, realtimeContext)

	log.Printf("🤖 Gemini APIでタイトル・物語を同時生成中... (テーマ: %s)", theme)

	content, err := g.client.GenerateStoryContent(ctx, prompt)
	if err != nil {
		log.Printf("❌ タイトル・物語同時生成に失敗: %v", err)
		return route.Name, g.generateFallbackStory(route, theme), nil
	}

	log.Printf("✅ タイトル・物語同時生成完了: %s (物語: %d文字)", content.Title, len(content.Story))
	return content.Title, content.Story, nil
}

// buildStoryPrompt はタイトルと物語の同時生成用プロンプトを構築
func (g *geminiStoryRepository) buildStoryPrompt(route *model.SuggestedRoute, theme string, realtimeContext *model.RealtimeContext) string {
	spots := make([]string, 0, len(route.Spots))
	for _, spot := range route.Spots {
		if spot != nil && spot.Name != "" {
			spots = append(spots, spot.Name)
		}
	}

	weather := "晴れ"
	timeOfDay := "昼間"

	if realtimeContext != nil {
		if realtimeContext.Weather != "" {
			weather = g.translateWeather(realtimeContext.Weather)
		}
		if realtimeContext.TimeOfDay != "" {
			timeOfDay = g.translateTimeOfDay(realtimeContext.TimeOfDay)
		}
	}

	// TODO: プロンプト調整
	prompt := fmt.Sprintf(`

あなたの役割は、散歩ルート提案アプリでユーザーに対してルートを提案する際に、最後により魅力的な文章で表現する仕上げ役です。
以下の条件で、散歩のタイトルと物語を、必ず出力フォーマットに従った形式で生成しください：

【要件】
- タイトルは詩的で魅力的な表現
- 物語は散歩の魅力と発見を端的に表現
- 各スポットでの想定される魅力的な体験を織り込む
- 散歩の雰囲気を活かした文体
- 読者が実際に歩きたくなるような内容

【出力フォーマット】
タイトル: [15-25文字の思わず散歩したくなるような魅力的なタイトル]
物語: [150文字の散歩物語]

【散歩コース】
立ち寄るスポット: %s
テーマ: %s
天気: %s
時間帯: %s

上記のフォーマットに従って、日本語で出力してください。
ただし、生成した後に物語が150文字程度になっているかを確認し、収まっていなければ150文字程度に収まるように修正してください。`,
		strings.Join(spots, "、"),
		theme,
		weather,
		timeOfDay)

	return prompt
}

// translateWeather は英語の天気を日本語に変換
func (g *geminiStoryRepository) translateWeather(weather string) string {
	switch weather {
	case "sunny":
		return "晴れ"
	case "cloudy":
		return "曇り"
	case "rainy":
		return "雨"
	default:
		return "晴れ"
	}
}

// translateTimeOfDay は英語の時間帯を日本語に変換
func (g *geminiStoryRepository) translateTimeOfDay(timeOfDay string) string {
	switch timeOfDay {
	case "morning":
		return "朝"
	case "afternoon":
		return "昼間"
	case "evening":
		return "夕方"
	default:
		return "昼間"
	}
}

// generateFallbackStory はAPI呼び出しが失敗した場合のフォールバック物語を生成
func (g *geminiStoryRepository) generateFallbackStory(route *model.SuggestedRoute, theme string) string {
	spots := make([]string, 0, len(route.Spots))
	for _, spot := range route.Spots {
		if spot != nil && spot.Name != "" {
			spots = append(spots, spot.Name)
		}
	}

	if len(spots) == 0 {
		return fmt.Sprintf("%sの素晴らしい散歩をお楽しみください。新しい発見があなたを待っています。", route.Name)
	}

	return fmt.Sprintf("%sを巡る散歩の旅。%sで新しい出会いと発見が待っています。ゆっくりと散策を楽しみながら、その場所ならではの魅力を感じてください。",
		route.Name, strings.Join(spots, "や"))
}
