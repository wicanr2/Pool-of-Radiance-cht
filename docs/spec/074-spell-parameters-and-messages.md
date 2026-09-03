# Spec 074：法術參數表與效果訊息

狀態：CONFORMED（表的位置與版面、`+0`..`+9`、`+0Ah`、`+0Eh` 十二個欄位的
用途、41 段效果訊息與它們屬於哪一支常式、`+8` 三種處置）；
DRAFT（`+0Bh`..`+0Dh`、`+0Fh` 四個欄位、共用施法常式後半段的另一個呼叫）。
日期：2026-09-03。

## 一支常式，六十七種行為

spec 073 那 67 支處理常式裡，絕大多數自己只做兩件事：把一段字面訊息複製到
區域變數，然後呼叫 `08BCh`。以 Protection From Evil（編號 6）為例，整支就是

```
110B  55 89 E5 83 EC 0E     push bp / mov bp,sp / sub sp,0Eh
110E  A0 79 67              mov  al, ds:6779h        ; 目前法術編號
1111  50                    push ax
1112  B0 00 50 ×4           push 0 四次
1121  8D 7E F3 16 57        lea  di,[bp-0Dh] / push ss / push di
1126  BF FE 10 0E 57        mov  di, offset "is protected" / push cs / push di
112B  9A 34 06 BB 05        lcall 05BBh:0634h        ; 字串指派
1130  0E E8 88 F7           push cs / call 08BCh
1134  89 EC 5D CB           mov sp,bp / pop bp / retf
```

`08BCh` 收七個 word（`retf 0Eh`）：法術編號、四個覆寫參數、訊息遠指標。
處理常式一律把四個覆寫參數推 0，差異全部來自法術編號。

## 差異來自 `DS:3194h` 的參數表

一筆 16 bytes，編號 0 到 67 共 68 筆；編號 0 那筆整筆是零，走不到。
表在 START.EXE 的資料段（`3194h + 30640 = 43476`，檔案 47936 bytes，所以它
是靜態初始化過的，不是 BSS）。

讀它的碼一律寫成「基底加欄位位移」，例如 `[di+3196h]` 就是欄位 `+2`；
下面的欄位編號都已經換算回 `3194h`。整張表在
[`docs/audit/pool-spell-dispatch.json`](../audit/pool-spell-dispatch.json)
的 `parameters` 欄裡逐筆留著，含還沒解讀的欄位。

| 欄位 | 讀它的碼 | 用途 |
|---|---|---|
| `+0` | ov25 `270Ah`、`274Ch` | 0 牧師、1 法師、2 物品效果 |
| `+1` | ov15 五處、ov20 `0933h` | 法術等級 |
| `+2` | ov22 `0757h`、`0998h` | 基礎射程；FFh 另表示先擲命中 |
| `+3` | ov22 `0742h` | 每施法者等級再加的射程 |
| `+4` | ov22 `08A3h` | 效果的固定回合數 |
| `+5` | ov22 `088Eh` | 每施法者等級再加的回合數 |
| `+6` | ov13 `20F0h`、ov22 `07A3h` | 低四位是**目標模式**；ov22 只看它是不是零 |
| `+7` | ov13 `2241h`、`22CDh` | 範圍法術的形狀參數（依模式取低三位或低兩位）|
| `+8` | ov22 `095Fh` | 0 就不擲豁免 |
| `+0Eh` | ov13 `1E67h` | 沒得挑時的預設目標；為零就是施法者自己 |
| `+9` | ov22 `097Dh` | 豁免類別（spec 075）|
| `+0Ah` | ov22 `0A14h` | 掛進效果串列的效果碼（spec 069）|

### `+6` 的低四位是目標模式

overlay-13 的挑目標常式（`20ADh`）用它分派：

