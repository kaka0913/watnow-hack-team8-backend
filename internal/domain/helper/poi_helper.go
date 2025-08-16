package helper

import (
	"Team8-App/internal/domain/model"
	"math"
	"sort"
)

const earthRadiusKm = 6371.0

// HaversineDistance は2地点間の距離を計算する (km)
func HaversineDistance(p1, p2 model.LatLng) float64 {
	lat1 := p1.Lat * math.Pi / 180
	lng1 := p1.Lng * math.Pi / 180
	lat2 := p2.Lat * math.Pi / 180
	lng2 := p2.Lng * math.Pi / 180
	dLat := lat2 - lat1
	dLng := lng2 - lng1
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

// HaversineDistancePOI は2つのPOI間の距離を計算する (km)
func HaversineDistancePOI(poi1, poi2 *model.POI) float64 {
	return HaversineDistance(poi1.ToLatLng(), poi2.ToLatLng())
}

// FilterByCategory は指定されたカテゴリのPOIのみを抽出する
func FilterByCategory(pois []*model.POI, categories []string) []*model.POI {
	var filtered []*model.POI
	catSet := make(map[string]struct{})
	for _, c := range categories {
		catSet[c] = struct{}{}
	}
	for _, p := range pois {
		for _, cat := range p.Categories {
			if _, ok := catSet[cat]; ok {
				filtered = append(filtered, p)
				break
			}
		}
	}
	return filtered
}

// FindHighestRated は最も評価の高いPOIを見つける
func FindHighestRated(pois []*model.POI) *model.POI {
	if len(pois) == 0 {
		return nil
	}
	highest := pois[0]
	for _, p := range pois {
		if p.Rate > highest.Rate {
			highest = p
		}
	}
	return highest
}

// SortByDistance は基準地点からの距離でPOIスライスをソートする
func SortByDistance(origin *model.POI, targets []*model.POI) {
	sort.Slice(targets, func(i, j int) bool {
		distI := HaversineDistancePOI(origin, targets[i])
		distJ := HaversineDistancePOI(origin, targets[j])
		return distI < distJ
	})
}

// SortByDistanceFromLocation は基準座標からの距離でPOIスライスをソートする
func SortByDistanceFromLocation(origin model.LatLng, targets []*model.POI) {
	sort.Slice(targets, func(i, j int) bool {
		distI := HaversineDistance(origin, targets[i].ToLatLng())
		distJ := HaversineDistance(origin, targets[j].ToLatLng())
		return distI < distJ
	})
}

// RemovePOI はスライスから特定のPOIを削除する
func RemovePOI(pois []*model.POI, target *model.POI) []*model.POI {
	var result []*model.POI
	for _, p := range pois {
		if p.ID != target.ID {
			result = append(result, p)
		}
	}
	return result
}

// GeneratePermutations はPOIスライスの全ての順列を生成する
func GeneratePermutations(pois []*model.POI) [][]*model.POI {
	var result [][]*model.POI
	var helper func([]*model.POI, int)
	helper = func(arr []*model.POI, n int) {
		if n == 1 {
			tmp := make([]*model.POI, len(arr))
			copy(tmp, arr)
			result = append(result, tmp)
		} else {
			for i := 0; i < n; i++ {
				helper(arr, n-1)
				if n%2 == 1 {
					arr[0], arr[n-1] = arr[n-1], arr[0]
				} else {
					arr[i], arr[n-1] = arr[n-1], arr[i]
				}
			}
		}
	}
	helper(pois, len(pois))
	return result
}

// HasCategory はPOIが指定されたカテゴリのいずれかを持つかチェックする
func HasCategory(poi *model.POI, categories []string) bool {
	catSet := make(map[string]struct{})
	for _, c := range categories {
		catSet[c] = struct{}{}
	}
	for _, cat := range poi.Categories {
		if _, ok := catSet[cat]; ok {
			return true
		}
	}
	return false
}

// SortByRating は評価の高い順にPOIスライスをソートする
func SortByRating(pois []*model.POI) {
	sort.Slice(pois, func(i, j int) bool {
		return pois[i].Rate > pois[j].Rate
	})
}
