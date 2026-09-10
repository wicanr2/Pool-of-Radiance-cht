package main

import "testing"

// 平面圖的視窗原點是 `clamp(隊伍座標 - 5, 0, 5)`（overlay-30 offset 0，
// spec 119）。兩個夾限各要有樣本，不然「往回退五格」寫成別的也照樣過。
func TestAreaMapOriginClampsAtBothEnds(t *testing.T) {
	const span = 16
	for _, testCase := range []struct {
		name     string
		position int
		want     int
	}{
		{"貼左邊界，退五格會變負的，夾成 0", 0, 0},
		{"隊伍在 4，退五格是 -1，還是 0", 4, 0},
		{"隊伍在 5，退五格剛好是 0", 5, 0},
		{"隊伍在 6，視窗開始跟著走", 6, 1},
		{"隊伍在中段，原點就是座標減五", 8, 3},
		{"隊伍在 10，原點 5 是上限本身", 10, 5},
		{"隊伍在 11，退五格是 6，超過上限夾回 5", 11, 5},
		{"貼右邊界，一樣夾在 5", 15, 5},
	} {
		if got := areaMapOrigin(testCase.position, span); got != testCase.want {
			t.Errorf("%s：areaMapOrigin(%d) = %d，該是 %d",
				testCase.name, testCase.position, got, testCase.want)
		}
	}
}

// 上限的意義是「視窗右緣剛好落在地圖最後一格」——原點 5 加上 11 格等於 16。
// 這一條釘的是那個關係，改了視窗大小而沒改上限的話它會紅。
func TestAreaMapWindowReachesTheLastCellExactly(t *testing.T) {
	const span = 16
	if areaMapWindow+areaMapMargin != span {
		t.Errorf("視窗 %d ＋ 上限 %d ≠ %d：11 格就涵蓋不到最後一格了",
			areaMapWindow, areaMapMargin, span)
	}
	if got := areaMapOrigin(span-1, span) + areaMapWindow; got != span {
		t.Errorf("隊伍貼右邊界時視窗右緣在 %d，該剛好是 %d", got, span)
	}
	if areaMapCell*areaMapWindow != areaMapSize {
		t.Errorf("一格 %d × %d 格 = %d，該填滿那一框的 %d",
			areaMapCell, areaMapWindow, areaMapCell*areaMapWindow, areaMapSize)
	}
}
