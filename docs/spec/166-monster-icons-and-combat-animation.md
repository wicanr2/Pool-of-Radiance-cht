# Spec 166：怪物戰場造形與戰鬥動畫（彈道、閃光、倒下）

狀態：READY（造形來源、換色、畫法、動畫的圖格／路徑／時長／觸發點皆 exact；例外逐條標在表裡）
日期：2026-09-29；issue #63、#121

怪物在戰場上畫哪一張、什麼顏色，以及攻擊、施法、受傷、倒下時盤面上播的那幾段動畫。
音效（PC 喇叭）不在這一份。

## 輸入、工具與位址基準

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `START.EXE` SHA-256：`12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`；資料段基底在
  檔案位移 30640（`DS:2880h` 讀到 `33 34 35 1F` 當正對照，spec 149）。
- overlay 由 `workplace/ovr/overlay-NN.bin` 讀，SHA-256 見 `docs/audit/dos-ovr-manifest.json`：
  overlay-03 `5a3a18bd…`、overlay-11 `d1e1c62e…`、overlay-13 `4d53df20…`、overlay-32 `efc22ba8…`、
  overlay-33 `6acdf145…`。其餘（07、12、22、24、25）同一份清冊。
- 反組譯：`coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel`；位址一律是 overlay-local。
  far call 段號換算：段 = (overlay 的 `executable_file_offset` − 3B0h) ÷ 16，偏移是 stub 位移
  （例：overlay-33 在 `1820h` → `147h`，`147h:0039h` 是 entry 5；overlay-25 在 `1450h` → `10Ah`；
  overlay-32 在 `1780h` → `13Dh`）。
- 原版畫面：dosgolem 基準 `workplace/dosgolem-ref/85-c.idx`（第一場遭遇，provenance 見同目錄）。

## 1. 怪物的造形是 `CPICn.DAX`，顏色是圖本身的（exact）

`0Bh LOAD MONSTER` 的處理常式 overlay-03 `044Dh`：

```
04F0  9a 25 00 45 00          第三個運算元 → [bp-3]（造形編號）
0507  bf 48 04 / 0e 57 …      Pascal "CPIC"（0448h：04 43 50 49 43）複製進 [bp-13Eh]
051B  push 字串、[bp-3]、DS:6D49h
0529  9a 39 00 47 01          overlay-33 entry 5（01CAh）(槽 DS:6D49h, 區塊, 名字)
0571  a0 49 6d / 26 88 85 bf 00   記錄 +0BFh = 槽
```

overlay-33 entry 5（`01CAh`，`retf 8`）依名字分三條：

| 名字 | 位址 | 做法 |
|---|---|---|
| `CHEAD`／`CBODY` | `01FEh..02FAh` | 名字最後一個字是 'T' 就區塊 + 40h；站立圖進 `5DB8h + 槽×8`、區塊 + 80h 的動作圖進 `5DBCh + 槽×8` |
| `COMSPR`／`ICON` | `02FDh..0408h` | 同上，不換色；只有 `ICON` 在 `03B8h..0403h` 用 `0CE6h → 0D06h` 換色 |
| 其他（`CPIC`） | `040Bh..04EEh` | 名字接上 `Str(DS:52D4h)`（`040Bh..042Ch`），站立與動作兩張都用 `0CE6h → 0CF6h` 換色（`0462h..0485h`、`04CBh..04EEh`） |

`DS:52D4h` 是目前的檔案組，MONnCHA 用的也是它（spec 142 的 overlay-17 `0EB7h`）。換色表（`START.EXE`
資料段，全部 overlay 只有 overlay-33 讀，沒有寫入點）：

```
DS:0CE6h  00 01 02 03 04 05 06 07 08 09 0A 0B 0C 0D 0E 0F   ; 原色
DS:0CF6h  00 01 02 03 04 05 06 07 08 09 0A 0B 0C 08 0E 0F   ; CPIC：0Dh → 08h
DS:0D06h  00 01 02 03 04 05 06 07 08 09 0A 0B 0C 00 0E 0F   ; ICON：0Dh → 00h
```

