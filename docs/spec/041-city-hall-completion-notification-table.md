# Spec 041：City Hall 完成通知狀態表

狀態：READY（結構清冊、`4AC1h` 增量、十二槽的 producer 寫入點）；DRAFT（各槽完成條件與正常玩家路徑實跑）。
日期：2026-09-01。

## 問題與勘誤

`4AC1h` 不是單一委託旗標。ECL3/block8 的 `9D0Eh..9DBEh` 會逐槽掃描
`4AA6h..4ABFh`：值為 `FEh` 的槽才由 `9D63h ON GOSUB` 顯示一次完成通知，之後由
`9F5Ah SAVE TABLE FFh,4AA6h,[6E79h]` 將該槽 acknowledge。26 個通知分支中只有十個
執行 `ADD 1,[4AC1h] → [4AC1h]`。因此「十個 producer」是十類會提高公告進度的完成
通知，不是十個獨立儲存欄位，也不存在 clerk 單點授予 `4AC1h` 的 producer。

## 可重生證據

- 原始控制流：`docs/audit/dos-ecl3-block8-trace.json`，ECL3/block8 SHA-256
  `fd446439973994d9f887b8628b512054a822369eb4360769b21eb2224982f012`，位址基準
  `9900h`。
- 全 ECL 直接引用：`docs/audit/dos-city-hall-reward-state-refs.json`，由
  `cmd/pool-ecl-memory-audit -addresses 4AA6,...,4ABF` 重生。三個尚未完整解碼的 block
  仍列為 failure，因此這是直接 operand 引用的下界，不能用零列宣稱沒有間接 producer。
- 結構矩陣：`cmd/pool-city-hall-audit` 必須從 `9D63h` 的 26 條有序 edge 產生每槽
  address、target、第一段玩家文字及是否增加 `4AC1h`；刪除任一 edge 或 producer
  都必須使測試失敗。

## READY 契約

1. 狀態槽 `i` 對應 `4AA6h+i`，合法索引為 0..25；順序只能取自原始 ON GOSUB edge。
2. 掃描時只有值 `FEh` 會進通知；通知完成後該槽改為 `FFh`，避免重複演出。
3. `4AC1h` 只在矩陣標示的十槽各增加一，不能按所有 26 槽增加，也不能自行 cap 在 9。
4. 通知文字與增量只描述 clerk 的結算行為；各槽在其他 ECL 的真正完成條件仍須逐條
   追到 producer，未閉合者維持 raw address，不把文字標題冒充欄位名稱。

## 已定位的 producer（2026-09-02）

依 spec 055 的區域腳本定位，逐一在各區 ECL block 內找出寫 `FEh` 的指令，補上垂直鏈
的前半段。位址取自 `workplace/bin/pool-ecl-trace` 的完整展開，屬 `exact`：

| 槽 | 位址 | City Hall 通知（節錄）| producer | 寫入點 |
|---:|---|---|---|---|
| 0 | `4AA6h` | `ELIMINATING NORRIS THE GRAY` | 古托井 `ECL8/29` | `9E02h` |
| 1 | `4AA7h` | `WITH SOKAL KEEP IN OUR HANDS` | 索卡爾城堡 `ECL4/21` | `ADAAh`、`ADD0h` |
| 4 | `4AAAh` | `WE FIND THESE DISCOURSES VALUABLE` | 曼多爾圖書館 `ECL2/15` | `A004h` |
| 5 | `4AABh` | `AMUSED BY THE DESCRIPTIONS` | 同上 | `A0B5h` |
| 6 | `4AACh` | `THESE MAPS SHOULD HELP US` | 同上 | `A10Fh` |
| 7 | `4AADh` | `THESE HISTORIES CONTAIN MUCH USEFUL INFORMATION` | 同上 | `A282h` |
| 8 | `4AAEh` | `THE RECORDS PROVIDE INSIGHTS` | 同上 | `A321h` |
| 9 | `4AAFh` | `THIS MATERIAL IS OF SMALL VALUE` | 同上 | `A3B3h` |
| 10 | `4AB0h` | `YOUR SUCCESS AT PODAL PLAZA` | 波多廣場 `ECL1/18` | `A82Bh`、`A86Bh` |
| 11 | `4AB1h` | `ENDING THE GRAVEYARD MENACE` | 瓦海登墳場 `ECL4/10` | `B177h` |
| 18 | `4AB8h` | `COUNCILMAN CADORNA LEFT THIS PAYMENT` | 新菲蘭城／碼頭 `ECL3/0` | `AB8Ah` |
| 21 | `4ABBh` | `CLEARING OF THE SLUM AREAS` | 貧民區 `ECL2/20` | `B6B5h`（spec 042）|

因此本節原先要求的「至少四條垂直鏈」已有十二槽的 producer 端；仍待補的是各槽的完成
條件（producer 前的判斷）與正常玩家路徑實跑。

### 兩個不進本表的完成旗標

- 卡德納紡織廠 `ECL4/2` 在 `9A7Eh` 寫 `FEh` 到 **`4AE6h`**，不在 `4AA6h..4ABFh` 內。
  與槽 18 的文字對照（卡德納是「留下一筆付款」而非議會公告獎賞）可知，紡織廠屬議員
  私下委託，完成旗標與 City Hall 通知表分離。
- 波多廣場 `ECL1/18` 另在 `B248h` 寫 `FEh` 到 **`4AE7h`**，該處的前置是
  `COMPARE [4A34h],9` → `IF =`，並把 `[4A34h]` 設為 10。這與《軟體世界》攻略所述
  「處理掉十群隨機出現的怪物就算清理完畢」一致（見
  [`docs/reference/walkthrough-notes-softworld-001.md`](../reference/walkthrough-notes-softworld-001.md)）。

### 完成機制不只一種

Slums 用共用計數 helper（`B69Ch`，累加到 25），波多廣場則是 inline 的計數比較。
實作時不可假設所有區域共用同一種完成判定，要逐區取自各自的 producer。

## 後續玩家路徑閘門

producer 端已定位，剩下的是完成條件與正常玩家路徑：走到地圖事件、滿足該區條件、
看到 City Hall 通知、`4AC1h` 增量、存檔。達成前不得用測試直接注入 `4AC1h=4`
宣稱墓園委託已正常解鎖。
