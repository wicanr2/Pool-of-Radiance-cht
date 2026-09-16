# Goal：古托井那一場對回攻略與原版——諾里斯的觸發、編成、獎賞，與「先索寇再下井」那條順序（GitHub #37；spec 137）

主台帳：[issue #37](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/37)。
上一個 goal 的結案位置：[`issue-29-30-city-money.md`](issue-29-30-city-money.md)
（#29／#30 關：訓練所收 1000 金、只放對職業的門；武具店金幣等值付款、餘額重鑄）。

這一輪不是戰術（#22）也不是整張路線表（#26）：只判古托井前後那一段——探針一級下井
撞諾里斯八回合全滅，#29／#30 做好的「獎賞買板甲、回訓練所升級」因此走不到。攻略對這一段
有三句話可拿來對：井邊中立、市議會沒有懸賞、索卡爾城堡可用密語過巡邏。

## 現況（2026-09-16，`3a9fad8` 之後）

- **探針**：`TestMainlineProbeHouseRuleCommissionExperience`（seed 137）貧民窟 20 場 → 城區 (0,4) →
  古托井踩井兩場狗頭人（贏）→ 井底 GEO8/32 (10,3) 地形 12：`NORRIS×1 hp25 ac7 thac0 14`、
  `LIZARDMAN×5 hp11 ac4 thac0 16`、`KOBOLD LEADER×9 hp4 ac7`，15 隻全上盤面；兩發催眠放倒 6，
  隊伍命中 4/17（23%）、敵方 18/22（81%），第 8 回合全滅。之前三個 seed 一勝兩敗（playtest 補六）。
- **ECL8/29（exact）**：`9D34h` "YOU ARE SURROUNDED BY THE BANDIT BAND OF THE INFAMOUS NORRIS THE GRAY"
  → `9DA1h` WHAT DO YOU DO? → `9DB0h` FIGHT／SURRENDER；`9DF0h` LOAD MONSTER 32×1、57×5、1×9；
  贏 `9E08h`（Journal 50）、`9E54h`、`9EB2h`（藏身處可休息）；投降 `9F1Bh`（剝光財物、放回地面）。
  觸發格與旗標還沒讀（探針記到 `4A10=1`）。
- **攻略**（`docs/reference/walkthrough-notes-softworld-001.md`，只當檢查清單）：井邊中立、不搜藏處不進井
  就不打；「市議會沒有針對他的懸賞」（↔ spec 137 獎賞表槽 0 = Norris 250 金＋200 白金，Journal 50
  是他回覆市議會命令的信）；章節順序 貧民區 → 索卡爾城堡 → 古托井；索卡爾城堡「可用密語
  （LUX／SHESTNI／SAMOSUD）通過或連續打四群清空」。
- **spec 137**：建議順序第 2 條把諾里斯排在一級；第 4 條寫「索寇要塞的 50 隻要先讀 `AE5Dh` 的表
  與觸發條件（是否可用密語避開）」——還沒讀。鏡像的 verify 是古托井那一列的「諾里斯是擲骰」還在。
- 工具：`cmd/pool-text-inventory`／`docs/audit/dos-ecl-text-inventory.json`（ECL8/29、ECL4/21、ECL3/8 的句子
  與位址）、`cmd/pool-ecl-memory-audit`（旗標的讀寫點）、dosgolem（`-load-state`／`-save-state`、`-peek`
  讀戰鬥員記錄；`tools/dosgolem-training-fee.py` 是「複製 scratch、打補丁、走鍵序、讀記錄」的樣板）、
  `docs/audit/dos-ecl-scripted-fights.txt`（每場的 LOAD MONSTER）。

## 提示詞（可直接貼給 `/goal`）

> 目標：判清古托井那一段——諾里斯的觸發與選項、獎賞是不是市議會的委任、編成與數值是否與原版一致、
> 以及「一級先用密語過索寇要塞交件升二級、再下井」走不走得通；讓探針走到「獎賞買板甲、回訓練所升級」
> 那一行。主台帳 #37。
>
> 1. **先讀 ECL，不改 remake。** (a) ECL8/29 `9D34h` 的觸發：哪一格／哪個旗標進來、投降分支 `9F1Bh`
>    清了什麼（`Money`？物品？）、打贏 `9E54h` 寫哪個旗標。(b) ECL3/8 獎賞表槽 0 的觸發旗標，對回 (a)
>    的旗標——是不是同一個；攻略「沒有懸賞」對或錯，記回攻略筆記的交叉核對節。(c) ECL4/21：
>    密語（三個字串的比對點）與 `AE5Dh` 表的觸發條件——用密語能不能不打 (8,5) 那 50 隻就到交件
>    條件（spec 041 的索寇完成旗標）。每一條標位址、證據等級，寫進 spec 137（古托井列、建議順序 2／4）。
> 2. **拿原版當裁判。** dosgolem 走到井底那一格開打（或 `-load-state` 從探針的存檔），peek 戰鬥員
>    記錄：名稱、AC、HP、`+2Dh` THAC0 各一筆，對 remake 的 `NORRIS ac7 thac0 14`／`LIZARDMAN ac4 thac0 16`。
>    收據進 `docs/audit/`，測試讀收據。
> 3. **改探針的路線，不改規則。** 若 (c) 走得通：探針改成 貧民窟 20 場 → 索寇要塞用密語過巡邏 →
>    交件 → 訓練所 → 古托井；若走不通：留在原順序，把諾里斯那一場的結果（等級、命中率）記進
>    playtest。兩種都要讓 log 出現 outfit 買板甲與 `L2` 那一行，或記下在哪一場全滅。不強化隊伍、
>    不 forced-win、不改怪物數值。
> 4. **不可越線：** 戰術（站位、集火、包紮）歸 #22；路線表整欄歸 #26；不動 engine／CoAB；
>    動到畫面才對拍。
> 5. **每輪開跑前寫「這輪跟上輪差在哪」**；探針一趟 5 秒，跑之前先把假設寫下。
> 6. **台帳：** 每輪在 #37 留言；spec 137 狀態行；攻略筆記交叉核對節；探針多過一段就在 #22／#26
>    留言；收尾 `docs/worklist.json`（#37 那條的 pattern 對著 spec 137 的「諾里斯是擲骰」，判完要一起改）
>    → `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` → `-mode verify` →
>    `gh issue list --state open`。
> 7. **停止線：** 密語分支要讀到 ECL4/21 以外的 overlay 才判得清（另開 issue）；dosgolem 走不到井底
>    （先修驅動或用探針存檔 `-load-state`）；探針換順序後在索寇要塞或古托井以外的地方全滅（那是
>    #22 的成因，記下停）。

## 已知風險與待決

- 攻略是 1988 年雜誌的城內篇，會錯也會漏；「沒有懸賞」很可能是它錯——但要 ECL3/8 讀出來才算，
  不拿 Journal 50 的語意當證據。
- 索寇要塞的密語就算能過巡邏，交件條件可能仍要求打某一場（`AE5Dh` 表驅動的三場）；走不通就
  老實記下，不要為了讓探針過而改觸發。
- 蜥蜴人 AC 4 若原版也是 4，一級隊伍對這一場本來就是擲骰；那時結論是「順序」不是「數值」。
