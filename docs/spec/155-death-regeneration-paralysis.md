# Spec 155：倒下時的群組 13、再生、群組 5 其餘的碼、麻痺、編號 58 的先後、瞄準屍體

狀態：READY（下表每一條逐條讀過位元組並實作，送鍵測試加變異檢查）；〈卡點〉那兩條維持 unknown／
hypothesis。證據是位元組（exact），沒有原版執行期收據逐次對過。日期：2026-09-27。主台帳：GitHub issue #113。
接手 spec 153〈卡點〉與 spec 112 OPEN 的對應條目。

## 輸入

- DOS ZIP `Pool of Radiance (1988).zip`，SHA-256 `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`
  （`MONnSPC.DAX` 的碼由 `docs/audit/monster-atlas.json` 讀；`poolrad/items` 的型別表）。
- `START.EXE`（`workplace/START.EXE`），SHA-256 `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`：
  DS 常數在檔案位移 `30640 + DS 位移`。
- overlay 由 `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` | `0051h` |
| 12 | `d1b057432515b7daa2b83199d2588c131a120037d30117e664b01e4c6f2bcb7e` | `006Ch` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` | `0096h` |
| 22 | `967065cc35975465a7250026c63b8a5ae06b812b228abcfbbbd83d636538dda8` | `00E2h` |
| 24 | `e878166ef2069fcc2dad3d15915801bd9ee63bee47bba8a581f0f651f22f8714` | `0100h` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` | `010Ah` |
| 32 | `efc22ba82144187c4bfc5deadb2bb10a868c13e45f2f7ca63f54d7387b22dfc8` | `013Dh` |

- 反組譯：`coab-go-test:20260729` 的 GNU objdump 2.40（`-D -b binary -m i8086 -M intel`，從函式起點解）。
  far call 的 segment 以 (`executable_file_offset` − stub offset − 3B0h) ÷ 16 反查 overlay，stub offset
  `20h + i × 5` 是 entry i。位元組掃描是 38 顆 overlay 的全檔比對。

## 倒下的三個入口

三條都是同一個形狀：先摘 `DS:0C28h` 那十六個碼，再派發群組 13，`+10Dh` 還是 0 才離場。

| 入口 | entry 13 | 群組 13 | 還躺著 → ov32 entry 20 |
|---|---|---|---|
| 近戰（overlay-13 entry 4 `02FEh`，`0559h` 起只在 `+10Dh == 0` 時跑）| `05F5h` `9A 61 00 00 01` | `0603h` `B0 0D 50 … 9A 2F 00 00 01` | `0608h..0619h` |
| 法術傷害（overlay-24 entry 19，只在戰鬥中 `15FBh`）| `1610h` `0E E8 F1 F9` | `161Dh` `B0 0D 50 … 0E E8 C2 EC`（近呼叫，常數掃描看不到）| `1620h..1631h` |
| 當場死亡（overlay-12 `005Ah`）| `00C2h` | `00D0h` | `00E0h..00E6h` |

**entry 13（`1004h`）**：`100Ah` 從 1 起，`102Fh` `80 7E FF 10` 比到 16 才停，每一輪 `mov al, [di+0C27h]`
讀 `DS:0C28h..0C37h`、以 NULL 節點叫 entry 2（摘最早掛上的那一個，`+4` 立著就先以模式 1 叫處理常式）。
START.EXE 的那十六個 byte 是 `07 0B 1E 1F 20 33 34 35 36 3A 3B 5F 62 89 4A 4B`（同一塊讀 `DS:2880h` 得
`33 34 35 1F` 是正對照）。逃離盤面（entry 11 `0FC3h`）叫的也是它——`gamepack.EscapeStrippedEffects`
原本少了最後一格 `4Bh`，一併補上。

魅惑的 `0Bh` 被摘時跑模式 1（entry 14 `040Ah`：`+10Eh` 還原成節點 `+3` 位元 6、士氣 B3h 歸零），所以
被魅惑的怪物倒下時回到原本那一邊。

