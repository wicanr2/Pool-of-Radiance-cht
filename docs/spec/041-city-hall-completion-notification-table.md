# Spec 041：City Hall 完成通知狀態表

狀態：READY（結構清冊、`4AC1h` 增量、二十五槽的 `FEh` 生產者、三類完成條件
各自的實跑驗證、貧民窟那一條的完整玩家鏈）；DRAFT（各槽的正常玩家路徑實跑）。
日期：2026-09-06。

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

## 完成條件：`FEh` 那一條前面在比什麼（2026-09-03）

把每個 `FEh` 寫入點前面八條指令一起印出來（同一支線性掃描），完成條件就
讀得出來。下表的「條件」欄只寫程式碼真的在比的東西；區域名稱取自市政廳的
通知文字，不是推測。

| 槽 | 區域（取自通知文字）| 完成條件 |
|---:|---|---|
| 0 | 除掉 NORRIS THE GRAY | `ecl8/29 9E02h` |
| 1 | 索寇要塞 | `ecl4/21`：亡魂那一段走完（`AD9Eh` 清 `4A19`、`ADA4h` 寫 `4A26 = FFh`），兩個出口 `ADAAh`／`ADD0h` |
| 2 | 邪惡神殿旁 | `ecl1/24`：`LOAD MONSTER 04` 那一場 `COMBAT` 之後 `6DC7h <= 80h`（打贏）|
| 3 | 救回孩子 | `ecl6/1` 三個出口：入口 0 在 `4AA9 == 1` 時、打贏熊地精那一場（`A751h COMBAT`）、或帶走男孩（`B53Bh`）|
| 4..9 | 曼多爾圖書館六本書 | `ecl2/15`：各自對 `4A2Fh` 的一個位元（`01`／`02`／`04`／`08`／`10`／`80h`），沒拿過才寫 |
| 10 | 波多廣場 | `ecl1/18`：拍賣被喊停或結束，同時寫 `4A35h` |
| 11 | 瓦海登墳場 | `ecl4/10`：`LOAD MONSTER 46h` 那一場之後 `6E79 != 2`，並寫 `4A41 = FFh` |
| 12 | 科維爾宅的賊 | `ecl3/14`：`B359h` 在 `4AB2 < FEh` 時寫 |
| 13 | 斯托揚諾河 | `ecl7/23`：`4A52h` 位元 2 立起來之後（`AC4Ch`）|
| 14 | 提前完成任務 | `ecl6/25`：**要 `4AB4 == FDh`**，而 `FDh` 由 `ecl6/28` 寫——這一槽是兩段式 |
| 15 | 蜥蜴人 | `ecl8/16`：`4A5Dh >= 28h`（40）|
| 16 | 狗頭人 | `ecl8/13`：`4A8Dh == 5` 之後國王掉下去，改寫 `4A8D = 6` |
| 17 | 遊牧營地 | `ecl7/17`：`4A7Ch` 位元 0 |
| 18 | 卡德納的付款 | `ecl3/0`：`4AC8h == 1` 時城衛攔下 |
| 19 | 拿下城門 | `ecl2/9`：`A692h` 那段文字之後 |
| 20 | **結局** | `ecl5/7`：`LOAD MONSTER 42h`（TYRANITHRAXUS）那一場之後 `6DC7 != 81h` |
| 21 | 貧民窟 | `ecl2/20`：共用計數 helper 加到 25（spec 042）|
| 22 | —— | 沒有生產者 |
| 23 | 巴恩神殿 | `ecl1/24`：四筆 `LOAD MONSTER` 之後 `6DC7 <= 80h` |
| 24 | 卡德納是叛徒 | `ecl3/8` 兩處與 `ecl5/4`（`4A67h` 位元 3，找到報告）|
| 25 | 叛徒伏誅 | `ecl5/3`：`B15Ch` 一刀了結他 |

看得出三種形狀：**打贏一場架**（2、3、11、20、23）、**收集或旗標**（4..9、
13、17、24）、**計數**（21 的 25 場、15 的 40、14 的兩段式）。
所以「所有區域共用同一種完成判定」是錯的假設，要逐區取自各自的 producer。

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

**「收集或旗標」與「計數」那兩類的完成條件也逐條跑過了**（2026-09-06，
`TestFlagAndCountCommissionsNeedTheirCondition`）。每一條都從那一段的條件
判斷起跑，佈好前置狀態、跑到停下來，看該槽有沒有變成 `FEh`：

