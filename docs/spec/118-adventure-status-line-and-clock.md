# Spec 118：冒險畫面的狀態列與遊戲時鐘

狀態：READY（隊伍面板版面、狀態列格式、一步一分、轉向與被擋下來不加）；
DRAFT（換圖那一步為什麼不加、指令列 `AREA CAST VIEW ENCAMP SEARCH LOOK`）。
日期：2026-09-05。

## 怎麼量的

`tools/capture-dos-adventure.sh` 的鍵序接上二十二個 Return（把 Rolf 導覽按完）
再送方向鍵，逐鍵拍兩張（按下當時與靜止後）。基準圖：

| 檔案 | 狀態列 | 說明 |
|---|---|---|
| `06-free-move-0-4-west.png` | `0, 4 W 00:00` | 導覽結束、自由移動已開始的那一格 |
| `07-slums-14-4-west.png` | `14, 4 W 00:01` | 進貧民窟之後往西再走一步 |
| `08-slums-14-5-south.png` | `14, 5 S 00:02` | 左轉之後往南走一步 |

（都在 `docs/reference/original-dos/adventure/`。）

## 讀到的東西

**版面**：第一人稱框右邊是隊伍面板——`NAME` 靠左，`AC` 與 `HP` 靠右，
一列一個隊員；面板下面是狀態列 `X, Y F HH:MM`。朝向是一個字母，
與 spec 076 的 0 北 1 東 2 南 3 西一一對應（`02` 是 `W`＝3、`05` 是 `S`＝2）。

**時鐘一步一分**（exact，逐格數過）：

| 動作 | 位置 | 時鐘 |
|---|---|---|
| 導覽結束 | `(0,4)` W | `00:00` |
| 往西一步（`(0,4)` → 貧民窟 `(15,4)`，換圖）| — | `00:00` |
| 往西一步 | `(14,4)` W | `00:01` |
| 左轉 | `(14,4)` S | `00:01` |
| 往南一步 | `(14,5)` S | `00:02` |
| 右轉、再往西兩步（被牆擋住）| `(14,5)` W | `00:02` |

所以：**走成功的一步加一分；轉向不加；被擋下來的那一步不加。**
換圖那一步也沒加——那一步同時是 `(0,4)` 往西進貧民窟的區域交接
（remake 自己的換圖紀錄也是 `GEO3/0 (0,4) → GEO2/20 (15,4)`），
**為什麼不加還沒讀**，目前只有這一個樣本，所以列在 DRAFT。

**指令列**：自由移動時畫面最下面是
`AREA CAST VIEW ENCAMP SEARCH LOOK`。remake 這一列由 `drawCommandBar` 畫
（`adventureCommandList` 決定內容，spec 119）。

## 進位

時鐘是逐位的數，進位上限表在 `DS:35D4h`（spec 114 已解，紮營用同一份）：
分的個位滿十進到十位、十位滿六進到時。`GameTime.Normalise` 就是
overlay-20 `02B1h`。

## 契約

1. 狀態列格式固定 `%d, %d %s %02d:%02d`，朝向字母由 spec 076 換算。
2. AC 顯示檯面值（`60 − 內部值`，spec 080）；算不出來的保留建角基礎值，
   不顯示假的 0。
3. 走成功的一步呼叫 `advanceGameMinute()`：分的個位加一再正規化。
   轉向、被擋下來、腳本自己搬動隊伍（換圖）都不呼叫。
4. 時鐘存進 `Campaign.Clock`（`omitempty`），沒有這一欄的舊存檔載回來是
   `00:00`。

## 驗收

- `TestAdventureStatusLineMatchesTheDOSFormat`：四種朝向的格式。
- `TestGameClockAdvancesOneMinutePerStep`：一步一分，六十步進位到 `01:00`
  （那也是進位上限表的正對照）。
- `TestPartyPanelRowsUseTabletopArmourClass`：AC 是檯面值。

## 仍未閉合

- 換圖那一步為什麼不加分（只有一個樣本）。
- `AREA CAST VIEW ENCAMP SEARCH LOOK` 那一列的每一項行為。
- 時鐘除了顯示以外還被誰讀（原版有日夜、商店營業時間之類的用途）。
