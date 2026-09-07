# Pool of Radiance remake 現況

更新日期：2026-09-07。

## 2026-09-07 介面譯文補完，截圖與對拍都換成看得到訊號的做法

三件事，起因都是同一個：**判斷靠「看起來對不對」，而看起來對的東西可以是錯的。**

1. **截圖腳本是盲按加比圖。** 漏掉一次 Return 時前後兩張仍然不一樣（畫面確實
   動了，只是動到別的地方去），`cmp` 判定通過，於是整條流程偏掉十幾步，
   最後以「A 打不開平面圖」炸出來——而真正的原因發生在建角途中，
   拍出來的「自由移動」其實是肖像編輯器。
   `cmd/pool-game/screen_state.go` 讓遊戲每換一個畫面就寫下識別字
   （`-screen-state`），腳本等到那個畫面才往下按，走錯就停在走錯的那一步。

2. **字型稽核的掃描面有洞。** `pool-font-coverage` 只掃 `.go` 的字串常值，
   而介面字串早就搬進 game pack 的 locale 表了——整批不在稽核範圍內，
   而它每次都回「沒有缺字」。補掃之後抓到瞄準列的 `→` 畫不出來。

3. **第一人稱那一框的 100% 只是單元測試裡的合成圖。** 現在改成對打包好的
   AppImage 在真視窗裡的截圖比（`tools/appimage-dos-parity.sh`），
   原版那一側由 dosgolem 跑真的 `START.EXE` 產生，不再用 DOSBox。
   標題整張 99.79%（差的 134 格是 remake 自己加的按鍵提示），
   第一人稱框 88×88 100%。方法與兩個數字的來源在
   [`docs/audit/dos-parity-sample.md`](docs/audit/dos-parity-sample.md)。

介面譯文補完的是肖像編輯器、戰鬥造形設計、人物管理的回應與隊伍面板的欄名；
識別字（`PARTS`、`COLOR-1`）同時是分派鍵，所以只在畫的時候查表翻。

**還剩什麼**：自評挪到 60～70%，壓著它的四項（戰鬥數值沒對過原版、主線沒有
連續跑過一次、真機啟動結果沒回填、下冊附錄三張規則表沒接進 UI）帶驗收條件
列在 [`WORKLIST.md`](WORKLIST.md) 開頭。

## 2026-09-04 戰鬥中存得了檔，而存檔失敗會把視窗收掉

補「戰鬥中與戰鬥後的存讀檔抽樣」時撞到的兩個缺陷：

1. `stateForSave` 的閘只擋對話與服務，**沒擋戰鬥**。而 `tactical` 與
   `combatMonsters` 都不在 `Campaign` 裡——存了讀回來怪物整批消失，等於免費
   脫離戰鬥，而且 ECL session 停在「戰鬥進行中」那一點，兩邊對不起來。
   原版的 SAVE 在營地，戰鬥畫面沒有那個入口，所以照樣擋掉。
   這一條排在對話那一條**前面**：戰術地圖開著時 `cellEventPending` 仍是 true
   （戰鬥掛在格子事件底下），排後面玩家會看到「先把對話讀完」，而畫面上根本
   沒有對話。
2. **存檔失敗會把視窗收掉**。`Update` 回傳非 `Termination` 的 error 時 ebiten
   直接結束——玩家在對話中按一下 F10 遊戲就沒了。改成把理由寫進狀態列。

`TestSavingIsRefusedDuringCombatAndWorksAfterIt` 走真實按鍵：打到第一場架、
進戰術地圖、按 F10 應該被擋且不結束程式，打完之後再存一次、讀回來還要走得動
（「欄位全對但按下一步就卡住」是這類缺陷的樣子，所以真的走一步）。

## 2026-09-04 怪物名根本沒有翻譯管線

查「怪物一覽表的中文譯名」那一項時發現的：定名的工作做過了（glossary 有 43 種），
但**譯名從來沒有接到畫面上**——`MONnCHA` 的名字直接印在遭遇與戰鬥畫面，中文
畫面上會冒出 `SPECTRE ×2`。連 `SKELETON`、`TROLL` 這些早就定名的也一樣。

掃完八個 archive 得到 **103 種**名字，說明書那 43 種之外還有職業等級變體
（`4TH LVL FIGHTER`）、NPC（`FERRAN MARTINEZ`）與專名（`TYRANITHRAXUS`）。
逐條定名後放進 `internal/gametext/monsters.zh-TW.json`，由
`gametext.MonsterCatalogue` 載入。**exact 46 條、strong inference 57 條，
未定 0 條**，每一條都寫得出來源。

三個舊的待證項一併解掉。`MONnCHA` 證實 Spectre（`mon2/17`、`mon4/17`）、
Wight（`mon4/20`）、Wraith（`mon4/21`）都是可遭遇的怪物——會顯示在戰鬥畫面上，
所以不能停在「待證」，那樣玩家看到的是英文。Spectre ＝幽魂由遊戲內文字證實
（兩處敘述都這樣譯）；Wight ＝屍妖、Wraith ＝幽鬼採 AD&D 繁中通行譯名並標
`strong inference`，glossary 同步更新。

**可重用的教訓**：「這個東西已經定名了」與「玩家看得到的地方真的顯示中文了」
是兩件事。定名的台帳綠了不代表管線接上了——這一項的台帳寫著 43 種怪物逐條
決定，而那 43 種一個都沒接到畫面。查這類項目時，要從**玩家看得到的字**回推，
不要只看定名表。

## 2026-09-04 問密碼時把答案附在問句後面

原版要玩家翻說明書才知道要打什麼字。那些字是遊戲內 NPC 說過的，但隔了很多格，
忘了就過不去；而答案本來就寫在原版資料裡，所以直接顯示，不必另外維護一張表。

`eclInputAnswer` 從 `INPUT STRING` 的下一條往後掃，找
`03h COMPARE <剛寫進去的位址> <內嵌字面>`，掃到就把字面接在問句後面加括號。
ecl7/23 的密碼門已生效：問句變成
`YOU QUICKLY TRANSLATE THEM AND SAY...?` 後面接 `（NOKNOK）`。
**這是 remake 的擴充，原版沒有這個括號。**

索寇要塞那三處的原版寫法不同——答案先 `09h SAVE <字面> → 9890h`，再
`COMPARE <輸入> 9890h`，比的是變數不是字面——掃描順便記下 `SAVE` 的目的地
就接上了。`9890h` 依 `4A26h` 被寫兩次，兩個候選都列：ecl4/21 的
(12,6)／(10,7)／(12,10)／(12,11) 顯示 `（SAMOSUD／SHESTNI）`，(8,10)／(7,10)
顯示 `（LUX）`。玩家當下需要哪一個，靜態分不出來，不猜。

這一項順帶曝出治具的一個不精確：它在 block 21 硬送 `LUX`，而上面那四格要的
是 `SAMOSUD`／`SHESTNI`。治具因此改成**照著畫面上的提示打**——`answerHints`
解問句尾巴那個括號，多個候選就輪流試，解不出來才退回舊的寫死表。這也是玩家
實際會做的事，比治具自己猜準。

手札那一側順便機械複驗過：上冊 `docs/reference/manual/journal-vol1.md` 的
`## p.N` 涵蓋 p.1–54 無斷號，線索報導 1–58、酒店傳言 1–23 都連續，議會公告
18 則，附錄七節到齊（第七節跨頁所以用頁層級標題）。轉錄本身沒有缺口，還沒做的
是逐條回對掃描原頁的第二輪覆核。

## 2026-09-04 `11h PRINT` 是接續，不是取代

密碼確認框只顯示一個 `?` 的成因查到了，與原本懷疑的 `33h PRINT RETURN`
無關：**`11h PRINT` 被當成取代**。共用 VM 對 `11h`／`12h` 走同一支 handler
（`case 0x11, 0x12`），只把文字包成事件；分頁語意在消費端，而消費端先前用
「上一則以換行收尾才接上去」這個啟發式，對兩者一視同仁。

原版兩者不同。`12h PRINTCLEAR` 是新的一頁，`11h PRINT` 接在目前這一頁後面
——原版靠它把一句話拼起來，而且中間沒有任何等待玩家的指令：

    ecl7/23  A4A7 PRINTCLEAR "DO YOU REALLY MEAN"
             A4B9 PRINT      6E79h      ← 玩家剛打進去的字
             A4BD PRINT      "?"
             A4C1 GOSUB      9982h      ← [YES NO]

    ecl3/0   AC22 PRINTCLEAR "…IN YOUR JOURNAL YOU NOTE"
             AC5C GOTO       AC9Bh
             AC9B PRINT      "PROCLAMATIONS LXIV, LXXVIII, CIX, AND LIX."

市政廳那一句先前被拆成兩頁，第一頁以 `YOU NOTE` 結尾——**那個分頁是 remake
的 `RunUntilEvent` 一遇事件就返回造成的，不是原版行為**，而測試把它固化成
期望值。已改成一句並更新測試。

四段文字的原始 bytes 前後都不帶空白，所以分隔一定由繪製端補。`joinPrintedText`
補一個空格、標點開頭不補；**原版實際是空格還是換行沒有畫面證據**，標為
`layout-reconstructed`，寫在 spec 082 的「還沒讀」。

順帶關掉 spec 082 自己列的一個未讀項：兩頁之間是誰清的框——`11h` 已排除，
它是接續。

## 2026-09-04 密碼門那個「卡住」是量測工具自己造成的

GEO7/23 (1,1) 的選單卡住查完了：**遊戲沒有缺陷，是探索治具永遠答 NO。**

ecl7/23 的密碼門原版有三條出路——答對 `A4D6 GOTO 9BE3h` 開門；答錯
`A559 DAMAGE` 扣血、`A564 OR 4A51h #64` 設旗標再 `EXIT`，而入口 `A3E4`
檢查同一個 bit 就 `EXIT`，所以**這一格只有一次機會**；答 NO 才會
`A4C6 GOTO A3FAh` 跳回去重新輸入。無限迴圈只有「每次都答 NO」走得出來。

治具正是這樣走的：密碼輸入與選單**共用同一個 `menuTurn[key]`**，交替各
`++` 一次，選單於是永遠落在奇數（`% 2 == 1` ＝ NO），密碼永遠落在偶數
（`% 4` 只取得到 `SAMOSUD` 與 `SHESTNI`），正解 `NOKNOK` 永遠輪不到。
兩個週期咬死。

修的是治具：剛送出密碼的格子下一個選單一律確認；已知密碼的門直接說對的
字。那一趟的迴圈分布從「輸入字串 2668、格子選單 4045」變成「2、49」，並
走到門後的 `[MOVE QUICKLY AWAY DESTROY THE EQUIPMENT]`。世界巡迴 22 趟
仍是零硬失敗。

**可重用的教訓寫在這裡**：探索治具用「輪流選不同選項」來拿覆蓋率時，
兩種互動若共用同一個計數器，週期會互相咬死而永遠走不到某些組合；症狀是
「遊戲卡在某一格」，而那一格的原版控制流其實是收斂的。**下結論說遊戲卡住
之前，先確認治具送得出走得通的那組輸入。** 另外，原版那種「答錯就永久
關閉」的門，輪流試等於把唯一的機會用掉——已知答案就直接說。

## 2026-09-03 死掉的怪還在行動

