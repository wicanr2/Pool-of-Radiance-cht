# Spec 139：Q）UICK 自動戰鬥與 M）OVE 的兩層按鍵

狀態：READY（Q 的效果、SPACE 收回、`+10Fh` 的分派與跨戰鬥保留、M 之後才是方向鍵；
`TestQuickHandsTheTurnToTheAI`／`TestQuickCombatEarnsExperience` 從 `Update()` 送鍵驗；
`2` 的 Magic On／Off 與 AI 施法，`TestQuickMagicOffKeepsSpellsAndTwoTurnsItOn`，issue #64）；
DRAFT（Q 之後那兩個呼叫 `1Dh:1203h`、`26Bh:0F6h` 的內容）。日期：2026-09-15；
2026-09-25 補 `2`。主台帳：GitHub issue #27。

## 一句話

按 Q 把這一位交給電腦：記錄 `+10Fh = 1`，這一回合立刻由怪物 AI（overlay-09 entry 1）
走，之後每一場也是，直到戰鬥中按 SPACE 把全隊（NPC 除外）收回。說明書 p.40：
「敵人比你弱得多時，使用 QUICK 可縮短戰鬥時間，否則最好少用」。

## 輸入

- `overlay-08.bin` SHA-256 `932ce28178ffad8d180db6fdc130f08e540452e788535217979e33fc30da036f`，
  IDA 9.4，overlay-local offset。匯出：`docs/audit/ida-overlay08-player-command-loop.json`
  （`0307h..05E1h`）、`docs/audit/ida-overlay08-quick.json`（`120Eh..125Fh`）。
- 分派與 SPACE 的另一支：spec 096（overlay-08 entry 3 `01E4h`、overlay-09 entry 7 `0FC8h`）。
- 說明書 p.38（MOVE 之後才是方向鍵）、p.40（QUICK）、p.41（SPACE 解除）。

## 玩家指令迴圈：overlay-08 `0307h`（exact）

`0381h` 讀鍵之後逐個比：`M`（`0387h` → `09C3h` 移動）、`V`、`A`、`U`、`C`、`T`、
`Q`（`0447h`）、`D`、`2`（`0490h`）、SPACE（`04D8h`）、`10h`、`-`。

```
0447  al == 'Q' →
044B    120Eh(記錄)                    ; 見下
0455    1Dh:1203h                      ; 未讀
045A    26Bh:0F6h                      ; 未讀
045F    512h:29Eh(0C8h)                ; 延遲 200
0468    [bp-2] = 1                     ; 這一回合結束
046C    058h:25h(記錄)                 ; overlay-09 entry 1：AI 走這一回合
```

`120Eh(記錄)`：

```
1214  記錄 +10Fh = 1                   ; 交給電腦
1222  runtime(+108h) 的目標 +0Ah/+0Ch 為 0 → 返回
1238  目標的 +10Eh == 自己的 +10Eh    ; 瞄著同伴
1251    → 目標清 0                     ; AI 不接手打自己人
```

SPACE（`04D8h`）：沿 `DS:5CF4h` 整條串列，`+84h < 80h`（不是 NPC）的記錄 `+10Fh = 0`。
怪物回合中按鍵走 overlay-09 entry 7 `103Eh`，同一件事（spec 096）。

## `+10Fh` 在記錄裡，不在 runtime（exact）

overlay-08 entry 3（`01E4h`）每一回合依記錄 `+10Fh` 分派：非零走 overlay-09 entry 1，
零走上面的指令迴圈。它是角色記錄的欄位，所以**跨戰鬥保留**、跟著存檔——Q 過一次，
下一場照樣是電腦打，直到 SPACE。

## 兩層按鍵（exact）

原版頂層的 `M` 是 M）OVE，進去之後才用八個方向鍵（說明書 p.38 畫的是數字鍵盤
8 9 6 3 2 1 4 7；aim 游標那一支 overlay-13 `3217h` 逐個比 `H I M Q P O K G`，
spec 127）。所以頂層的 `Q` 是 QUICK、移動中的 `Q` 是東南，兩個不衝突。

## `2`：Magic On／Off（exact，2026-09-25）

`0490h`：鍵是 `32h`（`2`）就把 `DS:6D23h` 反相，印 `Magic On`（`02E2h`）或
`Magic Off`（`02EBh`）；怪物回合中的按鍵檢查（overlay-09 entry 7 `0FF4h`）是同一件事，
字串在 `0FB5h`／`0FBEh`。`DS:6D23h` 全部 overlay 只有四處碰它（`objdump` 掃 `0x6d23`）：
這兩個反相、overlay-10 `1F3Bh` 每場開打**清成 0**、overlay-09 entry 4 `05C0h` 讀。

讀它的那一處是 AI 挑法術（spec 096〈entry 4〉）：`記錄 +84h <= 7Fh 而且 DS:6D23h == 0`
就不施。所以**交給電腦的隊員預設不放法術**，按 `2` 才放；隊伍裡的 NPC 與怪物（`+84h`
80h 以上）不看這個開關。兩個 `Roll(1, 7)` 在這道閘門之前，Magic Off 照樣擲。

## READY 契約

1. `Q`：該角色 `Quick = true`（`+10Fh`），若 AI 記著的目標是同伴就清掉，這一回合
   立刻由 AI 走；之後的每一場開打時這一位都由 AI 走。
2. SPACE：全隊非 NPC 的 `Quick = false`，當場生效（排在 AI 分派之前）。
3. 方向鍵要先按 `M`（或直接用數字鍵盤）；頂層的 `Q` 不移動。
4. Q 打完的仗照常發經驗值——經驗值只看怪物清單（spec 097），與誰下指令無關。
5. `2`：Magic On／Off 反相，狀態列印出來；每場開打是 Off。交給電腦的隊員（非 NPC）
   Off 時不施法、On 時照 spec 096〈entry 4〉挑法術，用掉的是自己記憶陣列裡的那一格。

## 實作對應

`cmd/pool-game/tactical.go`：`quick`／`releaseQuick`、`Moving`、`tacticalStepKeypad`；
`poolsave.Character.Quick`；訊息 `ui.statusQuick`／`ui.statusQuickOff`。測試
`cmd/pool-game/quick_test.go`。探索器的駕駛（`tactical_pilot_test.go`）走一步前先按 M。
`2` 在 `tacticalInput`（排在 AI 分派之前，同 SPACE）呼叫 `foe_cast.go` 的 `toggleMagic`，
開關存在 `tacticalState.Casting.MagicOn`（新的一場就是新的 state，等於每場清 0）；
訊息 `ui.statusMagicOn`／`ui.statusMagicOff`。測試 `cmd/pool-game/foe_cast_test.go`。
