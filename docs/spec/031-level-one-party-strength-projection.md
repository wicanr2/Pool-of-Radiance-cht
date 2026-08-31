# Spec 031：一級新角色的 `PARTYSTRENGTH` 投影

狀態：CONFORMED（一級 remake 建角 roster）；DRAFT（訓練、升級、裝備與匯入 DOS 角色）。
日期：2026-09-01。

## 範圍

將目前唯一可由 remake 正常 UI 建立的一級角色，投影成 Spec 030 的五欄公式並安裝到
同一個 ECL session。這不是完整角色成長規格；訓練與裝備系統完成時必須改由持久角色
衍生狀態提供，不得永久把所有角色鎖成一級。

## 三份原版正對照

三份 `.CHA` 均為既有 DOSBox 正常建角 oracle，285 bytes；位址是檔案內 record offset：

| 檔案／SHA-256 | class／STR／DEX | `+96h` | `+9Bh` | `+110h` | `+111h` | `+11Bh` |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| `FEM.CHA`／`69292e21…a379a6` | Fighter／14／13 | 0 | 0 | 40 | 50 | 7 |
| `HMU.CHA`／`8ba4b517…ddb86` | Magic-User／17／14 | 0 | 1 | 41 | 50 | 2 |
| `HTH.CHA`／`c81904d7…76cd` | Thief／13／16 | 0 | 0 | 40 | 52 | 4 |

完整雜湊由測試固定，不以表內省略字串作比較。Pool overlay-16 對 `+96h` 的 active-class
讀寫、HMU 的 `+9Bh=1`，以及同系列已驗證欄位表共同支持：`+96h` 是 Cleric level、
`+9Bh` 是 Magic-User level。這兩個名字對 Pool 為 `strong inference`；原始值與公式結果
仍為 exact。

shared engine `combat/ability` 的 STR hit／DEX defence 表來自另一個 Gold Box title；三份
Pool oracle 是本作正對照：

- `stored attack = 40 + StrengthHitAdjustment(StrengthIndex(STR, exceptional))`：
  STR 14→40、17→41、13→40。
- `stored AC = 50 + DexterityDefenceAdjustment(DEX)`：DEX 13→50、14→50、16→52。

這證明目前正常建角會用到的三個樣本；全值域仍由 shared table 的第二作品證據支持，
不得宣稱已逐值跑完 Pool DOS。

## Typed 投影

1. `gamepack.InitialCharacter` snapshot 保存姓名、目前 HP、STR、DEX、exceptional STR、
   class ID；建立 ECL session 後 immutable。
2. class ID 只接受現行建角 catalog 的組合；含 `cleric`／`magic-user` component 時對應
   level byte 為 1，否則 0。未知 class ID 失敗即關閉。
3. 每名角色依 Spec 030 公式取整數貢獻，party total 逐次加到 `uint8`，保留原版 wrap。
4. resolver 每次 `1Dh` 從 snapshot 計算並立即寫回；HP 使用 session 建立當下的
   `CurrentHP`。未來 session 中 HP 會變動時，必須改成有版本的 live roster seam，
   不能繼續宣稱 immutable snapshot 完整。

## 驗收與負對照

- DOS parser 對三份真實 `.CHA` 固定五欄及各自 contribution。
- synthetic snapshot 分別重現 Fighter／Magic-User／Thief 的 stored attack／AC 與 level。
- 未知 class、非法 STR／exceptional percentile 失敗即關閉。
- 真實 block8 的窄 fixture 由 `A592h` 進入同一控制流：strength 18 跳回一般委託，
  strength 19 顯示 Valhingen Graveyard 文字，且 `6E79h` writeback 與 request receipt
  均吻合。這是分支正／負對照，不冒充從主選單走到該旗標狀態的完整玩家路線。
- 正常玩家路徑日後抵達 `A5A8h` 時不再因缺 resolver 失敗；其後仍須依 Spec 028 對
  graveyard 選擇、`TREASURE` 與 `COMBAT` 各自建 READY 契約。