## 群組 13（`63h 64h 67h 4Bh 4Ah`）

| 碼 | 常式 | bytes | 做什麼 |
|---|---|---|---|
| `63h` WILD BOAR | entry 92 `2727h` | `26 80 BD 0C 01 05`、`26 80 7D 0E 06 / 73 18`、`B8 06 00 / 2B C2`、`26 80 BD 0C 01 04 / 75 04 / C6 46 FF 06`、`9A 8E 00 00 01`、`B0 5F 50 / B0 01 50 / B0 04 50 / 9A 48 00 00 01 / 30 E4 / 40`、`B0 63 50 … 9A 2A 00 00 01` | 狀態 5 而倒地計數 `+0Eh` 小於 6 → n = 6 − `+0Eh`；狀態 4 → n = 6；n > 0 → entry 22(記錄, n) 站起來，站得起來才掛 `5Fh`（持續 Roll(1, 4) + 1、FFh、有收尾）並摘掉 `63h` |
| `64h` TROLL | entry 93 `27D0h` | `A0 77 67 / 24 01 / 75 2B`、`A0 77 67 / 24 10 / 75 22`、`B0 66 50 / B0 03 50 / B0 06 50 / 9A 48 00 00 01` | `6777h` 沒有位元 0（火）也沒有位元 4 → 掛 `66h`（持續 Roll(3, 6)、FFh、有收尾）；不摘自己，每倒一次都掛 |
| `67h` VAMPIRE | entry 96 `28B8h` | `9A 8E 00 3D 01`、`9A 57 00 00 01` | ov32 entry 22(記錄, 3, 0) 換圖，再 ov24 entry 11(記錄, 3, "Turns gaseous and escapes")——entry 11 一進來 `0F19h..0F24h` `26 80 BD 0D 01 00 / 75 03 / E9 9F 00`：`+10Dh` 是 0 就整支返回 |
| `4Bh` | entry 68 `178Bh` | `26 80 BD 0D 01 00`（自己與 `6784h` 那一位）、`B0 3A 50 … 9A 2A`、`B0 4B 50 … 9A 2A` | 模式 0 而任一方已倒：摘夥伴（`DS:6517h[節點 +3]`）的 `3Ah`、摘自己 |
| `4Ah` | overlay-13 entry 29 `3A55h` | 同形 | 同上 |

- **倒地計數 `+0Eh` 就是打穿的點數**：overlay-25 entry 28（`2266h`，`gamepack.ApplyDamage`）在 `22E7h..22F2h`
  把「傷害 − 生命」寫進去；打穿 10 點以上直接狀態 6（`22AFh`），剛好 0 點是狀態 4（`2301h`）。所以野豬站起來的
  生命是 6 − 打穿點數（0..5 點），打穿 6 點以上不站。
- **吸血鬼的氣化逃走走不到**：三個派發點都在 `+10Dh = 0` 之後（近戰 `0559h`、法術 `1560h`、`005Ah` 的 `00ADh`），
  entry 11 看到 0 就返回。strong inference：群組 13 的呼叫端是常數掃描加上 `161Dh` 那一個近呼叫，沒有逐一排除
  以變數推群組的呼叫端（spec 112〈誰在問哪一組〉只有 `174Dh` 一個，是群組 2／3）。
- `4Bh`／`4Ah` 在 entry 13 已經先摘掉第一個；掛它們的是 STIRGE 的 `4Ch`（entry 69 `18D3h`，"Sucks some Blood"）等
  群組 2 的碼，remake 沒有接那些來源，所以這兩格在 remake 走不到（不阻擋玩家路徑）。

### 到期時的收尾

