# Spec 098：施法的共用機制（擲骰、施法者等級、處理常式的呼叫慣例）

狀態：CONFORMED（擲骰常式與引數順序、施法者等級的取法與那個覆寫旗標、
處理常式呼叫 `08BCh` 的形狀）；DRAFT（`08BCh` 自己在做什麼、四個覆寫參數的
語意、各法術的傷害）。日期：2026-09-03。

## 擲骰：overlay-24 的兩支

`0DE5h`（entry 8，stub `48h`）是全遊戲的擲骰：

```
def  cmpb $0, 0x8(%bp)          ; 顆數
df5  mov  al, 0x8(%bp)
e0a  mov  al, 0x6(%bp)          ; 面數
e0f  push ax
e10  lcall 05BBh:0C94h          ; Turbo Pascal 的 Random(n) → 0..n-1
e15  inc  ax                    ; → 1..n
e16  add  al, -0x3(%bp)
```

Pascal 由左往右推，所以 `[bp+8]` 是**先推的**、`[bp+6]` 是後推的：
**先推顆數、後推面數**，寫成 `Roll(顆數, 面數)`。

反面對照確認過這個順序：overlay-09 entry 1 推 `1` 再推 `8`，
接著 `cmp al, 8` 分岔。若順序相反就是 8d1，永遠等於 8，那條分支是死碼。

`0E30h`（entry 9，stub `4Dh`）是同一件事外加**把顆數記在 `DS:677Ah`**，
然後轉呼叫 `0DE5h`。法術的傷害走這一支，普通判定走 `48h`。

## 施法者等級：overlay-25 `26F8h`

引數是法術編號，回傳等級：

```
2709  mov  al, [di+3194h]              ; 參數表 +0（職業）
270d  cmp  al, 0 → 讀 DS:5CF0h→+96h    ; 牧師等級
271f  cmp  al, 1 → 讀 DS:5CF0h→+9Bh    ; 法師等級
2731  cmp  al, 2 → 12                  ; 物品效果一律 12 級
2739  cmpb $0, ds:6CB3h                ; 非零而且職業不是 2
2752  mov  byte ptr [bp-1], 6          ; → 等級一律當成 6
```

`DS:6CB3h` 由 **overlay-19**（人物檢視畫面：`Weapon`／`AC`／`THAC0`／
`Items`／`Spells`／`in Memory`／`in Spell Book`／`on Scroll`）在 `1AE2h`
與 `1BBDh` 設為 1、在 `1BDDh` 清回 0；overlay-22 的派發表初始化
（`33E1h`）也把它清成 0。所以**從人物畫面施法（戰術地圖之外）時，
施法者等級一律當成 6**，戰鬥中才用真正的職業等級。

## 處理常式怎麼呼叫 `08BCh`，四個覆寫參數各是什麼

`08BCh` 收七個 word（`retf 0Eh`）。由堆疊位移反推（先推的在高位）：

| 位移 | 推入順序 | 是什麼 | 誰用它 |
|---|---:|---|---|
| `[bp+12h]` | 1 | 法術編號 | 查參數表、取施法者等級 |
| `[bp+10h]` | 2 | **施法者等級的覆寫**，0 就用真的 | `08F2h` 的 `cmpb $0` 分岔 |
| `[bp+0Eh]` | 3 | 掛效果時一起傳出去的參數 | `0A3Dh` |
| `[bp+0Ch]` | 4 | **傷害**，0 就整段跳過 | `09DFh` 的 `cmpb $0` / `jbe` |
| `[bp+0Ah]` | 5 | 傷害的種類選擇，寫進 `DS:6777h` | `08E2h`，只在傷害非零時 |
| `[bp+6..8]` | 6、7 | 訊息遠指標 | `08C2h` 複製進區域變數 |

第二格是「施法者等級的覆寫」這件事由**鏡影術**釘住：`1A79h` 把 `Roll(1, 4)`
推在那一格而不是傷害那一格——鏡影的數量借等級那一格傳。

大多數常式四個覆寫參數都推 0，差異全部來自參數表（spec 074）。
會算傷害的那幾支就用第四格把結果傳進去。

Magic Missile（編號 15，`1429h`）是最短的一支：

```
1433  lcall 010Ah:00D4h                 ; 等級 = CasterLevel(編號)
1445  等級 ÷ 2                           ; 有號整數除法
1454  lcall 0100h:004Dh                 ; Roll(等級÷2, 4)
145d  等級 ÷ 2
1468  add  ax, bx                       ; 傷害 = (等級÷2)d4 + 等級÷2
146b  push 8                            ; 第四個覆寫參數
147e  call 08BCh
```

形狀就是 AD&D 的「N 發，每發 1d4+1」，**N ＝ 等級 ÷ 2 取整**。

### 未解：發數與說明書對不起來

