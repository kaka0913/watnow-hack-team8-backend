package model

// SuggestionRequest はルート提案に必要な全ての条件を保持する
type SuggestionRequest struct {
	UserLocation LatLng   // 必須：スタート地点
	Theme        string   // 必須：テーマ
	Scenarios    []string // オプション：指定がなければテーマ内の全シナリオが対象
	Destination  *LatLng  // オプション：nilでなければ目的地ありモード
}

// HasDestination は目的地が指定されているかどうかを判定する
func (r *SuggestionRequest) HasDestination() bool {
	return r.Destination != nil
}

// GetScenarios はシナリオリストを取得する（空の場合は空スライスを返す）
func (r *SuggestionRequest) GetScenarios() []string {
	if r.Scenarios == nil {
		return []string{}
	}
	return r.Scenarios
}

// HasSpecificScenarios は特定のシナリオが指定されているかどうかを判定する
func (r *SuggestionRequest) HasSpecificScenarios() bool {
	return len(r.Scenarios) > 0
}
