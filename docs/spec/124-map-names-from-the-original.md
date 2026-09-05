# Spec 124：地圖的名字要從原版自己的文字取

狀態：READY（方法、工具、跨封存檔互相印證的那幾個配對）；
DRAFT（其餘二十幾個目的地區塊還沒逐一命名）。
日期：2026-09-06。

## 為什麼不能用關鍵字

spec 055 早就警告過：市議會派任務時會把**所有地名唸一遍**，所以
「某個區塊的文字裡出現 SLUMS」證明不了那個區塊是貧民窟。
spec 009 也因此寫「terrain ID、wall art selector 與地名是不同資料層，
本 inventory 不替它們猜名稱」。

## 站得住的形狀：同一條控制流上的配對

唯一不靠猜的形狀是——**先印一句話，緊接著 `20h NEWECL <區塊>`**。
文字與目的地在同一個分支裡，關係是原版自己建立的，不是我們比對出來的。

`cmd/pool-map-names` 就做這件事：

1. 每個 ECL 區塊跑 `TraceGraphAtBaseWithCommands` 取控制流圖。
2. 找 `NEWECL`，目的地是常數的才算。
3. **沿反向邊往回走**，收沿路指令會印出來的字（操作元裡的封包字串
   ＋ `VERTICAL MENU`／`HORIZONTAL MENU` 的選項），記下隔了幾道指令。

輸出在 [`docs/audit/pool-map-names.json`](../audit/pool-map-names.json)：
73 個換圖點、28 個目的地區塊、0 個追不完的區塊。

### ⚠ `Graph.Edges` 只有分支邊

共用 engine 的 `Graph.Edges` **不含循序邊**。只用它回走的話，
73 個 `NEWECL` 一段文字都收不到——**而那看起來像「原版沒有提示語」，
不像工具有洞**。循序邊要自己補：除了 `EXIT`／`GOTO`／`RETURN`，
每一道指令都會落到 `Next`。補上之後 18 個換圖點收得到前置文字。
`TestBackwardWalkNeedsSequentialEdges` 守住這一條。

## 讀出來的名字

判準是**跨封存檔互相印證**：同一個目的地被兩個不相干的腳本指到，
兩邊的提示語講同一個地方。

| 區塊 | 名字 | 證據（距離 ＝ 隔幾道指令）|
|---:|---|---|
| **0** | **費蘭的文明區**（`THE CIVILIZED AREA OF PHLAN`）| ecl5/7 `A903h`：`FINALLY YOU ENTER THE CIVILIZED AREA OF PHLAN.`［9］；ecl4/21 `998Fh`：`YOU BOARD A BOAT.`［6］、`DO YOU WANT TO TAKE A BOAT BACK TO PHLAN?`［12］ |
| 6 | 瓦爾耶沃城堡（門的另一邊）| ecl2/9 `A5A0h`：`BUGBEARS WERE WAITING ON THE OTHER SIDE OF THE GATE.`［6］|
| 19 | 洞穴 | ecl6/25 `9CB9h`：`YOU ENTER THE CAVE.`［2］|
| 25 | 據點外面的野外 | ecl6/28 `A3AEh`：`YOU ESCAPE FROM THE OUTPOST.`［1］；`A15Bh`：`TOSSED OUT INTO THE WILDERNESS`［3］；ecl6/19 `996Eh`：`YOU ENTER THE ENORMOUS CAVE, SEARCH IT AND FIND NOTHING…YOU THEN LEAVE.`［1］|
| 28 | 海盜據點 | ecl6/25 `9F9Fh`：`THE RIDERS ESCORT YOU INTO THE OUTPOST.`［2］|

**區塊 0 就是第一張地圖**（GEO3/0，spec 009 的初始 spawn）。
兩個互不相干的封存檔各自把它叫成「費蘭」／「費蘭的文明區」，
所以 spec 009 的「不因編號最小就自動命名」這一條**現在有答案了**——
名字不是猜的，是原版在跳過去之前自己說的。

## 還沒命名的

28 個目的地裡只有 18 個換圖點收得到前置文字，其餘的 `NEWECL` 前面
十四層之內沒有任何字（多半是純狀態判斷之後直接跳）。那些要另外找證據，
**不要拿別的區塊的文字去湊**。

目的地不是常數的 `NEWECL` 這一支看不到，那是掃描面的洞，不是「原版沒有」。
