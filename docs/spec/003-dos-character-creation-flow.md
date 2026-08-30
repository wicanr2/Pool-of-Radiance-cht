# Spec 003：DOS 建角流程

狀態：READY（建角畫面順序、portrait／combat icon 與 285-byte CHA icon 欄位）；
日期：2026-08-31。能力擲值、年齡、金錢與職業限制公式仍是 DRAFT，不在本次實作
授權內。

操作契約同時對照 IBM PC Quick Start Card（垂直項目用 Home／End＋Enter；水平
命令按畫面白色字母）與原版 Rule Book 的 ICON 章：
<https://dosdays.co.uk/media/games/pool/PoolOfRadiance-ReferenceCard-PC.pdf>、
<https://www.mocagh.org/ssi/pool-manual.pdf>。實際選單、輸出 bytes 與順序仍以本
DOS build 的 runtime capture 為準。

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
6. 接受後要求 `CHARACTER NAME:`；提交非空名稱後進入 portrait editor：
   `HEAD / BODY / KEEP`。`H`、`B` 逐次換圖，`K` 接受。
7. combat icon editor 同時顯示 OLD／NEW 的 READY 與 ACTION：
   `PARTS / COLOR-1 / COLOR-2 / SIZE / EXIT`。
8. Parts 是 `HEAD / WEAPON / EXIT`；選 Head 或 Weapon 後以
   `NEXT / PREV / KEEP / EXIT` 循環並接受候選。Weapon 同時控制 body shape。
9. Color-1 是 `WEAPON / BODY / HAIR / SHIELD / ARM / LEG / EXIT`；Color-2
   將 HAIR 換成 FACE，其餘相同。每個部位以 `NEXT / PREV / KEEP / EXIT` 選色。
10. Size 對矮人顯示 `LARGE / KEEP / EXIT`；選 Large 後新 READY／ACTION 同時放大。
11. 頂層 Exit 顯示 `IS THIS ICON OK? YES NO`；Yes 寫出 `<NAME>.CHA` 與
    `<NAME>.SPC`，再返回 Party Creation Menu。

### 原版 Race → Class 清單

| Race | 依畫面順序的 Class |
|---|---|
| Dwarf | Fighter；Thief；Fighter/Thief |
| Elf | Fighter；Magic-User；Thief；Fighter/Magic-User；Fighter/Thief；Fighter/Magic-User/Thief；Magic-User/Thief |
| Gnome | Fighter；Thief；Fighter/Thief |
| Half-Elf | Cleric；Fighter；Magic-User；Thief；Cleric/Fighter；Cleric/Fighter/Magic-User；Cleric/Magic-User；Fighter/Magic-User；Fighter/Thief；Fighter/Magic-User/Thief；Magic-User/Thief |
| Halfling | Fighter；Thief；Fighter/Thief |
| Human | Cleric；Fighter；Magic-User；Thief |

六張同狀態畫面在
[`docs/reference/original-dos/character-classes/`](../reference/original-dos/character-classes/)。
這是原版 menu corpus，不由現代規則推導；順序也屬 UI 契約。

正確掛載下的畫面證據位於
[`docs/reference/original-dos/character-flow/`](../reference/original-dos/character-flow/)。
擲值具有隨機性，畫面只證明欄位、順序與一次 runtime 樣本，不把樣本數字寫成
固定規則。

## CHA icon 欄位（exact）

六個角色於同一 DOS session 由原版 UI 建立，每個只改一項；另以 ALLONE／
ALLTWO 將六部位各前進一色。所有 CHA 固定 285 bytes。差分得到：

| offset | 語意 | 證據 |
|---:|---|---|
| `BDh` | combat icon head selector | HEAD：`00→01` |
| `BEh` | combat icon weapon／body-shape selector | WEAPON：`00→01` |
| `BFh` | 未知，必須原值保存 | 本批均為 `00`，不可命名 |
| `C0h` | size | Small=`01`；Large=`02` |
| `C1h` | Body colors | low nibble Color-1；high nibble Color-2 |
| `C2h` | Arm colors | 同上 |
| `C3h` | Leg colors | 同上 |
| `C4h` | Hair／Face colors | low=Hair（Color-1）；high=Face（Color-2） |
| `C5h` | Shield colors | low=Color-1；high=Color-2 |
| `C6h` | Weapon colors | low=Color-1；high=Color-2 |

另由原版 UI 完成六種族與性別／單職角色得到：`2Eh` 是 race code
（Dwarf=`1`、Elf=`2`、Gnome=`3`、Half-Elf=`4`、Halfling=`5`、Human=`7`）；
`2Fh` 是 class code（Cleric=`0`、Fighter=`2`、Magic-User=`5`、Thief=`6`）；
`9Eh` 是 gender（Male=`0`、Female=`1`）。多職 class code 已以 Half-Elf 正常 UI
逐項建立；每筆都先由資料頁確認職業名稱，再配對同一次寫出的 285-byte CHA：

| Class | `CHA +2Fh` |
|---|---:|
| Cleric/Fighter | `8` |
| Cleric/Fighter/Magic-User | `9` |
| Cleric/Magic-User | `11` |
| Fighter/Magic-User | `13` |
| Fighter/Thief | `14` |
| Fighter/Magic-User/Thief | `15` |
| Magic-User/Thief | `16` |

逐筆 SHA-256 與固定 record offset 見
[`docs/audit/dos-multiclass-codes.json`](../audit/dos-multiclass-codes.json)。這些代碼
是原版持久格式契約，不由空號或 components 排列推測。

基線 `C1..C6 = 91 A2 B3 C4 E6 F7`；六個 Color-1 各前進一格後為
`92 A3 B4 C5 E7 F8`，六個 Color-2 各前進一格後為
`A1 B2 C3 D4 F6 07`。這同時證明 nibble 方向與 `F→0` wrap。parser 必須要求
恰為 285 bytes；寫回 icon 時只准改 `BDh..C6h` 的已知欄位，`BFh` 原值保存。

## 尚未閉合／禁止先猜

- Race／Class 組合限制、擲值與 age／gold／HP 公式。
- 返回上一層、取消、重擲與完成建角後加入隊伍的完整按鍵狀態機。
- portrait head／body 的 CHA offsets；本輪只證明玩家可見循環與 KEEP。
- `BFh` 的語意，以及 Head／Weapon selector 的合法上限。

本規格只授權 typed CHA icon codec 與上述 UI state machine；完整角色生成規則須另以
executable consumer／更多角色檔差分閉合。remake 不得只依 CoAB 建角流程外推。
