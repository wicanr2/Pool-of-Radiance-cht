package combat

import "testing"

// 視窗中心是原點加 3，所以原點 (5,5) 的中心是 (8,8)。

func TestRecentreViewportLeavesTheOriginAloneInsideTheMargin(t *testing.T) {
	origin := ViewportOrigin{X: 5, Y: 5}
	for _, point := range [][2]uint8{{8, 8}, {5, 5}, {11, 11}, {5, 11}, {11, 5}} {
		got, moved := RecentreViewport(origin, point[0], point[1], ViewportCursorMargin)
		if moved || got != origin {
			t.Fatalf("(%d,%d) 在餘裕 3 的框內卻捲了：%+v moved=%v", point[0], point[1], got, moved)
		}
	}
}

func TestRecentreViewportStepsUntilTheTargetIsTheCentre(t *testing.T) {
	// 餘裕 3 的框是 5..11；12 在框外，中心要一路走到 12，原點變成 9。
	got, moved := RecentreViewport(ViewportOrigin{X: 5, Y: 5}, 12, 12, ViewportCursorMargin)
	if !moved {
		t.Fatal("框外卻沒捲")
	}
	if got != (ViewportOrigin{X: 9, Y: 9}) {
		t.Fatalf("原點 %+v，預期 {9 9}", got)
	}
}

func TestRecentreViewportWithMarginZeroAlwaysCentres(t *testing.T) {
	// `Center` 用的餘裕 0：只要目標不是正中央就捲到正中央。
	got, moved := RecentreViewport(ViewportOrigin{X: 5, Y: 5}, 9, 7, ViewportCentreMargin)
	if !moved {
		t.Fatal("餘裕 0 時只差一格也該捲")
	}
	if got != (ViewportOrigin{X: 6, Y: 4}) {
		t.Fatalf("原點 %+v，預期 {6 4}", got)
	}
	if again, moved := RecentreViewport(got, 9, 7, ViewportCentreMargin); moved || again != got {
		t.Fatalf("已經置中還捲：%+v moved=%v", again, moved)
	}
}

func TestRecentreViewportClampsTheCentreLikeTheOriginal(t *testing.T) {
	// `0885h`／`08BDh` 的下界 3、`08A2h`／`08DAh` 的上界 2Eh／15h。
	low, moved := RecentreViewport(ViewportOrigin{X: 10, Y: 10}, 0, 0, ViewportCentreMargin)
	if !moved || low != (ViewportOrigin{X: 0, Y: 0}) {
		t.Fatalf("往左上夾制後原點 %+v moved=%v，預期 {0 0}", low, moved)
	}
	high, moved := RecentreViewport(ViewportOrigin{X: 0, Y: 0}, TacticalMaxX, TacticalMaxY, ViewportCentreMargin)
	if !moved {
		t.Fatal("往右下該捲")
	}
	want := ViewportOrigin{
		X: uint8(ViewportCentreMaxX - ViewportCentreOffset),
		Y: uint8(ViewportCentreMaxY - ViewportCentreOffset),
	}
	if high != want {
		t.Fatalf("往右下夾制後原點 %+v，預期 %+v", high, want)
	}
}

func TestRecentreViewportForceMarginSkipsTheInsideTest(t *testing.T) {
	// `0802h` 的 0FFh：餘裕當 0，但不看框——目標就是中心時原點不動，
	// 回傳的旗標仍是「重畫過」。
	origin := ViewportOrigin{X: 5, Y: 5}
	got, moved := RecentreViewport(origin, 8, 8, ViewportForceMargin)
	if !moved {
		t.Fatal("0FFh 一定要回「重畫過」")
	}
	if got != origin {
		t.Fatalf("目標就是中心，原點不該動：%+v", got)
	}
}

func TestViewportSpanCoversWhatTheOriginalRedraws(t *testing.T) {
	// 重畫迴圈兩層跑 0..6（七格），而中心在原點 +3——正好是正中央。
	if ViewportCentreOffset >= ViewportTileSpan {
		t.Fatalf("中心 +%d 落在 %d 格的視窗外", ViewportCentreOffset, ViewportTileSpan)
	}
}
