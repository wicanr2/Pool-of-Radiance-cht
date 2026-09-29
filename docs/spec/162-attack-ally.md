# Spec 162：打自己人之前的 "Attack Ally:"

狀態：READY（詢問、倒戈、@6E33 與兩個呼叫點逐條讀過位元組並實作，從 `Update()` 送鍵測試）。
證據是位元組（exact），另註者除外；沒有原版執行期收據逐次對過。日期：2026-09-27。
主台帳：GitHub issue #118。接手 spec 061〈跟著隊伍的非隊員〉表格最後一列。

## 輸入

- DOS ZIP `Pool of Radiance (1988).zip`，SHA-256 `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`；
  `poolrad/start.exe` `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`、
  `poolrad/game.ovr` `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638`。
- overlay 由 `workplace/ovr/overlay-NN.bin` 取（`docs/audit/dos-ovr-manifest.json` 的 `code_sha256`）：

| overlay | code SHA-256 | stub segment |
|---|---|---|
| 08 | `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f` | — |
| 10 | `b929c7040aaa399ba803911c8630e06a8e21b9e7d64a67def69c700d7fe14439` | — |
| 13 | `4d53df204756640fbae8fdc8743ada85ea6e323a9d4dcac496395600a64a2390` | `0096h` |

- `poolrad/ecl7.dax` `74b9aa32373d9bbc90ec615653f9cb99b7b2808759bdc36b8406111435e2e2e3`，
  block 17 `ad8605478679924bee121f74409d2a9c9865f52bc4729822c02143712e5a7a73`（`cmd/pool-ecl-trace -archive 7 -block 17`）。
- 反組譯：`coab-go-test:20260729` 的 GNU objdump（`-D -b binary -m i8086 -M intel`），位址是 overlay
  檔內位移。overlay-13 的 stub segment 是 (`0D10h` − `3B0h`) ÷ 16 = `0096h`，所以 entry 20（`2977h`，
  stub `84h`）是 far call `0096h:0084h`，entry 21（`352Ch`，stub `89h`）是 `0096h:0089h`。

## `2977h`：問不問、答了做什麼

參數（Pascal 由左到右推）：`[bp+0Ah]` 攻擊者、`[bp+6]` 目標，兩個都是遠指標，`retf 8`。

```
2983  9A B6 00 0A 01          010Ah:00B6h(目標)        ; overlay-25 entry 30：(+10Eh == 0) ? 1 : 0，對面的陣營值
2988  C4 7E 0A 26 3A 85 0E 01  cmp al, 攻擊者 +10Eh
2990  74 0B                    相等 → 回 1               ; 目標在對面，直接打
2995  26 80 BD 0F 01 00 74 07  攻擊者 +10Fh == 0 才往下   ; 不是 0（AI 在走）→ 回 1，不問
29A9  BF 69 29 0E 57 9A 34 06 BB 05                      ; 字串 cs:2969h = "Attack Ally: "（長 0Dh）
29BC  9A 3E 00 1D 01           011Dh:003Eh(0Dh, 0Ah, 0Fh) ; 讀一個鍵
29C1  3C 59 74 06              不是 'Y' → 回 0
29CF  C6 06 D6 6C 01           DS:6CD6h = 1
29D4  C4 3E 37 49 26 C7 85 66 06 01 00   [4937h]+666h = 1 ; ECL @6E33
29DF  沿 DS:5CF4h（下一筆 +104h）：
29F6    26 80 BD 0C 01 00 75 26          +10Ch == 0
2A01    26 80 BD 84 00 7F 76 1B          +84h > 7Fh
2A0C    26 C6 85 0E 01 01                +10Eh = 1（敵方）
2A15    +108h 指到的 runtime 子結構 +0Ah／+0Ch = 0   ; 追的目標清掉
2A36  9A BB 00 0A 01           010Ah:00BBh               ; overlay-25 entry 31，兩邊人數（spec 062）
```

`011Dh:003Eh` 是讀一個鍵、只比 `'Y'` 的那一支：spec 098 的 "Abort Spell? "（`0EC0h`）與 spec 097 的
"Do you wish to train?"（`2E5Ch`）都用它，呼叫端一律 `cmp al, 59h`。所以 **Y 以外的任何鍵都是「不打」**
（strong inference：`011Dh:003Eh` 本身沒有逐條讀，行為由三個呼叫端的形狀推出）。

### 誰會倒戈

`+84h` 位元 7 是「要判士氣」（spec 091）：怪物、ADD NPC 加進來的 NPC、競技場的複製品都有；玩家建的
角色是 0（spec 067）。`+10Ch == 0` 排除倒地、死亡、逃走與死靈術叫起來的（狀態 1，spec 098 `21C0h`）。
所以答 Y 之後倒戈的是**跟著隊伍的 NPC 與怪物**，被打的隊員自己不倒戈，其他隊員也不會。
原本就在對面的怪物寫 1 等於沒寫。

全遊戲 `+10Eh == 0` 的怪物只有 EFREETI（spec 061）；ADD NPC 加入的 NPC 則看劇情有沒有帶。