| 槽 | 起跑 | 條件成立 | 負對照 |
|---:|---|---|---|
| 4..9 曼多爾六本書 | `ecl2/15` `9FF4h`／`A0A5h`／`A272h`／`A31Ah`／`A3ACh`／`A0FFh` | 槽 `!= FFh`，順帶在 `4A2Fh` 記位元 `01`／`02`／`04`／`08`／`10`／`80h` | 槽已是 `FFh`（市政廳結算過）就不重寫 |
| 13 斯托揚諾河 | `ecl7/23` `AC35h` | `4A52h` 位元 2 立起來後 `AC4Ch` 結案 | 槽已是 `FFh` |
| 17 遊牧營地 | `ecl7/17` `A1DBh` | `4A7Ch` 位元 0 | 位元 0 沒立就整段跳過 |
| 15 蜥蜴人 | `ecl8/16` `9CBDh` | `4A5Dh >= 40` | **39 不算**（兩個方向都驗過）|
| 24 卡德納的報告 | `ecl5/4` `AD88h` | `4A67h` 位元 3 還沒立，寫的同時把它立起來 | 位元 3 已立就跳過 |
| 24 卡德納 市政廳 | `ecl3/8` `AF3Ah` | 槽 `< FEh` | 槽已是 `FFh` |
| 14 提前完成（第一段）| `ecl6/28` `9934h` | `6DD5h != 0` → 寫 `FDh` | `6DD5h == 0` 直接 `EXIT` |

**每一條都帶負對照**——少了負對照，測到的只是「這條 `SAVE` 執行得到」，
那對每一條都成立，什麼也沒證明。蜥蜴人那一條特別把 40 與 39 對調跑過一次，
兩個方向都失敗，門檻確實是 40。

槽 14 的兩段式另外單獨驗（`TestTheEarlyCompletionCommissionTakesTwoStages`）：
`ecl6/28` 寫 `FDh`，`ecl6/25` `9E57h` 只在讀到 `FDh` 時升成 `FEh`，同時寫
`4A11h = 1`；槽是 0（第一段還沒跑）或 `FFh`（結算過）都不動，`4A11h` 也不寫。

### 交差那一段：二十六槽逐條走過（2026-09-06）

`TestEveryCommissionHandsInAtCityHall` 逐槽走一次交差鏈，全程用真的按鍵：
從貧民窟出發（一場都不打）、把該槽設成 `FEh`、走回城區、進市政廳、走到職員
面前，然後檢查

1. **職員演出的是這一槽的通知**（前 24 個字元比對）；
2. **`4AC1h` 只在該加的槽加一**——十槽加、十六槽不加，名單取自
   `gamepack.ReadCityHallNotifications` 解 `9D63h` 的 ON GOSUB，不是抄一份表；
3. 有獎賞的把獎賞收完；
4. **該槽從 `FEh` 變成 `FFh`**（`9F5Ah` 的 `SAVE TABLE`）。

槽 22 沒有文字也沒有生產者（分支是 `RETURN`），所以它的期望是「什麼都不發生」。

負對照是 `TestCityHallSaysNothingWithNoCommissionDone`：一條都沒完成就去交差，
26 條通知一條都不該出現、`4AC1h` 不動、獎賞服務不開。**沒有它，上面那 26 條
證不了因果**——職員本來就會講話，而「文字裡有這一段」對一段夠長的獨白可能
永遠成立。把期望文字整體位移三槽再跑一次，26 條裡有 23 條失敗，比對確實會咬。

### 中間那一段：玩家自己走得到幾條（2026-09-06）

`TestPlayingTheWorldCompletesCommissionsOnItsOwn` 量的就是這個：**完全不給
主線旗標**，讓探索器從開場自己走十二趟，走完看有幾個槽被寫成 `FEh`。
與世界巡迴那一支的差別正在這裡——那一支會把 26 槽全部預設成 `FEh` 去解鎖
內容，所以它量不到這件事。

現在量得到**三條**：1（索寇要塞）、11（瓦海登墳場）、23（巴恩神殿）。
另外貧民窟（21）由 `TestTwentyFiveRealSlumsWinsEarnTheCityHallReward` 整條
連著跑，索寇要塞（1）另有 `TestSokalKeepOpensTheOtherBoatRoutes` 逐步驗過。

**這一支上線的第一次就抓到一個玩不下去的缺陷**：`36h ADD NPC` 沒有把職業
從記錄帶出來（NPC 不經建角，職業只在記錄的 `+2Fh` 與 `+96h`），WARRIOR 入隊
之後 `Pool character "WARRIOR" has unknown class ""` 讓整局停住。修掉之後
走得完的委任從一條變成三條——**那一條缺陷擋住的不只是一個查詢，是整條主線**。

剩下的槽還沒被探索器走到。門檻是量到的下限，不是目標；把它頂到實測值等於
再做一次單一亂數對齊的快照。

### 為什麼不是「多走幾趟就會多幾條」

試過兩種把成果帶到下一趟的作法，結論都寫在測試裡：