| 碼 | 常式 | bytes | 做什麼 |
|---|---|---|---|
| `5Fh` | entry 88 `25F9h` | `26 C6 45 04 00`、`26 80 BD 0D 01 00 / 74 1C`、`B0 06 50 … "Falls dead"（25EEh）/ E8 2C DA` | 還站著（`+10Dh != 0`）→ `005Ah(記錄, 6, "Falls dead")` |
| `66h` | entry 95 `285Fh` | `26 8A 45 32 / 50 / 9A 8E 00 00 01`、`B0 66 50 / 26 8A 45 03 / B8 01 00 / E8 8C D7` | entry 22(記錄, `+32h`) 以滿血站起來；站不起來 `0021h(記錄, 66h, 節點 +3, 1)` 下一回合再試 |

## 再生：`65h` → `3Bh` → `62h`

| 碼 | 常式 | bytes | 做什麼 |
|---|---|---|---|
| `65h` TROLL（群組 5、6）| entry 94 `280Dh` | `B0 62 50 … 9A A7 00 0A 01`、`B0 3B 50 … 9A A7`、`B0 3B 50 / B8 03 00 50 / B0 FF 50 / B0 01 50 / 9A 52 00 00 01` | 身上沒有 `62h`、`3Bh` → 掛 `3Bh`（持續 3、FFh、有收尾）；傷害不動 |
| `3Bh` | entry 54 `147Ch` | `B0 62 50 / 31 C0 50 / B0 FF 50 / B0 00 50 / 9A 52 00 00 01` | 不看模式：掛 `62h`（持續 0、FFh、不收尾）。`3Bh` 不在任何群組，只在摘掉時跑 |
| `62h`（TROLL 三回合後、VAMPIRE 生來就有）| entry 91 `26F5h` | `26 80 85 1B 01 03`、`26 3A 45 32 / 76 0F`、`26 88 85 1B 01` | `+11Bh += 3`（byte），無號大於 `+32h` 就墊回 `+32h` |

`62h` 在群組 19，派發端是 overlay-08 回合收尾的 `0869h` 迴圈：對 `DS:5CF4h` 串列上每一個 `089Ah..08A3h`
派群組 19，`08AEh` 再叫 overlay-24 entry 4（減計時、到期摘）。所以再生在減計時之前。`62h` 與 `3Bh` 都在
entry 13 的十六個碼裡，倒下就沒了（`3Bh` 摘的時候掛的 `62h` 接在尾端，同一輪就被摘掉）。

## 群組 5 其餘的碼

順序 `1Ch 29h 68h 78h 65h 73h 74h 77h 7Bh 60h 5Eh 3Ch 7Ah 75h`（spec 112）。派發端 overlay-13 `022Ch`，記錄是**目標**；
處理常式看的「行動者」一律是 `DS:5CF0h`（反應攻擊時是正在移動的那一位，不是出手的那一位）。

- **`1028h(記錄)`**：overlay-25 entry 45（彈藥查詢，spec 151）回非 0 而且指標非 NULL → 那一件；否則記錄 `+0CCh`。
  下表的「打中的那一件」是 `1028h(DS:5CF0h)`。
- **`0FB7h(記錄, 百分比)`**：`DS:5CF0h` 的 `+0CCh` 是 NULL → 不管；overlay-25 entry 33（距離）不大於 1 → 不管；
  Roll(1, 100) 大於百分比 → 不管；否則印 "Avoids it"（`0FADh`）、`6776h = 0`、`6780h = FFh`、`6781h` 減一。
- **型別表 `+7`**（`DS:54E7h + 型別 × 10h`）：`poolrad/items` 裡是 00h（24h、26h、55h、56h……）、01h（01h、08h、
  09h、13h、15h、1Ah、2Bh……）、80h（06h、07h、0Ch、14h、16h、17h、18h、21h、2Fh、57h、58h）三組。語意照 AD&D
  讀成揮砍／穿刺／鈍擊（strong inference），規則本身照位元組。