`NEWECL FF` 修完、GEO1/31 走得進去之後冒出來的兩類卡住，查完一類、修完一類。

**戰術地圖那類的根因是「離場的 combatant 每回合又拿到先攻」。** 死亡當場兩個
入口都把體型與分數歸零了，但 `startRound` 對名冊上每一格重擲，死者於是復活成
行動者。之後整條連鎖都不出聲：體型 0 的 mover 在目的格探測裡取不到佔格偏移，
出界與地形檢查那一整段被略過（那是原版行為），它一路走出盤面站到 `(35,25)`
（Y 上限是 `18h`＝24），`RequiredFacing` 對出界座標九個候選全不成立而回錯誤，
錯誤被 `Update` 收進狀態列，每個影格重試一次。從外面看只是「第 20 回合停住」。

修的是 `startRound`：離場者分數一律 0，骰子照擲、結果丟掉——`+3` 的每回合來源
還沒讀到，沒有證據前不動亂數流。`RequiredFacing` 改成出界當場失敗，順帶訂正
它「DirectionAny 恆真，搜尋一定會停」那句：界限檢查對 `DirectionAny` 一樣生效。
契約寫進 spec 062 第 7 條與 spec 058 第 6 條，未閉合的是原版清 `+3` 的位置。
**不是敵方 AI 改版引入的**——那一行只由 `e6fe452`（09-02 的回合迴圈）引入，
三個敵方 AI commit 都沒碰過它。世界巡迴 22 趟的硬失敗從 3 筆歸零。

**這次真正花時間的不是修，是讓錯誤說得出話。** 治具印的是
`tactical.Status`（`TURN ENDED`），而真正的原因在 `a.statusLine` 裡——
`Update` 把 `tacticalInput` 的錯誤收進狀態列，於是每個影格重試同一個失敗，
外觀是卡住、日誌裡沒有一則錯誤。硬失敗訊息現在帶 seed、狀態列、整份戰術名冊
與 ECL block；名冊一印出來，`(35,25)` 與四個體型 0 的槽同時就位。

當時還留著 GEO7/23 (1,1) 的選單卡住；那一條在隔天查完，根因是治具而不是
遊戲，見本檔最上面那一段。找重現花了 102 趟掃描才撈到一個
（seed 106、destination 2）——世界巡迴的隨機路徑不保證每輪都經過同一格，
所以「這一輪沒報」不等於修好了。

## 2026-09-03 破關那一場

`ECL5/7` 的 `A7DCh` 起是最後一戰：`LOAD MONSTER 42h` ＝ `mon5/66
TYRANITHRAXUS`（HP 80、AC 0、體型 `84h` 佔 2×2）。打贏之後 `A815h` 把
`4ABAh` 設成 `FEh`，那正是市政廳槽 20 的
`CONGRATULATIONS! YOUR QUEST IS OVER!`；結局腳本再把隊伍送回 `ecl3/0`。

擋在結局前面的缺口也修了：`finishCombat` 以前只把戰後腳本的文字套上去，
不分派邊界，所以戰鬥後面接的 `PROGRAM`／`TREASURE`／換區塊一條都不會跑。
改成走 `consumeInitialSearch` 之後，結局文字與回菲蘭的 `NEWECL 0` 都跑了。

還沒畫的是**結局過場**：`PROGRAM 8` 是 overlay-18 entry 1，字串在同一顆
overlay 的 `0111h..02A0h`（「Mortally wounded, the dragon roars!」到
「Noooo...」）。spec 081 先前寫「值 8 全遊戲沒有呼叫點」，那是用走得到的碼
掃出來的，而唯一的呼叫點就在掃不進去的那個區塊裡。

## 2026-09-03 第一條委任跑完了

貧民窟那條委任的完整迴圈通了：真的打 25 場（每一場都進戰術盤、清光敵人、
答 N 收尾）、`DS:4ABBh` 一場加一到第 25 場的 `FEh`、用原版的路走出貧民窟
回城區、從 `(3,4)` 往東進市政廳、走到 `(5,5)` 拿到
`Gold 250 / Platinum 50 / Jewelry 1`，`4AC1h` 由 0 變 1。負對照是「一場都
不打去交差」，文字空的、旗標不動。這是第一條從頭到尾走得完的委任。

獎賞也收下來了：ECL3/8 在通知之後還有獎賞選單（`9EFEh HORIZONTAL MENU`）與
戰利品服務（`9F28h TREASURE` ＋ `9F3Eh COMBAT`），挑 Share 分錢、挑 Exit 之後
`9F5Ah` 的 `SAVE TABLE FF` 把槽清成 `FFh`，角色錢包收到金 250 白金 50 首飾 1。
**這條委任從接到交差完整走得完。**

## 2026-09-03 生物種類與體型、四支法術、`NEWECL FF`

- 285-byte 記錄的 `+9Fh` 是**生物種類**、`+6Ch` 是**體型加一個大塊頭旗標**。
  種類由四個互相獨立的比較點釘住（死靈術只認 `0`、迷蛇術只認 `0Eh`、
  overlay-12 對 `4` 設旗標、魅惑與定身要求不大於 `1`）；體型由四個遮罩點
  （兩個 `and 7`、兩個 `and 7Fh`、一個 `cmp 80h`）證明是兩個欄位擠在一個
  byte 裡。魅惑人類、定身術兩個編號與迷蛇術因此接上，派發表六十七格接得出來
  的從 54 升到 62。被迷住之後的行為仍是近似：只讓它不再行動，沒有倒戈——
  那一段在還沒讀完的敵方 AI（overlay-09）裡。契約見 spec 098。
- 同一份 spec 又解出 `0100h:007Ah`（overlay-24 entry 18）是「把力量往上調」，
  記錄 `+10h` 是力量、`+16h` 是特殊力量百分位。由此抓到一個實作錯誤：
  變大術寫進 `DS:47A6h` 的 `12h` 是**十進位的 18**（要設成的力量值），
  不是效果碼；真正的效果碼是參數表 `+0Ah` 與 `1331h` 都指到的 `0Ch`。
  縮小術與編號 59 因此也接上。編號 60（`2F02h`）也讀完：`287Ch` 打一格、
  `2919h` 由施法者穿過目標拉一條射線逐格打，傷害 `Roll(1, 6) + 20`。
  力量那一組（變大術、力量術、編號 59）也真的會改到角色的力量與例外力量了。
  還缺的四支各自卡在一個還沒有的子系統，不是「再接一支就好」：臭雲術要
  盤面上的雲團物件、死靈術要戰鬥中新增戰鬥員、解除魔法要效果節點串列、
  恢復術要等級吸取的追蹤。前兩者的原版規則已經讀完寫進 spec 098。
  （臭雲術的原版規則 2026-09-05 已在 spec 121 全部閉合。）
- `NEWECL FFh` 是「不換區塊」的哨兵，證據在 spec 107：八個封存檔沒有編號
  255；全遊戲只有 `ecl1/24` 的出口表有 `FF`，而且與同一列的 `LOAD FILES`
  欄成對；拿原始 GEO 量兩張圖的邊界，走得出去的五格裡四格腳本明文處理過，
  剩下那一格兩欄都是 `FF` 且沒有守衛。**已修在共用 engine**
  （`eclvm.NoBlockChange`，2026-09-03）：世界巡迴那一類硬失敗歸零，
  修之前走不進去的 GEO1/31 現在走得進去。那一區一開放，又冒出兩類還沒查的
  卡住（敵方回合、`[YES NO]` 選單），治具目前只記 log。

## 2026-09-01 City Hall clerk、結構清冊與預設委託離場

- Spec 040 已由 `overlay-05:0E85h` 三個 far call 經 TPOV stub → overlay-21 entry →
  IDA Pro 9.4 code offset 的完整位址鏈閉合七種貨幣服務；先前把 far target 當成
  overlay-05 本地 offset 的暫時結果已作廢，未進入正式證據。原版七槽順序為 Copper、
  Silver、Electrum、Gold、Platinum、Gems、Jewelry；Pool 清空各角色 wallet，Share
  依高到低幣種與隊伍順序分配且超載餘額留池，Take 依幣種／角色／數量原子轉移。
  remake 已接 View／Take／Pool／Share／Exit 正常 UI、數量輸入接縫與容量檢查；
  七貨幣當時將存檔升為 schema 5；現行 schema 6 保留同一欄位，schema 1..4 的
  `gold`／`pooled_gold` 仍確定性遷移到索引 3，
  新舊欄位同時出現時失敗即關閉。全專案 `go test ./...` 與 `go vet ./...` 已通過。
- Spec 041 修正 `4AC1h` 的剩餘模型：`9D63h` 是 `4AA6h..4ABFh` 的 26 槽完成通知
  dispatcher，槽值 `FEh` 才顯示一次並在結算後改為 `FFh`；只有十個通知子程式增加
  `4AC1h`。`cmd/pool-city-hall-audit` 現從原始有序 edge 重生狀態位址、target、文字、
  增量位址矩陣，並有刪 edge／改 producer 的失敗即關閉測試。另以全 ECL operand
  audit 固定各槽直接 producer／consumer 下界；三個 decoder failure 仍明列，零引用
  不得解讀成不存在。
- Spec 042 已由 ECL2/block20 真 bytes 閉合 Slums 槽 `4ABBh`：`B69Ch..B6BBh`
  在值小於 `FEh` 時逐次加一，達 25 寫 `FEh`，之後保持不變。direct-entry 第 24／25
  次臨界測試已通過，但只證明 helper，不冒充正常 Slums 完成。game pack 新增八個
  archive／29 blocks 的 ECL catalog，保留 archive namespace、拒絕重複 member／block，
  並以副本隔離 runtime mutation；`cmd/pool-game` 啟動時已載入這份 catalog。
- Spec 043 以 IDA Pro 9.4 的 `overlay-03:0D80h..0ED4h` 訂正 LOAD FILES／PIECES：
  LOAD FILES 第一欄是目前 archive 的 GEO block，第二欄在該 handler 未被消費，不能
  當 archive；Slums `LOAD PIECES 2,4,1` 則精確填 slots 1..3。game pack 與前端已接
  三槽 WALLDEF／8X8D 載入，局部 `FFh` 替換仍在保存 slot state 前失敗即關閉。
- Spec 044 修正跨 archive 存檔：schema 6 新增 `ecl_archive`，Load 從完整 ECL catalog
  重建正確 namespace。ECL3 舊路徑及 ECL2/block20＋GEO2/block20＋`4ABBh=24` 的
  F10／Load round-trip 均通過；schema 5 依其 `map_archive` 確定性遷移。
- Spec 045 已閉合原版 archive controller：START resident `1D7h:090Fh` 保存舊
  `52D4h` 並切換 selector，overlay-07 特殊分派呼叫它；正常 ECL3/block0 在
  `9955h..9965h` 執行 `LOAD FILES FF,FF,7F → SAVE 2,6E12h → NEWECL 20`。
  共用 engine 新增作品中立 catalog resolver，Pool adapter 才解讀 `6E12h`；真檔測試
  已由 ECL3 起跑並落到 ECL2/block20、GEO2/block20、WALLDEF2 slots 2/4/1。
  尚未完成的是 Slums 25 個實際戰鬥結果與回 City Hall 結算，不得把入口接通寫成
  Slums 全區完成。

