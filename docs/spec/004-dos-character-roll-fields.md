# DOS 角色擲值與持久欄位

狀態：READY（建角擲值、年齡、能力修正、Gold、HP 與持久欄位）
日期：2026-08-31

## 輸入與位址空間

- `START.EXE`：47,936 bytes，SHA-256
  `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`。
- `GAME.OVR`：232,379 bytes，SHA-256
  `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638`。
- 通用 TPOV parser 得到 38 overlays、774 entries、223,317 code bytes、4,529
  relocation offsets。下列位址都是 **DOS `GAME.OVR` 解出的 overlay-local offset**，
  不是 file offset、MZ linear address 或 runtime segment:offset。
- IDA Pro `9.4`，image `ida-pro-9.4-idapython:locked-v1`；UID/GID `1000:1000`。
  最小探針以 overlay-00 SHA-256
  `6a284dad2159b70e6f877c11dfa8c2e16d5dac57956b2ac8fb477b06f4316631`
  驗證輸出 schema、輸入雜湊、非空檔案與擁有權後，才分析角色 overlays。

## 模組定位

- overlay-16：17,440 code bytes，SHA-256
  `a142d8a8f3b3c46a7755cf231228e7981c5b9f77721313ce79ab6c0aec105b94`。
  raw Pascal strings 位於 `04DEh` `Pick Race`、`0503h` `Pick Alignment`、
  `0532h` `Keep this character?`、`0548h` `Character name:`，故可證實它擁有
  正常建角流程；不以 IDA 自動函式名承載語意。
- overlay-19：11,004 code bytes，SHA-256
  `4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9`。
  Pascal strings 從 `0000h` 起包含 `(NPC)`、`Age`、六能力名稱，`05C4h` 起包含
  `HP`、`THAC0`、`Damage`。它是角色資料顯示 consumer 的強推論，仍須逐欄閉合。

IDA 由 TPOV entry seeds 加保守 sweep 匯出 overlay-16 38 functions、overlay-19
41 functions。部分 entry seed 落在既有 code/data 邊界而未建立（分別 1、8 筆），
所以這份函式數是目前可重生清冊，不可冒稱完整語意函式全集。

## 已證實的直接存取與欄位候選

以下有原版完成角色 `.CHA` 單變因／樣本 bytes 與 overlay 直接存取；各列的語意
等級仍分開標示，不因位址已證實就自動升格：

| CHA offset | 形狀 | 證據 | 結論 |
| --- | --- | --- | --- |
| `10h..15h` | 六個 byte | overlay-16 以能力索引加到 record base，再讀寫 `[es:di+10h]`；原版角色樣本逐 byte 對應畫面六能力順序 | STR、INT、WIS、DEX、CON、CHA；`exact` |
| `30h` | little-endian word | overlay-16 `0F23h／0F86h／0FD9h／107Fh／1178h／11CAh` 寫 `[es:di+30h]`；overlay-19 `0184h` push 同一 word 到顯示鏈 | age 欄位；`exact` 欄位，公式 DRAFT |
| `32h` | byte | overlay-16 `1CC6h` 初始化、`1D3Ch` 從 `B1h` 複製，後續依 modifier 調整；同一次 Elf／Thief 資料頁顯示 HP `7`，最終 `.CHA +32h` 也是 `7` | 畫面 HP；`exact` |
| `8Eh` | little-endian word | overlay-16 `1D00h..1D17h` 計算後寫 `[es:di+8Eh]`；同一次資料頁顯示 GOLD `100`，`.CHA +8Eh` 是 word `100` | 畫面 Gold；`exact`，產生公式 DRAFT |
| `B1h` | byte | overlay-16 `1D2Ch` 寫入 local `4209h` 回傳值，`1D34h` 讀回；同一角色為 `5`，其畫面 HP 是 `7`；`1DDEh..1DE1h` 另依 divisor 調整 | 未套完整 modifier 的 HP roll／class HP accumulator；`strong inference`，精確公式 DRAFT |
| `11Bh` | byte | overlay-16 `1DBEh..1DC5h` 複製調整後 `32h`；同一角色兩者同為 `7` | max/current HP 鏡像候選；`strong inference` |

`1D00h..1DE1h` 明確分成 Gold 與 HP 兩條鏈：前段把計算結果寫入 word `8Eh`；
local `4209h` 的回傳再寫 `B1h` 並複製到 `32h`，local `3F01h` 的 signed 結果只
調整 `32h`，最後複製至 `11Bh`，另對 `B1h` 作 divisor 處理。未閉合 `4209h`、
`3F01h` 的 caller contract、表格與 divisor 來源前，不把骰法寫成 READY。

