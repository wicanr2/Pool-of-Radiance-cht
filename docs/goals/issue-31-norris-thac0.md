# Goal：怪物 THAC0 的來源對回原版——諾里斯的 154（GitHub #31；spec 051／063／097）

主台帳：[issue #31](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/31)。
上一個 goal 的結案位置：[`issue-35-deployment-templates.md`](issue-35-deployment-templates.md)
（#35 關：四場樣板 528 bytes 逐 byte 相同、寫入端只有 `1A99h`／`14CFh`）。

## 現況（2026-09-16，`696f5e9` 已推送）

- `MON8CHA` 第 32 筆 NORRIS THE GRAY 的 `+110h = 154`（0x9A）；remake `MonsterRecord.THAC0()` 是
  `60 − Raw[110h]` → −94，`ResolveHit` 什麼都打得中（補七量到他命中率 88%／81%）。同場蜥蜴人
  `+110h = 44`（THAC0 16）、狗頭人首領 41 都正常。他的記錄 `+2Fh = 2`、`+96h.. = [0 0 5 …]`，
  是 5 級戰士（一版 THAC0 16，內部 44）。
- spec 063 已讀出：overlay-25 entry 1（`0000h`）**用武器重算** `+110h`——
  `record[+110h] = record[+2Dh]`（基礎 THAC0）＋ 型別表旗標的兩個修正 ＋ 武器 `+32h`；
  沒有武器就不動 `+110h`。spec 051 讀出 overlay-25 `0DF4h` 排怪時把 `+0A2h/+0A4h/+0A6h`
  抄到 `+114h..`。**還沒讀**：排怪那一段有沒有叫 entry 1（有武器的怪物 `+110h` 是不是每場
  都從 `+2Dh` 重算）、諾里斯的 `+2Dh` 是多少、`+110h` 的 154 是樣板殘值還是別的編碼。
- 工具：`cmd/pool-monster-inventory`（八個 MONnCHA 的記錄）、`tools/ida.sh`（overlay-25）、
  `docs/audit/ida-overlay25-combat-stat-recalc.json`（`0E30h..1010h`）、dosgolem（`-peek` 讀
  `6517h` 表指到的記錄可以直接讀執行期 `+110h`，古托井那一場走得到：主線探針 136 的路線）。
- remake 讀點：`cmd/pool-game/tactical.go` 兩處 `state.THAC0[index] = uint8(60 - record.THAC0())`。

## 2026-09-16 收在哪

#36 先判讀：樣板 `+110h` 是殘值（42/43 筆 bit 7 ＝ `+2Dh + 109`），開打時 overlay-10 `1380h`
對每一隻叫 entry 7 把 `+2Dh` 抄進 `+110h`（沒武器再加力量修正，`12AEh` 自己查 `+0AAh`），
dosgolem 獸人家二十隻執行期逐隻等於 `+2Dh`。remake `CombatThac0Internal` 照算，諾里斯
154 → 46（表面 14），`TestNorrisTHAC0ReadsTheBaseFieldNotTheTemplate`、
`TestMonsterTHAC0MatchesTheOrcHomeRuntimeReceipt`；補七諾里斯那一列重量。#31／#36 關。

## 提示詞（可直接貼給 `/goal`）

> 目標：怪物的 THAC0 照原版算，諾里斯那一場敵方命中率回到正常；主台帳 #31。
>
> 1. **先讀原版，不改 remake。** 兩件事：(a) 八個 MONnCHA 掃出 `+110h` 不在 20..60 的記錄，
>    連同 `+2Dh`（基礎 THAC0）、`+2Fh`／`+96h..`（職業與等級）、`+0CCh`（有沒有武器）列成表進
>    `docs/audit/`；(b) overlay-25 `0DF4h` 排怪那一段讀完——它有沒有呼叫 entry 1（`0000h`）讓
>    有武器的怪物從 `+2Dh` 重算 `+110h`、沒武器的怎麼辦、`+2Dh` 本身從哪來（樣板？職業表
>    `spec 097`？）。每一條標位址、bytes、證據等級，寫進 spec 063（或 051／097）。
> 2. **拿原版當裁判。** dosgolem 走主線探針 136 的路線到古托井打諾里斯（或任何 `+110h` 異常的
>    怪物的那一場），開打那一幀讀 `6517h` 表指到的記錄 `+110h`／`+2Dh`／`+0CCh`，記下原版執行期
>    的 THAC0；同一幀讀蜥蜴人的當正對照（要等於 44）。
> 3. **改 remake。** 照讀出來的規則算怪物 THAC0（有武器就走 spec 063 那條、`+2Dh` 為底），
>    `MonsterRecord.THAC0()` 的呼叫端一起換；一條測試釘住諾里斯 THAC0 = 16（內部 44）與正對照
>    蜥蜴人 16；補七諾里斯那一列重量（`battle_tally_test.go`）。
> 4. **不可越線：** 不改 MONnCHA 資料、不對單一怪物硬編碼、不動 engine／CoAB；沒有畫面變動就
>    不用對拍。
> 5. **每輪開跑前寫「這輪跟上輪差在哪」**；先一筆記錄（諾里斯）再全表。
> 6. **台帳：** 每輪在 #31 留言；spec 063／051 狀態行；收尾 `docs/worklist.json` →
>    `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` → `-mode verify` →
>    `gh issue list --state open`（鏡像那一條的 verify 是「還沒有 `TestNorrisTHAC0`」）。
> 7. **停止線：** `0DF4h` 那一段跨 overlay 呼叫兩層以上還讀不到 THAC0 的來源；dosgolem 走不到
>    古托井（先修它，另開 issue）；`+2Dh` 也是異常值（那就是資料問題，另開 issue 討論怎麼處理）。

## 已知風險與待決

- 154 有 bit 7：可能是「基礎 THAC0 + 128」之類的旗標編碼，也可能只是樣板殘值（spec 051
  的 `+114h..` 就是殘值）；靠 (b) 分辨，不要先猜。這件事單獨開了 **#36** 判讀（讀 `0DF4h`、
  bit 7 清單、一筆執行期收據），#31 的修法由 #36 的結論決定——步驟 1(b) 與 2 就是 #36。
- `+2Dh` 若也不對，remake 要不要照職業表算（spec 097 有訓練所的表）是規則問題，先回報。
