# Spec 131：戰場的地形圖塊

狀態：READY（三個圖塊集的檔名、形狀與 item 數；`PresentationCode` 就是圖塊集
裡的序號；室內／野外兩段的切點）；DRAFT（切點由哪一段程式碼選、序號
`19h..21h` 在室內段沒有 producer 的理由、色盤從哪裡取）。
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

66 筆類別分兩段，**0..31 是室內、32..65 是野外**：

| 段 | 類別碼 | `PresentationCode` | 圖塊集 |
|---|---|---|---|
| 室內 | `00h..19h` | `00h..18h` | `DUNGCOM.DAX` item 0..24 |
| 室內 | `1Ah..1Fh` | `22h..27h` | `RANDCOM.DAX` item 0..5 |
| 野外 | `20h..41h` | `00h..21h` | `WILDCOM.DAX` item 0..33 |

切點是從序號範圍推的，不是從程式碼讀的：室內段的序號正好蓋滿 DUNGCOM 的
25 個加 RANDCOM 的 6 個，野外段正好蓋滿 WILDCOM 的 34 個，**三個檔一個
item 都不多不少**。序號 `19h..21h`（9 個）在室內段沒有出現，在野外段則是
WILDCOM 自己的 item——這也是「兩段各有自己的序號空間」的旁證。

哪一段由誰選還沒讀出來。

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
- `cmd/pool-game/combat_screen.go`：`combatTerrainTile` 依上表挑圖塊，
  `drawCombatBoard` 先鋪地再畫造形；載不出來退回色塊。
- 換主題時 `switchTheme` 丟掉圖塊與造形的快取，否則會留在舊色盤上。