### 2026-08-31 勘誤

前一版因只有未配對樣本，把 `+32h` 推作金錢單位、`+B1h` 推作畫面 HP。新的同源
runtime anchor（資料頁 SHA-256
`865e4612c68a93aeacf712d6f307fc3d5f871dbf3ac7cfbf00aa5f26eb5adabc`；CHA
SHA-256 `cc8febdd1f9f8c2dc0ee7c752bddca90b1960b0b9cce8a33f6cdb19f66c9471a`）
直接否定舊解釋：HP `7`=`+32h`，Gold `100`=word `+8Eh`，而 `+B1h`=`5`。
舊推論形成原因保留於此，後續不得再引用它。

## READY 公式與執行順序

下列均由 IDA Pro 9.4 對 overlay-16 的指令、resident data segment 原始表格與
同源畫面／CHA anchor 交叉驗證，位址仍是 overlay-local；resident 表格則標為
`DS:`，不可混作同一位址空間。

### 年齡

`0EC6h..11CAh` 以 `race*28 + class*4 + DS:3DD3h` 取四 byte 記錄：little-endian
基準年齡、骰數、骰面。單職年齡是 `base + roll(count,sides)`。多職代碼不呼叫
亂數 helper，而取主導職業記錄的 `base + count*sides`：`8／9／11／12` 用 Cleric，
`13／15／16` 用 Magic-User，`14` 用 Thief。這個「多職取最大值」是原版指令契約，
不是現代規則推測。六種可選種族的完整表已逐 byte 固化於
`internal/creation/rolls.go`；未出現在本作選單的 race／class 記錄不授權 UI。

### 六能力與限制順序

`11EBh..12B8h` 先對 STR、INT、WIS、DEX、CON、CHA 各擲 `3d6`，再套種族修正：
Dwarf `CON+1／CHA-1`；Elf `DEX+1／CON-1`；Halfling `STR-1／DEX+1`。程式也有
race code 6 的 `STR+1／CON+1／CHA-2`，但該種族不在 Pool 選單，只保留為證據，
不對玩家暴露。

其後 `12BBh..1BB7h` 依序套用：年齡修正 → `DS:3D1B` 種族／性別最小最大值 →
`DS:3EF9` 的 `classCode*6+ability` 職業最低值。Cleric 多職集合（Pascal set bytes
位於 overlay `0512h`，本作可見代碼為 `8／9／11`）另把 WIS 提至 13。STR 最終為
18 且 Fighter level slot `+98h>0` 時擲 `1d100` exceptional strength，再依
`DS:3D1F + gender` 的種族／性別上限截斷。年齡臨界值來自 `DS:3EA9` 每種族
五個 word；本函式使用前四個，最後一個是壽命界線而非這段能力修正輸入。

### Gold

`1C10h..1D17h` 對每個 active class slot 擲該職業起始金錢，再計算
`sum*10/classCount`（整數除法）寫入 word `+8Eh`：Cleric／Druid `3d6`，
Fighter／Paladin／Ranger `5d4`，Magic-User `2d4`，Thief `2d6`，Monk `5d4`。
Elf／Thief runtime anchor 的畫面 Gold 100 與 `+8Eh=100` 對應。

### HP 與 CON

local `4209h..42BEh` 對 active class slot 擲 hit dice：Cleric `1d8`、Druid `1d8`、
Fighter `1d10`、Paladin `1d10`、Ranger `2d8`、Magic-User `1d4`、Thief `1d6`、
Monk `2d4`。一級單一職業骰若小於 `floor(2*sides/3)` 就提升至該值；各職業
結果先相加寫 `+B1h`。

local `3F01h..3F80h` 對每個 active class 依 CON 加 modifier：CON 3=`-2`、
4..6=`-1`、7..14=`0`、15=`+1`、16+=`+2`。只有 class code 恰為 pure Fighter
（`2`）才於 CON 17 再 `+1`、CON 18 再 `+1`；多職 Fighter 不取得這兩級額外值。
caller `1D31h..1DE1h` 的畫面 HP 是
`max(1,(sumHitDice+sumConModifiers)/classCount)`，寫 `+32h` 並鏡像到 `+11Bh`；
最後 `+B1h` 改寫為 `sumHitDice/classCount`。所有除法皆為原版整數除法。

## 實作邊界

本規格授權注入式 dice roller、上述原版表格與純資料角色生成器。畫面上的
重擲／KEEP 狀態、把結果完整序列化進 CHA、AC／THAC0／damage 等衍生欄位仍須各自
沿垂直鏈閉合，不因本規格 READY 就宣稱完整建角已完成。