- Spec 032 以 Pool overlay-03 dispatcher `346Eh/3474h` 與 handler `1A81h..1EA4h`
  固定 `27h TREASURE` 的八欄 numeric request。共用 engine `91801a5` 已提供 inline、
  fail-closed 的 typed request；Pool 真 block8 `A780h` 抽樣得到七欄全零、
  `ItemBlock=33h`，再停於 `A791h COMBAT`。Spec 033～036 已把該 request 接到五筆
  `ITEM3/33h` record、原版式 Take／Exit 服務、負重檢查與 schema 3 引入的 inventory；
  raw request 本身仍只代表待領戰利品，必須成功 Take 並存檔後才算玩家取得。
- Spec 012 已勘誤：overlay-07 **entry 27** 與 ECL **opcode `27h`** 只是編號碰巧相同，
  前者為座標 wrapper、後者為 TREASURE；舊的合併追查指示已刪除，後續分兩條證據鏈。
- Spec 033 固定 `ITEM3.DAX` 雜湊與 block `33h` 的 315-byte payload；Pool handler 的
  `3Fh` 複製迴圈及真檔共同證明它是五筆 63-byte record。四筆名稱是
  `Clerical Scroll With 2 Spells`，第五筆是
  `Two-Handed Sword +1 +3 vs. Undead`。typed loader 已接妥；未證實欄位、卷軸法術、
  裝備規則仍維持 DRAFT；Take UI 已依 Spec 034～036 接妥。
- Spec 034 由 Pool `overlay-05` 原始 bytes 固定戰後戰利品選單：`0E85h` 提供
  View／Take／Pool／Share，Take Items 走 `0CF0h → 0BCAh`；成功後才從 `DS:676Eh`
  的 `+2Ah` next chain 移除並釋放 63-byte 節點。Spec 035 已閉合角色接收 helper；
  remake 同樣只在容量檢查與存檔成功後移除 pending loot。
- Spec 035 已沿 TPOV runtime segment/control record 鏈定位 overlay-06 item receiver、
  overlay-19 overload predicate 與 overlay-25 carry table。remake 現保存完整 63-byte
  inventory record，依 16 格與力量負重判斷；schema 3 的 schema 2 遷移仍保留，
  Take 只有在持久化成功後才移除 pending loot。Spec 036 又證明墓園的 `24h COMBAT`
  在該狀態呼叫 overlay-05 post-combat，因此 UI 結束後從 COMBAT 後方續跑而不重播 loot。
- Spec 037 修正「F10／Load 已足以保存戰役」的過期斷言：schema 3 實際只有 roster、
  HP、Gold 與 inventory，會遺失 map 與所有 ECL 旗標。共用 engine `142b245` 現提供
  作品中立、驗證後才原子替換的 `BlockSessionSnapshot`；Pool schema 4 保存 GEO identity、
  `(x,y,facing)`、current block、PC／stack、numeric／string memory、compare、亂數續點與
  pending lifecycle entries；Spec 040 加入 schema 5 七貨幣，Spec 044 再升 schema 6
  保存獨立 ECL archive。穩定玩家
  邊界的 F10／Load round-trip 已接妥；任意對話、
  神殿、戰利品或未完成戰鬥中途續點仍為 DRAFT，不能宣稱全情境存讀檔完成。
- Spec 038 新增全 ECL memory-reference audit，固定墓園入口的三個欄位：`4AC1h >= 4`、
  `4AB1h != FFh`、`4A96h != FFh`。`4A96h` 在接受後由 `A792h` 寫 `FFh`；`4AB1h`
  的目前唯一直寫在 ECL4/block10 吸血鬼戰鬥結果分支寫 `FEh`，仍會通過墓園 gate，
  不得先命名成完成旗標。Spec 039 已另證明 `4A39h..4A3Fh` 是墓園七種戰利品累積量，
  不是七項任務；正常墓園驗收真正前置是閉合 `9FAEh..A4D1h` 十個 `4AC1h` 進度
  producer，而不是在測試直接注入 4 或拿戰利品槽數代替任務數。

- Spec 026 已由同一正常按鍵 session 從標題、原版建角、Rolf、Sune、City Hall 公告
  與 `NEWECL 8` 走到 clerk office：`(4,5)` 外部提示、`(5,5)` clerk 第一頁、
  `4A01h=1`／`4A06h=1`、script block 8 與 GEO3/block0 均有精確斷言。
- Spec 027 以 Pool overlay-03 `0FA2h..0FEEh` 的 IDA Pro 9.4 bytes 閉合 opcode
  `35h SAVE TABLE`：`memory[wordAddress(op1)+numeric(op2)] = numeric(op0)`，位址採
  16-bit wrap。共用 engine `7e93050` 已實作 value／base／index、write receipt
  與三種失敗即關閉測試；正式 pseudo-version 為
  `v0.0.0-20260831165626-7e9305036c43`。
- `cmd/pool-city-hall-audit` 已從固定 block8 trace 重生
  `docs/audit/dos-city-hall-structure.json`：墓園戰利品七個有序槽／四個唯一 target、commission
  十六個有序入口、十個 `4AC1h` 增量 producer，以及五個 external service call 均有
  hash gate 與刪除 evidence 的負對照。清冊補出舊規格漏列的 `9F28h TREASURE`／
  `9F3Eh COMBAT`，另三個是 `A5A8h PARTYSTRENGTH`、`A780h TREASURE`、`A791h COMBAT`。
- Spec 029 已把「全新隊伍、無 reward」獨立成 READY：正常按鍵由 clerk 引言依序走過
  slums、Sokal Keep、old Phlan books/maps 三項委託、`THESE ARE ALL...` 與 `AF7Ch EXIT`，
  並斷言回到 `(5,5)` 地城移動。完整 reward／commission 條件矩陣仍屬 DRAFT Spec 028，
  不得把這條預設分支冒充整個 City Hall 服務完成。
- Spec 030 已用 IDA Pro 9.4 固定 Pool opcode `1Dh`：dispatcher `3404h..340Dh` →
  handler `13A0h..14B1h`，沿 party `+104h` 鏈讀 `+96h/+9Bh/+110h/+111h/+11Bh`，
  依原版整數公式累加為 byte 並 inline 寫回 destination。shared engine `4b10d7c` 已提供
  fail-closed typed resolver，Pool／CoAB 正式依賴升到
  `v0.0.0-20260831181703-4b10d7c5a302`。
- Spec 031 以 `FEM/HMU/HTH.CHA` 三份原版 class oracle 固定一級 Fighter／Magic-User／
  Thief 的五欄與 contribution；Pool 正常建角 roster snapshot 已接 resolver。真 block8
  `A592h` 正／負 fixture 證明 18 跳過、19 進 Valhingen Graveyard。訓練、升級、裝備與
  DOS 角色匯入投影仍是 DRAFT，不能把一級 snapshot 當完整角色成長。

## 2026-08-31 存檔生命值決定

- 使用者採用 schema 2：Pool 角色明確保存 `max_hp`、`current_hp` 與未臆測命名的
  原版 `status` byte；schema 1 的 `hp` 讀取時確定性遷移成最大／目前 HP 相同、狀態 0。
  新建角色以滿血建立，建角預覽顯示 `HP current/max`。契約與原版 Heal 證據見 Spec 018。
- CoAB 本來就有 `HitPoints`、`MaxHitPoints` 與健康狀態，不套用 Pool schema；依使用者
  授權已把正式 engine 相依升到 `0819c64` 並完成全套回歸與實際遊戲編譯。
- Cure Light／Serious／Critical Wounds 的 100／350／600 GP 與 `1d8`／`2d8+1`／
  `3d8+3` 已 CONFORMED。overlay-04／19／21 證明付款先嘗試目前角色完整支付，個人
  不足才嘗試 pooled money 完整支付，兩者不合併；玩家 Heal UI、原子保存、失敗回滾
  與真實 ECL3 正常按鍵路徑均已通過。其他狀態治療仍 fail-closed。

## 已證實

- DOS 來源 ZIP SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- ZIP 的 168 個非目錄檔案、解壓總長 1,582,291 bytes，已由
  `cmd/pool-input-manifest` 逐檔固定 size／CRC32／SHA-256；可重生報表為
  `docs/audit/dos-input-manifest.json`，工具不解壓也不修改來源。
- ZIP 有 169 筆，內含 47,936-byte `START.EXE`、232,379-byte `GAME.OVR`，
  以及 `ECL1..8`、`GEO1..8`、`WALLDEF1..8` 等 DAX。這只證明檔案 inventory，
  尚未證明 executable 版本、DAX consumer 語意或遊戲完成度。
- `珍009-光芒之池.rar` SHA-256：
  `209265086b6ad98d28bb51bb689737b48eb54ce0396ecb773e52ebdd5676409e`；
  magic 是 RAR4。它只作歷史中文化線索，不是 DOS 行為 oracle。
- `amiga/` 中八檔實際大小都是 174,848 bytes，符合 D64 容器形狀；檔名中的
  `amiga` 尚未由內容證明，不把它當平台事實。
- 共用 engine `0d6de308a19c` 已提供 `dax`、`ecl` packed text／menu record 與通用
  `tpov` codec；Pool 是第二作品
  consumer，作品位址、文字與劇情不得回填 engine。
- 2026-08-31 使用者確認：ECL／runtime 應以系列共用 engine 沿用，並授權參考或複製
  CoAB 程式加速；硬限制是不能干擾現行 CoAB remake。實施順序固定為「只在獨立
  engine 新增相容 API與測試 → Pool 先使用 → CoAB 測試只讀回歸」，不得把 CoAB
  game code 直接搬進 Pool，也不得把 CoAB 位址／旗標誤升格為 Gold Box 通則。
  Pool 全 corpus 現有 26 個完整解碼 block、14,724 條可達指令，opcode 集合與 CoAB
  handler table 高度重疊，證明重用方向成立；剩餘三個 graph failure 與每項作品副作用
  仍須各自閉合。engine 第一個新增切片是作品中立 operand numeric／address／text 求值，
  CoAB 現行程式未修改。
- 共用 engine `0819c64` 已提供 fail-closed `eclvm` 核心：控制流、比較、算術、
  SAVE／GETTABLE、ON branch、文字與選單 continuation。Pool production seam
  `gamepack.NewInitialEventMachine` 已只白名單 Rolf 路徑實際走到的 `0C/0D/0E/2D/31/3A`；
  真實 `ECL3/block0 B06Eh` 測試逐次提供 Return 後可跑到 `AE85h EXIT`，七頁文字與
  最後 `C04B/C04C/C04D = 0/4/3` 均吻合。production 前端現已在真實 ScriptBlock
  存在時逐 VM boundary 消費：SAVE 更新位置、Return menu 等按鍵、DELAY 形成 34 frame、
  文字更新 dialogue、EXIT 才完成導覽；Docker／Xvfb 正常 Begin 測試跑到 `(0,4,3)`。
  手寫 `TourStep` 僅保留給無原始 script 的合成 UI fixture，不是正式遊戲路徑。
  engine 後續已加入同一 VM 的 entry 切換與 `RunUntilEvent`：後者逐 instruction 在
  第一個 observable event／menu／EXIT 邊界停下，避免先跨過 COMBAT 再事後標記。
  Pool `04ae5b4` 已將此契約接到初始 map cell lifecycle；未處理的事件會設為 pending
  並停止移動。pending 只代表失敗即關閉，尚不代表該事件已可遊玩。
  未提交或只用於稽核的 VM clone／全圖 sweep 不列為現行完成度，必須待 deterministic
  報表、測試與正常玩家可達性分開驗收後再更新本節。
  後續已補 deterministic opcode `08h RANDOM` 與可複製 RNG continuation；Pool 原始
  `9A0Eh RANDOM 19 → 6E79h` 不再需要 passthrough。全 engine 測試已通過。