`CPIC1..8.DAX` 的 218 張站立與動作圖裡沒有一個像素是 0Dh，所以**怪物的顏色就是圖上的顏色**；
「哥布林那一類是紅的」是那一張圖本來就是紅的，不是另一份配色。

**同狀態核對**：dosgolem 基準 `85-c`（第一場遭遇）戰場 (32,56) 那一格的 222 個不透明像素，與
`CPIC1.DAX` 區塊 2 逐像素相同（222／222）；拿 `CBODY.DAX` 區塊 2 套預設配色去比，連輪廓都對不上。
`CPIC1`／`CPIC2`／`CPIC4` 的區塊 2 是同一張圖。

之前 remake 把 `IconBlock` 當成 `CBODY.DAX` 的身體編號、套玩家的預設六組配色，**怪物畫錯的是圖，
不只是顏色**。

### 圖的大小與畫法（exact）

`CPICn` 的區塊有 24×24、48×24、24×48、48×48 四種（最後一戰的 TYRANITHRAXUS 是 `CPIC5` 區塊 42h，
48×48）。畫的時候（overlay-33 entry 6 `04FEh`，`retf 0Ah`）：

```
04FE  (槽 [bp+6], 動作 [bp+8], 朝向 [bp+0Ah], X [bp+0Ch], Y [bp+0Eh])
0504  圖 = 5DB8h + 動作×4 + 槽×8
0568  80 7e 0a 03 / 76 2d     朝向 > 3 → 18E:0052 左右翻過來再畫（0594h）
0606..062E  18E:004D(畫面, 圖, …, X×3, Y×3)   以 8 像素為單位，從那一格的左上角往右下鋪
```

X、Y 是戰鬥員那一格減去視窗原點（`5FA8h`／`5FF0h`，overlay-32 `0068h`）；呼叫端是 overlay-32 entry 14
（`0AF6h` 的 `0BD9h`）與重畫盤面的 `02CCh`／`0397h`／`0A6Fh`，朝向是 runtime `+108h` 的 `+9`。所以大型怪物的
圖只畫一次、蓋過它佔的幾格；朝向 4..7 的人（隊員也一樣）是左右翻著畫的。

「18E:0052 是左右翻」是 strong inference：它是 resident 的圖形常式，沒有逐行讀；證據是 overlay-25
entry 23 用它做「翻轉」的那一格，而箭的東北與西北正好是同一張圖翻過來（下一節）。

### 競技場的複製品（exact）

overlay-07 `1B20h..1B36h` 以 Pascal "cpic"（`1AA9h`）、區塊 0Bh、槽 `DS:6D49h` 叫 entry 5；
`1BCFh..1BD5h` 把槽寫進複製品記錄的 `+0BFh`。記錄照抄上場的人，**造形不照抄**：畫的是這一組
`CPIC` 的區塊 0Bh。spec 150 那一列「畫的是照抄的造形」由這一條取代。

## 2. `COMSPR.DAX` 的圖示槽（exact）

開場初始化 overlay-11（`002Dh` 起）：

```
04FE..052F  for i = 0..0Bh: overlay-33 entry 5(槽 i + 0Dh, 區塊 i, "COMSPR")
0531..0546  overlay-33 entry 5(槽 19h, 區塊 19h, "COMSPR")
```

所以槽 0Dh..18h 是 `COMSPR` 區塊 0..0Bh、槽 19h 是區塊 19h；動作圖都是區塊 + 80h。

## 3. 三支動畫常式（overlay-25，exact）