| 碼 | 常式 | bytes | 做什麼 |
|---|---|---|---|
| `68h` THRI-KREEN | entry 97 `28FAh` | `B0 3C 50 / 0E / E8 AD E6` | `0FB7h(記錄, 3Ch)`：兩格外、60% |
| `78h` HILL／FIRE GIANT | entry 114 `2BE1h` | `26 80 7D 2E 57 / 74 0A`、`26 80 7D 2E 58 / 75 0D`、`B0 32 50 … E8 96 E3` | `+0CCh` 那一件的型別是 57h／58h → `0FB7h(記錄, 32h)` |
| `65h` | 見上 | | 掛 `3Bh` |
| `73h` SKELETON | entry 109 `2A95h` | `80 BD E7 54 00 / 74 19`、`24 01 / 08 C0 / 74 0E`、`99 / B9 02 00 / F7 F9` | 打中的那一件存在，`+7` 是 0 或位元 0 立著 → 減半 |
| `74h` MUMMY | entry 110 `2AF8h` | `26 80 7D 32 00 / 7E 0E` | 打中的那一件 `+32h` 有號大於 0 → 減半 |
| `77h` SPECTRE、VAMPIRE、MUMMY、JUJU、FERRAN | entry 113 `2B96h` | `26 80 7D 32 00 / 75 1B`、`26 80 7D 2E 00 / 7F 0B`、`26 80 7D 73 04 / 7D 05`、`C6 06 76 67 00` | 打中的是 NULL 或 `+32h` 是 0 → 行動者 `+2Eh`（種族）有號大於 0 就歸零；否則 `+73h` 有號小於 4 歸零 |
| `7Bh` WRAITH | entry 117 `2D99h` | `26 80 7D 32 00 / 75 1A`、`26 80 7D 31 B1 / 75 10` | NULL → 歸零；`+32h` 0 而 `+31h` B1h（Silver）→ 減半；`+32h` 0 → 歸零 |
| `60h` WIGHT | entry 89 `2634h` | `74 14`、`26 80 7D 32 00 / 75 0F`、`26 80 7D 31 B1 / 74 05` | NULL → 歸零；`+32h` 0 而 `+31h` 不是 B1h → 歸零 |
| `5Eh` JUJU ZOMBIE | entry 87 `25A1h` | `8A 85 E7 54 / 24 81` | 打中的那一件存在，`+7 & 81h` → 減半 |
| `3Ch` | entry 126 `31A9h` | 空常式 | 無 |
| `7Ah` MUMMY | entry 116 `2D4Ch` | spec 153 | `+0CCh` 型別 56h → 3d8、骰數 3；`6777h & 9` → 加骰數 |
| `75h` 不死生物 | entry 111 `2B36h` | `26 80 7D 2E 55 / 75 11`、`B0 01 50 / B0 06 50 / 9A 4D 00 00 01 / 30 E4 / 40` | `+0CCh` 型別 55h → 傷害 = Roll(1, 6) + 1（entry 9，骰數 1）|

減半都是 `A0 76 67 / 30 E4 / 99 / B9 02 00 / F7 F9`（byte 零延伸除 2）。

**`06h` 的 `6777h = 9`**（overlay-12 `0227h` `C6 06 77 67 09`，無條件）：近戰每一下在 `0210h` 把 `6777h` 清 0，
群組 4 的 `06h` 改成 9。群組 5 裡讀它的只有 `7Ah`（加骰數）；倒下時的群組 13 `64h` 也讀它——帶 `06h` 打倒的巨魔
不會再站起來。

## 麻痺：`43h 44h 45h`（overlay-12 `15F7h`）

```
15F7  f(記錄 [bp+0Ah]; 持續 [bp+8]; byte [bp+6])     retf 8
15FDh 目標 = 記錄 +108h 的 +0Ah（攻擊者當下的目標）
1612  entry 7(目標, 類別 0, 修正 0)                   ; B0 00 50 / B0 00 50——[bp+6] 整支沒有讀
1625  成功 → 返回
1629  ov25 entry 26(目標, "is Paralyzed"（15EAh）, 1)
1648  entry 10(目標, 34h, 持續, 0Ch, 0)              ; B0 34 50 / FF 76 08 / B0 0C 50 / B0 00 50
```

