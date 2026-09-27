# Spec 154：ADD NPC 加入當下的記錄、MONnSPC、橫掃（`+6Bh`）

狀態：CONFORMED（ADD NPC 當下的記錄 209 bytes 逐位元組對上 dosgolem 雇 WARRIOR 那一刻；
開打那一幀 40 筆 combatant 的 `+6Bh` 全部符合 entry 7 尾段）；READY（橫掃 overlay-13 entry 10，
碼 exact，remake 盤面接上，原版沒有掃到的那一幀收據）。日期：2026-09-27。主台帳：GitHub issue #107
（#97 留下的五條，spec 147〈NPC〉「仍未閉合」那一表）。

## 輸入

- DOS ZIP：`Pool of Radiance (1988).zip`，SHA-256
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- overlay（`workplace/ovr/overlay-*.bin`，均對 `docs/audit/dos-ovr-manifest.json`；overlay-local
  file offset、base 0；`coab-go-test:20260729` 的 `objdump -D -b binary -m i8086 -M intel`）：

| overlay | SHA-256 |
|---|---|
| 03 | `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f` |
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` |
| 09 | `6b47e49d1a2e73427b9863560350965d6d5b43d3dd89df89d9b7f43a68409258` |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` |
| 17 | `f92fed1bcf009daa8638ba0ab1f8f9a680c0ad3b2b43b799faa9383b02f7d1e4` |
| 23 | `7ce2992587b64fc751c6de34b6eb54dab5025b098d26a9b6fbd2868f93b2f2d8` |
| 25 | `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e` |

- far call 反查（`segment = (executable_file_offset − stub_offset − 3B0h) ÷ 16`）：`0096h` = overlay-13
  （`52h` 是 entry 10 `0E8Ch`、`66h` entry 14 `17F5h`、`6Bh` entry 15 `1883h`），`00FCh` = overlay-23
  （`25h` 是 entry 1 `0000h`），`010Ah` = overlay-25（`2Ah` entry 2 `0762h`、`43h` entry 7 `0BBEh`）。
- 執行期：`docs/audit/dosgolem-npc-combat-runtime.json`（`tools/dosgolem-npc-combat-runtime.py`；
  shots SHA-256 `322bf49e…68775f`，dosgolem checkout `fix/pool-wipe-anykey` `2ab6f65`，`start.exe`
  `12811cbc…10d9f`）。同一份鍵序跑兩次，收據逐欄相同。

## ADD NPC 加入時寫了什麼（exact）

`36h ADD NPC`（overlay-03 `2EDBh`，spec 091）的完整順序：

| 位址 | bytes | 作用 |
|---|---|---|
| overlay-17 `1244h` | `E8 49 FC` | `0E90h` 讀 MONnCHA、MONnSPC（`1051h`，鏈在 `+7Fh`）、MONnITM（`112Eh`）同一個 block（spec 142） |
| overlay-17 `1250h..125Ch` | `B8 00 01`…`26 88 85 AB 00` | `+ABh = Random(100h)` |
| overlay-17 `1261h..13B1h` | `26 88 85 BC 00`、`26 88 85 BB 00` | 戰鬥造形：隨機挑一個職業，`+BCh`、`+BBh` 查 `DS:08DBh..0909h` 的表（擲骰） |
| overlay-17 `13BDh` → `13EDh` | `E8 2D 00` | 接到 `DS:5CF4h` 串列尾、`+BFh` 取第一個空的隊伍順序、`[4937h]+67Ch` 加一、`DS:5CF0h` 指向它 |
| overlay-17 `150Fh` | `9A 25 00 FC 00` | overlay-23 entry 1（spec 072）：`+2Dh`、`+73h`、`+A1h`、豁免、法術格數、盜賊技能照職業等級重算 |
| overlay-03 `2F21h..2F25h` | `26 88 85 84 00` | `+84h` 士氣（spec 091） |
| overlay-03 `2F2Ah..2F40h` | `26 C6 85 0E 01 00`／`01` | `+10Eh` 陣營 |
| overlay-03 `2F46h..2F4Eh` | `FF 36 F2 5C`、`9A 43 00 0A 01` | overlay-25 entry 7：`+110h..+11Ch`、`+102h`、`+100h`、`+6Bh` 寫回**這一位自己的記錄** |
| overlay-03 `2F53h..2F5Bh` | `9A 2A 00 0A 01` | overlay-25 entry 2（`0762h`）：只重畫隊伍名單（`07BEh` 起沿 `5CF4h` 印名字、AC、HP），不寫記錄 |

