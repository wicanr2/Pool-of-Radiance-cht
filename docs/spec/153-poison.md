# Spec 153：中毒與效果群組 12／6／4 的其餘碼

狀態：READY（中毒的來源、致死、緩毒術之後 `4Eh`／`0Fh`／`16h` 三支、解毒的兩條路、群組 12 的
`3Dh 6Fh 7Dh`、群組 6 除 `65h` 外的全部、群組 4 的 `03h 06h`、怪物身上的靈魂鎚與致病收尾都逐條讀過並
實作，送鍵測試加變異檢查）；DRAFT（下面〈卡點〉那幾條）。證據是位元組（exact），沒有原版執行期收據逐次
對過。日期：2026-09-27。主台帳：GitHub issue #106。

## 輸入

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`（`MONnSPC.DAX` 的毒碼、參數表）。
- overlay 由 `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 04 | `d948ce6bc533470ac1fa44a7787c2ce5006462cc33ada1cf61ffd128da8a91af` | `0035h` |
| 12 | `d1b057432515b7daa2b83199d2588c131a120037d30117e664b01e4c6f2bcb7e` | `006Ch` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` | `0096h` |
| 22 | `967065cc35975465a7250026c63b8a5ae06b812b228abcfbbbd83d636538dda8` | `00E2h` |
| 24 | `e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714` | `0100h` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` | `010Ah` |
| 32 | `efc22ba82144187c4bfc5deadb2bb10a868c13e45f2f7ca63f54d7387b22dfc8` | `013Dh` |

- 反組譯：`coab-go-test:20260729` 的 GNU objdump 2.40（`-D -b binary -m i8086 -M intel`，
  `--start-address` 從函式起點解，線性掃描在字串之後會錯位）。far call 的 segment 以
  (manifest `executable_file_offset` − 3B0h) ÷ 16 反查 overlay，stub offset `20h + i × 5` 是 entry i。
  位元組掃描是 38 顆 overlay 的全檔比對。

## 中毒是當場死亡

DOS 版的中毒不是逐時扣血。唯一掛 `37h` 的地方是 overlay-12 `1553h`（全部 overlay 掃
`B0 37 50` 九處，只有 `15B5h` 後面接 entry 10 `9A 52 00 00 01`；其餘是讀或摘，見〈誰碰 37h〉）：

```
1553  f(豁免修正: byte [bp+6]; 記錄: far [bp+8])          retf 6
1559  les di,[bp+8]; les di,es:[di+108h]; les ax,es:[di+0Ah]
1567  DS:6784h／6786h = 目標                            ; 攻擊者當下的目標
157D  9A 43 00 00 01   entry 7(目標, 類別 0, 修正)        ; 豁免
1582  08 C0 / 75 5E    成功 → 返回
1598  "is Poisoned"（153Dh）、ov25 entry 20(0Ah, 0) 停一下、ov37 entry 13
15B5  B0 37 50 / 31 C0 50 / B0 FF 50 / B0 00 50 / 9A 52 00 00 01
                       entry 10(目標, 37h, 持續 0, 等級 FFh, 不收尾)
15CE  B0 06 50 … "is killed"（1549h）/ E8 76 EA   005Ah(目標, 6, 訊息)
```

`005Ah`（exact）：

```
0084  ov25 entry 20：印「<名字> <訊息>」
008C  +10Ch 是 7／6／8 → 結束                          ; 3C 07 / 3C 06 / 3C 08
00A5  +10Ch = 狀態；+10Dh = 0；+11Bh = 0
00C2  ov24 entry 13(記錄)；00D0h 群組 13(記錄)
00E0  +10Dh == 0 → ov32 entry 20(記錄)                 ; 從盤面上拿掉
00F5  不在戰鬥（4954h != 5）→ ov25 entry 2(5CF0h)       ; 重畫
```

