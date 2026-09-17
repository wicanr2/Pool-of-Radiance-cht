package main

// #24／#20（spec 114／069／022）：紮營畫面一打開就跑這一區的 ECL 入口 2，把
// 「會不會被打擾」寫進 `6DD2h`／`6DD3h`；時鐘每次推進都投影到 `49C6h..49CCh`，
// 腳本的白天判斷才讀得到真的小時。
//
// 呼叫順序是 exact：overlay-03 的紮營常式 `312Ah` 先 `3134h push ds:4948h`
// （入口 2 的 header，spec 022 的 `DS:4944..494C` 表）、`3139h` 呼叫 VM runner，
// **之後**才 `3141h` far call overlay-15 entry 1（紮營畫面；休息迴圈
// overlay-20 entry 3 `0C45h` 在那裡面），被打斷才 `3151h push ds:494Ah` 走入口 3。

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// restEntryValues 直接跑某一個區塊的入口 2，回傳它寫下的週期與門檻。
// 這是規則層的對照：每一區的入口 2 各自有自己的條件，值要對得上腳本。
func restEntryValues(t *testing.T, application *app, archive uint8, block uint16,
	x, y uint8, presets map[uint16]uint16) (period, threshold int) {
	t.Helper()
	catalogArchive, ok := application.eclCatalog.Archive(archive)
	if !ok {
		t.Fatalf("ECL%d archive is absent", archive)
	}
	session, err := gamepack.NewDOSECLArchiveSession(catalogArchive, block, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	geoMap, ok := application.geometryCatalog.Map(gamepack.MapKey{Archive: archive, BlockID: uint8(block)})
	if !ok {
		t.Fatalf("GEO%d/%d is absent", archive, block)
	}
	for address, value := range presets {
		session.Machine().Memory[address] = value
	}
	if _, err := gamepack.RunInitialSessionRestEntry(session, geoMap.Grid,
		gamepack.Spawn{Map: geoMap.Key, X: x, Y: y}); err != nil {
		t.Fatal(err)
	}
	return int(session.Machine().Memory[gamepack.RestInterruptionPeriodAddress]),
		int(session.Machine().Memory[gamepack.RestInterruptionThresholdAddress])
}

// 四個區的入口 2 寫下的值，逐條對回腳本（位址在 spec 137／114）。
func TestRestEntryTwoWritesEachAreasInterruption(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	for _, row := range []struct {
		name      string
		archive   uint8
		block     uint16
		x, y      uint8
		presets   map[uint16]uint16
		period    int
		threshold int
		why       string
	}{
		// `ecl2/20 9A0Eh`：`@6E82`（地形）是 0（街上）而且還沒清完 → 24／24。
		//
		// **入口 2 讀的是 `@6E82`，不是 `@C04F`**：那一格是入口 0／1 每次移動時
		// 用 `AND 127 @C04F → @6E82` 留下的（spec 015 `9965h`）。正常玩的時候
		// 它自然是最新的，治具沒走過路，所以這裡明寫。
		{"貧民窟街上", 2, 20, 0, 4, map[uint16]uint16{0x6E82: 0}, 24, 24, "街上會被城衛隊趕"},
		// 同一支的第三條：地形不是 0（屋內，(11,0) 是 `81h`、遮罩後 1）→ 0／0。
		// 這是「街上 24／24」的負對照：少了它，24／24 有可能是每一格都一樣。
		{"貧民窟屋內", 2, 20, 11, 0, map[uint16]uint16{0x6E82: 1}, 0, 0, "屋內沒人趕"},
		// 同一支：`4ABB >= 254`（清完了）→ 0／0。
		{"貧民窟清完", 2, 20, 0, 4, map[uint16]uint16{0x6E82: 0, 0x4ABB: 0xFE}, 0, 0, "清完就沒人管"},
		// `ecl3/0 9A5Eh → 9A63h`：`4ABA < 254` 且 `4A07 == 0` → 1／101。
		// 門檻 101 大於百分位的上限，所以「每一刻都問、一定中」。
		{"城區", 3, 0, 1, 4, nil, 1, 101, "城裡睡在街上一定被趕"},
		// 同一支：通關之後 → 0／0。
		{"城區通關後", 3, 0, 1, 4, map[uint16]uint16{0x4ABA: 0xFE}, 0, 0, "通關就不管了"},
		// `ecl4/21 9A29h`：`4A03 <= 4`（巡邏還在）→ 2／1。
		{"索寇要塞", 4, 21, 8, 14, nil, 2, 1, "巡邏還在"},
		{"索寇巡邏清完", 4, 21, 8, 14, map[uint16]uint16{0x4A03: 5}, 0, 0, "巡邏清完"},
	} {
		t.Run(row.name, func(t *testing.T) {
			period, threshold := restEntryValues(t, application, row.archive, row.block, row.x, row.y, row.presets)
			if period != row.period || threshold != row.threshold {
				t.Fatalf("%s 的入口 2 寫下 %d／%d，腳本是 %d／%d（%s）",
					row.name, period, threshold, row.period, row.threshold, row.why)
			}
		})
	}
}

// 接線：玩家按 E 打開紮營畫面，那一刻入口 2 就該跑過。打開之前兩個值是 0——
// 沒有這個負對照，「打開之後是 1／101」證明不了是 E 這一步做的。
func TestOpeningCampRunsTheAreaRestEntry(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	if application.spawn.Map.Archive != 3 || application.spawn.Map.BlockID != 0 {
		t.Skipf("開場沒有停在城區，而是 GEO%d/%d", application.spawn.Map.Archive, application.spawn.Map.BlockID)
	}
	before := [2]uint16{
		application.eventMachine.Memory[gamepack.RestInterruptionPeriodAddress],
		application.eventMachine.Memory[gamepack.RestInterruptionThresholdAddress],
	}
	if before != [2]uint16{0, 0} {
		t.Fatalf("紮營之前就有打斷設定 %v，負對照不成立", before)
	}
	if err := press(application, ebiten.KeyE); err != nil {
		t.Fatal(err)
	}
	if !application.campOpen {
		t.Fatalf("按 E 沒有打開紮營畫面：%q", application.statusLine)
	}
	got := application.restInterruption()
	// 城區 `ecl3/0 9A63h`：還沒通關且 `4A07 == 0` → 1／101。
	if got.Period != 1 || got.Threshold != 101 {
		t.Fatalf("打開紮營之後城區的打斷設定是 %d／%d，腳本 `9A63h` 是 1／101", got.Period, got.Threshold)
	}
}

// 時鐘要投影給 ECL：走一步一分鐘，七位都要寫。
func TestGameClockProjectsIntoECLMemory(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	hourAddress := uint16(gamepack.ClockECLBaseAddress + gamepack.TimeDigitHour)
	if got := application.eventMachine.Memory[hourAddress]; got != 0 {
		t.Fatalf("開場的小時是 %d，該是 0", got)
	}
	// 推到 14:00——腳本的白天判斷用的就是這個門檻（`ecl3/0 9BAEh`、
	// `ecl2/9 ADAAh` 的馬車商人、`ecl3/0 9920h` 的晚上鎖門、`ecl4/21 AE48h`）。
	for minutes := 0; minutes < 14*60; minutes++ {
		application.advanceGameTime(1)
	}
	if got := application.gameTime[gamepack.TimeDigitHour]; got != 14 {
		t.Fatalf("推了十四小時之後時鐘的小時是 %d", got)
	}
	if got := application.eventMachine.Memory[hourAddress]; got != 14 {
		t.Fatalf("ECL 讀到的小時是 %d，時鐘是 14——投影沒接上（#20）", got)
	}
	// 七位都要寫，不是只寫小時：原版 overlay-20 entry 2 寫的是整個時間。
	for digit := 0; digit < gamepack.TimeDigits; digit++ {
		address := uint16(gamepack.ClockECLBaseAddress + digit)
		if got, want := application.eventMachine.Memory[address], uint16(application.gameTime[digit]); got != want {
			t.Fatalf("第 %d 位投影成 %d，時鐘是 %d", digit, got, want)
		}
	}
}

// 換 machine（換圖、讀檔）之後那七格是 0，要補投影；不補就等於把時間倒回午夜。
func TestGameClockSurvivesANewEventSession(t *testing.T) {
	application := bootCityParty(t, dosZIPForTests)
	for minutes := 0; minutes < 15*60; minutes++ {
		application.advanceGameTime(1)
	}
	archive, ok := application.eclCatalog.Archive(3)
	if !ok {
		t.Fatal("ECL3 archive is absent")
	}
	session, err := gamepack.NewDOSECLArchiveSession(archive, 0, 0x9914)
	if err != nil {
		t.Fatal(err)
	}
	if got := session.Machine().Memory[gamepack.ClockECLBaseAddress+gamepack.TimeDigitHour]; got != 0 {
		t.Fatalf("全新的 machine 小時是 %d，該是 0（負對照）", got)
	}
	if err := application.configureEventSession(session); err != nil {
		t.Fatal(err)
	}
	if got := session.Machine().Memory[gamepack.ClockECLBaseAddress+gamepack.TimeDigitHour]; got != 15 {
		t.Fatalf("接上新 session 之後 ECL 的小時是 %d，時鐘是 15", got)
	}
}