overlay-23 entry 1 裡 remake 接上的三格（`0009h..009Dh`）：

```
0009  +2Dh = 0，i = 0..7：+2Dh = max(+2Dh, [3C16h + i × 0Bh + 等級])  ; 等級 0 那一列八格都是 40
0066  +73h < 等級 → +73h = 等級                                         ; 有號，只往上
007C  i = 2（戰士）：等級 > 6 → +A1h = 3，否則 2                        ; 每一筆都寫
```

entry 7 另外兩格 remake 以前沒寫（exact）：

- `+100h` 手數：`0C0Ch`（`26 C6 85 00 01 00`）清 0，`0D57h..0D6Eh`（`26 88 85 00 01`）對穿戴中的
  每件加型別表 `+1`，與 spec 144 的 `readiedSlots` 同一段。
- `+6Bh` 橫掃上限：見下一節。

### 執行期收據（exact）

`tools/dosgolem-npc-combat-runtime.py`：建一名 HERO → 導覽 → 訓練所（ecl3/11）競技場 (7,2) →
`n`（不決鬥）→ `y`（找夥伴）→ 開價 "THERE IS A FINE WARRIOR WHO WILL JOIN YOU FOR 1 SHARE" → `y`。
ADD NPC（ecl3/11 `9F1Ch`，`@6E79 = 67h`、`@6E7A = 55`）一返回就沿 `DS:5CF4h` 讀新的那一筆 11Dh bytes。

remake 同一條（`TestAddNPCRecordMatchesTheHireRuntimeReceipt`：跑同一條 `9F1Ch`、同樣兩個變數）
**209 bytes 逐位元組相同**。不比的 76 bytes 都是原版執行期才有的：遠指標（`+C8h..+FFh`、
`+104h..+10Bh`）、兩次擲骰（`+ABh`、`+BBh`／`+BCh`）、隊伍順序 `+BFh` 與造形配色 `+C0h..+C7h`。

| 欄位 | 樣板 | 原版加入後 | 來源 |
|---|---:|---:|---|
| `+2Dh` 基礎 THAC0 | 40 | 41 | overlay-23 `0009h` |
| `+84h` 士氣 | FFh | 9Bh | `2F21h`（55 ÷ 2 ∣ 80h） |
| `+10Eh` 陣營 | 1 | 0 | `2F34h` |
| `+110h` THAC0 | 149 | 41 | entry 7（手上的武器沒有加值，力量 16 不加命中） |
| `+111h`／`+112h` AC | 50／48 | 57／54 | entry 7 |
| `+119h` 傷害加值 | 66h | 1 | entry 7（武器那一支） |
| `+11Ch` 腳程 | 12 | 9 | entry 7（盔甲與負重） |
| `+102h` 負重 | 0 | 801 | entry 7 |
| `+100h` 手數 | 0 | 2 | entry 7 |
| `+6Bh` 橫掃上限 | 2 | 2 | entry 7 尾段 |

負對照：樣板的 `+2Dh` 與 `+110h` 都不等於原版加入後的值（測試裡有這一條，否則收據證明不了什麼）。
十四個 NPC 來源（spec 091 的八個呼叫點，雇傭兵表展開）裡 overlay-23 改得到 `+2Dh` 的是
WARRIOR 40 → 41、HERO 與 PRINCESS FATIMA 42 → 43；`+73h`、`+A1h`、豁免五格與樣板都相同。

## MONnSPC 效果串列（exact，資料上是空的）

`0E90h` 在 `1051h` 用同一個 block 讀 `MON<n>SPC.DAX`，9 bytes 一節鏈在 `+7Fh`（spec 142），ADD NPC 與
LOAD MONSTER 同一支。資料：十四個來源全部沒有節點——`MON3SPC.DAX` 在 ZIP 裡根本沒有（十個雇傭兵與
DIRTEN 都在 archive 3），其餘四個 archive 的 SPC 沒有那些 block。收據的 WARRIOR `+7Fh` 是 0。
remake 照樣接上（`applyAddNPC` 呼叫 `loadMonsterEffects`），`TestAddNPCCarriesTheMonsterEffectList`
用換過的載入器驗接線，並逐一確認十四個來源的資料是空的（哪天資料對不上，那一條會先開口）。