| entry | 位址 | 做什麼 |
|---|---|---|
| 23 | `18E1h`（`retf 8`） | (翻轉 `[bp+6]`, 第幾格 `[bp+8]`, 動作 `[bp+0Ah]`, 槽 `[bp+0Ch]`)：把 `5DB8h + 動作×4 + 槽×8` 那一張抄進 `DS:6D29h` 那張合成圖的第幾格；翻轉非 0 就先經 `18E:0052`（`1953h`） |
| 24 | `1A30h`（`retf 2`） | (槽)：依序 (動作 0, 格 0, 不翻)、(0, 1, 翻)、(1, 2, 翻)、(1, 3, 不翻) 叫 entry 23 |
| 25 | `1A7Dh`（`retf 0Ch`） | (起 X `[bp+10h]`, 起 Y `[bp+0Eh]`, 終 X `[bp+0Ch]`, 終 Y `[bp+0Ah]`, 格數 `[bp+8]`, 毫秒 `[bp+6]`)：畫一道 |

spec 161 把 overlay-12 `1D03h`／`2CAAh` 寫成「聲音」，那兩個是 entry 24（組合成圖），聲音是別的呼叫
（`27F:0000`）。

### entry 25 的每一步（exact）

```
1A9F  [bp-94h] 整表填 08h
1AF4..1B2C  overlay-31 0196h 初始化走訪器（起 ×3 → 終 ×3），再反覆 02C4h 走一步：
            每叫一次把走訪器 +19h（這一步的方向）存進表，走不動（回 0）的那一次也存，然後停
1B3B  存的筆數 < 2 → 整支返回（同一格不畫）
1CDD  delay 為 0 時只在 X、Y 都是 3 的倍數才畫（所有呼叫端 delay 都非 0，這一條沒用上）
1D03..1D5E  畫 6D29h 的第 [bp-0B3h] 格、顯示、Delay(毫秒)、擦掉
1D63..1D70  格數 +1，到 [bp+8] 歸零
1D75..1DB5  筆數 +1，照**這一筆**的方向（DS:274Ah／2753h）移一步
1E62  筆數 < 總數而且沒出框 → 再畫
1F4D..2027  畫在終點 ×3、Delay、擦掉
```

`DS:274Ah`／`2753h` 是 (dx, dy)：`00 01 01 01 00 FF FF FF 00`／`FF FF 00 01 01 01 00 FF 00`（方向 0..7
從北順時針，8 不動）。因為筆數先加一才取方向，**第一筆的方向從來沒用上**，最後一筆（08h）不動：
往東兩格（6 步）畫在 0、1、2、3、4、5、5，再在終點 6 畫一次，共 8 次。

出了 7×7 視窗（`1DB9h..1DD5h`，相對 0..12h 以外）原版會把視窗捲過去（`13D:0061`）；remake 的視窗
不跟著捲，出框的那幾步照樣佔時間，只是畫不出來（停止線）。

## 4. 誰叫它們

### 攻擊（overlay-13，exact）

攻擊包裝 entry 15（`1883h`，spec 051／059 的那一支）：

```
1889..18E5  目標被出手不到三次 → 轉身面向攻擊者（已接，spec 059）
18F6..191C  攻擊者轉向目標，以動作圖重畫（未接，停止線）
1931  Delay(64h)                                              （未接，停止線）
193A..1955  帶著彈藥（[bp+0Ah] 非 NULL）→ entry 24（268Eh）(彈藥, 目標, 攻擊者)
1958..1981  攻擊者 +0CCh 那一件是型別 2Fh → 以那一件再叫一次 268Eh
19D2  1404h 擲命中與傷害
```

`268Eh` 依那一件的型別（`+2Eh`）與攻擊者看目標的方向（`261Bh`）挑圖，再叫 entry 25：

| 型別 | 位址 | 圖 | 格數 | 毫秒 |
|---|---|---|---|---|
| 09h 1Ch 15h 1Fh 49h | `26DBh..2779h` | 方向奇數：槽 0Eh（COMSPR 1），方向 3／5 用動作圖，5／7 左右翻；方向偶數：槽 0Dh + 方向 mod 4（COMSPR 0 或 2），方向 ÷ 4 是動作圖 | 1 | 0Ah |
| 02h 07h 14h | `277Ch..27A1h` | entry 24(槽 10h)（COMSPR 3） | 4 | 32h |
| 55h 56h | `27A3h..27C3h` | entry 24(槽 11h)（COMSPR 4） | 4 | 32h |
| 其餘 | `27C5h..2804h` | 槽 14h（型別 2Fh 是 15h）的站立、動作兩格（COMSPR 7／8） | 2 | 14h |