| 碼 | 群組 | 常式 | bytes | 持續 |
|---|---|---|---|---|
| `43h` THRI-KREEN | 3 | entry 62 `16A9h` | `B0 02 50 / B0 08 50 / 9A 48 00 00 01`（先擲再豁免）| Roll(2, 8) |
| `44h` GHOUL | 2、3 | entry 63 `16CDh` | `26 80 7D 2E 02 / 74 11`（目標是精靈就整支不做）、`B8 3F 00` | 3Fh |
| `45h` DRIDER | 3 | entry 64 `16FAh` | `B0 01 50 / B0 09 50 / 9A 48 … / 05 0A 00`、`B0 FE 50`（推了，但沒有人讀）| Roll(1, 9) + 0Ah |

`34h` 就是定身術的碼：處理常式 entry 48 `1352h` 只叫 ov25 entry 34，是群組 7（反應攻擊否決，`DS:2880h`）與
remake 既有的「定住就跳過回合」。解除：解除魔法照 `+3` 的 0Ch 擲（spec 098）、到期、倒下時 entry 13 摘。
Pool 的法術表沒有 Remove Paralysis。怪物側 `1B36h`、`1C5Ah` 另有兩處掛 `34h`（別的特殊攻擊），不在本 spec。

## 編號 58（overlay-22 `2E02h`）的先後

```
2E13  entry 15(表上第一格, 37h)：有 → 677Dh = 1、摘 16h、677Dh = 0、結束
2E3F  225Bh（解病鏈）回非 0 → 結束
2E4E  entry 21(表上第一格, Roll(1, 4) + 8, 0)：回非 0 → "is Healed"（2DF8h）
```

`225Bh`（`677Dh = 1`）：entry 15 摘 `22h`；entry 15 摘 `2Bh`，有就再摘 `2Ch`、`1Fh`；entry 15 摘 `32h`，有就再摘
`39h`。任一個 entry 15 回 1 結果就是 1。

**entry 21（`175Dh`，`f(記錄, 量, 旗標)`）**：法術的三個呼叫端（`1074h`、`2E62h`、`2FA7h`）都推旗標 0。

```
1770  +10Ch 不在 {0, 1, 4, 5}（CS:173Dh，首 byte 33h）→ 回 0
1796  旗標 0 → 身上有 32h 就回 0
17B9  +11Bh += 量（byte），無號大於 +32h 就墊回 +32h
17F4  +10Dh == 0：狀態 5 → 4；狀態 4 而不在戰鬥中 → entry 1(4Eh, 記錄, NULL, 1) 站起來
183B  回 1
```

## `20AEh` 收不收屍體

玩家瞄準走 `1E09h` → `352Ch(施法者, 射程, 0, 範圍旗標, &確定, &輸出)`；Manual 是 `2DD3h`，參數照原樣往下傳：

```
2F80  游標那一格有站著的人（[bp-35h] > 0）→ 目標 = DS:6517h[那一位]
2F9F  否則地形類別是 1Fh → 走 DS:6634h 那張屍體表（7 bytes 一筆，+4／+5 是 X／Y），相符的蓋掉前一個
302A  距離大於射程或類別 FFh → 不能選
3062  目標非 NULL → 1087h（能不能打）
306D  第四個引數是 1 才擋「自己」與「屍體（類別 1Fh）」
30DE  目標是 NULL 而且不是範圍法術 → 不能選
```

施法那一條的第四個引數是 `1E09h` 推的 `B0 00 50`，所以**法術瞄得到屍體**。Next／Prev 的候選是 `32C3h` 以
`0138h:003Eh` 從位置表做的近鄰查詢，倒下的人已從位置表拿掉（strong inference），只有 Manual 瞄得到。