## `+6Bh` 與橫掃（exact）

### 誰寫

entry 7 尾段 `1000h..102Eh`：

```
1003  26 80 BD 98 00 00   cmp byte [+98h], 0 ; jle 1026h   ; 戰士等級
100E  26 80 7D 2E 00      cmp byte [+2Eh], 0 ; jle 1026h   ; 種族碼
1018  26 8A 85 98 00 / 1020 26 88 45 6B      +6Bh = +98h
1029  26 C6 45 6B 01                           +6Bh = 1
```

只看戰士那一格；聖騎士、遊俠不算。七名預設人物 8、8、1、1、1、1、4 全部符合
（`TestRecomputeCombatFieldsMatchesThePremadeCharacters` 把它列進衍生欄位）；開打那一幀 40 筆
combatant（HERO、WARRIOR、兩名 LEVEL 3 MU、十二名 6TH LVL FIGHTER、十二名 AIDES、十二名 NOMAD；
上限 1、3、6 都有）全部符合（`TestSweepLimitMatchesEveryCombatantInTheRuntimeReceipt`）。

### 誰讀

整個 GAME.OVR 以 `es:[di+6Bh]` 定址記錄的有五處。寫：entry 7 的 `1020h`／`1029h`；overlay-12
`0B8Fh`（`26 88 45 6B`，`+6Bh = +98h`，同一段把 `+10Fh` 設 1、`+76h` 設 0）；overlay-22 `210Dh`
（`26 C6 45 6B 00`，同一段設 `+10Eh`、`+10Fh = 1`、`+76h = 2`）。後兩處是另外兩種建立 combatant
的路，用途沒有追，remake 沒有對應。讀只有一處：

```
overlay-13 entry 1（每回合初始化，spec 062）
0073  26 8A 45 6B        runtime +5 = 記錄 +6Bh
007A  26 88 45 05
```

runtime（`+108h`）`+5` 的讀取端全在 overlay-13 entry 10（`0E8Ch..1084h`，字串 `0E85h` = "sweeps"）：

```
0E99  al = 攻擊者 +113h ; cmp al, runtime +5 ; jb 0EAFh    ; 第一形態剩的次數 >= 上限 → 不掃
0EAF  目標 +73h != 0 → 不掃                                  ; 生命骰不到一個才掃得到
0EC8  010Ah:00C5h（overlay-25 entry 33，距離）!= 1 → 不掃
0EDD  010Ah:00C0h(攻擊者, 1)（entry 32，隔壁的對手）→ DS:6CD7h 名單；數 +73h == 0 的 n
0F4D  n <= +113h → 不掃
0F62  n = min(n, runtime +5)
0F77  entry 20(攻擊者, "sweeps", 0Ah, 1)
0FB3  目標不在名單第一格就與第一格對調
0FEF  名單逐格：還有額度、+73h == 0 → entry 14、+113h = 1（1048h）、entry 15（彈藥 NULL）、額度減一
107A  回 1
```

呼叫端三條，都在一般攻擊之前問它，回 1 就不打一般攻擊、這一次行動用掉：

| 位址 | bytes | 路 |
|---|---|---|
| overlay-08 `0DA0h` | `9A 52 00 96 00` | 走進敵人那一格；成立在 `0DA9h` 寫出參 1 |
| overlay-09 `0E80h` | `9A 52 00 96 00` | 電腦；成立在 `0E8Fh` 叫 overlay-25 entry 34 |
| overlay-13 `2CD0h` | `E8 B9 E1` | 瞄準按 T（`2C17h`，`[bp+0Eh] == 1`）；成立在 `2CDDh` 叫 entry 34 |

反應攻擊（spec 059）直接進 entry 15，不經過這一支。

攻擊核心 `1404h..17F2h` 收尾在 `17D0h`（`26 C6 45 05 00`）把攻擊者的 runtime `+5` 清 0：**同一回合
出過手的人再輪到自己時掃不了**（下一回合 entry 1 才從 `+6Bh` 抄回來）。收據對得上：開打那一幀
WARRIOR 的 runtime `+5` 是 0，而它的記錄顯示已經動過（`+0CCh` 換了一件、第一形態 1d8+1 → 1d6、
負重 801 → 793，看起來是換弓射了箭——這一句是 strong inference）；還沒動的 HERO 與怪物都等於
各自的 `+6Bh`。（`14F9h` 那條「砍無助者」捷徑跳過 `17D0h`，remake 沒有那一條。）