說明書寫的是「魔法師每昇兩級可多擲出一個魔法飛彈，第 3 或第 4 級擲 2 個，
第 5 或第 6 級擲 3 個」，也就是 `(等級+1) ÷ 2`。
碼算的是 `等級 ÷ 2`：偶數等級兩者相同（6 級 3 發），
**奇數等級碼少一發，而且第 1 級算出來是 0 發、0 點傷害**。

三個可能的出口已經關掉兩個：

- **`08BCh` 裡沒有下限。** `09DFh` 是 `cmpb $0, [bp+0Ch]` / `jbe`——
  傷害是 0 就整段跳過，連套用都不做。
- **引數順序沒讀反。** Fireball（`262Eh`）推 `等級` 再推 `6`，
  說明書寫的正是「1-6 點 × 施術者等級」，兩邊逐字相符。
  同一支 `26F8h` 回的是**沒有加工的職業等級**。

剩下的可能是「原版就是這個行為，說明書寫得比較寬鬆」。
**這裡照碼接**：原始執行檔是一手資料，說明書是二手的，而 remake 要重現的是
遊戲的行為。`internal/gamepack.CastSpell` 算的就是 `等級 ÷ 2`，
測試把第 1 級的 0 傷害釘住，所以哪天驗出不同的結果會是紅的而不是靜悄悄的。

**決定性的實驗（還沒做）**：在原版拿一個第 1 級法師對敵人施魔法飛彈，
看傷害是不是 0。做完再回來改這一條與 `SpellIDMagicMissile` 那一段。

## Fireball：引數順序的正對照

Fireball（`262Eh`）的傷害是

```
26fd  mov  al, [bp-1]      ; 施法者等級
2700  push ax              ; 顆數
2701  mov  al, 6
2703  push ax              ; 面數
2704  lcall 0100h:004Dh     ; Roll(等級, 6)
```

說明書寫「其程度＝1-6 點×施術者等級」，兩邊逐字相符。這同時確認了
`Roll(顆數, 面數)` 的順序與「`26F8h` 回的是沒有加工的職業等級」。

## Fireball 的第二個入口

Fireball（`262Eh`）先設 `DS:677Eh = 1`，然後分兩條：法術編號等於 `40h`
時等級是 `Roll(1,3) × 2 + 1`（也就是 3、5、7 隨機），否則照 `26F8h` 取。
編號 `40h` 與 47 共用這支常式，所以那是另一個來源的火球（捲軸或魔杖）。
接著它把範圍交給 overlay-31 entry（`0138h:003Eh`）挑格子。

## 已經讀出來的十九支

| 編號 | 法術 | 位址 | 算法 |
|---:|---|---|---|
| `01h` | Bless | `0FF5h` | 推施法者的 `+10Eh`（哪一邊）給 `0F35h`，整邊掛效果 |
| `03h` | Cure Light Wounds | `1051h` | 治療 `Roll(1, 8)`，直接呼叫 `0100h:0089h`，不走 `08BCh` |
| `09h` | Burning Hands | `1178h` | 傷害＝施法者等級，沒有擲骰 |
| `0Fh` | Magic Missile | `1429h` | `Roll(等級÷2, 4)` ＋ 等級÷2 |
| `14h` | Shocking Grasp | `14BFh` | `Roll(1, 8)` ＋ 等級 |
| `15h` | Sleep | `1513h` | 額度 `Roll(4, 4)` 生命骰，逐個目標依 HD 扣 |
| `2Fh` | Fireball | `262Eh` | `Roll(等級, 6)` |
| `33h` | Lightning Bolt | `2B75h` | `Roll(等級, 6)` |
| `04h` | Cause Light Wounds | `108Fh` | 傷害 `Roll(1, 8)` |
| `20h` | Mirror Image | `1A6Fh` | `Roll(1, 4)` 推在施法者等級那一格 |
| `28h` | Cause Disease | `231Dh` | 四個覆寫參數 `0／1／0／0`，只掛效果 |
| `02h` | Curse | `1026h` | 與祝福術同一條 `0F35h`，訊息 `is Cursed` |
| `1Ch` | Spiritual Hammer | `19A8h` | 四個覆寫參數 `0／1／0／0`（生出鎚子那段未讀）|
| `27h` | Cure Disease | `2300h` | 轉呼叫 `225Bh`：拿掉六個病痛類的效果碼 |
| `2Ah` | Prayer | `249Dh` | `(哪一邊 << 4) + 等級` 推在等級覆寫那一格 |
| `37h` | Slow | `2BC7h` | 推效果碼 `27h` 走 `2724h`，範圍法術 |
| `0Eh` | Friends | `13C8h` | `Roll(2, 4)` 加在記錄 `+15h`（魅力）上，上限 25 |
| `25h` | Cure Blindness | `21E8h` | 解掉效果碼 `21h` |
| `2Bh` | Remove Curse | `2508h` | 解掉效果碼 `24h`，並清掉物品的 `+36h` |

