# Spec 074：法術參數表與效果訊息

狀態：CONFORMED（表的位置與版面、`+0`／`+6`／`+7`／`+8` 四個欄位的用途、
41 段效果訊息與它們屬於哪一支常式）；DRAFT（其餘十個欄位、豁免類別 1..3
各是什麼、共用施法常式後半段的三個 resident 呼叫）。日期：2026-09-03。

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

## 差異來自 `DS:3196h` 的參數表

`08BCh` 從頭到尾用 `di = 編號 × 16` 去取這張表：

| 位址 | 指令 | 用途 |
|---|---|---|
| `0997h` | `cmp byte ptr [di+3196h], 0FFh` | `+0` 是 FFh 就先擲命中 |
| `095Eh` | `cmp byte ptr [di+319Ch], 0` | `+6` 是 0 就不擲豁免 |
| `097Ch` | `mov al, [di+319Dh]` → `0100h:0043h` | `+7` 是豁免判定的第二個引數 |
| `0A13h` | `cmp byte ptr [di+319Eh], 0` | `+8` 大於零才掛效果 |

表在 START.EXE 的資料段（`3196h + 30640 = 43478`，檔案 47936 bytes，所以它
是靜態初始化過的，不是 BSS），一筆 16 bytes，編號 0 到 67 共 68 筆；編號 0
那筆整筆是零，走不到。整張表在
[`docs/audit/pool-spell-dispatch.json`](../audit/pool-spell-dispatch.json)
的 `parameters` 欄裡逐筆留著，含還沒解讀的欄位。

### `+0`：要不要先擲中

值為 FFh 的正好五個：Cause Light Wounds、Shocking Grasp、Cause Blindness、
Cause Disease、Bestow Curse。**規則書裡要碰觸到對手才生效的就是這五個**，
一個不多一個不少。走這條路時 `08BCh` 在 `099Eh` 依序呼叫
`010Ah:0043h`、`0100h:002Fh`（帶常數 `0Bh`）與 `0100h:003Eh`（帶對手記錄的
`+111h`，也就是 spec 051 的內部 AC），結果為零就把效果整個取消。

### `+6`／`+7`：豁免

`+6` 是類別，0 代表不擲。分佈是 0 共 52 個、1 共 9 個、2 共 4 個、3 共 2 個。
不擲的那 52 個包含 Magic Missile 與 Sleep——**這兩個在 AD&D 裡正好都不給
豁免**，而 Fireball（2）、Lightning Bolt（2）、Hold Person（1）都要擲。
類別 1..3 分別對到哪一種豁免還沒讀，`0100h:0043h` 未反組譯。

### `+8`：效果碼

值為零的正好是十三個「打完就結束、不留狀態」的法術：Cure／Cause Light
Wounds、Burning Hands、Magic Missile、Shocking Grasp、Knock、Cure Blindness、
Cure Disease、Dispel Magic ×2、Remove Curse、Fireball、Lightning Bolt、
Restoration。留下狀態的一個都不在裡面——**Cure Blindness 在（它是把狀態拿掉）、
Cause Blindness 不在（它留下失明，碼 `21h`）**。

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

### 57..67 這十一個無名編號有訊息

名稱表只有 56 筆（spec 068），但派發表有 67 格。無名的那些帶著訊息與效果碼：

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

指向「物品與怪物的特殊效果借用同一條派發路徑」。**呼叫端還沒找到**，
所以這一節是 DRAFT。

## 十個還沒解讀的欄位

`+1`..`+5`、`+9`..`+0Dh` 都有人讀，讀的地方在
overlay-09、overlay-13、overlay-15、overlay-19 與 overlay-22——不只在施法路徑
上，所以它們的語意要到那些呼叫端去讀，不能從施法常式推。`+0Eh` 與 `+0Fh`
沒有任何 `[di+disp]` 形式的存取。

| 欄位 | 讀它的地方 |
|---|---|
| `+1` | overlay-22 `0743h`、`0783h` |
| `+2` | overlay-22 `08A3h` |
| `+3` | overlay-22 `088Eh` |
| `+4` | overlay-13 四處、overlay-22 `07A4h` |
| `+5` | overlay-22 `0AC9h`、`0C40h` |
| `+9` | overlay-13 `2451h`、overlay-19 `1BF5h` |
| `+0Ah` | overlay-13 `24ABh` |
| `+0Bh` | overlay-09 `0300h` |
| `+0Ch` | overlay-09 `031Dh`、overlay-13 `1E68h` |
| `+0Dh` | overlay-09 兩處、overlay-13 兩處 |

## 順帶解出來的東西

- `DS:6B88h` 是施法目標的個數，`DS:6B89h` 起是一串遠指標（一格 4 bytes，
  1 起算）。它就接在派發表（`6A7Ch..6B87h`）後面。
- `DS:6779h` 是目前法術編號，處理常式與 `08BCh` 都讀它。
- `DS:6777h` 由 `08BCh` 在開頭依第三個參數設定、結束時清零。
- Bless（編號 1）不直接呼叫 `08BCh`，先經過 `0F35h`：那一支用角色記錄的
  `+10Eh` 比對敵我，把不同陣營的目標從清單上清掉，再呼叫 `08BCh`。
  所以 `+10Eh` 是陣營 byte。
