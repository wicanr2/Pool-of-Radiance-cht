# Spec 113：Pool 的 game pack 與 adapter

狀態：READY（pack 的分檔與合併順序、header、presentation、字串表，
以及前端怎麼取用都已實作並有測試護著）。
日期：2026-09-05。

## 為什麼要有它

共用 engine（`golden-box-remake-engine`）是**作品中立**的：它只認
`engine.Pack` 這個結構，不認識菲蘭、不認識任何一條 Pool 的字串。作品的內容
一律留在作品這一側，由 pack 提供。反過來也成立——Pool 的 pack 不得抄 CoAB 的
地名、位址或劇情資料，那是兩份各自獨立的內容。

## 分檔與合併順序

pack 拆成數個 JSON，載入時**依檔名排序合併**，所以數字前綴就是合併順序：

| 檔案 | 內容 |
|---|---|
| `internal/gamepack/pack/00-core.json` | header（`schema_version`／`id`／`default_locale`）、`presentation`、`events` |
| `internal/gamepack/pack/20-locale.en.json` | 英文字串表 |
| `internal/gamepack/pack/20-locale.zh-TW.json` | 繁中字串表 |

header 欄位只能出現在一個分檔裡（engine 的 `mergeHeader` 會擋重複宣告），
所以 `00-` 是唯一放 header 的地方。

`id` 是 `pool-of-radiance.phlan`。`presentation` 是原生 320×200、放大兩倍——
與 `cmd/pool-game` 的 `logicalWidth = 640`／`logicalHeight = 400` 一致。

## adapter

| 位置 | 做什麼 |
|---|---|
| `gamepack.PackPartNames()` | 列出分檔，順序就是合併順序 |
| `gamepack.Pack()` | 用 `engine.LoadPackPartsFS` 合併，只解析一次 |
| `gamepack.LocaleTable(locale)` | 取一個語言的字串表；語言不存在回錯 |
| `cmd/pool-game` 的 `packMessage(id, locale)` | 用 `messageKeys[id]` 查字串 |
| `cmd/pool-game` 的 `(*app).text(id)` | 繁中查不到就退回英文 |

`messageID` 與它的**出處註解**留在 `cmd/pool-game/text.go`：說明書頁碼、
原版字串的位址這些是證據，屬於程式碼那一側；JSON 沒有註解，塞進去會掉。
`messageKeys` 由 `workplace/packgen`（一次性遷移工具，不進版控）從舊的
`messages` 表產生，key 的規則是 `msgTitleHint → ui.titleHint`。

## 刻意不放進 pack 的東西

- **建角規則**：Pool 的建角是從原版 `START.EXE` 的資料段直接重生的
  （spec 003／004）。搬進 JSON 會把「逐位元組對得上原版」降級成「有人抄了一份
  數字進去」，那是**降低**保真度。engine 的 `character_creation.templates`
  是給沒有原始資料可讀的作品用的。
- **事件與地圖**：Pool 直接跑原版的 ECL 與 GEO。把它們抄進 pack 會變成
  第二份真相，而且抄錯一格的症狀是「測試綠、玩家走不到」（CoAB 那邊已經踩過，
  見該專案 `CLAUDE.md` 的第 4 點）。
- **`search`**：Pool 這一側的搜尋分鐘數還沒從原版讀出來，沒有證據就不宣告。

## 護欄

| 測試 | 擋什麼 |
|---|---|
| `TestGamePackLoadsThroughTheSharedEngine` | pack 進得了 engine 的載入器；id、schema、320×200 ×2 都釘住 |
| `TestGamePackPartsAreListedInMergeOrder` | 分檔照檔名排序，header 在 `00-` |
| `TestGamePackLocalesHaveTheSameKeys` | 兩個語言的鍵完全一樣；查不存在的語言要回錯 |
| `TestGamePackCarriesNoAzureBondsContent` | pack 裡不准出現 CoAB 專有的識別字（配一條正對照，確認掃描面沒有洞）|
| `TestMessageKeysAndThePackAgree` | `messageKeys` 與字串表一一對應，沒有孤兒 |
| `TestEveryMessageHasATraditionalChineseString` | 每一則都有英文與繁中 |

## 下一批

字串之外，engine 的 `Pack` 還模型化了 `text_rules`、`option_rules`、
`music_*`、`combat_*` 那幾組。它們要不要搬，判準一律是**搬進去會不會降低
保真度**：Pool 能從原版靜態讀出來的，留在讀取端；只有 remake 自己決定的
（配色、按鍵提示、暫定值那類）才進 pack。