除了魔法飛彈那一條，其餘六支與說明書逐字相符（Burning Hands「每級 1 點」、
Shocking Grasp「1d8 加每級 1 點」、Fireball「1-6 點×施術者等級」）。

## 催眠術的花費表

`1513h` 先把 `DS:677Eh` 設為 1（範圍法術），額度 `DS:47A6h = Roll(4, 4)`，
再走目標清單，依目標的 `+73h`（最高職業等級，怪物就是生命骰）扣：

| `+73h` | 1 以下 | 2 | 3 | 4 | 5 | 6 以上 |
|---|---:|---:|---:|---:|---:|---:|
| 花費 | 1 | 2 | 4 | 6 | `+2Eh` 為 0 花 10，否則 20 | 20 |

額度不夠就放不倒。六段以上一律 20，而額度上限是 16，所以**生命骰 6 以上
的怪物催眠術完全無效**——說明書寫的「一定程度以上的敵人，催眠術將會完全
失效」就是這一格。

掛上去的效果碼是 `35h`，overlay-15 的名稱鏈把它叫 **"Funky--"**（spec 069）。

## 二十五支是純泛型的，不必逐支讀

六十七支裡有一大批**只做兩件事**：把一段字面訊息複製到區域變數，
然後推「法術編號 ＋ 四個零」呼叫 `08BCh`。它們沒有自己的算法——射程、
持續、豁免與掛哪一個效果全部來自參數表（spec 074）。

版型（以 Protection From Evil `110Bh` 為例）：

```
55 89 E5 [83 EC nn]      push bp / mov bp,sp / sub sp,nn
A0 79 67 50              mov al, ds:6779h / push ax     ; 法術編號
(B0 00 50) × 4           push 0 四次                     ; 四個覆寫參數
8D 7E nn 16 57           lea di,[bp-nn] / push ss / push di
BF lo hi 0E 57           mov di, 訊息位移 / push cs / push di
9A 34 06 BB 05           lcall 05BBh:0634h              ; 字串指派
0E E8 lo hi              push cs / call 08BCh
89 EC 5D CB              mov sp,bp / pop bp / retf
```

`internal/gamepack.ParseGenericSpellHandlers` 逐位元組比對整個版型，
認出 **25 格**：

| 編號 | 法術 | 訊息 |
|---:|---|---|
| 5、11 | Detect Magic | `is affected` |
| 6、7、16、17 | Protection From Evil／Good | `is protected` |
| 8 | Resist Cold | `is cold-resistant` |
| 18 | Read Magic | `is affected` |
| 19 | Shield | `is shielded` |
| 22 | Find Traps | `is affected` |
| 24 | Resist Fire | `is fire resistant` |
| 25 | Silence, 15' Radius | `is silenced` |
| 29 | Detect Invisibility | `is affected` |
| 30、50、63 | Invisibility | `is invisible` |
| 31 | Knock | `Knock-Knock` |
| 33 | Ray of Enfeeblement | `is weakened` |
| 38 | Cause Blindness | `is blind` |
| 44 | Bestow Curse | `has been cursed!` |
| 45 | Blink | `is blinking` |
| 52、53 | Protection From Evil／Good, 10' Radius | `is protected` |
| 54 | Protection From Normal Missiles | `is protected` |
| 61 | （無名）| `is paralyzed` |

**要比對整個版型，不能只看「有沒有呼叫 08BCh」**：會算傷害的那幾支也呼叫
`08BCh`，只看呼叫會把它們一起收進來，然後傷害就消失了。測試同時釘住
「二十五格」與「魔法飛彈那幾支不在裡面」兩個方向。

加上逐支讀出來的十九支，六十七格裡**接得出來的有 44 格**。
數字由 `TestImplementedSpellCount` 釘住——改了實作沒改這裡就會紅。

## 解病術：拿掉效果，不掛效果

`2300h` 整支只是 `call 225Bh`。`225Bh` 逐個問「目標中了這個嗎」再拿掉：

```
2275  lcall 0100h:006Bh(目標, 22h)      ; 有沒有這個效果
228D  lcall 0100h:006Bh(目標, 2Bh)
22AB  lcall 0100h:002Ah(目標, 2Ch)      ; 拿掉
22C1  lcall 0100h:002Ah(目標, 1Fh)
22D1  lcall 0100h:006Bh(目標, 32h)
22EF  lcall 0100h:002Ah(目標, 39h)
```

所以 `0100h:006Bh` 是「有沒有這個效果」、`0100h:002Ah` 是「拿掉這個效果」。

### 解除類與施加類的效果碼互相對得上