屍體表由 ov32 entry 20（`0E11h`，倒下的三個入口都叫它）寫：runtime `+13h` 是 0 時 `6673h` 加一，記下記錄、X、Y、
原地形，地形改成 1Fh（原本是 1Eh 雲就不改）。entry 22 站起來時 ov32 entry 21 以 `[bp+6] = 1` 把那一筆清掉。
逃走與投降走 entry 11，不進屍體表。

效果那一側：`08BCh` 的 `090Ch..0A62h` 只跳過表上清成 NULL 的那一格，整段與 entry 20 都沒有看 `+10Ch`／`+10Dh`
（exact），所以瞄到屍體的緩毒術照樣掛 `16h`、`1846h` 把人扶起來。

## remake 的對應

| 原版 | remake |
|---|---|
| entry 13 ＋ 群組 13 | `app.combatantDown`（`cmd/pool-game/death_effects.go`）：近戰（`resolveAttackSwings` 倒下那一段）、法術（`applySpellDamage`）、`poisonKill`（`005Ah`）|
| `63h`／`5Fh` | `gamepack.BoarRallyHitPoints`、`BoarRallied`；`deathTeardown` 的 "Falls dead" |
| `64h`／`66h` | `gamepack.TrollRevival`、`TrollRisesRetry`；`deathTeardown` 以 `MaxHitPoints` 站起來 |
| `65h`／`3Bh`／`62h` | `gamepack.TrollWounded`（群組 5、6）、`RegenerationBegins`、`Regenerate`；`startRound` 在 `tickEffects` 前叫 `regenerate` |
| 群組 5 其餘 | `gamepack.MeleeTargetEffects`；`app.meleeTargetGroup`（`29h` 之後）|
| `6777h` 在近戰 | `meleeDamageAfterEffects` 清 0、`06h` 寫 `CreatureBaneDamageFlags`（`tacticalState.SpellDamage.Flags` 就是 `DS:6777h`）|
| 麻痺 | `gamepack.ParalysisAttacks`、`NewParalysisNode`；`app.paralysisSpecialAttack`（群組 3 在毒之後、群組 2 在吸取之後）|
| 編號 58 | `neutralizeOnBoard`（戰場，整支）、`applyFieldEffect`（營地）；`gamepack.CureDiseaseChain` |
| entry 21 | `gamepack.HealByEntry21`；戰場 `app.healOnBoard`（通用的治療也改走它）|
| 屍體表 | `tacticalState.Corpses`、`corpseAt`、`forgetCorpse`（`poisonRecover` 站起來時）；`confirmManualAim` |
| 08BCh 不看狀態 | `castEffectOnly` 拿掉「體型 0 就跳過」 |

測試（`cmd/pool-game/death_effects_test.go`，從 `Update()` 送鍵）：`TestWildBoarFightsOnAndThenFallsDead`、
`TestTrollGetsBackUpUnlessBurnt`、`TestTrollRegeneratesThreeRoundsAfterBeingHurt`、
`TestUndeadWeaponRulesReadTheHittingItem`、`TestMonsterParalysisHoldsTheTargetExceptElvesAgainstGhouls`、
`TestGreaterHealCuresDiseaseInsteadOfHealing`、`TestSlowPoisonReachesAPoisonedCorpseByManualAim`；純規則在
`internal/gamepack/death_effects_test.go`（`68h`／`78h` 的 `0FB7h`、`75h`、`7Ah` 走 `MeleeTargetEffects` 直接驗）。
變異檢查逐條拿掉規則重跑，十六個全紅（2026-09-27 實跑）：野豬站起來、`5Fh` 倒地、巨魔的 `66h`、`06h` 的
`6777h`、`66h` 站起來、再生的接點、`3Bh` 的收尾、群組 5 其餘、麻痺的兩個形態、編號 58 戰場的解病與治療、營地的
解病、Manual 收屍體、`08BCh` 不看狀態、屍體表的記錄。

