# Spec 024：City Hall commission 公告分派

狀態：CONFORMED；日期：2026-08-31。
實作：ECL 腳本本身，由共用 engine 的 `eclvm` 直接跑——**remake 這一側沒有市政廳專屬的程式碼**，前端只提供文字框與選單（`cmd/pool-game` 的事件消費）。玩家路徑的驗證在 `cmd/pool-game/coverage_test.go`。

## 範圍

本切片只閉合 `4AC1h=1..9` 時，City Hall 牆上新增哪一則 proclamation；不把
`AD29h` 起的 Skullcrusher 離隊事件或尚未定位的 clerk 頒發 commission 流程混進來。

證據輸入、雜湊、IDA 位址空間與 trace 契約沿用 Spec 023。可重生的
`docs/audit/dos-ecl3-block0-trace.json` 同列保留原始 ECL 位址、opcode、operand bytes、
packed 原文與控制流邊。

## 原版分派

`AC55h` 比較 `4AC1h` 與零；非零時：

1. `AC60h SUBTRACT` 計算 `4A18h = 4AC1h - 1`。
2. `AC69h ON GOSUB` 依零起算索引跳到九個子程式之一。
3. 子程式把下表原文寫入 string-memory `985Eh` 後 `RETURN`。
4. `AC8Ah PRINT "PROCLAMATION"`，`AC96h PRINT [985Eh]`，`AC9Ah EXIT`。

| `4AC1h` | 子程式 | `985Eh` 原文 |
|---:|---:|---|
| 1 | `ACBFh` | `CI.` |
| 2 | `ACC9h` | `CXXVI AND CX.` |
| 3 | `ACDAh` | `CXXXIV.` |
| 4 | `ACE7h` | `CLIV.` |
| 5 | `ACF2h` | `CXIV.` |
| 6 | `ACFDh` | `CCIV.` |
| 7 | `AD08h` | `CXXIX.` |
| 8 | `AD14h` | `CCI.` |
| 9 | `AD1Eh` | `CXIV.` |

分派位址、九筆 literal、`4AC1h-1` 與輸出順序均為 `exact`。第 5 與第 9 筆同為
`CXIV.` 是原始 bytes 的結果，不應自行「訂正」。`4AC1h` 的上游授予時機不在本切片，
因此不得由這份表猜測 commission 劇情語意。

## 驗收

- 從真實 `ECL3/block0` 建立新 session，投影真實 City Hall `(3,4,facing 2)` 狀態。
- 九個 `4AC1h` 值逐一走 entry 1、Return menu 與所有文字 boundary 至 `EXIT`。
- 每筆必須得到共同公告前言與精確的 `PROCLAMATION <表列原文>`；預設零值仍由
  Spec 023 的正常按鍵測試作控制組。
- 不新增 engine alias、作品 hardcode 或手寫公告替代路徑；驗收直接執行原始 ECL。

`TestCityHallCommissionSelectsOriginalProclamation` 已依正式舊格 entry 0 → 移動 →
新格 entry 1 順序，對九個值各建立獨立 session 並走到 `EXIT`；九筆皆通過。