| 施加 | 效果碼 | 解除 |
|---|---:|---|
| Cause Blindness（38，純泛型）| `21h` | Cure Blindness（`21E8h`）|
| Bestow Curse（44，純泛型）| `24h` | Remove Curse（`2508h`）|
| Cause Disease（40，`231Dh`）| `2Ch` | Cure Disease（`225Bh`）|

左邊的碼來自**參數表 `+0Ah`**，右邊來自**處理常式裡的字面值**，
兩條路各自解出來卻落在同一個碼上。測試把這三對釘住。
碼與 overlay-15 的名稱鏈對得上：**`2Ch` 是 Cause Disease、`32h` 是
Dreaded Mummy Disease、`1Fh` 是 Helpless**——解病術拿掉的正是那幾個病
以及它們造成的無助，兩份獨立的證據互相印證。

## 還沒接的一支（形狀已知）

| 編號 | 法術 | 位址 | 形狀 |
|---:|---|---|---|
| `39h` | （無名）| `2DB7h` | 先問 `0100h:006Bh(目標, 2Ah)`，回 0 才走泛型那條 |

Prayer 的 `<< 4` 值得記一筆：**等級覆寫那一格被拿來塞兩個東西**——
高四位是哪一邊、低四位是等級。這與鏡影術借同一格傳「幾個影像」是同一招。
隊伍這一邊的 `+10Eh` 是 0，所以對玩家而言那一格就等於施法者等級本身。

## 挑目標：overlay-13 `1E09h`

`1E09h` 收（輸出結構的遠指標、`[bp+0Ah]`、旗標 `[bp+0Ch]`、法術編號 `[bp+0Eh]`），
輸出結構是 **`+0..+3` 目標的遠指標、`+4` X、`+5` Y**。

```
1e13  cmpb $0, [bp+0Ch]
1e25  lcall 00E2h:002Fh(施法者, 編號)     ; 旗標 0：先取射程
1e3d  call  352Ch                        ; 互動的挑目標介面
1e51  mov   es:[di+0Ah], ax              ; 選中的存進施法者 runtime 的 +0Ah
...
1e67  cmpb $0, [di+31A2h]                ; 旗標非 0：看參數表的 +0Eh
1e6e  → 預設目標就是施法者自己
1ea3  lcall 013Dh:006Bh / 0070h          ; 取目標的 X 與 Y 填進輸出結構
```

所以**參數表 `+0Eh` 決定沒得挑的時候預設打誰**（為零就是施法者自己）；
選中的目標會存進施法者 runtime 子結構的 `+0Ah`。

`352Ch` 是那個互動介面（`Next Prev Manual`、`Range = `、`Target `、
`Already been targeted`、`Attack Ally:` 都是它的字串）。已經讀出來的部分：

**射程從裝備中的武器來**（`3563h..3598h`）：`[bp+12h]` 傳進來的射程若是
`FFh`，就去讀角色記錄 `+CCh` 那一格裝備中的物品、取它的型別，
查型別表的 `+0Ch` 再減一；0 與 `FFh` 都墊成 1（spec 065）。

**距離由 `261Bh` 算**：

```
261b  x1 = 013Dh:006Bh(A) / y1 = 013Dh:0070h(A)
2643  x2 = 013Dh:006Bh(B) / y2 = 013Dh:0070h(B)
2659  n = 0
2671  while 0138h:0034h(x1, y1, x2, y2, n) == 0: n++
267f  回傳 n
```

`0138h:0034h` 是 overlay-31 entry 4（code `0579h`，912 bytes）。
它先把座標夾在 **0..31h × 0..18h**（50 × 25，就是戰場的大小），
`n` 為 `FFh` 時當成 8，然後拿 `n` 去索引 `DS:274Ah`（X 位移）與
`DS:2753h`（Y 位移）那兩張方向表。

**`n` 是「第幾步」還是「第幾個方向」還沒定案**——兩種讀法都與這幾行相容，
而它們給出的距離完全不同。要把 `0579h` 那 912 bytes 讀完才寫得定。
在那之前 remake 用戰場座標的切比雪夫距離，是自己的選擇。

remake 目前做的是這個介面的最小版本：N／P 或方向鍵換人、Enter 確定，
預設停在繞得過去的最近敵人身上，**格子游標（Manual）那一半沒有**。

## 不做

- 不從 AD&D 規則書補傷害公式。原版有自己的算法，逐支讀出來才寫。
- **沒讀過的法術硬失敗**，而且前端不列出來——玩家看得到的清單就是實際
  做得到的事。安靜地不做事與正確地不做事在報表上分不出來。
- 選目標還是 remake 自己的作法（傷害打繞得過去的最近敵人、治療打自己）。
  原版讓玩家自己瞄（overlay-08 的 `View Aim`），那條還沒讀。
