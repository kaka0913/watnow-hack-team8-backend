package model

// SuggestionRequest はルート提案に必要な全ての条件を保持する
type SuggestionRequest struct {
	StartLocation       *Location        `json:"start_location" validate:"required"`        // 必須：スタート地点
	DestinationLocation *Location        `json:"destination_location"`                      // オプション：目的地なし（お散歩モード）の場合はnull
	Mode                string           `json:"mode" validate:"required,oneof=destination time_based"` // 必須：モード
	TimeMinutes         int              `json:"time_minutes"`                              // modeが"time_based"の場合に必須
	Theme               string           `json:"theme" validate:"required"`                 // 必須：テーマ
	Scenarios           []string         `json:"scenarios,omitempty"`                       // オプション：指定がなければテーマ内の全シナリオが対象
	RealtimeContext     *RealtimeContext `json:"realtime_context"`                          // オプション：リアルタイムコンテキスト（天気、時間帯など）
}

// UserLocation 後方互換性のため、StartLocationをLatLng形式で取得
func (r *SuggestionRequest) UserLocation() LatLng {
	if r.StartLocation == nil {
		return LatLng{}
	}
	return LatLng{
		Lat: r.StartLocation.Latitude,
		Lng: r.StartLocation.Longitude,
	}
}

// Destination 後方互換性のため、DestinationLocationをLatLng形式で取得
func (r *SuggestionRequest) Destination() *LatLng {
	if r.DestinationLocation == nil {
		return nil
	}
	return &LatLng{
		Lat: r.DestinationLocation.Latitude,
		Lng: r.DestinationLocation.Longitude,
	}
}

// HasDestination は目的地が指定されているかどうかを判定する
func (r *SuggestionRequest) HasDestination() bool {
	return r.DestinationLocation != nil
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
