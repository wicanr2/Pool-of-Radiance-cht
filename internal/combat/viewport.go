package combat

// 戰術地圖的捲動視窗（overlay-32 `07D4h`）。
//
// 原版的戰術畫面一次只看得到 **6×6 格**：重畫迴圈兩層都數到 6，每畫一格
// 螢幕座標加 3、地圖座標加 1，資料讀 `[DS:6674h] + 7 + y*32h + x`。
// 視窗左上角就是戰術地圖 record 的 `+2`／`+3`（[ViewportOrigin]），
// 所以視窗中心是原點加 3。
//
// `07D4h(X, Y, 餘裕)` 的形狀：
//
//	07DE  cx = [DS:6674h]+2 + 3 ；cy = [DS:6674h]+3 + 3
//	0802  餘裕 0FFh 當 0 用
//	0813  框 ＝ [cx−餘裕, cx+餘裕] × [cy−餘裕, cy+餘裕]
//	084C  餘裕 ≠ 0FFh 且 (X,Y) 在框內 → 回 0（不捲、不重畫）
//	0875  否則把 cx 一格一格朝 X 移，夾在 3..2Eh；cy 朝 Y 移，夾在 3..15h
//	08E5  新的 cx−3／cy−3 寫回 `+2`／`+3`，重畫整個 6×6 視窗，回 1
//
// **餘裕就是「離中心多遠才捲」**：Manual 的格子游標用 3（走到視窗邊緣才捲），
// `Center` 與火球術的雲心用 0（一定捲到正中央）。`0FFh` 是「不看框、直接
// 重畫」，餘裕仍當 0。
const (
	// ViewportTileSpan 是視窗的邊長（格）。
	ViewportTileSpan = 6
	// ViewportCentreOffset 是中心相對於原點的位移（`07EAh`／`07F9h` 的 +3）。
	ViewportCentreOffset = 3

	// 中心的夾制範圍，逐個對應 `0885h`／`08A2h`／`08BDh`／`08DAh` 的常數。
	ViewportCentreMinX = 3
	ViewportCentreMaxX = 0x2E
	ViewportCentreMinY = 3
	ViewportCentreMaxY = 0x15

	// ViewportForceMargin 是 `0802h` 那個特別值：餘裕當 0，而且跳過
	// 「已經在框內」的檢查，所以一定會重畫。
	ViewportForceMargin = 0xFF

	// ViewportCursorMargin 是 Manual 格子游標用的餘裕（overlay-13 `2E43h`），
	// ViewportCentreMargin 是 `Center` 與火球術用的（`3734h`／overlay-22 `26EEh`）。
	ViewportCursorMargin = 3
	ViewportCentreMargin = 0
)

// stepCentre 是 `0875h`／`0892h` 那兩個 while 迴圈：一格一格朝目標移，
// 碰到夾制邊界就停。兩邊的條件都拿目標與中心直接比，不是比框。
func stepCentre(centre, target, low, high int) int {
	for target < centre && centre > low {
		centre--
	}
	for target > centre && centre < high {
		centre++
	}
	return centre
}

// RecentreViewport 重現 overlay-32 `07D4h`：把 6×6 視窗捲到讓 (x, y) 落在
// 距離中心 margin 格的框裡。第二個回傳值就是原版的 AL——true 代表捲了
// 並且重畫過，false 代表本來就在框內、原點一個位元組都沒動。
//
// 比較照原版以**有號位元組**進行（`cmp al, …` 配 `jl`／`jg`）。
func RecentreViewport(origin ViewportOrigin, x, y uint8, margin uint8) (ViewportOrigin, bool) {
	centreX := int(int8(origin.X)) + ViewportCentreOffset
	centreY := int(int8(origin.Y)) + ViewportCentreOffset

	span := int(margin)
	if margin == ViewportForceMargin {
		span = 0
	}

	targetX, targetY := int(int8(x)), int(int8(y))
	if margin != ViewportForceMargin &&
		targetX >= centreX-span && targetX <= centreX+span &&
		targetY >= centreY-span && targetY <= centreY+span {
		return origin, false
	}

	centreX = stepCentre(centreX, targetX, ViewportCentreMinX, ViewportCentreMaxX)
	centreY = stepCentre(centreY, targetY, ViewportCentreMinY, ViewportCentreMaxY)
	return ViewportOrigin{
		X: uint8(centreX - ViewportCentreOffset),
		Y: uint8(centreY - ViewportCentreOffset),
	}, true
}