型別名稱是 strong inference（以 24h 長劍、49h 箭、1Ch 弩矢、55h 聖水、56h 油瓶這幾個已知的錨點
對 Gold Box 的型別表）：09h 飛鏢、15h 標槍、1Fh 矛、02h 手斧、07h 棍棒、14h 鎚、2Fh 投石索、08h 匕首。

### 施法與射線（overlay-22，exact）

| 位址 | 做什麼 |
|---|---|
| `0D67h..0E24h`（entry 5，共用施法常式） | 挑到目標（`[6A78h]` 之後 `[bp+6]` 非 0）而且在戰鬥中：entry 24(槽 12h)（COMSPR 5），施法者轉向瞄準點、以動作圖重畫，entry 25(施法者 → `DS:6CADh`／`6CAEh`, 4, 1Eh)，然後才派發處理常式。物品施法（overlay-19 `1BD8h`）也叫這一支 |
| `2906h`／`2922h`、`2AA8h`（射線 `2919h`） | 拉射線之前與每打完一格之後以 entry 24(槽 13h)（COMSPR 6）重組合成圖；每一段在打那一格（`2AC3h`）之前 entry 25(這一段的起點 `2A0Eh`, 停下的那一格, 4, 32h) |
| `3167h..3185h`（吐息 `3092h`） | entry 24(槽 13h)，entry 25(吐的人 `[bp-2]`／`[bp-3]` → 算好的那一點, 4, 32h)，再拉射線 |

### 怪物的接近效果（overlay-12，exact，spec 161）

| 位址 | 圖 | 路徑 | 格數 | 毫秒 |
|---|---|---|---|---|
| `1D03h`／`1D42h` 石化凝視 | 槽 12h 四格 | 自己 → 目標 | 4 | 2Dh |
| `1E11h` 鏡子反射 | 同上 | 目標 → 自己 | 4 | 2Dh |
| `1F3Ah`／`1F79h` 魅惑凝視 | 槽 12h 四格 | 自己 → 目標 | 4 | 2Dh |
| `2CAAh`／`2CE9h` 噴酸 | 槽 17h 四格，只用第一格 | 自己 → 目標 | 1 | 1Eh |

### 閃光（overlay-25 entry 26 `2041h`，exact）

戰鬥中（`DS:4954h == 5`）：旗標 `[bp+0Ah]` 非 0 用槽 16h（COMSPR 9）、否則槽 17h（COMSPR 0Ah）組四格；
entry 20(記錄, 字串, 0Ah, **0**) 印名字與那一句（不停拍，字留在右欄）；在目標那一格（`13D:007A`
→ `5FA8h`／`5FF0h`）逐格畫、每格 Delay(46h)，輪數旗標 1 是遊戲速度 + 1、旗標 0 是 1；旗標 0（或
遊戲速度 0）之後 `21BAh` 再等一拍。

旗標 0 的呼叫端只有 overlay-24 entry 19（`14ECh`，法術與效果的傷害）：在扣血（`14FBh`）之前印
"takes N points of damage "（`127Dh`／`1284h`，一點是 `1297h`），接上種類：`DS:6777h & F7h` 是 1／2／4／10h
接 "from Fire"／"from Cold"／"from Electricity"／"from Acid"（`13DEh..14A5h`），**另外** `6777h & 8`
等於 `6777h` 本身（含 0）接 "from Magic"（`14A8h`）。傷害是 0 整段跳過（`137Ah`）。

### 倒下（overlay-32 entry 20 `0E11h`，exact）

