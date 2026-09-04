# Spec 008：建角完成、角色庫與 Party Creation Menu

狀態：READY（原版畫面順序、remake 自有保存契約、DOS 三檔匯出）
日期：2026-09-05

## 原版證據

- 原版正常 UI 的 Party Creation Menu 截圖 SHA-256：
  `22bcb0126e52be62b397a18b64f06542ddfc346aa229d1efefc1503bf4932c71`。
  畫面依序列出 `CREATE NEW CHARACTER`、`ADD CHARACTER TO PARTY`、
  `LOAD SAVED GAME`、`EXIT TO DOS`，底部提示 `CHOOSE A FUNCTION`。
  那是**空隊伍**的畫面，不是完整清單——見下一節。
- **隊伍非空的同一張選單**（2026-09-05 由 `tools/capture-dos-adventure.sh`
  在 DOSBox 內走到，`docs/reference/original-dos/adventure/04-party-menu-with-party.png`，
  SHA-256 `4230ad4f0d7f29c1bc766472b6d05cb0a5832f986ac3f1d098930dc5189e9f17`）
  **只有九項**，順序是 `CREATE NEW CHARACTER`、`DROP CHARACTER`、
  `MODIFY CHARACTER`、`VIEW CHARACTER`、`ADD CHARACTER TO PARTY`、
  `REMOVE CHARACTER FROM PARTY`、`SAVE CURRENT GAME`、`BEGIN ADVENTURING`、
  `EXIT TO DOS`。**沒有 `LOAD SAVED GAME`，也沒有 `TRAIN CHARACTER`。**
  那張圖的狀態是「剛用 `C)REATE` 建好一名 1 級角色、`A)DD` 進隊伍，還沒存過檔」。
- 原版 Rule Book 的「Creating A Party Of Characters」明寫最多 6 名 player characters，
  遊戲內總共可控制 8 名，另兩格保留給途中加入的 NPC：
  <https://www.mocagh.org/ssi/pool-manual.pdf>。目前 remake `party` 只保存玩家角色，
  因此 fail-closed 上限是 6；NPC slots 待事件／招募垂直鏈另行實作。
- Spec 003 的 runtime 路徑已觀察：icon 最終確認 `YES` 後產生同名 `.CHA/.SPC`，
  再回到上述選單。建角不等於自動加入隊伍。
- overlay-16 local `1E79h` 呼叫 portrait editor、`1E7Dh` 呼叫 icon editor；返回後
  `1E81h..1E96h` 組字串並呼叫外部 exporter，最後釋放 285-byte 暫存 record。
- 33 份原版 UI 角色檔全部是 285 bytes。`.SPC` 不是每角色固定 blob：本機 corpus
  有 9／18／36 bytes，且每 9 bytes 是一個鏈節點；同一 DOS session 的多個角色可
  共享相同首節點。不能把 `BASE.SPC` 複製給新角色。

## 十一個指令與它們的可見規則

中文說明書 p.8..p.10（`docs/reference/manual/manual-vol2.md`）逐項介紹了
人物管理選擇項，按敘述順序是：

| 鍵 | 指令 | 說明書怎麼寫 |
|---|---|---|
| C | CREATE NEW CHARACTER | 創造一名人物，存放到「人物名單」內 |
| D | DROP CHARACTER | 從人物名單中**永遠**除掉，一切資料都消失；會再確認一次 |
| M | MODIFY CHARACTER | 重新調整屬性與生命力；經驗點數必須為 0、身上不能有金錢以外的物品 |
| T | TRAIN CHARACTER | 花 1000 金幣昇級 |
| V | VIEW CHARACTER | 檢視資料，以及交換錢幣或裝備 |
| A | ADD CHARACTER | 從人物名單加入隊伍，最多 6 人 |
| R | REMOVE CHARACTER FROM PARTY | 移回人物名單，人還在 |
| L | LOAD SAVED GAME | 叫出存下的進度（原版 A..J 十個位置）|
| S | SAVE CURRENT GAME | 存下目前進度（同上十個位置）|
| B | BEGIN ADVENTURING | 離開選擇項，開始冒險 |
| E | EXIT TO DOS | 跳回 DOS |

**可見規則**（p.8「由於第一次進入遊戲的你沒有任何人物，因此只會有其中的
四項」、p.9「若你未將任何人物加入隊伍，則大部份的人物處理選擇項都將不會
出現」）：隊伍是空的時候只留 C、A、L、E ——正好就是上一節那張截圖上的四項，
兩份證據互相印證。

**隊伍有人的時候是另一組**，而它不是「十一項全開」：原版截圖上是
C、D、M、V、A、R、S、B、E 九項。L 與 T 兩項只在這一組裡消失。

- **L)OAD 的規則已經照原版接上**：`emptyOnly`，隊伍有人就不顯示也不接受按鍵。
  兩張截圖一正一反（空隊伍有、非空沒有），這一條是**已證實**。
- **T)RAIN 是已知的偏差**：原版兩張截圖都沒有它。原版每個指令各有一個啟用旗標
  （`T` 的在 `DS:06D4h`，spec 097），那個旗標從哪裡設還沒讀出來，所以分不出
  「一律不顯示」與「有條件才顯示」（例如經驗值足夠才亮）。remake 目前一律顯示
  ——藏起來玩家就昇不了級。讀出旗標來源之前不要當成照原版接的。