1. **帶 `FEh`（做完但沒交差）更差**：委任 3→1 條、地圖 13→7 張，而且跑得更快
   ——探索器更早就沒地方去了。`FEh` 會改變港務長的選單與各區出口。
2. **帶「交差完」的狀態（槽 `FFh` ＋ `4AC1h` 加一）確實把世界打開了**：
   地圖 13→22 張、ECL block 13→19 個。**但委任還是三條。**

第二個結果才是重點：**解鎖更多地圖不等於解得開更多委任**。剩下的條件要密碼
（`NOKNOK`／`SAMOSUD`）、要湊六本書、要打滿 40 隻蜥蜴人、要在拍賣會挑對選項
——那些是「玩家知道要做什麼」，不是「走得到那一格」。要讓探索器自己解開，
得先給它那些知識，而那會讓它不再是探索器。

### 開放缺陷：探索器在 `GEO8/29` 的選單答了四千次出不來

兩階段版的探索測試會撞到（第二階段帶 `4AC1h = 2` 與槽 1／11／23 為 `FFh`）。
現場的量測值（探索器的卡住回報現在會把這些一起印出來）：

```
GEO8/29 (7,7)  選單 [YES NO]  游標 0
文字   RUNGS ARE SET IN THE SIDES OF THE WELL WALL.  DO YOU WANT TO CLIMB DOWN?
載入的地圖 GEO8/29   地形碼 1   C04F 1   4A10 0   9802 1   PC $AF71
```

**原版那一段已經讀完**：

- 每一格從 `9A23h` 起：`AND #127,[C04Fh] → 6E82h`、`GETTABLE $AFCE → 9800h`、
  `9A4Bh ON GOTO` 十六支，記錄尾（fall-through）正好是 `9A81h`。
- 表裡**只有地形碼 1 的值（18）超出 16 支的範圍**，所以踩在地形碼 1 的格子上
  一定走 fall-through 到 `9A81h`。
- `9A81h COMPARE [4A10h],#0 ; IF <> ; GOTO $A1CC`——井的問句要 `4A10h` 非零。
- `AF71h HORIZONTAL MENU` 有兩個呼叫端：井那一條（`A23Bh`）與 `AE83h`。
  `AE83h` 的下一條是 `AE87h SAVE #1 → 4A10h`、`AEAFh LOAD FILES #32`、
  `AEB6h LOAD PIECES`、`AEBDh GOTO $A1CC`。

**量到的值排除了井那一條**：`4A10h` 是 0，`9A81h` 的 `IF <>` 不成立，走不到
`A1CC`。而 `PC` 停在 `AF71h`，所以 VM 是從 **`AE83h`** 那個呼叫端進來的——
`AE87h` 沒跑（`4A10h` 還是 0），表示那個選單**從來沒有被答成「是」**。

**已經修掉的一件事**（順帶，與這個迴圈是否同一件還沒證實）：
`consumeInitialTransitionResources` 先前看到 `WaitingForMenu` 就整個返回，
**把同一個 result 裡、發生在選單之前的資源事件丟掉**。`AEAFh` 正是
「先 `LOAD FILES #32` 換地圖、再問問題」這種形狀，地圖因此不會換。
改成「停在選單之前的資源事件照樣套用，只是不續跑」。修完之後這個迴圈仍在，
所以它不是唯一的成因。

### 已經量過、可以排除的

量了「探索器實際送哪一個索引」（在選單分支加暫時的 log 跑 seed 166／rotate 3）：

```
第 66777 步 游標 0 want 0 menuStall 1
第 66783 步 游標 0 want 0 menuStall 3
第 66789 步 游標 0 want 0 menuStall 5   …每次都一樣，隊伍始終停在 (7,7)
```

三個先前的假設因此都不成立：

1. **不是「沒答成是」。** 探索器每一次送的都是索引 0（YES）——`flags` 非 nil
   那條路本來就強制 YES／NO 一律答 YES。
2. **不是選單的存放位址解錯。** `AF71h` 的 `Header[0]` 解出來就是 `$9802`，
   與腳本 `AF80h COMPARE $9802,#0` 對得上。
3. **`9802h = 1` 不代表上一次答了 NO。** `AEC1h RANDOM #9 → $9802` 把同一個
   位址當暫存用，所以暫停當下的值證明不了答案。

現場穩定的事實只剩三條：每次都送 YES、隊伍完全沒有移動、`menuStall` 每次
出現選單就加 2（表示答完立刻又是選單，中間沒有回到走路）。

**下一步要的是一次完整的指令級追蹤**：從答完那一刻起，記錄跑到哪些位址、
是哪一條把 `4A10h` 寫回 1、以及這一格是被誰重新觸發的。前面幾輪都是靠推論
挑一個點去量，已經連續推翻三次——這一段要的是整條路徑，不是再挑一個點。