### 對既有探針的影響

倒下時 entry 13 會摘掉睡眠 `35h`。作弊通關那一趟（`TestMainlineProbeCheatMenuToEnding`）裡被自己的催眠術放倒
又被 OGRE 打倒的隊員，鎖 HP 拉起來之後不再睡著、接著施法，戰鬥走向從貧民窟那一場起改變；之後有一扇鎖著的門
（GEO2/20 (7,3)）第一輪 BASH／PICK／KNOCK 都沒開，駕駛原本只會一直選 EXIT、在門前繞到 guard 用完。
改成跟原版玩家一樣回來再撞（`walkAllowing` 整輪試過又退出過一次就重來一輪），這一趟照樣走到結局；收據多了
一趟同一組旗標下的 貧民窟 (15,4) → 城區 (0,4) 往返（兩個檢查點），其餘檢查點的順序與旗標不變。

## 與原版不同、寫明的幾處

| 原版 | remake | 理由 |
|---|---|---|
| 倒下時狀態依打穿點數分 4／5／6（entry 28）| 近戰與法術倒下一律寫 5（既有行為，未改）；野豬的 n 由打穿點數直接換算 | 倒下狀態的分流不在本 issue；`63h` 只需要 n |
| 編號 58 的 Roll(1, 4) + 8 只在治療那一支擲 | CastEffect 放出去時就擲（`spell_cast.go`）| remake 的施法先擲好整組數值；亂數次數多一次 |
| 屍體格的地形改成 1Fh，雲散時還原的是雲記下的原地形 | 屍體可選的條件是「那一格現在不是雲（1Eh）」| remake 不改盤面地形；死在雲裡、雲散之後那一具原版選不到，remake 選得到 |
| Manual 停在屍體上時底列印名字（`3012h` ov25 entry 5）| 不印 | 狀態列只有一行 |
| `0FB7h` 的 "Avoids it" 之後群組 5 其餘的碼照跑（`75h`／`7Ah` 會把傷害改回非 0）| `29h`／`68h`／`78h` 擋掉就不再往下 | 帶 `29h`／`68h`／`78h` 的目標沒有同時帶 `75h`／`7Ah` 的（monster-atlas）|

## 被推翻的斷言

| 舊寫法 | 現況 | 依據 |
|---|---|---|
| `1004h` 摘十五個碼（`DS:0C28h..0C36h`，morale.go 舊版）| 十六個，最後一格 `4Bh` | `102Fh` `80 7E FF 10` |
| 戰場上的治療直接加、不封頂（remake 舊行為）| entry 21：狀態 {0, 1, 4, 5}、`32h` 不治、封頂 `+32h`、倒著的瀕死改昏迷 | `1770h..183Bh` |
| 瞄不到屍體、`08BCh` 跳過體型 0 的格子（remake 舊行為）| Manual 查屍體表；`08BCh` 只跳過 NULL | `2F9Fh`、`306Dh`、`092Fh` |

## 卡點

- **重掛 `4Eh` 的 `+3`**（spec 153）：`1A1Fh` 以 NULL 節點讀 `0000:0003`，那是中斷向量表 INT 00h 的段位址高位元組。
  Turbo Pascal RTL 開機時把 INT 00h 換成自己的除零處理常式（strong inference），所以這個 byte 是 RTL 程式碼段的
  高位元組，取決於載入位址——unknown。remake 維持 FFh（解除魔法解不掉）。
- **`6775h` 的殘值**（spec 153〈677Ah 與 6775h〉）：只在 `6777h` 為 0 的 1 點傷害（`0Fh`／`2Ch` 的收尾）碰上帶
  `69h`／`6Ah` 的目標時有差，remake 當 0（hypothesis）。