所以豁免失敗 = 「is Poisoned」「is killed」、狀態 6（死亡，`combat.DeadState`）、生命 0、身上一個持續 0
（永久，spec 069）的 `37h`。`37h` 本身的處理常式是空的（entry 50 `1373h`，`55 89 E5 89 EC 5D CA 0A 00`），
它只是給別人問的旗標。

### 四個毒碼（群組 3）

| 碼 | 處理常式 | 推給 `1553h` 的修正 | MONnSPC 裡帶它的 |
|---|---|---|---|
| `40h` | entry 59 `1667h` | `B0 00 50`：0 | GIANT SNAKE、HUGE SCORPION、MEDUSA、WYVERN |
| `41h` | entry 60 `167Dh` | `B0 04 50`：+4 | POISONOUS FROG |
| `42h` | entry 61 `1693h` | `B0 02 50`：+2 | LARGE SCORPION |
| `46h` | entry 65 `1721h` | `B0 FE 50`：−2（entry 7 `0D8Fh` `8A 46 06 / 98` 以 `cbw` 讀）| PHASE SPIDER |

（怪物清單出自 `docs/audit/monster-atlas.json`；帶毒碼的怪物全部有第二攻擊形態。）

四個碼只在群組 3（`40h 41h 42h 43h 44h 45h 46h 4Fh 55h 56h 57h`），不在群組 2。overlay-13 的攻擊區段
在 `02FEh` 扣完血之後：

```
1735  26 80 BD 0D 01 00 / 74 12      目標 +10Dh == 0（已離場）→ 不派發
1740  8A 46 EB / 30 E4 / 40 / 50     al = [bp-15h] + 1               ; 形態 1 → 群組 2、形態 2 → 群組 3
174D  9A 2F 00 00 01                 02E2h(群組, 攻擊者)
```

所以**只有第二攻擊形態**命中、目標還在場上時才會中毒；每帶一個毒碼擲一次豁免（`014Dh` 各問一次）。
`43h 44h 45h`（`15F7h`，"is Paralyzed" 掛 `34h`）是同一個形狀的麻痺，不在本 spec。

### 誰碰 37h

| 位置 | 做什麼 |
|---|---|
| overlay-12 `15B5h` | 掛（上面）|
| overlay-12 `0797h` | `16h` 的收尾問有沒有（下面）|
| overlay-12 `29F7h`／`2E37h` | `6Fh`／`7Dh` 的 `0000h(37h)`：群組 9 裡擋掛 `37h`（spec 112）|
| overlay-22 `1873h` | 緩毒術問有沒有（沒有就整支不做）|
| overlay-22 `2E10h` | 編號 58 用 entry 15 摘 |
| overlay-04 `0754h`／`07C0h` | 神殿 Neutralize Poison 問、摘；`05DBh` 起死回生也摘 |

`37h` 沒有別的來源（literal 掃描 exact；以變數傳碼給 entry 10 的呼叫端沒有逐一排除，整體為 strong
inference）。城堡毒荊棘樹籬那一種「直接死」走的是 ECL 那一側（spec 084〈不經過這個入口的死亡〉），
不掛 `37h`。

## 緩毒術：把人暫時扶起來

overlay-22 `1846h`（spec 098〈不是純泛型的那幾支〉）在 `08BCh` 之後還有兩步，全部對表上第一格
（`DS:6B89h`）：

```
185B  26 80 BD 0C 01 01 / 75 0A      +10Ch == 1 → 清表、結束
187B  9A A7 00 0A 01                 010Ah:00A7h(記錄, 37h)：沒中毒 → 結束
1887  +11Bh == 0 → +11Bh = 1
1898  A0 79 67 50 / B0 FF 50 / B0 01 50 / B0 00 50 / B0 00 50 … E8 01 F0
                                     08BCh(法術, 等級 FFh, 1, 0, 0)       ; 掛 16h，+4 = 1
18BB  B0 4E 50 … 31 C0 31 D2 52 50 / B0 01 50 / 9A 25 00 00 01
                                     entry 1(4Eh, 記錄, NULL, 模式 1)     ; 4Eh 的常式
18D2  B0 0F 50 / B8 0A 00 50 / B0 FF 50 / B0 01 50 / 9A 52 00 00 01
                                     entry 10(記錄, 0Fh, 0Ah, FFh, 1)
```

