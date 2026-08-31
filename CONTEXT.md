# Pool of Radiance remake 現況

更新日期：2026-08-31。

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
  `.CHA +10h..+15h` 六能力、`+30h` age、`+32h` HP 與 word `+8Eh` Gold 已由
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
  順序的 Party Creation Menu，`A` 加入最多六名玩家角色、`L` fail-closed 載入、F10 保存
  後離開；Xvfb 正常按鍵已走到 Library 1／Party 1，再以 `B` 進入已證實的
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
共用 VM；Spec 021 的 `0Ah` 與字串比較也已依真實 consumer 接線。正式引擎鎖版為
`v0.0.0-20260831132230-0a028e0956c1`；初始地圖 sweep 已無靜態錯誤。

Sune 選單後的同一 VM 分支已由 Spec 017 閉合到服務入口：YES（選單索引 0）先令
`6E79h=0`、進入 `AA63h`，執行 opcode `1Ch CLEARMONSTERS`、
`SAVE 1 → 6DE2h`，再抵達 opcode `24h COMBAT`。共用 engine 現以作品中立的
`MonstersCleared` 訊號聚合到 external boundary；Pool 只有在該訊號與 `6DE2h=1`
同時成立時才進 Sune 神殿，其他 COMBAT 不會誤路由。NO（索引 1）則顯示
`THEN YOU MUST LEAVE.`，保存 `49F0h → C04Bh`、`49F1h → C04Ch`、
`FFh → 6DE1h`，呼叫 `2C90h` 後 `EXIT`。

已推翻的斷言：`CALL 2C90h` **不是神殿 healing 本體**。它在 Rolf 34-step 導覽的
每一步及 Sune NO 分支都出現，現階段只能列為 redraw／movement service 候選；未追完
overlay dispatcher 前不得命名，也不得把所有 `CALL` 自動續跑。opcode `24h COMBAT`
對應 overlay-03 entry 50（`2E90h`），其 far call `010A:00F2` 已由 MZ header
`0x3B0` 與 TPOV manifest 精確反查至 overlay-25 entry 42（`2C81h`）。後續證據訂正：
entry 42 是通用選擇／鏈結處理，不是 temple dispatcher。38 份 overlay 的 raw far-call
全掃只有 overlay-03 `18B9h` 的 `9A 25 00 35 00` 指向 overlay-04 entry 1；該 callsite
位於 opcode `15h VERTICAL MENU` handler，先檢查並清除 runtime state `+5C4h`。
ECL `6DE2h` 對 `+5C4h` 的映射仍是 `strong inference`，不可單獨冒稱 exact；但
ECL pattern、唯一 temple call 與原始 Pascal 選單字串已足以完成 Spec 017 的服務入口。
engine／Pool 全測試與 CoAB 唯讀 `internal/ecl`、`internal/game` 回歸均通過。下一步是
逐項閉合 overlay-04 的 Heal／View／Pool／Appraise 規則，優先完成玩家可見的 Heal
價格、HP／狀態與 Gold 垂直鏈；不是繼續把 overlay-25 誤當神殿 dispatcher。
