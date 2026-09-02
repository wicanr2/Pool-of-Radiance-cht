package combat

// SortNearbyCells 重現 overlay-31 `002Eh`：對 DS:6674h 結果表做交換排序，
// 成本小的排前面；成本相同時朝向索引小的排前面，但斜向不得越過正向
// （原版以 `朝向 mod 2` 比較，偶數是四個正向、奇數是四個對角）。
//
// 這個比較關係不是全序，所以不能換成一般的排序函式；必須照原版的
// 兩層迴圈逐對交換，結果才會一致。
func SortNearbyCells(cells []NearbyCell) {
	if len(cells) <= 1 {
		return
	}
	for i := 0; i < len(cells)-1; i++ {
		for j := i + 1; j < len(cells); j++ {
			if nearbyCellPrecedes(cells[j], cells[i]) {
				cells[i], cells[j] = cells[j], cells[i]
			}
		}
	}
}

func nearbyCellPrecedes(candidate, incumbent NearbyCell) bool {
	if candidate.Cost < incumbent.Cost {
		return true
	}
	if candidate.Cost != incumbent.Cost {
		return false
	}
	if candidate.Facing >= incumbent.Facing {
		return false
	}
	return candidate.Facing%2 <= incumbent.Facing%2
}