```
20e5  mov  al, [bp+0Ch]                ; 法術編號
20f0  mov  al, [di+319Ah]              ; 參數表 +6
20f4  and  al, 0Fh                     ; ← 目標模式
20fb  jne  ...                         ; 0 → 目標就是施法者自己
2112  cmp  ax, 0Fh                     ; 0Fh → 走 1E09h 的挑目標介面
21ff  cmp  ax, 8 / 0Eh                 ; 範圍那一組，設 DS:677Eh = 1
```

**分組本身就是語意證據**：

| 模式 | 支數 | 是什麼 | 樣本 |
|---:|---:|---|---|
| `0` | 21 | 作用在施法者自己 | 法術護盾、偵測魔法、尋找陷阱 |
| `4` | 30 | 挑一個目標 | 治療輕傷、燃燒之手、魅惑術 |
| `6`／`7` | 3 | 定身術那一族 | Hold Person |
| `8` | 1 | 直線 | 閃電束 |
| `9` | 5 | 以一點為中心的範圍 | 催眠術、臭雲術、解除魔法 |
| `0Ah` | 4 | **整邊** | 祝福、詛咒、急速、緩速 |
| `0Bh` | 2 | 大範圍 | 火球術 |
| `0Fh` | 1 | 挑目標介面 | 沉默術 |

模式 `0Ah` 那四支與**逐支讀出來的處理常式對得上**：祝福與詛咒的碼走的正是
`0F35h` 那條整邊的路（spec 098）。兩邊獨立解出來的東西落在同一組，
任何一格讀錯分組就會散掉。

## `+0`／`+1`：職業與等級

overlay-25 的 `26F8h` 用 `+0` 決定施法者等級要讀哪裡：0 讀角色記錄的 `+96h`
（牧師等級）、1 讀 `+9Bh`（法師等級）、2 直接給 12。分佈是牧師 25、法師 31、
物品 11——**25 + 31 = 56，正好是名稱表的筆數**（spec 068）。

`+1` 是法術等級。與照說明書整理的中文目錄逐筆比對，56 筆裡 55 筆相符；
唯一的差異是 **Restoration 在表裡是牧師第 7 級**，而它在說明書的法術章裡
根本沒有條目。隊伍的最高職業等級是 6（spec 071），第 7 級法術記不起來，
所以它是神殿服務而不是可施放的法術。目錄已依表更正。

### `+2`／`+3`：射程

`0723h` 那一段算的是 `射程 = +2 + +3 × 施法者等級`，單位是格；算出來是 0
而 `+6` 非零就墊成 1，`+2` 是 FFh 也是 1。戰術地圖外那個分支（`ds:6CB3h`
非零）把施法者等級當成 6。

解出來的數字與規則書逐條相符：Bless 6 格、Detect Magic 3 格、Hold Person
6 格、Silence 15' Radius 12 格、Fireball 10 加每級 1、Lightning Bolt 4 加
每級 1；Cure Light Wounds 與 Animate Dead 算出來是 0，靠 `+6` 墊成 1，
也就是碰觸。**Magic Missile 與 Sleep 的每級增量是 4，規則書是 1**——
原版就是這樣寫的，不改。

### `+2`：要不要先擲中

值為 FFh 的正好五個：Cause Light Wounds、Shocking Grasp、Cause Blindness、
Cause Disease、Bestow Curse。**規則書裡要碰觸到對手才生效的就是這五個**，
一個不多一個不少。走這條路時 `08BCh` 在 `099Eh` 依序呼叫
`010Ah:0043h`、`0100h:002Fh`（帶常數 `0Bh`）與 `0100h:003Eh`（帶對手記錄的
`+111h`，也就是 spec 051 的內部 AC），結果為零就把效果整個取消。

### `+4`／`+5`：持續回合數

`0875h` 那一段就是 `持續 = +4 + +5 × 施法者等級`：

```
0875  al = 法術編號
0879  dx = 施法者等級          ; 010Ah:00D4h
088D  al = [di+3199h]          ; +5
0893  dx = +5 × 等級
08A2  al = [di+3198h]          ; +4
08A8  ax = +4 + dx
```

