# Spec 131：戰場的地形圖塊

狀態：READY（三個圖塊集的檔名、形狀與 item 數；`PresentationCode` 就是圖塊集
裡的序號；圖塊集由戰場模式選，見〈誰選圖塊集〉）；DRAFT（`149Ah` 兩個參數的
確切語意、色盤從哪裡取）。
日期：2026-09-08。

## 結論先講

戰術地圖是生成的（spec 060），**鋪在每一格上的圖不是**。三個 `*COM.DAX`
就是圖塊集，每個 item 24×24——正好戰場一格，也正好與戰鬥造形同寬，
所以造形直接蓋上去就對齊。

| 檔案 | item 數 | 用途 |
|---|---|---|
| `DUNGCOM.DAX` | 25 | 地城戰鬥 |
| `WILDCOM.DAX` | 34 | 野外戰鬥 |
| `RANDCOM.DAX` | 6 | 隨機遭遇 |

三個檔都是單一區塊（block 1），`24×24`，**不遮罩**——地城的地板整格就是
色號 0，當成遮罩色會讓地板變透明。

## 輸入

- `Pool of Radiance (1988).zip` SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`
- 格位類別表：`docs/audit/ida-start-combat-cell-class-table.json`
  （`START.EXE` `DS:2758h` 起 264 bytes，66 筆 × 4）
- 原版畫面：使用者提供的 DOS 戰鬥截圖（地城戰鬥；黑色地板 ＋ 灰色碎石牆）

## 對應規則

地圖裡每格存的是**格位類別碼**，類別表第四個欄位 `PresentationCode`
才是圖塊序號（spec 053 已把該欄位定位成「傳給 tactical tile drawing
service」的 exact consumer；這裡補的是那個 service 的圖從哪裡來）。

66 筆類別依生成它的建構器分兩群：

| 群 | 類別碼 | `PresentationCode` | 誰寫 |
|---|---|---|---|
| 室內 | `00h..19h` | `00h..18h` | 室內四支建構器（spec 060） |
| 隨機物件 | `1Ah..1Fh` | `22h..27h` | 室外 `0000h` 寫 `1Ah..1Dh` |
| 野外 | `20h..41h` | `00h..21h` | 室外 `0C7Ah`／`0DB0h`／`0F3Fh` |

外加兩邊共用的平地類別 `17h`（序號 `16h`）：室內西帶鋪地板、室外 FillChar
整面填的都是它。

### 誰選圖塊集

overlay-10 `12E5h` 在配置戰術地圖之前先載圖塊，分岔條件與生成器是同一個
`DS:495Bh == 1`（bytes exact，`docs/audit/ida-overlay10-tactical-map-build.json`）：

| 位址 | 條件 | 字串（`12CDh` 起） | `149Ah` 參數 |
|---|---|---|---|
| `12F7h..1307h` | `495Bh == 1` | `DungCom` | `(0, 18h)` |
| `1313h..1323h` | 其餘 | `WildCom` | `(0, 21h)` |
| `132Dh..133Dh` | 兩邊都跑 | `RandCom` | `(22h, 5)` |

所以序號 `00h..21h` 在室內戰場取 DUNGCOM、在野外戰場取 WILDCOM，`22h` 起一律
取 RANDCOM 的 item（序號 − 22h）。同一個序號 `16h` 在室內是地城的黑地板、
在野外是野外的平地。`149Ah` 兩個參數當「起點、終點」讀是 strong inference：
DUNGCOM 25 個（0..18h）、WILDCOM 34 個（0..21h）正好對上；RANDCOM 的
`(22h, 5)` 形狀不同，確切語意沒有讀。

## 交叉核對：通行性與圖的樣子

`EntryThreshold` 是規則，圖塊是美術，兩者沒有義務一致；但如果序號對錯了，
兩者會系統性地對不上。逐格量 DUNGCOM 的色號比例（色號 0 是黑、7 與 8 是
兩階灰）：

- 室內 25 筆裡有 **23 筆方向一致**——不可進入（`FFh`）的圖以灰為主，
  可進入（`01h`）的圖以黑為主。
- 兩個例外（類別 `01h`、`0Eh`）的灰佔比是 42% 與 47%，落在對角牆那種
  半黑半灰的格子上。這是判準太粗，不是序號對不上。
- 最乾淨的一筆：室內地板類別 `17h`（`IndoorFloorBuilderClass` `16h` 加一，
  spec 060）的序號是 `16h`，那個 item **整格 576 個像素全是色號 0**。
  原版地城戰鬥的地板就是純黑。

## remake 現況

- `internal/assets/combat_terrain.go`：`ReadCombatTerrainTiles`，item 數不對
  就失敗，不拿別一款金盒子的圖塊頂上。
- `cmd/pool-game/combat_screen.go`：`combatTerrainTile` 依〈誰選圖塊集〉挑圖塊
  （戰場模式記在 `combat.TacticalGrid.Outdoor`），
  `drawCombatBoard` 先鋪地再畫造形；載不出來退回色塊。
- 換主題時 `switchTheme` 丟掉圖塊與造形的快取，否則會留在舊色盤上。