### 兩個呼叫點

| 位置 | 情境 | 回 0 之後 |
|---|---|---|
| overlay-08 `0D76h`（`FF 76 10 FF 76 0E FF 76 0C FF 76 0A 9A 84 00 96 00 08 C0 74 63`） | 移動時走進有人的那一格：`0BF1h` 叫 `0D30h`，那一支先看武器（`0D36h..0D68h`，spec 151），再叫 `2977h` | `0D7Dh` 跳 `0DE2h`：不打，完成旗標不動，留在移動裡 |
| overlay-13 `2C3Fh`（`80 7E 0E 01 75 1B … E8 35 FD 08 C0 75 07 C4 7E 0A 26 C6 05 00`） | 瞄準列的 Target（`36CDh` 按 `T`）叫 `2C17h`，只在旗標 `[bp+0Eh] == 1` 時問 | `2C46h` 把完成旗標清 0，`376Dh` 看到 0 就留在瞄準列 |

`2C17h` 的旗標來自 `352Ch` 的 `[bp+10h]`：overlay-08 A）IM 在 `03D1h` 推 1（`B0 01 50`），施法的
`1E09h` 推 0（`1E2Bh` `B0 00 50`）。所以**用武器瞄才問，施法瞄自己人不問**。

`352Ch` 只在目標搆得到時才把 Target 放上選單（spec 151），所以 remake 先看射程與 Target 在不在，
再問。

## @6E33 與 DS:6CD6h

- **@6E33**：class 1 位移 `(2A00h + 6E33h × 2) mod 10000h = 666h`。`cmd/pool-disp-scan -addresses 666,5AA`
  （`5AAh` 正對照掃出 overlay-14 那四條）只有兩筆：overlay-10 `1F76h`
  `C4 3E 37 49 31 C0 26 89 85 66 06`（開打時歸零）與 overlay-13 `29D8h`（答 Y 立 1）。
  `cmd/pool-ecl-memory-audit -addresses 6E33` 只有一筆讀取：ECL7 block 17 `9ABAh`
  `COMPARE @6E33, 0 / IF = RETURN`，後面把 `@4A7C` 的位元 1 立起、位元 0 清掉。它是 `AFA3h` 與
  `B376h` 兩個 `GOSUB 9ABAh` 的目標，兩者都緊接在 `COMBAT` 之後——那兩場打完，腳本看隊伍有沒有打過
  自己人。
- **DS:6CD6h**：全部 overlay、`game.ovr` 與 `start.exe` 的位元組裡 `D6 6C` 只出現在 `29D1h` 這一條寫入，
  沒有讀的一方（`pool-disp-scan` 絕對定址也只有這一筆）。remake 不做（strong inference：沒有字面
  位址的間接存取沒有排除）。

## remake

`cmd/pool-game/attack_ally.go`：

- `needsAllyPrompt` 是 `2983h..299Bh`：同一邊、而攻擊者不是 AI 在走（`AIDriven`，記錄 `+10Fh`）。
- 走進去撞到（`tacticalInput` 的 `MovementAttack`）：先看武器，再問；Y 走 `bumpAttack`。
- A）IM 的 Target（`resolveAimedAttack`）：射程、Target 在不在之後問；Y 走 `strikeAimedTarget`，
  其他鍵回到瞄準列。施法的瞄準不經過這裡。
- `turnAlliesHostile` 是 `29CFh..2A36h`：@6E33 = 1；狀態 0 而 `+84h > 7Fh` 的每一格改到對面、追的目標清掉。
  `+10Eh` 寫在記錄上，所以隊伍裡的 NPC 同步改 `Party[i].Side` 與記錄 `+10Eh`（戰後像怪物那樣結算、
  再被 `1164h` 摘掉，spec 167）；怪物那一條改 `combatMonsters` 的記錄 `+10Eh`，戰後的經驗值因此算它（overlay-05 entry 2
  `0068h` 只跳過 `+10Eh != 1` 的）。
- 開打時（`deploy` 收尾、`setupMorale` 之前）@6E33 歸零。

問句畫在指令列那一條基線（`ui.attackAllyPrompt`）。

測試（`cmd/pool-game/attack_ally_test.go`，ECL4 block 10 的真實記錄）：

- `TestBumpingAnAllyAsksAttackAlly`：走進 EFREETI 那一格先問；N 什麼也不做；Y 之後 EFREETI 在對面、追的
  目標清掉、記錄 `+10Eh` 是 1、@6E33 是 1、挨了一下，隊員沒有倒戈，經驗值算進 EFREETI。
- `TestAimingAtAnAllyAsksAttackAlly`：A）IM 瞄一個隊員按 ENTER 先問；ESC 留在瞄準列；Y 打到隊員（他仍在
  隊伍那一邊）、EFREETI 倒戈。
- `TestAttackingAFoeDoesNotAsk`、`TestCombatStartClearsAttackedAlly`。

## 未閉合

- 倒戈的隊伍 NPC 在戰後的經驗值與分帳：由 spec 167 接手（#122）。
