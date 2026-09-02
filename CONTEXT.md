# Pool of Radiance remake 現況

更新日期：2026-09-01。

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
  Pool `3b17d57` 已將此契約接到初始 map cell lifecycle；未處理的事件會設為 pending
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