`08BCh` 的第二個覆寫參數 `1` 就是節點 `+4`：`16h` 到期時要叫處理常式。

### `4Eh`：扶起來（overlay-12 entry 71 `19F5h`、overlay-24 entry 22 `1869h`）

```
19FB  ov24 entry 22(記錄, +11Bh)：回非 0 → 結束
1A13  0021h(記錄, 4Eh, 節點 +3, 1)       ; 站不起來：重掛一個持續 1 的，下一刻再試
```

entry 22（`1869h`，`retf 6`）：

```
187B  ov32 entry 15／16(記錄)：x、y
1890  ov32 entry 21(記錄, x, y, 1)：回 0 → 回 0
189C  +10Ch = 0；+10Dh = 1；+11Bh = 參數
18B9  戰鬥中 → ov32 entry 22(記錄, 3, 0)          ; 換回站著的圖
18D1  +10Eh == 1 → "stands up and grins"（1848h），否則 "gets back up"（185Ch）
1915  ov25 entry 20(0Ah, 1)、ov25 entry 31；回 1
```

ov32 entry 21（`1091h`）：

```
1097  80 3E 54 49 05 / 74 03 / E9 D2 01   不在戰鬥 → 回 1
10AB  索引 = 0C61h(記錄)；5E88h[索引] = +6Ch & 7Fh；5E85h／5E86h = x、y   ; 體型與位置放回位置表
1104  0CB9h(記錄, 8, &佔用者, &地形)       ; 方向 8：探自己的佔格（spec 058）
1107  佔用者 != 0 → 失敗；地形 == 0（盤面外）→ 失敗；DS:2758h[地形 × 4] == FFh → 失敗
1125  失敗：5E88h[索引] = 0、回 0
1138  成功：回 1（[bp+6] 為 1 時另外把 6634h 那一張屍體表上的這一筆清掉，再 03A2h 重建佔用）
```

`19F5h` 被 `1846h` 叫的時候節點是 NULL，`1A1Fh` `26 8A 45 03` 讀到的是 `0000:0003`（中斷向量表）的一個
byte——它只會存進重掛的 `4Eh` 的 `+3`（解除魔法讀的等級），值取決於 RTL 的載入位址（unknown）。

### 時間到了：`0Fh`、`16h`

| 碼 | 處理常式 | bytes | 做什麼 |
|---|---|---|---|
| `0Fh` | entry 17 `05F7h` | `B0 0F 50 / 26 8A 45 03 50 / B8 0A 00 50 / E8 0E FA`、`26 80 BD 1B 01 01 / 76 2D`、`C6 06 77 67 00`、`9A 7F 00 00 01` | `0021h` 重掛（持續 0Ah）；回 1 而且 `+11Bh` 大於 1 → `6777h = 0`、entry 19(記錄, 1, 0, 0) 打 1 點；不在戰鬥就重畫 |
| `16h` | entry 23 `078Bh` | `B0 37 50 … 9A A7 00 0A 01 / 74 1C`、`B0 06 50 … "dies from poison"（077Ah）/ E8 96 F8`、`C6 06 7D 67 01`、`B0 0F 50 … 9A 2A 00 00 01`、`C6 06 7D 67 00` | 還中毒 → `005Ah(記錄, 6, "dies from poison")`；然後 `677Dh` 立著摘 `0Fh`（它的收尾不重掛、不扣血）|
| `4Eh` | entry 71 `19F5h` | 上面 | 再試一次扶起來 |

`0021h`（`55 89 E5 83 EC 02 / 80 3E 7D 67 00 / 74 06`）：`DS:677Dh` 立著就回 0，否則 entry 10 掛
（碼, 持續, 等級, 有收尾 1）、回 1。`677Dh` 只在治療那幾條路摘節點時立起。

