# Pool of Radiance remake 現況

更新日期：2026-08-31。

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
  `pool-remake-initial-rolf-event.png`；GEO／wall／第一頁事件來源是 exact，wrapped
  traversal 目前僅是跨作品 strong inference。Return 後的七頁導覽與 scripted movement
  尚未閉合，remake 會明確停住而不提前放行自由移動。DOS CHA／SPC exporter
  未冒充完成：
  33 份 CHA 均為 285 bytes，但 SPC 是 9-byte 節點鏈且 corpus 有 9／18／36 bytes，不能
  複製單一模板。初始 map、wall material 與 Rolf 第一頁已接，但地名、Pool 專屬視錐／
  背景 oracle、後續導覽、移動與遊戲內存檔仍未接。
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
WALLDEF／8X8D 素材 identity 與 Rolf 第一頁已 READY；地名、移動變體、Pool 專屬
第一人稱視錐／背景及其餘導覽仍須以 DOS runtime／executable 閉合。
首條玩家垂直鏈固定為標題 → 建角／建隊 → 第一個正常可操作地圖 → 事件／戰鬥 →
存檔／讀檔。