- 113／113 個 DOS DAX 已由 engine `dax.Parse` 成功解析，合計 1,245 blocks；
  這只關閉 container shape 閘門，不代表 payload semantic parity。
- `GEO1.DAX..GEO8.DAX` 合計 29 blocks；29／29 payload 均為 `0x402` bytes，
  已由 engine `geometry.Parse` fail-closed 解成 16×16 四平面，重生報表在
  `docs/audit/dos-geo-inventory.json`。這只證明 GEO 結構與共用 engine contract
  相容；尚未證明哪個 block 是 Phlan 起始地圖，也未證明該圖採 bounded、wrapped
  或 dungeon-door 移動語意。
- Pool-owned `internal/gamepack.ReadDOSGeometryCatalog` 已將八份 archive／29 組原始
  `(archive, block ID)` 接成 typed catalog，保留 prefix、拒絕缺檔／重複 identity，
  並以值副本隔離 runtime map mutation。這項只完成結構 adapter，不替任何 map 命名。
- `START.EXE` 是 MZ，`GAME.OVR` 以 `TPOV!` 開頭；配合 overlay/runtime 字串，
  Borland／Turbo Pascal overlay family 目前是 `strong inference`，精確版本未知。
- 未修改 DOS 程式可用固定輸入抵達標題與主選單；兩張穩定畫面及雜湊已保存。
- `TITLE.DAX` 恰有兩個 320×200 picture blocks；block 1 經共用 engine 解碼、
  標準 EGA 色盤與最近鄰 2× 呈現後，和 DOSBox oracle 的 AE 為 0。
- ECL 映射基準的舊斷言 `9914h` 已推翻：`overlay-07` loader／address classifier／
  resolver 證明 raw payload 映射到 `9900h..B6FFh`；前五個 command-set headers
  佔 20 bytes，所以 `9914h` 只是第一條指令。修正後既有 decoder 完整走過
  26／29 blocks、14,724 條 reachable instructions；剩餘 3 筆才是真待研究缺口。