整條線：中毒倒下 → 緩毒術把生命墊到 1、狀態 0、站起來 → 每十分鐘（戰鬥中每十回合）`0Fh` 扣 1 點，
扣到 1 為止 → `16h`（`+4 + +5 × 等級`）到期時還沒解毒就再死一次，這次印 "dies from poison"。

## 解毒

| 位置 | 前提 | 做什麼 |
|---|---|---|
| 神殿 Neutralize Poison（overlay-04 entry 8 `0736h`，spec 115）| 身上有 `37h` | `677Dh = 1`，摘 `37h`、`16h`、`0Fh`，`677Dh = 0` |
| 編號 58（overlay-22 `2E02h`）| 表上第一格有 `37h` | entry 15 摘 `37h`（印 "is Cured"）、`677Dh = 1` 摘 `16h`（`078Bh` 順手摘 `0Fh`）、結束 |
| 起死回生（overlay-04 `0515h`）| 狀態 6 或 1 | 摘 `20h`、`37h`，狀態 0、生命 1 |

前兩條**都不動 `+10Ch`**：中毒倒下又沒緩毒的人，解了毒還是死的，要起死回生。編號 58 沒中毒時走
`2E3Fh` 的解病鏈 `225Bh`，解到東西就結束；什麼都沒解到才 `2E4Eh` Roll(1, 4) + 8 治療。

## 戰鬥內外

| | 戰鬥中 | 戰鬥外 |
|---|---|---|
| `37h` 的來源 | 第二形態命中 | 無 |
| 計時 | 每回合減一 | overlay-20 逐分鐘（spec 069）|
| `4Eh` 扶起來 | 原地要站得下（ov32 entry 21），不然每回合再試 | 一定站得起來 |
| `0Fh` 扣血 | entry 19，受傷打斷照走 | entry 19、重畫 |

## 群組 12／6／4 的其餘碼（#106）

處理常式的簽章與累加格見 spec 112。以下 exact。

### 群組 12（豁免，overlay-24 entry 7 `0DB2h`）

| 碼 | 常式 | bytes | 做什麼 |
|---|---|---|---|
| `3Dh` | entry 55 `149Ch` | `A0 77 67 / 24 01 / 08 C0 / 74 46`、`80 06 74 67 04` | `6777h & 1`（火）才做：豁免 +4 |
| `6Fh` | entry 105 `29F4h` | `80 3E 88 67 00 / 75 05 / C6 06 74 67 64` | 類別 0 → 豁免骰寫成 100 |
| `7Dh` | entry 119 `2E1Fh` | 同上 | 同上 |

`6Fh`／`7Dh` 前面的 `0000h(37h)`、`0000h(34h)` 只在 `6775h` 等於那個碼時寫 `6775h`／`6776h`，在豁免的時點
沒有讀者（`6776h` 由 entry 19 `1344h` 重寫）。

### 群組 6（法術傷害，overlay-24 entry 19 `1351h`）

順序 `71h 3Dh 7Ah 3Ch 5Bh 0Ah 14h 69h 6Ah 70h 72h 76h 11h 5Dh 65h 1Ch`。#89／#96 接過 `0Ah 14h 11h 1Ch`。