D、M、T、V、R 五項要先指定對象；remake 以 1..6 選人，選中的那一位在右欄
標上 `>`。看不見的指令連按鍵都不接受：畫面上沒有那一列卻按得動，等於多開了
一個原版沒有的入口。

**D 與 R 的差別是這一節最容易做錯的地方**：D 是「從人物名單中永遠除掉」，
隊伍與名單兩邊都要不見；R 只是「移回人物名單」。兩個都從隊伍拿掉一個人，
畫面上看起來一樣。

`TestEmptyPartyShowsOnlyTheFourOriginalEntries`、
`TestHiddenPartyMenuEntriesIgnoreTheirKey`、
`TestDropAsksFirstThenClearsBothLists`、
`TestRemoveKeepsTheCharacterInTheRoster` 與
`TestModifyRefusesExperiencedOrLoadedCharacters` 釘住這幾條。

### 與原版的差異

- 原版的 L 與 S 有 A..J 十個存檔位；remake 只有一份 JSON，所以 S 就是把它
  寫出去、L 就是讀回來。十個位置要等 remake 的存檔管理有介面再談。
- V 開的是 remake 的裝備畫面（`cmd/pool-game/equipment.go`）。原版的
  VIEW CHARACTER 版面還沒反組譯，所以那個畫面不宣稱與原版一致。

## Remake 保存契約

在 DOS exporter 的角色主體全部衍生欄位，以及 `.SPC` session 鏈 key／生命週期尚未
閉合前，remake 使用作品專屬、版本化 JSON：

- 本規格實作當時 schema 固定為 `pool-remake-state/1`；現行 schema 4 與 1→2→3→4
  遷移由 Spec 018／035／037 管理，未知版本仍 fail-closed。
- `character_library` 保存已完成但尚未入隊的角色；`party` 保存已加入角色。
- 每個角色保存姓名、race／gender／class／alignment IDs、擲值結果、portrait selectors、
  combat icon selectors、size 與六部位雙色。
- 寫檔採同目錄 temporary file → rename，避免 F10 自動保存留下半份 JSON。
- 建角確認後先寫入角色庫，保存成功才回 Party Creation Menu；保存失敗留在確認頁並
  顯示錯誤，不得假裝角色已建立。
- F10 在可保存狀態先保存再離開。`LOAD SAVED GAME` 只接受現行或明訂可遷移的 schema；它不是 DOS
  `.CHA/.SPC` importer，也不得標示成原版相容存檔。

## DOS 三檔匯出

原版在 icon 確認 `YES` 之後寫出角色檔，再回到 Party Creation Menu
（spec 003 第 11 步）。remake 在同一個時機做同一件事，實作是
`internal/character/export.go` 加 `cmd/pool-game/dos_export.go`。

三個檔與原版預設人物的形狀相同：285 bytes 的角色記錄、63 的倍數的物品鏈、
9 的倍數的效果串列（spec 069）。副檔名 `.CHA` 與 `.SPC` 有 spec 003 的原版證據；
`.ITM` 是由預設人物三個一組的檔名（`chrdatd4.sav`／`.itm`／`.spc`）推的，
屬**強推論**。

### 契約

1. **只寫有出處的欄位。** 匯出是「疊在一份 base 上」，不是「從零組一份」；
   remake 沒解出來的位元組原封不動留著。NPC 疊在自己的 285-byte `Record` 上，
   remake 自己建的角色疊在 spec 063 的建角基礎值上
   （`character.NewDOSRecordBase`）。
2. **疊完一定要重算。** remake 的角色模型只存輸入（能力值、等級、背包），
   THAC0、AC、負重與移動力是現算現用的。不跑 `gamepack.RecomputeCombatFields`
   （overlay-25 `0E36h`，spec 063）就會寫出一份戰鬥數值全是 0 的記錄——
   那份檔案載得進原版，但那個人打不到東西。
3. **物品照抄。** remake 重建不出那 63 bytes，`Raw` 長度不對就失敗，不補零。
4. **效果節點只寫 `+0` 的代碼。** `+1`..`+4` 在 spec 069 仍是 DRAFT，
   remake 的存檔模型也沒有存它們；`+5..+8` 的遠指標是上次執行的位址，
   重新載入沒有意義，走訪照檔案順序。
5. **匯出失敗不影響角色。** remake 的真相是自己的 JSON 存檔，三個檔是匯出。

### 驗收

`TestExportDOSRecordReproducesThePremades` 把七名原版預設人物讀進 remake 的
角色模型再寫回去，285 bytes 一個位元組都不差；接著只改現在生命，差異必須
正好落在 `+11Bh` 一格——否則「輸出等於輸入」可能只是因為根本沒寫。

### 仍未閉合

- `+6Dh..+71h` 的豁免表與 `+73h` 的生命骰：remake 只從記錄讀，沒有從職業與
  等級算的產生端，所以匯出時保留 base 的值。從零建的角色那五格是 0。
- `+0ABh` 是 `random(100h)`，語意未閉合，不假造定值。
- `+76h` 驅散不死欄：七名預設人物都是 0（含一名 6 級牧師），寫入端還沒找到。