解出來的數字與規則書逐條相符：Bless 固定 6 回合、Find Traps 固定 30、
Protection From Evil 每級 3、Shield 每級 5、Mirror Image 每級 2、
Blink 與 Prayer 每級 1、Sleep 每級 5、Resist Cold 與 Enlarge 每級 10、
Strength 每級 60、Haste 與 Slow 是 3 加每級 1。兩個欄位都是零的
（Invisibility、Charm Person、Animate Dead）不自己結束。

### `+8`／`+9`：豁免

`+8` 決定要不要擲，0 就不擲。分佈是 0 共 52 個、1 共 9 個、2 共 4 個、3 共 2
個。不擲的那 52 個包含 Magic Missile 與 Sleep——**這兩個在 AD&D 裡正好都不給
豁免**，而 Fireball（2）、Lightning Bolt（2）、Hold Person（1）都要擲。
非零的值還會一路傳給 `0100h:007Fh` 與 `0100h:0084h`，所以它不只是布林。

**三種處置讀出來了**，在 overlay-24 entry 19（code `133Ah`，段 `0100h` 的
stub 表第 19 格 `007Fh`）。共用施法常式 `08BCh` 先在 `096Bh` 擲一次豁免
（overlay-24 entry 7），把布林結果、規則值、傷害與目標一起交給它：

```
133A  push bp; ...
1341  al = [bp+0Ah]            ; 傷害
1344  ds:6776h = al
1354  cmpb $0, [bp+6]          ; 豁免成功嗎
1358  je  137A                 ; 沒成功 → 傷害不動
135A  al = [bp+8]              ; 規則值
135D  cmp $1 ; jne 1368
1361  ds:6776h = 0             ; 規則 1：完全無效
1368  cmp $2 ; jne 137A
136C  ds:6776h = ds:6776h / 2  ; 規則 2：減半（整數除法）
137A  ds:6776h 是 0 就整段跳過
```

| 規則 | 豁免成功之後 | 表裡是誰 |
|---:|---|---|
| 0 | 不擲 | 52 支，含 Magic Missile、Sleep |
| 1 | 完全無效 | 9 支，全是狀態類：Charm Person、Reduce、Hold Person ×2、Ray of Enfeeblement、Cause Blindness、Cause Disease、Bestow Curse、編號 61 |
| 2 | 傷害減半 | 4 支：**Fireball、Lightning Bolt**、編號 60、編號 64 |
| 3 | 傷害不動 | 2 支：Silence 15' Radius、Stinking Cloud（兩支本來就不造成傷害）|

規則 2 剛好落在火球術與閃電術上，規則 1 剛好全是狀態類——這與規則書的
「豁免減半」「豁免無效」一致，是這段反組譯的語意交叉核對。

`+9` 是**豁免類別**，索引角色記錄 `+6Dh` 起那五個目標值（spec 075）。
67 個編號裡 65 個是 4（對法術），只有兩個是 0（癱瘓／毒／死亡）：
Stinking Cloud 與編號 61 的癱瘓效果。**規則書裡毒霧要擲毒、癱瘓要擲癱瘓**，
兩個例外都例外在該例外的地方。

### `+0Ah`：效果碼

值為零的正好是十九個「打完就結束、不留狀態」的編號，具名的十三個是
Cure／Cause Light Wounds、Burning Hands、Magic Missile、Shocking Grasp、
Knock、Cure Blindness、Cure Disease、Dispel Magic ×2、Remove Curse、
Fireball、Lightning Bolt、Restoration。留下狀態的一個都不在裡面——
**Cure Blindness 在（它是把狀態拿掉）、Cause Blindness 不在（它留下失明，
碼 `21h`）**。