| 碼 | 常式 | bytes | 做什麼 |
|---|---|---|---|
| `71h` EFREETI | entry 107 `2A30h` | `24 01`、`FE 0E 76 67`、`3A 06 7A 67 / 73 06` | 火：`677Ah` 次，每次傷害 −1，無號小於骰數就墊回骰數 |
| `3Dh` 抗火戒指 | entry 55 `149Ch` | `80 2E 76 67 02`、同上的墊底、`A0 77 67 / 24 08 / 75 07 … E8 0F EB` | 火：每骰 −2 墊底；`6777h` 位元 3 沒立 → `0000h(0)` 歸零 |
| `7Ah` MUMMY | entry 116 `2D4Ch` | `C4 3E F0 5C / 26 C4 85 CC 00`、`26 80 7D 2E 56`、`B0 03 50 B0 08 50 9A 4D 00 00 01 / A2 76 67`、`24 09`、`00 06 76 67` | 行動者手上是型別 56h → 傷害 = 3d8（entry 9，骰數變 3）；`6777h & 9` → 加骰數 |
| `3Ch` | entry 126 `31A9h` | 空常式 | 無 |
| `5Bh` JUJU ZOMBIE | entry 85 `2535h` | `80 3E 79 67 0F / 75 29`、`E8 B7 DA`、"is unaffected"（`2527h`）、`24 04` | 魔法飛彈 → 歸零並印一句；否則 `6777h & 4` → 歸零 |
| `69h`／`6Ah` | entry 99／100 | `2910h(32h／0Fh)` | 魔法抗性：`6775h != 0` 或 `6777h & 8` → 1d100 不大於門檻就歸零（門檻算法 spec 112〈群組 9〉）|
| `70h` FIRE GIANT | entry 106 `2A17h` | `24 01 / … E8 D6 D5` | 火 → 歸零 |
| `72h` VAMPIRE | entry 108 `2A75h` | `24 04`、`99 / B9 02 00 / F7 F9` | `6777h & 4` → 減半 |
| `76h` VAMPIRE | entry 112 `2B76h` | `24 02`、同上 | `6777h & 2` → 減半 |
| `5Dh` JUJU ZOMBIE | entry 86 `2581h` | `24 01`、同上 | 火 → 減半 |
| `65h` TROLL | entry 94 `280Dh` | `B0 62 50 … 9A A7`、`B0 3B 50 … 9A A7`、`B0 3B 50 / B8 03 00 50 / B0 FF 50 / B0 01 50 / 9A 52` | 身上沒有 `62h`、`3Bh` → 掛 `3Bh`（持續 3、有收尾）；不改傷害 |

**byte 減法會繞回去。** `3Dh` 的 `sub 6776h, 2` 之後比的是無號 `jae`：單顆骰擲出 1（或燃燒之手的
1 級、骰數 1）時 `1 − 2 = 0FFh`，比骰數大，不墊——傷害變成 255。照搬。

#### `677Ah` 與 `6775h`

`DS:677Ah` 只有 overlay-24 entry 9（`0E30h`：`8A 46 08 / A2 7A 67` 之後轉呼叫 entry 8）寫，全部 overlay
掃 `7A 67` 只有 `0E3Ah` 一處寫。所以它是**最近一次 entry 9 擲的骰數**。走 entry 9（`9A 4D 00 00 01`）的
傷害：近戰 overlay-13 `01B4h`、致輕傷 `10A5h`（1）、魔法飛彈 `1454h`（等級 ÷ 2）、電擊之握 `14E1h`（1）、
火球 `2704h`（等級）、編號 65 `3024h`（2），以及 overlay-12 的幾支怪物特殊攻擊（`184Fh`、`1A53h`、
`1A88h`、`2B65h`、`2CFCh`、`2D7Bh`）。燃燒之手不擲骰、閃電束與射線走 entry 8，讀到的是上一次的值。
原版是 DS 段全域，跨戰鬥留著；開機時 Turbo Pascal 把全域清成 0（strong inference）。

`DS:6775h` 只有 entry 20 `1673h` 寫、`0000h` 清（`75 67` 全掃）：掛完一個效果之後它留著那個碼。
`69h`／`6Ah` 在群組 6 看它；remake 施法的傷害種類全都立著位元 3，一定擲，`6775h` 的殘值只在
`6777h` 為 0 的傷害（`0Fh`／`2Ch` 的 1 點）上有差，remake 當 0（hypothesis）。

### 群組 4（近戰傷害骰之後，攻擊者，overlay-13 `021Eh`）

順序 `1Dh 03h 06h`。目標是攻擊者 `+108h` 的 `+0Ah`（`0144h`、`01CFh`），與群組 10 同一個讀法（spec 112）。

