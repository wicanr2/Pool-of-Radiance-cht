# Spec 049：Pool 285-byte 角色／怪物戰鬥欄位

狀態：READY（HP、AC、THAC0、第一組傷害骰、Movement 的 record 解碼）；
DRAFT（命中／傷害運算、initiative、攻擊次數、特殊攻擊與戰術回合）。
日期：2026-09-01。

## 輸入、工具與位址空間

- DOS `GAME.OVR`：SHA-256
  `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638`。
- `overlay-17.bin`：SHA-256
  `f92fed1bcf009daa8638ba0ab1f8f9a680c0ad3b2b43b799faa9383b02f7d1e4`。
  IDA Pro 9.4 匯出保存於
  `docs/audit/ida-overlay17-monster-loader-candidates.json`。
- `overlay-19.bin`：SHA-256
  `4694cb51c5ead4a8e6a155767b8601261c6d74444458ae5c7b6afe54f5bbdca9`；
  角色資料頁 entry 1 與其 local formatters 的 IDA Pro 9.4 匯出保存於
  `docs/audit/ida-overlay19-character-sheet.json`、
  `docs/audit/ida-overlay19-character-stats.json`。
- `overlay-25.bin`：SHA-256
  `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e`；
  AC／HP formatter 匯出保存於
  `docs/audit/ida-overlay25-ac-hp-formatters.json`。
- 所有下列指令位址均為 **overlay-local file offset、base 0、16-bit metapc**；
  沒有把它們混作 runtime segment:offset 或 `GAME.OVR` file offset。輸出保留原始
  bytes、運算元與函式邊界，未以推測改名覆蓋原定位。

## 285-byte unified record（exact）

overlay-17 `0ACDh..0E64h` 的 MON loader 在 `0C1Bh` push `011Dh`，接著呼叫
`05BBh:1508h`，把 DAX block 的 285 bytes 完整複製到角色 record。載入後只清除
數個 runtime linked-list pointer，另載入 63-byte ITM 與 9-byte SPC 節點；因此
MON*CHA 與玩家 CHA 使用同一份 285-byte record layout。這只證明資料形狀與共用
layout，不會讓尚未追到 consumer 的 offset 自動成為已知語意。

## 顯示 consumer 與欄位契約（exact）

overlay-19 entry 1（`004Ah`）呼叫 local `0607h` 顯示角色 combat stats。原始 Pascal
labels 位於 `05BDh AC`、`05C4h HP`、`05CBh THAC0`、`05D4h d`、
`05D6h +`、`05D8h -`、`05DAh Damage`、`05E9h Encumbrance`、
`05FBh Movement`。其欄位消費如下：

| record offset | 形狀 | 原版 consumer | typed 語意 |
| --- | --- | --- | --- |
| `32h` | byte | overlay-25 `0978h` HP formatter 以 `+11Bh` 與 `+32h` 比較，決定受傷顏色 | max HP |
| `102h` | word | overlay-19 `07F4h` 顯示於 Encumbrance label 旁 | encumbrance；本輪不供怪物戰鬥使用 |
| `110h` | byte | overlay-19 `0673h..0681h` 顯示 `60 - value` 於 THAC0 label 旁 | THAC0 internal encoding；typed 值為 `60-value` |
| `111h` | byte | overlay-25 `08F3h` formatter 取 `value-60` 的絕對值；`value>60` 時加負號 | AC internal encoding；typed 值為 `60-value` |
| `115h` | byte | overlay-19 `06FDh` 接到 damage 字串 | 第一組 damage dice count |
| `117h` | byte | overlay-19 `0728h`，置於 `d` 後 | 第一組 damage die sides |
| `119h` | signed byte | overlay-19 `075Ah..07C9h` 依正負加 `+`／`-` 並顯示絕對值 | 第一組 damage bonus |
| `11Bh` | byte | overlay-25 `0978h` HP formatter顯示此值；小於 `+32h` 時用受傷顏色 | current HP |
| `11Ch` | byte | overlay-19 `081Dh` 讀入 movement，再依物品效果調整後顯示 | base movement |

`+32h` 與 `+11Bh` 的方向亦修正 Spec 004 早期保守描述：建角完成時兩者相等，
所以單一樣本只能看出鏡像；HP formatter 明確顯示 `+11Bh`，並以 `+32h` 作上限比較，
故 current／max 的方向現已閉合。

## 真實資料抽樣

`MON2CHA.DAX` 的 Slums 遭遇兩筆 ORC：

| block | HP current/max | AC | THAC0 | damage | movement |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 13 | 5/5 | 6 | 20 | `2d4-1` | 9 |
| 4 | 5/5 | 6 | 19 | `1d8` | 9 |

上述數值由 typed 公式解碼真實 block；它們只證明這兩個 block，不把「ORC 必然是
同一組數值」寫成全域斷言。ECL 仍以 block identity 區分兩筆同名模板。

## 實作與停止線

1. `MonsterRecord` 保留完整 `Raw [285]byte`，只對上表已閉合且戰鬥需要的欄位提供
   accessors；不得因數值看似 AD&D 合理就命名其他 byte。
2. AC／THAC0 必須按原版 internal encoding 解碼，不可直接把 raw byte 當畫面值。
3. damage bonus 必須按 signed byte 解碼；`FFh` 是 `-1`，不是 255。
4. 本規格尚未授權 attack roll、damage roll、turn order 或勝敗 continuation；這些
   必須另追 Pool 戰鬥 consumer 並成為 READY，不能用現代規則或 CoAB offset 補洞。

## 驗收

- 合成 record 逐欄驗證 signed bonus、`60-value` 編碼及 HP current/max 方向。
- 真實 `MON2CHA.DAX` block 13／4 固化上表抽樣。
- Pool 全套 `go test ./...`、`go vet ./...`；測試仍以本機共用 engine 的臨時
  `modfile` 執行，不修改 game pack 的版本依賴。