非零的值就是掛進角色效果串列（spec 069）的效果碼：Bless `01h`、Curse `02h`、
Detect Magic `05h`、Protection From Evil `08h`、Protection from Good `09h`、
Hold Person `34h`、Sleep `35h`、Haste `27h`、Slow `2Ah`、Strength `26h`。
掛之前先經過 overlay-22 的 `07C7h`（`08BCh` 在 `0A34h` 呼叫，帶編號與碼），
那一支還沒讀完。

## 效果訊息

41 支處理常式各自帶一段 Pascal 字串，就放在自己第一個 byte 的前面。
往回找長度 byte 這個手法會有偽陽性，所以**每一段都再驗一次「本體真的用到
它」**（字串位址要以 16-bit 立即數出現在該常式的碼裡）：41 段全部通過，
沒有一段落空。

| 編號 | 法術 | 訊息 |
|---:|---|---|
| 1 | Bless | `is Blessed` |
| 2 | Curse | `is Cursed` |
| 5, 11, 18, 22, 29 | 偵測系五個 | `is affected` |
| 6, 7, 16, 17, 52, 53, 54 | 防護系七個 | `is protected` |
| 21 | Sleep | `falls asleep` |
| 23, 49 | Hold Person | `is held` |
| 30, 50 | Invisibility | `is invisible` |
| 34 | Stinking Cloud | `Creates a noxious cloud` |
| 43 | Remove Curse | `'s item is un-cursed` |
| 48 | Haste | `is Hasted` |

沒有訊息的十二支，全部是傷害或治療類——它們印的是數字，不是狀態。

### 57..67 這十一個編號是物品效果

`+0` 全部是 2，施法者等級固定 12，`+1` 全部是 6。名稱表沒有它們，但訊息與
效果碼有：

| 編號 | 訊息 | 效果碼 |
|---:|---|---|
| 57 | `is Speedy` | `27h`（與 Haste 同碼）|
| 58 | `is Healed` | `00h` |
| 59 | `is stronger` | `26h`（與 Strength 同碼）|
| 61 | `is paralyzed` | `34h`（與 Hold Person 同碼）|
| 62 | `is Healed` | `27h` |
| 63 | `is invisible` | `47h` |
| 64 | （共用 Fireball 的常式）| `00h` |
| 67 | `is Reading` | `04h` |

**呼叫端還沒找到**，所以哪個物品對到哪個編號仍是 DRAFT。

## 八個還沒解讀的欄位

`+6`、`+7`、`+0Bh`..`+0Fh` 都有人讀，讀的地方在 overlay-09、
overlay-13、overlay-19 與 overlay-22——不只在施法路徑上，所以它們的語意要到
那些呼叫端去讀，不能從施法常式推。

| 欄位 | 讀它的地方 |
|---|---|
| `+6` | ov13 四處、ov22 `07A4h`（只知道非零時會把射程墊成 1）|
| `+7` | ov22 `0AC9h`、`0C40h` |
| `+0Bh` | ov13 `2451h`、ov19 `1BF5h` |
| `+0Ch` | ov13 `24ABh` |
| `+0Dh` | ov09 `0300h` |
| `+0Eh` | ov09 `031Dh`、ov13 `1E68h` |
| `+0Fh` | ov09 兩處、ov13 兩處 |

## 順帶解出來的東西

- `DS:6B88h` 是施法目標的個數，`DS:6B89h` 起是一串遠指標（一格 4 bytes，
  1 起算）。它就接在派發表（`6A7Ch..6B87h`）後面。
- `DS:6779h` 是目前法術編號，處理常式與 `08BCh` 都讀它。
- `DS:6777h` 由 `08BCh` 在開頭依第三個參數設定、結束時清零。
- Bless（編號 1）不直接呼叫 `08BCh`，先經過 `0F35h`：那一支用角色記錄的
  `+10Eh` 比對敵我，把不同陣營的目標從清單上清掉，再呼叫 `08BCh`。
  所以 `+10Eh` 是陣營 byte。
- 角色記錄 `+96h` 是牧師等級、`+9Bh` 是法師等級，與 spec 070 的職業陣列
  （`+96h..+9Dh`）一致。
