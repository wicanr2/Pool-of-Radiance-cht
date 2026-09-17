package gamepack

import "github.com/wicanr2/golden-box-remake-engine/eclvm"

// BlockLoadWrites 是原版每載入一個 ECL 區塊時、在新區塊的入口開跑之前做的記憶體寫入
// （overlay-07 entry 3，`01C8h..0333h`；spec 106〈載入區塊時清掉的兩段〉）。
//
//   - `0237h` `[4937h]+5C2h = FFh` → `6DE1h`
//   - `0244h`／`024Fh` `[4937h]+5A4h`／`+5A6h = 0` → `6DD2h`／`6DD3h`（休息打斷設定，spec 114）
//   - `025Ah` `[4933h]+1CAh = 0` → `49E5h`
//   - `02D1h` `[4933h]+1CCh = 1` → `49E6h`（遭遇距離的走法，spec 078）
//   - `02DFh..0302h` class 0 `4A00h..4A1Fh = 0`
//   - `0304h..0327h` class 1 `6E79h..6E82h = 0`
//
// 最後兩段在 `02D8h` 看 `DS:4959h`，非 0 時跳過一次。dosgolem 量到讀檔那一條路不寫
// `4959h`、讀回的值留著（`docs/audit/dosgolem-block-load-clear-cases.json`）；remake 讀檔是
// `Restore`，本來就不套這份宣告，所以不另外模擬那個旗標。位元組 exact；跨 archive、
// 同 archive 與讀檔後第一次換區都有 dosgolem 收據。
func BlockLoadWrites() []eclvm.MemoryFill {
	return []eclvm.MemoryFill{
		{Start: 0x6DE1, Count: 1, Value: 0xFF},
		{Start: 0x6DD2, Count: 2, Value: 0},
		{Start: 0x49E5, Count: 1, Value: 0},
		{Start: 0x49E6, Count: 1, Value: 1},
		{Start: 0x4A00, Count: 0x20, Value: 0},
		{Start: 0x6E79, Count: 10, Value: 0},
	}
}

// declareBlockLoadWrites 讓 session 每次 `NEWECL` 換區都照原版寫。每一個建 session 的
// 建構子都要呼叫——漏掉一個，那條路徑上的旗標就會帶著上一個區塊的值走（#41）。
func declareBlockLoadWrites(session *eclvm.BlockSession) error {
	return session.SetBlockLoadWrites(BlockLoadWrites()...)
}