- DOSBox 資料目錄必須是 `C:\POOLRAD\`；直接掛成 `C:\` 會到建角資料頁才
  假性要求 disk 3。修正掛載後，預設 Dwarf／Male／Fighter／Lawful Good 可正常
  產生非零能力值與頭像。原先誤讀成 `MAX BONUS?` 的文字經放大後訂正為
  `HEAD / BODY / KEEP` portrait editor；其後 READY／ACTION combat icon、Parts、
  雙色六部位、Size 與 Exit 均已走通。285-byte CHA 的 `BDh..C6h` 已由原版 UI
  單變因差分閉合（Spec 003 READY）；擲值公式已另於 Spec 004 READY。
- 六種族職業清單已由正常 UI 逐張擷取並寫成 typed catalog；原版 Race codes 是
  `1,2,3,4,5,7`，單職 Class codes 為 Cleric `0`／Fighter `2`／Magic-User `5`／
  Thief `6`，Gender 位於 `9Eh`（Male `0`／Female `1`）。七種多職持久碼均已用
  正常 UI 的同源畫面／CHA 配對閉合，不以排列猜測。
- 通用 TPOV parser 已對本 build 解出 38 overlays／774 entries；IDA Pro 9.4
  最小探針通過後，角色建立定位到 overlay-16，角色資料顯示定位到 overlay-19。
  `.CHA +10h..+15h` 六能力、`+30h` age、`+32h` max HP 與 word `+8Eh` Gold 已由
  同一次原版資料頁＋最終 CHA 與直接存取交叉證實。早先將 `+32h／+B1h` 推作
  Gold／HP 的說法已被 runtime anchor 否定並在 Spec 004 保留勘誤；`+B1h` 現只作
  raw class HP accumulator，最後除以 active class count。七種多職代碼已由 Half-Elf
  正常 UI 逐項閉合；overlay-16 與 resident 表格也已閉合年齡、`3d6`、種族／年齡／
  職業能力限制、exceptional STR、Gold、hit dice 與 CON modifier。Spec 004 已 READY，
  注入式 dice roller 與純資料角色生成器已實作；完整畫面／CHA 串接仍待完成。
- `cmd/pool-game` 是第一支 Ebitengine 正常入口：本機 ZIP → typed `TITLE.DAX` →
  標題 → 主選單 → Race／Gender／Class／Alignment → Spec 004 角色資料頁 → 1..15-byte
  姓名 → HEAD／BODY／KEEP portrait editor → combat icon editor。Spec 007 已用
  IDA Pro 9.4 閉合 Head 0..13、Weapon 0..31、Size 與六個雙色欄位；CHEAD／CBODY
  184／184 blocks 全部 fail-closed 解碼，remake 由正常玩家路徑顯示 READY／ACTION
  真實素材並可調 Head、Weapon、Size 與六部位雙色。Xvfb 逐鍵截圖已涵蓋三個 editor；
  Spec 008 已另接版本化 remake 角色庫與 atomic save：icon 確認後保存角色、回到原版
  順序的 Party Creation Menu，`A` 加入最多六名玩家角色；`L`／F10 現依 Spec 037 在
  穩定玩家邊界恢復／保存 schema 4 campaign。Xvfb 正常按鍵已走到 Library 1／Party 1，
  再以 `B` 進入已證實的
  `GEO3/block 0, (15,1), facing 6`。初始 `LOAD PIECES 127,127,127` 已由
  overlay-03 handler、overlay-30 `LoadWallSet` 原始字串／bytes 與真實 DAX 共同閉合為
  `WALLDEF3 block 0, slot 1`；其三筆 records 對應 `8X8D3 blocks 101/102/103`。
  共用 engine 已補上原版多-record selector `0→10` 工作值特例，Pool typed adapter
  與正常 `B` 畫面現在可解析原版 wall stamps。Spec 010 又由 ECL3/block 0 閉合
  `COMPARE [4AC5h],1` 首次閘門、handler `B06Eh`、事件位置 `(15,1), facing 3`、
  monster 12、Rolf 第一頁 packed text 與單一 Return 選項。Xvfb 正常按鍵截圖為
  `pool-remake-initial-rolf-event.png`；GEO／wall／第一頁事件來源是 exact。Spec 011
  進一步閉合四張 34-byte GETTABLE table：正常 Return 後依 34-step 位置／朝向動畫，
  在 steps 5／7／20／25／32／33 依 selector 顯示七頁，最後於 `(0,4), facing 3`
  清狀態並 ECL `EXIT`。typed adapter、按鍵狀態機與 Tyr 停靠截圖均已驗證；每步約
  150ms 是 hardware-spec approximation。wrapped traversal 目前僅是跨作品 strong
  inference，自由移動仍不提前放行。DOS CHA／SPC exporter
  未冒充完成：
  33 份 CHA 均為 285 bytes，但 SPC 是 9-byte 節點鏈且 corpus 有 9／18／36 bytes，不能
  複製單一模板。初始 map、wall material 與完整 Rolf 導覽已接，但地名、Pool 專屬
  視錐／背景 oracle、Rolf 初次 APPROACH 圖像、地圖事件 dispatch 與遊戲內存檔仍未接。
- HEAD1..8／BODY1..8 共 16 archives 已全掃：109／109 blocks 可由共用 engine
  picture decoder fail-closed 解碼；Spec 006 已閉合建角固定使用 HEAD3／BODY3、
  14／12 筆稀疏 block selector，並以 DOS capture 全像素零差異證明 88×40＋88×48
  是保留 index 0 的不透明無縫垂直合成。remake 只暴露原版這 14×12 組合，沒有把
  全部 109 張任意交叉組合冒充建角選單。

## 尚未知／不阻擋目前盤點

- DOS 發行版精確 revision、compiler／linker／overlay 精確版本（family 已有強推論）。
- 剩餘三個 ECL graph 失敗的 record／控制流成因，以及非 TITLE picture payload 語意。
- 歷史中文 RAR 的字碼、修改範圍、可執行檔差異與授權狀態。
- D64 實際平台、檔案系統內容及其與 DOS 版的關係。

## 現行驗證策略

先做唯讀 inventory、雜湊與 fail-closed codec 驗證；再建立 READY spec，才實作
玩家行為。GEO archive／block shape、正常新遊戲初始 map identity 與初始
WALLDEF／8X8D 素材 identity 與完整 Rolf 34-step 導覽已 READY；地名、移動變體、
Pool 專屬第一人稱視錐／背景與 Rolf 初次 APPROACH 圖像仍須以 DOS runtime／
executable 閉合。
Spec 012 已用 IDA Pro 9.4 證實 overlay-03 `30FAh` 的 `401Fh` dispatch 唯一呼叫
overlay-07 entry 27；後者依 facing `0/2/4/6` 將 X／Y 在 `0..15` 間 wrap，並更新
`6A0Fh/6A0Eh`。2026-08-31 後續 corpus 掃描推翻「401Fh 是尚待尋找的玩家輸入
producer」：dispatcher 的 helper 以 low byte `40h`、high byte `1Fh` 反向組成
`401Fh`，原始 ECL operand 是 `1F40h`；唯一 decoder-validated 候選位於
`ECL7.DAX` block 17、payload `B69Ah` 的 opcode `27h` 第四運算元。這仍未證明該
wrapper 是一般玩家前進或完整 collision policy；自由移動繼續失敗即關閉，下一步改為
確認 opcode 27 handler 在 Pool 的 operand consumer 與 overlay-07 entry 27 的座標
語意，不再沿錯誤的「opcode 1F／CALL producer」假說追查。
首條玩家垂直鏈固定為標題 → 建角／建隊 → 第一個正常可操作地圖 → 事件／戰鬥 →
存檔／讀檔。

Spec 014 已開放 Rolf EXIT 後的第一張地圖基本操作：左右方向鍵以八方向步進轉向，
上方向鍵僅在 cardinal `0/2/4/6` 時以前進方向查 `CanMoveDungeonWrapped`，座標採
16×16 wrap。Pool 真實 GEO3/block0 的 `(0,4)` 四向結果為 N/E/W 可通、S 阻擋；
Xvfb 正常路徑已驗 `(0,4,facing3)` 左轉至 facing2，再前進到 `(1,4)`。GEO bytes、
入口與 wrapper wrap 是 Pool exact；door detail consumer 仍是跨作品 strong inference，
事件尚未執行，因此只能稱「基本 GEO walk」，不能稱完整自由移動。

Spec 015 已把第一張地圖移動接到原始 cell lifecycle：同一 Rolf VM session 保留記憶體，
每次成功移動同步 `C04B..C04F` 後切到 ECL3/block0 entry `9914h`。正式 `(0,4,3)` 左轉／前進至
`(1,4,2)` 的第一格執行 15 條指令並在 `997Dh EXIT` 返回，無事件副作用。尚未接的
事件格不再被安靜略過：一旦結果包含文字、選單或 external event，前端設 pending 並
停止移動。engine `RunUntilEvent` 現已逐 instruction 在第一個 observable event 暫停，
不會先跑過 COMBAT 才事後標 pending。下一步是各 boundary 的 frontend continuation，
而不是建立座標 hardcode 表。

Spec 016 與 `docs/audit/pool-initial-cell-sweep.json` 已完成隔離的 16×16×四方向掃描：
entry 0 的 1,024 樣本全在 `997Dh EXIT`；這推翻「entry 0 單獨完成 terrain dispatch」。
以同一副本接 entry 1、seed 1 的舊基線為 840 `EXIT`、156 個真文字／external event、
28 個 fail-closed error。Spec 019 以 Pool dispatcher 訂正 opcode handler 後，`20h`
跨 block session 已使 12 個錯誤歸零，首次重跑為 852 `EXIT`／156 event／16 error；
其中 4 筆在目的 block 新暴露的 `14h COMPARE AND` 已由 Spec 020 的 Pool handler
證據閉合並接入共用 VM。Spec 021 再接通 `0Ah LOAD CHARACTER` 與 string-memory
`COMPARE` 後，現行為 856 `EXIT`／168 event／0 error；新增的 12 筆是真實 City Hall
文字事件，不是空表現事件或放寬錯誤。
另有每格一致的空
`PRINTCLEAR`／`PICTURE 255`，已分列為表現事件，
不灌進玩家事件數。幾何 BFS 可達 226／256 格；它不執行途中 ECL，不能冒充玩家可達性。

正常按鍵路徑已從 Rolf 結束 `(0,4,3)` 前進至 `(1,4,2)`，再轉北前進至 `(1,3,0)`；
同一 VM 的第二次 RANDOM 為 7，entry 1 依 terrain `87h` 顯示
`YOU ARE WELCOMED BY PRIESTESS JOY OF SUNE.`。空 `PRINTCLEAR` 與 `PICTURE` 由前端
消費後續跑，真文字會暫停移動並等待 Return。這是第一個正常玩家可達的 post-Rolf
cell event；後續 `DO YOU SEEK HEALING?`、原始 YES／NO menu 與 Sune 神殿服務入口
已可逐段繼續。YES 會進入原版 `Heal／View／Pool／Appraise／Exit` 五項選單；Exit 從
同一 VM 的 `AA6Bh` 續跑，寫入 `6DE1h=FFh`、消費 `PICTURE 255` 後 `EXIT`。Heal 等四項
服務的價格、HP／狀態與金錢副作用尚未 READY，現階段保持失敗即關閉。`20h` 已依
Pool `0CDDh` handler READY 並接入共用 block session；Spec 020 的 `14h` 亦已接入
共用 VM；Spec 021 的 `0Ah` 與字串比較也已依真實 consumer 接線。初始地圖 sweep
已無靜態錯誤；現行正式引擎版本統一見本節末，不再把當時鎖版冒充目前版本。

Sune 選單後的同一 VM 分支已由 Spec 017 閉合到服務入口：YES（選單索引 0）先令
`6E79h=0`、進入 `AA63h`，執行 opcode `1Ch CLEARMONSTERS`、
`SAVE 1 → 6DE2h`，再抵達 opcode `24h COMBAT`。共用 engine 現以作品中立的
`MonstersCleared` 訊號聚合到 external boundary；Pool 只有在該訊號與 `6DE2h=1`
同時成立時才進 Sune 神殿，其他 COMBAT 不會誤路由。NO（索引 1）則顯示
`THEN YOU MUST LEAVE.`，保存 `49F0h → C04Bh`、`49F1h → C04Ch`、
`FFh → 6DE1h`，呼叫 `2C90h` 後 `EXIT`。

已推翻的斷言：`CALL 2C90h` **不是神殿 healing 本體**。它在 Rolf 34-step 導覽的
每一步及 Sune NO 分支都出現，現階段只能列為 redraw／movement service 候選；未追完
overlay dispatcher 前不得命名，也不得把所有 `CALL` 自動續跑。先前把 opcode `24h`
掛到 overlay-03 `2E90h`／entry 50，再追到 overlay-25 entry 42 的鏈條已被原始 dispatcher
推翻：Spec 017／019 證明 `24h` 呼叫 `186Ch`，`2E90h` 是 opcode `39h` handler。
因此 overlay-25 entry 42 只保留通用選擇／鏈結處理證據，不能再當 temple 或 COMBAT
consumer。38 份 overlay 的 raw far-call 全掃只有 overlay-03 `18B9h` 的
`9A 25 00 35 00` 指向 overlay-04 entry 1；該 callsite 位於 opcode `15h VERTICAL MENU`
handler，先檢查並清除 runtime state `+5C4h`。
ECL `6DE2h` 對 `+5C4h` 的映射仍是 `strong inference`，不可單獨冒稱 exact；但
ECL pattern、唯一 temple call 與原始 Pascal 選單字串已足以完成 Spec 017 的服務入口。
engine／Pool 全測試與 CoAB 唯讀 `internal/ecl`、`internal/game` 回歸均通過。Heal
價格、HP／狀態與 Gold 垂直鏈已由 Spec 018 閉合；下一步是 City Hall proclamations
之後的 clerk／commission 分支，以及神殿 View／Pool／Appraise，而不是繼續把
overlay-25 誤當神殿 dispatcher。

Spec 023 已關閉 City Hall 的預設公告分支。正常按鍵由 Sune 離開後走到 `(3,4)`，
依序停在 `ABD2h` 外部文字、`AF1Ch` 單項 Return menu、`AC22h` 公告前言與
`AC9Bh` 公告編號，最後由 `ACBEh EXIT` 回到移動。trace 新增可重生的
`menu_destination`，證明 `AF1Ch` 寫入 `9801h`。IDA Pro 9.4 的 overlay-07
classifier／完整 resolver 又證明 `9801h`（bank 2）與 `4AC1h`（bank 0）是不同
storage；先前「兩者可能 alias」的工作假說已推翻，不得據此改共用引擎。預設
`4AC1h=0` 略過 clerk／commission 是原版控制流；下一個切片才以非零狀態閉合該分支。

Spec 024 已接續閉合 `4AC1h=1..9` 的公告分派：`AC60h` 先減一到 `4A18h`，
`AC69h ON GOSUB` 選九個 literal 子程式，回來由 `AC8Ah/AC96h` 印出
`PROCLAMATION` 與 `985Eh`。九筆真實 ECL session 均依舊格 entry 0 → 新格 entry 1
走到 `EXIT`；第 5 與第 9 筆同為 `CXIV.` 是原始 bytes，不自行更正。這仍不證明
clerk 在何時寫入 `4AC1h`；下一步追該 producer 與 `AD29h` Skullcrusher 離隊事件。

Spec 025 推翻 Spec 019 的不完整斷言「NEWECL 目的 block 只跑 entry 0」。原版
overlay-03 lifecycle controller 在 `3741h` 呼叫 entry 0，因 NEWECL 設定的
`4391h=1` 於 `3778h` 回圈，再由 `3658h` 呼叫 entry 4；Pool 的 transition sequence
因此是 `0→4`。共用 engine 保留預設 `[0]`，由 Pool adapter 設 `[0,4]`，pending
entry 可跨玩家 boundary 與 Clone 保存。正常按鍵在 City Hall 公告後前進到 `(4,4,2)`，
script block 變 8；block 8 entry 4 的 `LOAD FILES 0,0,0` 仍載入 GEO3/block0，證明
script identity 不等於 geometry identity。`LOAD PIECES 127,127,127` 保留現行 wall set；
其他 selector 尚未 READY。engine／Pool 全測試與 CoAB 玩家／game／ECL 抽樣已通過。
共用引擎正式版本現為 `v0.0.0-20260901045011-d59f339e7600`；Pool 的 `go.mod` 已鎖定
該已推送版本。本機隔離回歸以同 commit 的唯讀 engine mount 驗證，未把未提交來源
混入結果。

Spec 046／048／049 已把 Slums 第一個真實 `COMBAT` 邊界接到 Pool 前端，但沒有假造
戰鬥結果。八個 `MON*CHA` archive 全掃共 172 blocks，全部是 285-byte record，
Pascal name 皆合法；`ECL2/block20 9E5Dh` 的 ID 13 與 4 在 `MON2CHA.DAX` 均精確為
`ORC`。Pool 現依目前 ECL archive 載入 raw-preserving typed record，staging 顯示
`ORC ×1 / ORC ×3`，PC 留在 `9E6Dh`、`4ABBh` 不變，Enter 亦不能越過。IDA Pro 9.4
已從 Pool 自己的角色資料頁 consumer 閉合 unified 285-byte record 的 max/current HP、
AC、THAC0、第一組 damage dice／signed bonus 與 movement；兩筆 ORC 的真檔抽樣分別是
`5/5 HP, AC 6, THAC0 20, 2d4-1, move 9` 與
`5/5 HP, AC 6, THAC0 19, 1d8, move 9`。基礎命中／傷害已由後述 Spec 050 閉合；
裝備／effect modifier、initiative、特殊攻擊、勝利／逃跑／全滅與戰後 continuation
仍是下一個 READY 切片；
真實全滅時依專案規則停止測試，不強求通過。

Spec 050 已沿 Pool `overlay-13:1404h..1883h` 的 attack span 與
`overlay-24:0CB5h／0DE5h／0E30h` 閉合基礎命中、dice core 與傷害公式。typed primitive
保留原版 internal THAC0／AC encoding：roll 1 直接 miss、roll 20 改成 score 100，
再比較 `score + THAC0Internal + signedModifier >= effectiveACInternal`；傷害依序消耗 NdS、加
signed bonus、負值歸零後才套 caller 明示倍率。狀態／法術 modifier、attack rate 的
裝備／effect 覆寫、initiative、特殊攻擊與 status transition 尚未閉合，所以 combat staging 仍不會自動
勝利或續跑 ECL。全專案 Docker／Xvfb `go test ./...` 與 `go vet ./...` 已通過。

Spec 051 又閉合雙 attack slot 的 base source 與 phase rounding：record `+A1h/+A2h`
分別供 slot 1／2，經裝備／effect（仍 DRAFT）後，以
`(encodedRate + DS:6CD7h bit0) / 2` 寫入本 phase 的 `+113h/+114h` remaining count；
攻擊迴圈由 slot 2 倒走到 slot 1。typed game-pack 現可 fail-closed 取得兩槽 base rate
及交錯排列的 damage dice，combat primitive 保留 byte wrap。Slums 兩筆 ORC 真檔皆為
base rate `2/0`，任一 phase 得到 primary 1、secondary 0；這仍不取代 initiative、effect、
deployment 與玩家／AI 回合。

同一 Spec 051 已補閉 phase counter 生命週期：overlay-10 combat setup 在建立 combatant
runtime 前把 `DS:6CD7h` 清零；overlay-08 回合邊界 `0879h` 以 byte `inc` 增加一次，
再重算全體 combatants。該函式同時持有 `Your Teammate is Dying`／`Continue Battle:`
提示，支持回合語境；typed `AdvanceAttackPhase` 固定 `255→0` wrap。戰鬥中途仍禁止
存檔，因此未增加 campaign schema；將來若開放，counter 必須保存為 combat continuation。

Spec 052 已再閉合先攻排序的可實作核心。overlay-13 entry 1 對有效 combatant 取一個
DEX signed modifier 加 `1d6`，先做 minimum-one，再依驚訝旗標減 6，最後把
`<0` 或 `>20` 轉成 0。overlay-08 沿 combatant 鏈選 runtime `+3` 最大者，同值以每筆
`1d100` 決勝，較大或完全相等都由後者取代；全零才回傳無行動者並進回合邊界。
`ResolveInitiativeScore` 與 `SelectInitiativeActor` 已依此加入純規則，但各種行動如何
消耗／重設 `+3`、玩家／AI 行動與勝敗 continuation 仍未 READY，不能因
先攻排序已完成就自動結算 Slums 戰鬥。

同一 Spec 052 隨後以原始 far call `010A:0057`、MZ header `3B0h` 與 TPOV control table
把 modifier producer 精確映射到 overlay-25 entry 11：它讀已由 Spec 004 證實的
record `+13h` DEX，依原版分段表回傳 `-4..+5`。overlay-13 entry 19 亦證實施法時以
spell table byte `/3` 為 casting cost；目前先攻大於 cost 時相減，否則固定留 1。
typed `DexterityInitiativeModifier`／`ApplyCastingTimeInitiative` 已接妥。另已確認
overlay-08 combat command 的 `D` 會把先攻設為 1；但通用 runtime-clear entry 34 同時被
攻擊、移動與死亡分支呼叫，尚不能把每個 callsite 都當成攻擊者消耗。下一輪必須先辨認
各參數是 attacker 或 target，再閉合一般攻擊／移動／防禦的生命週期。

攻擊參數身分現已由 overlay-13 `1404h` 與 `1883h` 兩層閉合：內層 `arg_A` 與 wrapper
`arg_E` 都是 attacker，target pointer 會被寫入 attacker runtime `+0Ah/+0Ch`。每次攻擊
後掃描 `+113h/+114h`；任一 remaining attack slot 非零就保留目前先攻，兩槽皆零才對
attacker 呼叫 entry 34，讓 `+3` 歸零。`InitiativeAfterAttackSlots` 已實作這個 initiative
投影；entry 34 還會清 `+0/+7/+6`，完整 tactical runtime 接線時必須同步處理。下一個
未閉合範圍縮為移動／防禦及死亡 callsites，不能再把一般攻擊列為未知。

Spec 053 已把移動預算從 base record 接到每步扣除：record `+11Ch`（Spec 049 已證實為
base movement）在特定角色類別先加一個尚未命名的 combat global word，結果經 byte
wrap、1..96 clamp、乘 2，再交給 effect code 12，最後成為 runtime `+6`。Move 畫面以
`+6/2` 顯示；direction 0..7 中偶數 cardinal 扣 2、奇數 diagonal 扣 3，不足則歸零。
正常走一步不修改 initiative `+3`，仍回到同一角色 command loop。typed
`InitialMovementBudgetBeforeEffects` 與 `SpendMovementStep` 已接妥；effect code 12
與目的格 attack／entry-threshold 分派亦已由本節後文閉合。剩餘是 global bonus 語意、
`2758h` 兩個路徑欄的玩家語意、移動後反應攻擊完整 gate 與戰術 runtime 位置提交。

Spec 053 的 effect code 12 accumulator 亦已閉合。overlay-24 dispatcher 對 code `12h`
固定依序套 effect IDs `27h/2Ah/3Ah`；overlay-12 handler table 與三支原始 handler 證實
它們分別對 movement accumulator 做 byte double、整數 halve、zero。ID `27h` 另有 effect
record bit 與 actor word side effect，ID `3Ah` 也會清 runtime movement，因此 typed
`ApplyMovementEffectIDs` 只明確承諾 accumulator 投影。三個 ID 的 spell／item producer
尚未閉合，不把它們猜名為 Haste／Slow／Hold。移動剩餘缺口是 global bonus 語意、
地形成本、碰撞與移動觸發 attack wrapper，不再把 effect code 12 accumulator 列為未知。

effect producer 再追後，ID `3Ah` 已 exact 閉合為 held 狀態：新增 `3Ah` 的同一函式顯示
`is held fast`，但具體 Hold spell／怪物能力仍未區分。IDs `27h/2Ah` 分別有 Haste／Slow
的 strong inference：`is Slowed` 路徑移除 `27h`、`is Hasted` 路徑移除 `2Ah`，且數值
handler 正好是 double／halve；因尚缺新增 ID 與 spell 名稱的同一條 call chain，程式與
schema 繼續保留數字，避免把強推論偽裝成 exact 名稱。速度效果移除後會立即重跑
effect code `12h`，未來 effect runtime 不能只在 combat setup 時計算一次。

overlay-22 的另一層 `DS:6A78h` far-pointer table 已證實採 selector × 4 分派；
`is Hasted`／`is Slowed` handlers 分別位於槽 48／55。這兩個 selector 不是 effect
IDs `27h/2Ah`，不可拿表槽位替 effect 命名。現有 strong inference 等級不變；
若要升格 exact，仍須找到「法術選擇 → 新增 effect ID」的同一條 producer chain。

Spec 053 也已閉合目的格 probe 的玩家 Move 分派。overlay-13 entry 6 回傳目標與格位
類別；目標 ID 非零時，overlay-08 先經 `DS:6517h` 取目標並進 attack wrapper，不經
`2758h` gate。無目標時才查 `[格位類別×4+2758h]` 第一 byte，只有 entry threshold
`<= runtime movement +6` 才提交方向步；threshold 太高便阻擋。實際提交仍只扣
cardinal 2／diagonal 3，不能把 threshold 重複扣除。typed `ResolveMovementProbe`
已固定攻擊優先、相等可進、超額與 `FFh` 阻擋。

START.EXE 的 MZ loader 映射已把 `DS:2758h` 固定到 file offset 40712；該處是完整
`66×4 = 264` bytes 戰術格位類別表，戰術地圖 cell record `+7` 保存其索引。四欄現依
原始位置保留為 entry threshold、兩個尚未命名的 path bytes 與 presentation code：
`+0` 的 Move gate 與 `+3` 傳入 tactical tile drawing service 是 exact；`+1/+2` 的
原始值及 overlay-31 consumer 已證實，但玩家語意仍 DRAFT。Pool game pack 新增嚴格
66×4 typed parser 與完整原始 table fixture，不接受其他版本或截短形狀。

overlay-13 entry 5 的位置提交另已閉合八方向 delta：
X=`[0,+1,+1,+1,0,-1,-1,-1]`、Y=`[-1,-1,0,+1,+1,+1,0,-1]`，typed
`AdvanceTacticalCoordinate` 保留原版 byte wrap。Y 表最後三 bytes 與第一筆 cell class
的 `01 00 FF` 共用原始儲存位置，已在 Spec 053 明示，不能誤判為擷取錯位。位置提交後
確會呼叫反應攻擊掃描：候選是附近敵對側、須有 runtime `+7`，成功時先清該 byte 再以
候選攻擊 mover；但 `+7` producer、overlay-25 entry 6 與 overlay-32 entry 13 的兩道
predicate 尚未閉合。故目前只完成資料／規則 primitive，戰術畫面、occupancy、反應攻擊
與勝敗 continuation 仍保持失敗即關閉。

2026-09-01 暫停恢復後已用同一 `coab-go-ebiten:1.24` 容器、Xvfb、唯讀 Pool／engine
掛載重跑 `go test -p 1 ./...` 與 `go vet ./...`，全數通過。第一次 `--network none`
因空的 module cache 無法取得已鎖定的 Ebiten／`x/image` 而在 setup 階段停止；開放網路
下載同一鎖定版本後乾淨重跑成功，該次失敗分類為工具環境，不是產品測試失敗。本收據
只證明目前 remake 內部與編譯期檢查通過，不升格 DOS 同狀態 parity 或完整戰鬥可玩性。

2026-09-02 使用者把下一個產品閘門改為：先整理《軟體世界》說明書中的 Journal／日誌，
完成全文繁中翻譯、集中術語表與未翻譯數為 0 的覆蓋報表，再接遊戲內 UI。README 現況
已同步到 Spec 048～053，並以 `docs/audit/remake-screenshot-manifest.json` 固定七張正常
玩家路徑重拍圖的來源提交、尺寸與 SHA-256；戰鬥目前沒有正常可達 runtime 截圖，不能
用 direct-entry 畫面替代。

2026-09-02 說明書 Journal corpus 上冊完成（Spec 054）。`珍009-光芒之池.rar` 的 68 張
跨頁掃描已確認是軟體世界珍藏版 9 的官方繁中說明書，上冊即 Adventurers Journal，
所以這批工作是轉錄與校對，不是重新翻譯。上冊 p.1–54 已逐頁對掃描原圖轉錄成
`docs/reference/manual/journal-vol1.md`，頁碼連續無缺，含線索報導 1–58、酒店傳言
1–23、議會公告 18 則與七節附錄（金錢換算、法術表、裝備、昇級經驗、對抗不死怪物、
職業裝備、武器表），未轉錄數 0；`docs/reference/manual/glossary.md` 固定當年譯名並
標出原書自身的異名，其中 Yulash（城市）與 Yarash（巫師）在內文都譯成「亞拉斯」，
接線時必須依上下文判別。來源清冊 `docs/audit/manual-scan-manifest.json` 登記 68 張
掃描的 SHA-256 與尺寸。RapidOCR 對這份掃描會穩定掉繁體特有字（放大兩倍實測無效），
因此 OCR 只留作行序與覆蓋率對照，跑到 25／68 後停止以歸還機器資源；corpus 的內容
一律以逐頁校讀為準。下冊操作手冊與上冊第二輪覆核仍未做，corpus 也尚未接進遊戲內
Journal／UI，不得據此宣稱遊戲內中文化完成。

2026-09-02 下冊「操作手冊」p.1–57 亦已全文轉錄為
`docs/reference/manual/manual-vol2.md`，頁碼連續無缺，六章（進入遊戲、開始冒險、
人物創設與組隊、探索與旅行、戰鬥、魔法）全部涵蓋，含安裝流程、密碼盤用法、人物管理
選單、戰鬥造形編輯（PARTS／COLOR／SIZE 與六部位雙色）、時間制（3-D 一步 1 分鐘、
邊走邊搜 10 分鐘、陸地一步 12 小時；ROUND＝1 分鐘、TURN＝10 分鐘）、命中公式
（攻擊者 THAC0 －防禦者 AC ＜ 1～20 亂數）、記憶／鬆弛時間（15／30／45 分鐘與
4／6 小時）與逐條法術說明。這些是原版行為的第二來源，可與反組譯結果交叉核對，但
說明書不是 oracle：與 DOS 實機衝突時以實機為準。譯名對照表已補上種族、職業、屬性、
九種陣營、六種人物狀況、各畫面指令與法術譯名。跨冊譯名衝突（Phlan 的菲蘭／弗蘭／
蘭城、Sokal Keep 在下冊拼成 Kosal Keep、戰鬥回合 ROUND／TURN 混用）已就地註記未統一，
取捨列為 WORKLIST 待辦。corpus 仍未接進遊戲內 Journal／UI。

2026-09-02 跨冊譯名已定案（`docs/reference/manual/glossary.md` 的「定案譯名」）。
取捨用四條依序套用的原則：出現次數多者勝、次數相同取音近且不撞字者、再相同時內文
優於地圖標註、最後取較短者。14 組決定包含 Phlan＝菲蘭、Sembia＝桑比亞、
Braccio＝巴西歐、Valjevo＝瓦傑渥（城堡同字）、Werner von Urslingen＝魏納·烏斯林根、
Thentia＝珊提亞（避免與「蘭」撞字）、Sokal Keep＝索卡爾城堡、Kobold＝小妖魔、
Magic-User＝魔法師、Thief＝賊。另外分開 Yarash（巫師，亞拉斯）與 Yulash（城市，
尤拉斯）——上冊 p.11 把城市寫成亞拉斯是原書誤譯；並固定 ROUND＝戰鬥回合（1 分鐘）、
TURN＝普通回合（10 分鐘），下冊 p.42 的混用不採。轉錄正文一律維持原書用字，本表只
約束 game pack 與 UI。專案程式碼與 JSON 目前尚無任何中文譯名，因此本次定案沒有既有
實作要回改。怪物 42 種的中文譯名與「遊戲內實際字串是否與說明書一致」仍待原版證據。

2026-09-02 任務系統與戰術 occupancy 各推進一段。ECL 側：對 29 個 block 全部展開靜態
trace（26 成功，失敗三筆與既有 `failed_blocks` 一致），以「自載 `LOAD FILES`＋原始
字串」定位出城內八區腳本（spec 055），再由各區找出寫 `FEh` 的指令，補上 City Hall
十二個槽的 producer 端（spec 041 由此升為 READY）。`MENDOR` 只出現於 `ECL4/21`、
`MANTOR` 全無，遊戲原始資料只認 Mendor。程式碼側：`internal/gamepack/cityhall.go`
直接由 `ECL3/block 8` 的 `9D63h ON GOSUB` 解出二十六條通知分支並提供
pending／acknowledge／增量判定，八個測試通過；既有的 commission 測試測的是派發端
（`4AC1h` → 公告字號），兩者合成完整循環，且其期望字號全部落在說明書第四章轉錄的
18 則公告內，這是說明書 corpus 第一次被原版資料證明可接線。戰鬥側：occupancy
整條鏈打通（spec 056 升為 READY）。`START.EXE` 每個 overlay stub 段開頭有 32 bytes
描述子，其 file offset／code size／relocation size／entry 數四個欄位對 38 顆 overlay
全部與 `ovr-manifest.json` 相符，因此 far call 落在哪一顆 overlay 是查表得到的：
`0138h` 是 overlay-31、`013Dh` 是 overlay-32。產生端 overlay-31 `0912h` 以體型類別
展開四個候選格，逐一與場上每個 combatant 的四個佔格兩兩比對，取成本最小者寫進
`6674h` 結構；佔格偏移表在 `DS:2860h`，只有四列——1 格、直向 2 格、橫向 2 格、2×2。
結果每筆 3 bytes 的語意是 `+0` combatant 索引、`+1` 成本、`+2` 攻擊該目標所需的
朝向；篩選端 overlay-25 entry 32 再依 `+10Eh` 留下對立陣營、原地前移壓縮、索引
匯出到 `6CD7h`。`sub_579` 同時解出：它是朝向弧判定，方向表在 `DS:274Ah`／`DS:2753h`
（0 為上，順時針八方向，第九項代表不指定），四個正向是以「前方一格」為頂點的
90 度錐形、四個對角是同一頂點的象限，起點自身與頂點無條件成立。進入時的界限
檢查（X 0..49、Y 0..24）也定了盤面尺寸，X 是較寬的橫軸。`internal/combat` 因此
新增 `facing.go` 並重寫 `occupancy.go`，兩張表都逐位元組照抄原始資料，
`go vet` 與全部測試通過。`sub_419` 也隨之解出（spec 057）：它是 Bresenham
直線追蹤，走訪器 26 bytes，直走每步 2、斜走每步 3，預算上限 `budget×2+1`；
地圖 record 的 `+6` 是「跳過地形判定」旗標，格子自 `+7` 起、列距 50，
與 X 上限 49 一致；每一輪先判地形再判預算，起點格自己也要判。步進方向由
`DS:25D4h` 的 3×3 表查出，與方向環互相印證。`internal/combat/trace.go`
實作走訪器與追蹤，11 個新測試通過。直線追蹤讀的兩欄就是既有 `DS:2758h`
格位類別表（`gamepack` 已有 66×4 的權威解析）的 `PathByte1` 與 `PathByte2`，
spec 057 閉合的是這兩欄的用途；戰鬥層直接消費 `gamepack.CombatCellClass`，
不另建副本。
`sub_2E` 是結果表的交換排序：成本遞增，成本相同時比朝向，但斜向不得越過
正向——比較關係不是全序，所以照原版兩層迴圈實作而不是換成排序函式。

移動命令因此整條接起來（spec 058）。目的格的兩個 byte 是 **overlay-32
entry 19（`0CB9h`）** 產生的，不是 overlay-13 entry 6；後者做的是「離開威脅區」
的反應攻擊，觸發條件精確地是「移動前鄰接、移動後不鄰接」的差集。探測會把
mover 的四個佔格各推一格，回報撞到的 combatant 與最難進的目的格類別；
類別 0 代表盤面外，原版問玩家要不要離開戰鬥，不是擋住。順帶定出戰術層
的 DS 版面：佔用格陣列在 `6039h`（50 寬，1250 bytes），結尾正好接上
`6517h` 的 combatant 遠指標表第一筆，而 `6674h` 是戰術地圖的遠指標。
`internal/combat/destination.go` 實作逐格查詢、目的格探測與
`ResolveDestination`，全部測試通過。

反應攻擊的閘門同日閉合（spec 059）。候選是差集而不是鄰接：原版把 mover 暫時
往該方向推一格、再查一次鄰近敵人、然後復原，兩份名單相減。閘門依序是
mover 的 `+10Dh`、對手身上沒有 `DS:2880h` 那四個致能效果碼（`33h`／`34h`／
`35h`／`1Fh`，由 overlay-25 entry 27 沿 record `+7Fh` 的效果串列搜尋）、
overlay-13 entry 11 的效果否決查詢、兩個狀態碼查詢，最後在「目前朝向的前後
兩格」這五個朝向裡找一個讓 mover 落進朝向弧。攻擊槽依 `+A1h` 與
`+113h`／`+114h` 選出，每個對手最多打一次。`internal/combat/reaction.go`
實作可測的那幾段；`DS:677Ch` 否決旗標的 producer 仍未閉合。

戰術地圖的來源也定了（spec 060）：**它是戰鬥開始時生成的，不是從資料檔載入**，
所以 DAX 盤點裡找不到 consumer 不是儀器有洞。`DS:6674h` 在全部 overlay 與
`START.EXE` 裡只有兩處被寫——overlay-10 `134Bh` 的 `GetMem(4E9h)` 與 overlay-08
`004Bh` 的釋放後清零。`4E9h` ＝ 1257 ＝ 7 ＋ 50×25，由配置大小獨立證實了
先前由定址反推的版面。室外戰場整面填類別 `17h`（可進入、不擋路徑，與目的格
探測的初值同一個常數）；室內戰場則由地城幾何生成，掃 13×5 個地城格、每格做
三次牆面查詢。

投影是斜的：`X = 21 + 6dx + 5dy + subB`、`Y = 10 + 5dy + subA`，算完才檢查界限，
超界整格不寫；寫入的類別是建構器類別**加一**，正好對上格位類別表折疊過的
1-based 索引。四支建構器也解完了——西帶、北帶、西北角、東北角，各自負責
不重疊的一塊，兩個角落是自動接圖表。`internal/combat/indoormap.go` 把視窗、
雙向牆面查詢、四支建構器與投影接成可呼叫的 `GenerateIndoorTacticalGrid`。
原版不清空這塊記憶體，投影不到的格子保留配置時的內容，實作以
`UnpaintedCellClass` 明確標出而不是靜靜地填 0。

仍未閉合：室外那四支建構器、大地圖表 `35E2h` 的形狀、`DS:495Bh` 模式值的
完整清單，以及牆面值 1 與 3 的玩家語意。

部署與回合迴圈跟著閉合（spec 061、062），戰術戰鬥因此第一次真的可以打。
overlay-32 entry 20 是佔用格重建：整面清零之後沿 combatant 鏈把每個人的四個
佔格寫成索引加一，所以 0 代表空格，正好對上目的格探測讀到 0 就當空的判定。
視窗原點在戰術地圖 record 的 `+2`／`+3`，畫面格由「絕對格減原點」得到。
部署走 `11×6` 樣板，一組四個，索引由陣營與遭遇型態選出，投影與室內建構器
同一條斜投影但 X 原點是 22；四個失敗出口分別是超界、已被佔用、地形不可進入、
以及樣板該格留白。樣板本體是執行期填的，位置落在 `START.EXE` 檔尾之後，
靜態讀不到，所以 remake 的候選位置是**暫定值**，標在畫面上而不是假裝已知。
回合迴圈在 overlay-08：每輪重算佔用格、依先攻順序輪流行動，回合結束時推進
垂死計數（狀態 5，第九輪轉狀態 6），再判勝敗；場上敵人清空時原版問
`Continue Battle:`，不是立刻結束。`internal/combat/deployment.go`、`round.go`
把這兩段實作成可測的形狀。

`cmd/pool-game` 的遭遇畫面因此改成按 Enter 進入戰術戰鬥：回合、先攻、八方向
移動與預算、攻擊、傷害、倒地與勝敗都會實際跑，勝利後由停在 `COMBAT` 邊界的
ECL PC 續跑戰後腳本（與 treasure、temple 共用同一條 `RunUntilEvent` 路徑），
戰敗依 spec 046 契約 5 不續跑——原版的戰後段不能用自動勝利代替戰鬥結果。
那條契約由一個刻意不給 `eventSession` 的測試釘住，defeat 一旦走到續跑就會
panic。畫面截圖由 `tools/capture-tactical-preview.sh` 在容器內以 Xvfb 實拍，
腳本內含三個回歸斷言（F5 要改變畫面、移動鍵要改變畫面、結束回合要改變畫面）。

仍未接：敵方 AI 的行動（目前只有玩家這側會動）、反應攻擊接進移動提交、
`Continue Battle:` 提示的 UI。隊伍的 AC／THAC0／傷害骰仍是暫定值——角色記錄
還沒有那三項，畫面上以 `PLACEHOLDER` 標明，不得當成 parity 證據。

敵方回合接上之後戰鬥才是雙向的。`foeTurn` 用 spec 056 的鄰近成本表挑成本最小的
敵對目標，每一步反查原版方向表朝它前進、過 `ResolveDestination`，撞上目標就走
spec 050／051 的攻擊。產生鄰近成本表的 overlay-31 `0912h` 本身也一併實作了
（`internal/combat/nearby.go`）：展開雙方佔格、過朝向弧、走直線追蹤、取最小成本，
朝向未指定時自 0 起找第一個成立的方向。`OpposingNearbyAt` 接上 overlay-25
entry 32 的陣營篩選，`LeavingOpponentsAfterStep` 是 spec 059 那個「暫時推一格、
查詢、復原」的差集。

敵方回合走的是原版的骨架（spec 096）：先問武器搆得到誰，搆得到就打，搆不到
才照戰術模式那一列的五個相對方向依序試、第一個進得去的就走；基準方向是目標的
方位（`combat.RequiredFacing`）。仍有三處近似——挑哪個目標、五個方向多一道
「要離目標更近」的閘門、全不合用時的繞路備案——畫面上還是以 `PROVISIONAL AI`
標明。目前的擷取路徑沒有 ECL 排出來的遭遇，盤面上只有隊伍，所以敵方回合沒有
截圖佐證，只有單元測試；manifest 裡寫明了這一點。

繁中化開始接進遊戲畫面。`internal/etenfont` 是倚天 16x15 Big5 點陣字的
`font.Face` 轉接（與 CoAB remake 的同名套件同源；那種與作品無關的東西本該收進
共用 engine，但要動另一個 repo，先各自持有一份）。`-lang zh` 沒有給字型時失敗
即關閉——內建的 7x13 沒有漢字，硬跑會整片留白，而留白看起來像繪製壞掉。
預設 `auto`，沒給字型就照舊跑英文。

已中文化的畫面：標題提示、人物管理選擇項、建角的種族／性別／職業／陣營四個
選單、人物資料頁、姓名輸入與各階段提示。用詞全部取自官方中文說明書
（`docs/reference/manual/manual-vol2.md`），不是重新翻譯：CREATE NEW CHARACTER
是「創造新人物」、character library 是「人物名單」、六項屬性是力量／智慧／睿智／
敏捷／體質／魅力。兼職依 p.14「××／×× 的選擇項代表兼職」由組成職業合成，
不另外列表。譯名對照的鍵是選項 ID 不是英文標籤，`creation.Flow.OptionIDs` 與
`Options` 一一對應且有測試釘住；另一個測試走過六個種族的四個階段，確認每個實際
出現的選項都有譯名。

兩件已知限制：字型目錄沒有 `spcfont.15` 時全形標點會退回半形（截圖 manifest 有
記）；戰術畫面的四行資訊加上功能鍵列，在 15px 的漢字字型下高度不夠，那一頁的
版面要重排之後才能中文化。遊戲內的敘述文字（ECL packed text）與 Journal 都還
沒有接。

原版文字的可讀性也一併解決了：ECL block 裡一個明碼句子都沒有，敘述文字是 6-bit
packed、以 `0x80` 長度前綴標記。`cmd/pool-name-audit` 兩種都掃，並用虛詞判斷解出
來的字串是不是真的散文——6-bit 解碼會把圖形也解成有字母有空白的東西。
說明書定案的 17 個專名裡，九個由遊戲文字證實，八個只在說明書出現；
`Sokal Keep` 的拼法因此確定，而遊戲寫的是 `MAGIC USERS`，沒有連字號。

遊戲內敘述文字的翻譯管線接起來了。原版文字是 6-bit packed 的 ECL 運算元；
`cmd/pool-text-inventory` 從每個 block 的進入點做控制流追蹤，只取真正是指令
運算元的文字，因此位置是指令位址、內容是原版真的會顯示的那一句。第一版漏掉
一整類——選單的選項字串不在 `Operands` 裡而在選單記錄裡，補上之後總量是
1,731 句、110,473 個字元，失敗的仍是既有的三個 block（ECL5/7、ECL7/17、ECL7/22）。

`internal/gametext` 以原文整句為鍵存譯文，原版 block 不修改。用整句而不是指令
位址當鍵，是因為同一句常出現在多個 block，以位址為鍵會逼人把同一句翻好幾次，
而且改一處忘另一處時畫面會一半中文一半英文。有測試拿盤點檔逐條核對每個原文
都真的存在；它當場抓到一條抄漏結尾引號的條目。

對話框換行改用 `wrapDisplay`：以半形格數計算，漢字兩格、ASCII 以空白斷詞、
行首不放收尾標點。原本的 `wrapASCII` 只看空白，中文一整段沒有空白。

碼頭那條主線的機制解開了（spec 102）。港務長 `9C43h` 有兩道閘門：面向要是北
（`@C04D == 0`），而且手上不能有票（`@4A01 == 1` 就不開口）。第一次找他時
`@4AA7 < 254`，他免費給一張去索寇要塞的票並把 `4A01` 寫成 1；要塞裡對費蘭選
「說謊」會把 `4A01` 寫成 255，255 同時通得過港務長與碼頭兩道閘門。選「說實話」
或**打贏亡魂那一場**都會把 `4AA7` 寫成 254（`ADAAh`／`ADD0h`）並把 `4A26` 寫成
255 讓亡魂不再出現，所以順序只能是先說謊再了結，而且中間不能回城區——`4AA7`
還沒開到 254 時港務長會把票再發一次。

航線選單存的是 0 起算的游標，直接寫進 `@4AC4`：SOKAL=0 到索寇要塞、EAST=1 到
野外圖 27 的 (9,29)、WEST=2 到野外圖 26 的 (7,29)、BAY=3 到野外圖 26 的
(13,27)、NONE=4 不上船。`internal/gamepack/harbour_master_test.go` 直接拿原版
ECL 跑這五條，不靠探索器。

`cmd/pool-ecl-trace` 補了 `branch_index`／`branch_targets`：`25h ON GOTO` 的目標
**原始順序**只能從這裡讀，`edges` 是排序過的，拿它推「第幾個選項跳到哪」會得到
自洽但錯的結論。城區的地點分派 `9B4Ch` 因此確定是 28 個目標、索引 0 起算，
索引 1 是碼頭、2 是港務長，索引 0／4／11–16／18 都指向「這一格沒有地點」。

ECL 的 active-character 視窗補上了金錢那一格。原版 `5CF0h` 是指標，指到那個人的
285-byte 記錄本身（spec 021），`39h WHO` 挑完人改的就是這個指標（spec 090）；
腳本對 `6BC3h`（白金）的加減因此直接落在本尊身上。remake 先前只在
`0Ah LOAD CHARACTER` 投影姓名與士氣，`6BC3h` 永遠是 0，於是任何要付錢的腳本都
停在「你的白金不夠」——船資一枚白金的港務長、賭場的 `'YOU HAVE' n 'PP.'`
都走不下去。remake 的記憶體是 map，沒有那種疊合，改用
`gamepack.CharacterBinding` 做「選進來抄進去、換人之前抄回來」，收尾再 `Flush`
一次；`cmd/pool-game` 的 window 直接讀寫 `state.Party`。

野外的移動方向修好了。那張八支 `26h ON GOSUB`（ecl7/26 `9A18h`）的索引**從 0
起算**，順序是北、東北、東、東南、南、西南、西、西北，所以四方位是 0、2、4、6；
remake 先前用 1、3、5、7，於是每一步都走成斜的——往東走一步，X 與 Y 會同時加一。
這跟城區分派那個坑是同一類：`ON GOTO`／`ON GOSUB` 的目標順序與起算基準只能從
`cmd/pool-ecl-trace` 的 `branch_targets` 讀，靜態圖的 `edges` 排序過。

野外的可通行判定還是空的：`9A53h..9A90h` 拿 `DS:035Fh` 去比兩張表，而 `035Fh`
在 29 個 ECL 區塊裡只有三處 `COMPARE`、沒有人寫，`9A36h` 的 `CALL @C01B`
在 `2Dh` 分派器裡也沒有對應分支。所以隊伍現在可以一直往東走出地圖範圍。
排除過的可能與兩張表的內容記在 spec 105。

ECL 的位址空間讀出來了（spec 106）：指令裡的位址不是直接的 DS 偏移，直譯器先
分類再決定去哪一塊記憶體、用 byte 還是 word。`4900h..4CFFh`、`6B00h..6EFFh`、
`9700h..98FFh` 各是一塊 word 陣列，`9900h..B6FFh` 是**目前這個 ECL 區塊的
payload**（所以 `GETTABLE` 讀得到區塊自己的位元組，而 `NEWECL` 換掉之後同一個
位址讀到的是新內容；主線旗標在另外兩塊，不隨換區消失）。其餘落在 class 4，
那些不是記憶體是引擎暫存器：`033Dh` 是朝向、`00FBh`／`00FCh` 是兩個 word、
`C04Bh + n` 對到 `DS:6A0Bh + n`。

野外可通行判定那一格 `035Fh` 因此有了答案：它在 class 4，而取值函式對它的
case 是空的，回傳沒有初始化的堆疊區域變數。原版比的是堆疊剩下的東西，
remake 讓它一直是 0——兩邊都沒有可靠的擋路行為，這一格不是 remake 少做了什麼。

野外的移動接對了：**一步是兩件事一起發生**——引擎照一般規則在載入的 GEO 上走
一格（進野外時 `LOAD FILES 6, 6, 0`，走的是 GEO block 6），那一區的 ECL 入口 0
同時把野外座標 `49C3`／`49C4` 往前推一格。先前前端把野外整條移動路徑換掉，
隊伍在 GEO 上不動、也不檢查牆，於是野外 Y 一路走到 34、35（範圍只到 33）。

判斷依據：野外那三個區塊沒有 `CALL @C01E`（由腳本走一格）卻有七次
`CALL @C018`（用 `6A0Bh`／`6A0Ch`／`6A0Dh` 重算面前的牆），而寫 `C04Bh..C04Dh`
（＝ `DS:6A0Bh..6A0Dh`）的地方都在地點腳本裡、是進區域前擺位置用的。
`TestAWildernessStepMovesBothPositions` 釘住兩個位置要一起動、牆擋住時一起不動。

野外因此變成真正的樞紐了：從城區買往東的船票、上碼頭、落在圖 27 的 (9,29)，
地點表當場派工（問「要不要搭船回文明區」），答應就換到 ECL block 20。
`TestAWildernessLocationDispatchesItsScript` 把這一條從頭釘到尾。
三張野外圖的 `NEWECL` 目標合起來是 0、1、2、10、13、14、16、17、18、19、22、
24、25、26、27、28 共 16 個區塊——這是走到其餘區域的入口。

`2Eh DAMAGE` 的豁免參數讀反了，現在改回來：呼叫端（overlay-03
`2BCEh..2BD6h`）先推 `運算元5 & 7` 再推 `旗標 & 1Fh`，而豁免常式
（overlay-24 entry 7、`0D61h`）把第一個參數 `cbtw` 之後加進 d20、第二個存進
`DS:6788h` 再索引 `record[+6Dh + 類別]`。所以**旗標低五位是修正、運算元 5 的
低三位才是類別**。先前反過來讀，於是資料裡低五位等於 10 的旗標會被當成第 10
個類別（`+6Dh` 只有五格）而失敗即關閉——治具走到野外後面的區域時出現四次。

野外那三張圖的邊界與地點腳本也讀完了（spec 105）。邊界是三張以 `DS:033Dh`
（八方位，0 起算）分派的 `ON GOTO`：`49C4 <= 2` 擋掉帶北向分量的三個方向、
`49C3 <= 2` 擋掉帶西向分量的三個方向、`49C3 == 15` 的東北與東直接跨圖
（東南要 `49C4 != 31`）。「拒絕」就是 `SAVE 255 → @6DC9`。
46 個地點格共用 **20 支腳本**（25 八支、26 八支、27 四支），每一支的入口與
第一句原文都列進 spec 105，名字取自原版而不是攻略。
