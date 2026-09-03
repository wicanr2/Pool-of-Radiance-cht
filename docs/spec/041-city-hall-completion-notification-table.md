# Spec 041：City Hall 完成通知狀態表

狀態：READY（結構清冊、`4AC1h` 增量、二十五槽的 `FEh` 生產者、貧民窟那一條的
完整玩家鏈）；DRAFT（其餘各槽的完成條件與正常玩家路徑實跑）。
日期：2026-09-03。

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

## 每一槽的 `FEh` 生產者（2026-09-03，完整）

`cmd/pool-ecl-memory-audit` 只給「哪裡碰到這個位址」，不給寫進去的值。這一張表
是把八個封存檔的每一個區塊做線性指令掃描（`ecl.ScanKnownInstructions`），挑出
`09h SAVE` 與 `35h SAVE TABLE` 裡**目的落在 `4AA6h..4ABFh` 而且值是 `FEh`** 的
指令。線性掃描的好處是**三個靜態解不開的區塊也掃得到**——槽 20 的生產者就在
其中一個（`ecl5/7`），先前逐條展開的方法看不到它。

| 槽 | 位址 | City Hall 通知（節錄）| 寫 `FEh` 的地方 |
|---:|---|---|---|
| 0 | `4AA6h` | `THE COUNCIL HAS AWARDED A BONUS FOR YOUR ELIMI` | ecl8/29 `9E02h` |
| 1 | `4AA7h` | `WITH SOKAL KEEP IN OUR HANDS, WE CAN USE BOATS` | ecl4/21 `ADAAh`、ecl4/21 `ADD0h` |
| 2 | `4AA8h` | `YOU HAVE CLEARED THE AREA NEXT TO THE EVIL TEM` | ecl1/24 `9B3Eh` |
| 3 | `4AA9h` | `BIVANT WAS DELIGHTED WITH THE CHILD'S RESCUE.` | ecl6/1 `993Dh`、ecl6/1 `A7BDh`、ecl6/1 `B57Ah` |
| 4 | `4AAAh` | `WE FIND THESE DISCOURSES VALUABLE.` | ecl2/15 `A004h` |
| 5 | `4AABh` | `THE COUNCIL WILL BE AMUSED BY THE DESCRIPTIONS` | ecl2/15 `A0B5h` |
| 6 | `4AACh` | `THESE MAPS SHOULD HELP US TO LOCATE SEVERAL LE` | ecl2/15 `A282h` |
| 7 | `4AADh` | `THESE HISTORIES CONTAIN MUCH USEFUL INFORMATIO` | ecl2/15 `A321h` |
| 8 | `4AAEh` | `THE RECORDS PROVIDE INSIGHTS INTO MUCH THAT WA` | ecl2/15 `A3B3h` |
| 9 | `4AAFh` | `THIS MATERIAL IS OF SMALL VALUE.` | ecl2/15 `A10Fh` |
| 10 | `4AB0h` | `YOUR SUCCESS AT PODAL PLAZA IS NOTED.` | ecl1/18 `A82Bh`、ecl1/18 `A86Bh` |
| 11 | `4AB1h` | `THE COUNCIL HAS VOTED A SPECIAL PRIZE FOR ENDI` | ecl4/10 `B177h` |
| 12 | `4AB2h` | `THE COUNCIL WAS PLEASED BY THE ELIMINATION OF` | ecl3/14 `B361h` |
| 13 | `4AB3h` | `WITH THE CLEARING OF THE STOJONOW RIVER, WE CA` | ecl7/23 `AC53h` |
| 14 | `4AB4h` | `WE WERE PLEASANTLY SURPRISED THAT YOU COMPLETE` | ecl6/25 `9E6Ah` |
| 15 | `4AB5h` | `YOUR ELIMINATION OF THE LIZARDMEN MENACE HAS G` | ecl8/16 `9CCFh`、ecl8/16 `A362h` |
| 16 | `4AB6h` | `THIS LOSS OF THE KOBOLD FORCES WILL BE A MAJOR` | ecl8/13 `A502h` |
| 17 | `4AB7h` | `WE WERE MUCH RELIEVED BY THE NEWS YOU BRING FR` | ecl7/17 `A1F6h` |
| 18 | `4AB8h` | `COUNCILMAN CADORNA LEFT THIS PAYMENT FOR SOME` | ecl3/0 `AB8Ah` |
| 19 | `4AB9h` | `YOUR TAKING THE GATE WILL ENABLE OUR FORCES TO` | ecl2/9 `A702h` |
| 20 | `4ABAh` | `CONGRATULATIONS! YOUR QUEST IS OVER! TYRANTHRA` | ecl5/7 `A815h` |
| 21 | `4ABBh` | `YOUR CLEARING OF THE SLUM AREAS PERMITS US TO` | ecl2/20 `B6B5h` |
| 22 | `4ABCh` | （沒有文字） | **沒有** |
| 23 | `4ABDh` | `CONGRATULATIONS, YOU MAY KEEP ALL YOU FOUND IN` | ecl1/24 `ABC8h` |
| 24 | `4ABEh` | `PORPHYRYS CADORNA IS A TRAITOR TO THE CITY. IF` | ecl3/8 `A2BAh`、ecl3/8 `AF45h`、ecl5/4 `ADA6h` |
| 25 | `4ABFh` | `THE COUNCIL HAS NOTED THE PASSING OF THE TRAIT` | ecl5/3 `B19Ah` |

**二十六槽裡二十五槽有生產者，只有槽 22（`4ABCh`）沒有**——它的通知文字也是空的
（`9D63h` 的第 22 個目標是 `A4DBh RETURN`），所以那一格在原版就是空的，不是漏掃。

值得單獨記的兩件事：

- **槽 20 是結局**：`CONGRATULATIONS! YOUR QUEST IS OVER! TYRANTHRAXUS IS
  DEFEATED!`，生產者是 `ecl5/7` 的 `A815h`。
- **槽 24 有三個生產者**（`ecl3/8` 兩處與 `ecl5/4` 一處），是唯一由市政廳自己
  也會寫的槽。

同一次掃描另外量到三個**不是 `FEh`** 的寫入，它們是進行中的狀態不是完成：
`ecl6/1` 的 `A862h` 寫 `80h`、`AF3Bh` 寫 `01`；`ecl6/28` 的 `993Ch`／`B5F1h` 寫
`FDh` 到槽 14；`ecl3/8` 的 `AA2Ah` 寫 `01` 到槽 10。**`FDh` 與 `FEh` 差一階**，
把它們一起當成完成會讓通知早一步跑出來。

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

**貧民窟那一條（槽 21）已經整條走完**：真的打 25 場、`4ABBh` 一場加一到 `FEh`、
走出貧民窟回城區、進市政廳走到職員面前、`4AC1h` 由 0 變 1、拿到獎賞
（金 250 白金 50 首飾 1，與 `B5EDh`／`B604h`／`B61Bh`／`B632h` 四張表在槽 21
的原值逐項相符）、挑 Share 分錢、挑 Exit 之後 `9F5Ah` 把槽清成 `FFh`。
驗收是 `TestTwentyFiveRealSlumsWinsEarnTheCityHallReward`，負對照是
`TestCityHallPaysNothingBeforeTheCommissionIsDone`（一場都不打去交差，
沒有獎賞、`4AC1h` 不動）。

其餘各槽剩下的是完成條件與正常玩家路徑：走到地圖事件、滿足該區條件、
看到 City Hall 通知、`4AC1h` 增量、存檔。達成前不得用測試直接注入 `4AC1h=4`
宣稱墓園委託已正常解鎖。