| 碼 | 常式 | bytes | 做什麼 |
|---|---|---|---|
| `03h` | entry 7 `0141h` | `26 80 BD 9F 00 04 / 75 0A`、`80 06 76 67 02` | 目標 `+9Fh == 4` → 傷害 +2 |
| `06h` | entry 9 `01C9h` | `3D 0A 00`／`3D 09 00`／`3D 0C 00`／`3D 04 00`、`00 06 76 67`、`C6 06 77 67 09` | 0Ah +1、9／0Ch +2、4 +3；另寫 `6777h = 9` |

## 怪物側

- **靈魂鎚**（`17h`，overlay-12 entry 24 `07F6h`）：整支只讀寫傳進來的記錄——`+0C8h` 物品串列、
  `+0C7h` 件數、overlay-25 entry 17／18 接上摘下、`0916h` 重算——**沒有 `+10Eh` 的判斷**（exact）。
  所以怪物施法者一樣拿到鎚子，節點到期收回。remake 的怪物物品串列是 `FoeItems`（spec 142）。
- **致病**（`22h` `0BE3h`、`2Bh` `10EDh`、`2Ch` `1177h`、`0021h`）：同樣只讀寫記錄（`+10h`、`+11Bh`、
  效果串列），怪物身上一樣到期收尾（exact）。
- **中毒那一串**（`0Fh`、`16h`、`4Eh`、`005Ah`、entry 22）：也不看 `+10Eh`；entry 22 依 `+10Eh` 選
  那一句話（怪物是 "stands up and grins"）。

## remake 的對應

| 原版 | remake |
|---|---|
| `1553h` 與四個毒碼 | `gamepack.PoisonAttackSaves`、`app.poisonSpecialAttack`（`resolveAttackSwings` 的第二形態那幾下，排在 `55h`／`56h` 前面）|
| `005Ah` | `app.poisonKill`（盤面）、`mapPoisonTeardown`（地圖）|
| 緩毒術 `18BBh..18E5h` | `slowPoisonAftermath`（戰場，`castEffectOnly` 之後）、`slowPoisonAftermathInCamp`（營地，`campSpellEffect`）；`CastSpell` 的緩毒術補上 `EffectParameter = 1` |
| entry 22 ＋ ov32 entry 21 | `poisonRecover`、`standsOnItsOwnCells`（`combat.ProbeDestination` 方向 8）|
| `0Fh`／`16h`／`4Eh` 的收尾 | `gamepack.PoisonTeardownOf`；戰場 `poisonTeardown`（`partyEffectTeardown` 開頭，隊員與怪物都跑）、地圖 `mapPoisonTeardown`（`advancePartyEffects`）|
| 神殿 Neutralize Poison | `internal/temple` 既有（摘三個碼）|
| 編號 58 | `CastEffect.NeutralizesPoison`：戰場 `neutralizeOnBoard`、營地／物品 `applyFieldEffect` |
| 群組 12 其餘 | `SaveRollEffects.applyCode` 的 `3Dh 6Fh 7Dh` |
| 群組 6 | `SpellDamageEffects.Apply` 改成照整組順序跑；`Dice`／`CasterLevel`／`ActorWeaponType` 三個新輸入 |
| `677Ah` | `app.diceCount`：近戰每一下、`gamepack.SpellDamageDice` 那五支施法時寫；`7Ah` 擲 3d8 時改 3 |
| 群組 4 其餘 | `MeleeDamageAfterAttackerEffects(串列, 傷害, 目標 +9Fh)` |
| 怪物的靈魂鎚、致病 | `grantSpiritualHammer` 的怪物分支、`foeEffectTeardown`（力量記在 `tacticalState.FoeStrength`）|