```
0E17  不在戰鬥中 → 聲音、等一拍，返回
0E2F..0E5F  這個人已經在屍體表（6634h，最多 8 筆；只記隊員，見 0F4Bh）→ 整支返回，不畫也不等
0EAC..0F3E  他佔的每一格（5E88h 的體型、2858h 的偏移）畫 DS:5E78h，顯示
            5E78h = 5DB8h + 18h × 8：槽 18h 的站立圖，COMSPR 0Bh（紅底骷髏）
0F43..0FF9  runtime +13h 為 0 才記進屍體表、那一格地形改 1Fh
0FFD..1006  聲音、等一拍
100B..1055  從盤面上擦掉、重畫
```

呼叫端：overlay-13 `0619h`（攻擊）、overlay-24 `1631h`（法術與效果）、overlay-12 `00E6h`（石化與毒死）；
前兩條在群組 13 之後才看 `+10Dh`：還是 0 就叫 entry 20，被救起來就只等一拍（overlay-13 `0620h`、
overlay-24 `1638h`）。

## remake

| 原版 | remake |
|---|---|
| `CPIC` + `Str(52D4h)`、換色表 | `internal/assets/monster_icon.go`（`ReadMonsterCombatIcon`） |
| `LOAD MONSTER` 的造形、競技場複製品 | `cmd/pool-game/tactical.go`（`monsterBoardIcon`、`boardIconFor`）；`stagedMonster.Archive` |
| entry 6 的畫法（左上角往右下、朝向 > 3 翻） | `cmd/pool-game/combat_screen.go` 的 `drawCombatBoard`、`combat_animation.go` 的 `drawBoardPicture` |
| entry 23／24／25、閃光、骷髏 | `cmd/pool-game/combat_animation.go` |
| 攻擊包裝的彈道 | `missile.go` 的 `resolveWeaponAttack` → `weaponMissile` |
| 施法、射線、吐息 | `cast.go` 的 `castSpell`、`spell_targets.go` 的 `castSpellRay`（`combat.SpellRay.Segment`）、`breath.go` |
| 凝視、噴酸 | `approach_effects.go` |
| entry 19 的受傷閃光 | `cast.go` 的 `applySpellDamage` → `hurtNotice`；物品穿戴的傷害 `item_page_trade.go` |
| entry 26 旗標 1 | `combat_notice.go` 的 `turnedNotice` |
| 倒下 | `death_effects.go` 的 `combatantDown` → `downBeat` |

**動畫不動規則與擲骰。** remake 的戰鬥邏輯在按鍵那一個影格算完，畫面靠停拍佇列一則一則放
（combat_notice.go）；動畫也是一則，停拍期間 `tacticalInput` 不讀鍵。原版的動畫都在擲骰之間，
不消耗亂數，所以放進佇列的先後就是原版的先後，擲骰順序不變。Delay 的毫秒換成 60 Hz 影格、
無條件進位。

輪到骷髏之前，倒下的人照樣畫在盤面上（原版在 `1016h` 之後才擦掉）；輪到骷髏時不畫他的造形。
前一則是不停拍的 entry 20（`Kept`）時，那幾行字在動畫期間照樣留在右欄。

## 驗收

- `internal/assets/monster_icon_test.go`：`CPIC1` 區塊 2 的兩列逐像素等於 dosgolem `85-c` 那一格；
  `CPIC5` 區塊 42h 48×48；換色表只動 0Dh。
- `cmd/pool-game/combat_animation_test.go`：走訪器每一步與輪格、型別挑圖、玩家射箭的彈道排在攻擊
  之前而且停完不改任何盤面數值、骷髏與屍體表、兩種閃光的槽與長度、受傷那一句的種類。
- `arena_duel_test.go`：複製品的造形是 `CPIC` 區塊 0Bh。
- 發行包對拍：`combat` 那一張的 remake 側是 F5 預覽（盤面上沒有怪物），怪物造形與動畫不在那一張裡；
  數字見 commit message。

## 停止線

攻擊者轉向目標的動作圖與 `1931h` 的 Delay(100)、施法者的動作圖、彈道出框時捲動視窗、聲音——
寫在 `docs/audit/stop-line.md`。