資料面：`+73h == 0` 的只有 KOBOLD／KOBOLD LEADER、GOBLIN GUARD／GOBLIN LEADER（archive 1、2、7、8）與
archive 6／7 的 GUARD、BANDIT、BUCCANEER、MERCHANT。貧民窟的狗頭人與哥布林是戰士兩級以後最常掃到的。

### remake

- `gamepack.SweepLimit`（`+6Bh` 規則）、`gamepack.SweepTargets`（`0E8Ch` 的判定與順序）；
  `recomputeFromBase` 尾段寫 `+6Bh` 與 `+100h`，所以怪物、NPC、DOS 匯出的角色都有。
- 盤面（`cmd/pool-game/sweep.go`）：`resolveWeaponAttack` 一開頭先問 `sweep`，三條出手的路都經過它。
  上限：NPC 讀記錄 `+98h`／`+2Eh`，玩家角色用職業等級與種族碼，怪物讀 staged 記錄；
  `tacticalState.Sweep` 記下這一回合出過手的人（`resolveAttackSwings` 結尾），那時上限是 0。
  每一刀是 `resolveAttackSwings(一下第一形態)`，第一個人另外吃這一相位剩的第二形態（`+114h` 不動）。
- 驗證：`TestFighterSweepsGoblinsBelowOneHitDie`（戰士 3 級、三隻哥布林圍著：一則 "sweeps"、
  三隻各一下；同一回合再打照一般攻擊；下一回合上限回來）、`TestSweepNeedsMoreLimitThanAttacks`
  （一級戰士不掃、兩級只砍兩隻）、`TestSweepTargetsFollowsEntry10`（四個不掃的分支與順序）。

## 隊伍強度（`1Dh PARTYSTRENGTH`）

`partyStrengthRecord` 的 NPC 分支原本排在職業等級那一條之後：NPC 從記錄帶出了等級（addnpc.go），
所以一直走到玩家那一支，被當成能力值全 0 的角色算。現在 NPC 先走，讀記錄的 `+96h`／`+9Bh`、
`+110h`／`+111h`，後兩格由同一支 entry 7 從記錄與物品現算——原版每次換裝都會重跑 entry 7，
所以記錄上的值恆等於現算的值，舊存檔裡的樣板殘值也拿得到對的數字。

## 舊存檔

- 記錄欄位：`migrateNPCRecords`（讀檔時與 `migrateNPCMorale` 一起跑）把 overlay-23 的三格、`+10Eh`
  與 entry 7 的結果寫回每一位 NPC。三者只由記錄本身、陣營與物品決定，新存檔再跑一次得到同一組值。
- **物品欄空的舊 NPC 不補 MONnITM，也不升 schema。** schema 8 在 #97（2026-09-26）之前就已經在用
  （2026-09-11 起），所以 #97 前後的存檔 schema 相同；現在升 schema 只分得出「#107 以後」，分不出
  手上這些檔案裡哪一份是「那時還沒載」、哪一份是「後來交給別人了」。後一種補了就是複製物品，
  而 remake 允許 NPC 交易（`item_page_trade.go`）。影響面是 #97 之前雇的 NPC 開打時沒有裝備；
  重新雇一次（或讀 #97 以後的存檔）就有。

## 還沒接

| 項目 | 等級 | 為什麼 |
|---|---|---|
| ADD NPC 的兩次擲骰（`+ABh`、戰鬥造形 `+BBh`／`+BCh`）與 `+BFh`、`+C0h..+C7h` | exact（碼） | 不影響規則；remake 的造形另有來源。亂數流因此與原版不同步（原版加入一名 NPC 多耗三次 `Random`） |
| overlay-23 對 NPC 的法術格數與盜賊技能 | exact（碼）、未比對 | 十四個來源的施法者只有 DIRTEN、CURATE、ACOLYTE 與四名法師；`+0B2h..+0B7h` 會照表重寫（例：法師 7 級的表列是 0）。remake NPC 的施法讀記錄，這一段沒有接也沒有收據 |
| 戰鬥中 NPC 的 AI 換武器與扣箭 | — | 收據的 WARRIOR 在 HERO 之前就換成弓射了箭；remake 那一側的逐欄對拍不在這一份 |
