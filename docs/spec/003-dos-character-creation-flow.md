# Spec 003：DOS 建角流程

狀態：DRAFT；日期：2026-08-31。

## Oracle 環境勘誤

`POOL.CFG` 以 bytes 明確指定資料根目錄為 `C:\POOLRAD\`。DOSBox 必須 mount
該目錄的父層為 `C:`，再 `CD POOLRAD`；把檔案直接放在 `C:\` 時，標題與主選單
仍能顯示，但建角資料頁會錯誤要求插入 disk 3，且能力值全為 0。這是驗證環境
缺陷，不是遊戲規則。`tools/capture-dos-oracle.sh` 已依此修正。

## 已觀察的預設垂直鏈

從主選單按 `C` 進入 CREATE NEW CHARACTER，逐層接受預設項目：

1. Race：Dwarf、Elf、Gnome、Half-Elf、Halfling、Human；預設 Dwarf。
2. Gender：Male、Female；預設 Male。
3. Dwarf Class：Fighter、Thief、Fighter/Thief；預設 Fighter。
4. Alignment：Lawful／Neutral／Chaotic × Good／Neutral／Evil 九種；預設
   Lawful Good。
5. 角色資料頁顯示隨機 age、六能力值、gold、AC、THAC0、HP、damage、
   encumbrance、movement、狀態與頭像，並詢問 `KEEP THIS CHARACTER? YES NO`。
6. 接受後要求 `CHARACTER NAME:`；提交非空名稱後顯示
   `MAX BONUS? YES NO`。

正確掛載下的畫面證據位於
[`docs/reference/original-dos/character-flow/`](../reference/original-dos/character-flow/)。
擲值具有隨機性，畫面只證明欄位、順序與一次 runtime 樣本，不把樣本數字寫成
固定規則。

## 未閉合／禁止先猜

- `MAX BONUS?` 的可靠自動輸入與其資料語意。
- 其後戰鬥 sprite 的 ready／action 選擇、六部位調色、角色檔 offsets。
- Race／Class 組合限制、擲值與 age／gold／HP 公式。
- 返回上一層、取消、重擲與完成建角後加入隊伍的完整按鍵狀態機。

必須先從原版繼續走到 sprite／調色並配合 executable consumer／角色存檔差分，
本規格才可升為 READY。remake 不得只依 CoAB 建角流程外推 Pool 行為。
