package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"Team8-App/internal/infrastructure/database"
	"Team8-App/internal/repository"
	"Team8-App/internal/domain/model"

	"github.com/stretchr/testify/assert"
)

func TestCategoryAnalysis(t *testing.T) {
	// Supabaseクライアントを初期化
	supabaseClient, err := database.NewSupabaseClient()
	assert.NoError(t, err)

	poiRepo := repository.NewSupabasePOIsRepository(supabaseClient)

	// 京都河原町の位置
	kawaramachi := model.LatLng{
		Lat: 35.004573,
		Lng: 135.768799,
	}

	t.Run("各カテゴリのPOI数を確認", func(t *testing.T) {
		categories := []string{
			"公園", "寺院", "神社", "カフェ", "ベーカリー", "店舗",
			"観光名所", "文化施設", "自然スポット",
			"park", "temple", "cafe", "bakery", "store", "natural_feature", "place_of_worship",
		}

		for _, category := range categories {
			pois, err := poiRepo.FindNearbyByCategories(
				context.Background(),
				kawaramachi,
				[]string{category},
				5000, // 5km範囲
				50,   // 最大50件
			)
			assert.NoError(t, err)
			fmt.Printf("📊 カテゴリ '%s': %d件のPOIが見つかりました\n", category, len(pois))

			// 上位3件の詳細を表示
			if len(pois) > 0 {
				fmt.Printf("  🏆 上位POI:\n")
				for i, poi := range pois {
					if i >= 3 {
						break
					}
					fmt.Printf("    %d. %s (%s) - 評価: %.1f\n", 
						i+1, poi.Name, getCategoryDisplay(poi.Categories), poi.Rate)
				}
			}
			fmt.Println()
		}
	})
}

// getCategoryDisplay は配列形式のカテゴリを文字列に変換
func getCategoryDisplay(categories []string) string {
	if len(categories) == 0 {
		return "カテゴリなし"
	}
	
	categoryBytes, err := json.Marshal(categories)
	if err != nil {
		return fmt.Sprintf("%v", categories)
	}
	return string(categoryBytes)
}