測試（`cmd/pool-game/poison_test.go`）：`TestAPoisonousStingKillsOnAFailedSave`（A 出手；豁免失敗／過了／
第一形態三組）、`TestPoisonCodesCarryTheirOwnSaveModifiers`、`TestUndeadShrugOffPoison`、
`TestSlowPoisonHoldsThePoisonOnlyUntilItRunsOut`（營地 C 施法、時間推進）、
`TestNeutralizedPoisonDoesNotComeBack`（神殿選單）、`TestPoisonRecoveryWaitsForItsSpotInCombat`（ENTER 推回合）、
`TestFireResistanceRingRaisesTheSaveAgainstFire`、`TestSpellDamageGroupReadsTheDamageKind`、
`TestMeleeBaneCodesAddDamageToTheRightKind`（A 出手）、`TestSpiritualHammerAlsoArmsAFoeCaster`、
`TestCauseDiseaseAlsoWastesAFoe`。變異檢查逐條拿掉規則重跑，十七個全紅（2026-09-27 實跑）：毒的派發、
只限第二形態、41h 的修正、7Dh 的豁免、營地緩毒的後兩步、16h 毒發、0Fh 扣血、地圖收尾的接點、神殿摘 16h、
扶起來的佔格探測、3Dh 的豁免、70h、每骰墊底、群組 4 的 03h／06h、怪物的鎚子、怪物的收尾接點。

## 與原版不同、寫明的幾處

| 原版 | remake | 理由 |
|---|---|---|
| "is Poisoned"、停一下、"is killed" 兩句 | 狀態列一行兩句 | 狀態列只有一行 |
| 重掛的 `4Eh` 的 `+3` 是中斷向量表的一個 byte | FFh（解除魔法解不掉）| 原值取決於載入位址（unknown）|
| 死亡時派發群組 13（`00D0h`）| 不派發 | 群組 13 的五個碼沒讀（卡點）|
| `6775h` 的殘值 | 當 0 | 見〈677Ah 與 6775h〉|
| 戰場上 `20AEh` 收不收得到屍體 | remake 的瞄準只收站著的人，中毒倒下的隊員在戰場上施不到緩毒 | `20AEh` 對屍體的處理沒讀 |

## 被推翻的斷言

| 舊寫法 | 現況 | 依據 |
|---|---|---|
| 中毒是「逐時扣血」那一套（spec 098／073 舊版）| 豁免失敗當場死亡；逐時扣血的 `0Fh` 只在緩毒之後 | `1553h` 的 `15E1h` 直接叫 `005Ah(目標, 6, …)` |
| entry 22 問的是 overlay-38（spec 098 舊版）| `013Dh` 是 overlay-32（manifest 的 (`executable_file_offset` − 3B0h) ÷ 16）| stub segment 表 |
| 靈魂鎚只給隊員、怪物的致病到期不跑（remake 舊行為）| 兩支都不看 `+10Eh`，怪物一樣跑 | `07F6h`、`0BE3h`、`10EDh`、`1177h` 全段 |
| 編號 58 一律治療並解 `16h` 與病痛（remake 舊行為）| 中毒就只解毒；`16h` 只在那一支摘 | `2E08h..2E3Ch` |

## 卡點（DRAFT）

- **群組 13 在死亡時**（`005Ah` 的 `00D0h`，碼 `63h 64h 67h 4Bh 4Ah`）：沒讀，remake 不派發。
- **`65h`（TROLL）掛的 `3Bh`**：`3Bh` 的處理常式（再生）沒讀，`65h` 不接；群組 6 的傷害不受影響。
- **群組 5 的其餘碼**與 `06h` 寫給它的 `6777h = 9`：群組 5 只接了 `1Ch`、`29h`，`7Ah`／`65h` 在群組 5 那一次
  沒接。
- **麻痺**（`43h 44h 45h`，`15F7h`）與群組 2／3 的其餘特殊攻擊：同一個形狀，另開。
- **編號 58 沒中毒時**「解到病就不治療」：戰場那一支只解病、不治療，營地那一支兩樣都做，兩邊都沒照
  `2E3Fh..2E4Eh` 的先後。
