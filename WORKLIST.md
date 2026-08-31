# Pool of Radiance remake 工作清單

只列尚未完成且有驗收條件的工作。

## P0：可重現研究基線

- [x] DOS ZIP 的 168 個檔案已逐檔記錄 size、CRC32 與 SHA-256，固定 ZIP 雜湊、
  `START.EXE` 錨點、檔案數與解壓總長；報表為 `docs/audit/dos-input-manifest.json`，
  可由 `cmd/pool-input-manifest` 在不解壓、不修改來源的情況下重生。DOSBox oracle
  仍將 ZIP 複製到 tmpfs writable overlay，原始來源唯讀掛載。
- [x] 以 engine `dax` 掃描全部 DAX：113／113 成功，合計 1,245 blocks；結果在
  `docs/audit/dos-dax-inventory.json`。下一階段仍須依 payload consumer 分格式驗證。
- [x] 以 engine `geometry.Parse` 掃描 `GEO1..8`：29／29 blocks 皆為 `0x402` bytes
  且成功解成 16×16 四平面；重生報表與結構契約見
  `docs/audit/dos-geo-inventory.json`、Spec 009。Pool typed catalog 已依
  `(archive, block ID)` 接妥並通過真檔／fail-closed 測試；地名、正常入口與移動變體
  不在此完成項。
- [ ] 完成 `START.EXE`／`GAME.OVR` 的 compiler、linker 與 overlay 邊界。
  MZ／`TPOV!` 與 family 強推論已記於 `docs/re/dos-toolchain-baseline.md`；仍須以
  startup code、RTL helper bytes、overlay directory 與 IDA 位址空間交叉驗證。
  TPOV 結構層已可重生解出 38 overlays／774 entries；角色 overlay 的 IDA 9.4
  窄切片已完成。清冊現由共用 engine `tpov` 與 `cmd/pool-ovr-manifest` 直接從 ZIP
  重生，overlay ID 固定零起算 0..37；compiler／RTL fingerprint 仍未閉合。
- [x] 建立 DOSBox 正常啟動 oracle 與未縮放標題／主選單截圖。
  驗收：Docker/Xvfb 有界重播，輸入序列、畫面與 metadata 齊全。
- [x] 完成 `TITLE.DAX` typed consumer 與 PNG／總覽圖匯出；block 1 放大 2× 後
  與原版標題逐像素 AE=`0`，規格見 `docs/spec/001-dos-title-picture.md`。
- [ ] 閉合 Pool ECL record format：Spec 002 已訂正 payload mapping base
  `9914h → 9900h`，既有 decoder 現可完整走過 26／29 blocks、14,724 條 reachable
  instructions。只剩 ECL5/block 7、ECL7/block 17、ECL7/block 22 三筆；須逐筆判斷
  variable record、控制流 fallthrough 或真 opcode 缺口，不得以 byte 猜命令。

## P1：第一條玩家垂直鏈

- [ ] 反組譯並寫 READY 建角／建隊 spec，包含戰鬥 sprite、調色與角色檔。
  已用正確的 `C:\POOLRAD\` 掛載走通 portrait、READY／ACTION combat icon、Parts、
  雙色六部位、Size 與完成建角；285-byte CHA icon 欄位已有單變因差分，Spec 003
  對這一範圍已 READY。六種族職業清單、引導文字與 Race→Gender→Class→Alignment
  狀態機已實作；Spec 004 已 READY，閉合年齡、能力限制、exceptional STR、Gold、
  hit dice／CON／多職平均公式，並已有注入式 dice roller 與純資料生成器。剩餘驗收是
  `cmd/pool-game` 已由標題以正常按鍵走到完整公式資料頁，R 可重擲，F1／F2／ESC／F10
  已有接縫測試與 Xvfb 截圖。YES 後的 1..15-byte 姓名與 HEAD／BODY／KEEP portrait
  editor 已接正常玩家路徑；Spec 006 已閉合 `+BBh/+BCh`、1..14／1..12 wrap、
  HEAD3／BODY3 稀疏 block descriptor 與 88×88 零間隙不透明合成。Spec 007 又閉合
  CHEAD／CBODY 184 blocks、Head 0..13、Weapon 0..31、READY／ACTION × 大小 family
  與六部位雙色；remake 已接真實素材雙預覽及 Head／Weapon／Size／顏色熱鍵。剩餘驗收是
  原版 nested icon menu 細節仍待 polish；Spec 008 已接版本化 remake 角色庫、原子保存、
  icon 確認後回到 Party Creation Menu、Add／Load 與六名玩家角色上限，正常按鍵抓圖已
  走到 Party 1/6。剩餘驗收是 DOS 285-byte CHA＋多條鏈 export、完整 Party Creation Menu
  功能，以及 theme 下 sprite／tileset 同步切換。
- [ ] 解出第一張地圖的移動遮罩、第一人稱背景／視錐與第一個玩家事件。正常 Begin
  的 ECL3/block 0 → `LOAD FILES 0,0,0` 已閉合 `GEO3/block 0, (15,1), facing 6`，
  初始 `LOAD PIECES` 又閉合 `WALLDEF3 block 0` 與 `8X8D3 blocks 101/102/103`，
  並由正常 `B` 畫面解析 42 個可見原版 wall stamps。GEO／wall material identity 是
  exact；Spec 010 已閉合首次旗標、Rolf 第一頁、事件位置／朝向、monster 12 與 Return
  閘門。Spec 011 又閉合並實作四張 34-byte table、34-step scripted movement、六個停靠
  selector／七頁文字與最終 ECL `EXIT`；正常按鍵已走到 Tyr 停靠畫面。wrapped traversal
  仍是跨作品 strong inference；Rolf 初次 APPROACH 圖像、自由移動交接、地名、
  bounded／wrapped／door policy、背景與同狀態 DOS 畫面仍待證明。
  Spec 012 已再閉合 overlay-03 `401Fh` dispatch → overlay-07 entry 27，以及一組
  cardinal 座標的 16×16 wrap。後續已訂正 `401Fh` 來源：原始 ECL operand 是
  `ECL7/block17 B69Ah` opcode `27h` 第四參數 `1F40h`，經 helper 反向組成；不是
  opcode `1Fh` 或一般玩家 `CALL` producer。下一個窄切片改追 opcode 27 handler、
  `1F40h` operand consumer 與 `6A0Bh/6A0Ch` 座標用途，再閉合牆／門 gate；不能直接
  把該 wrapper 當完整 movement policy。
  Spec 014 已先接 Rolf EXIT 後方向鍵轉向與 cardinal GEO forward：真實 `(0,4)` edge
  抽樣及 Xvfb `(0,4,3) → turn → (1,4,2)` 通過。仍須接每格 ECL dispatch、門選單／
  解鎖 mutation 與 DOS 同狀態按鍵對拍，才可把「基本 GEO walk」升為完整自由移動。
  Spec 015 已再接同 session 的 ECL3/block0 entry 0 `9914h`；Spec 016 進一步依五入口
  ABI 接 entry 1 `99EBh` SearchLocation。deterministic sweep 的 1,024 樣本在 Spec 019
  接妥 `20h` 跨 block session 後首次為 852 EXIT／156 event／16 error；Spec 020
  接妥 `14h COMPARE AND` 後為 856 EXIT／156 event／12 error；Spec 021 再接通
  `0Ah LOAD CHARACTER` 與 string-memory `COMPARE` 後現行為 856 EXIT／168 event／0 error，
  新增 12 筆皆為 City Hall 真文字，且不把空 PRINTCLEAR／PICTURE 灌成事件。正常按鍵已從
  `(0,4)` 經 `(1,4)` 走到 `(1,3)`，顯示 Sune 女祭司原始文字與 healing 問句，並接上
  原始 YES／NO menu 游標。Spec 017 已接 `CLEARMONSTERS → SAVE 6DE2h → COMBAT`
  服務邊界、原版 `Heal／View／Pool／Appraise／Exit` 神殿選單與 Exit 後同 VM 續行；
  Spec 018 已將存檔升為 schema 2，分開最大／目前 HP 與 raw status，並對 schema 1
  提供無歧義遷移；三種 Wounds 治療、個人優先／pool fallback 付款、正常 UI 與存檔
  回滾已 CONFORMED。其餘狀態治療與 View／Pool／Appraise 仍須各自 READY。`20h`
  NEWECL 已由 Pool dispatcher／handler 閉合並接入共用 engine；opcode `14h` 亦已
  CONFORMED；Spec 021 已閉合 `0Ah` 本輪所需的 active-character 三欄投影。下一步是
  Spec 022 已由 Pool overlay-03 精確閉合 entry 0（舊格）→ movement → entry 1（新格）
  的順序並修正正常玩家接線；Spec 023 已把正常按鍵由 Sune 接到 City Hall，精確走過
  外部文字、原版單項 Return menu、公告前言與公告編號後回到移動。`AF1Ch` 的 menu
  destination 已訂正為 UI 暫存 `9801h`，不是 commission 狀態 `4AC1h`，不得建立假 alias。
  Spec 024 又以非零 `4AC1h=1..9` 閉合原版 `ON GOSUB` 與九則 proclamation，逐值
  執行真實 ECL 至 `EXIT`；這只完成已持有 commission 時的公告選擇。Spec 028 的
  block8 全 trace 已推翻「clerk 有單一 `4AC1h` producer」：十個後段進度子程式會各自
  把它加一。機器清冊已固定墓園戰利品 7 槽、commission 16 路與五個 external call；
  Spec 029 已 READY 並驗收全新隊伍依序顯示三項預設委託後由 `AF7Ch EXIT` 回到移動。
  Spec 030／031 已接第一個 external query `PARTYSTRENGTH`：真實 handler、五欄公式、
  三份一級 class oracle 與 block8 強度 18／19 正負門檻均通過。Spec 032 已接作品中立
  `TREASURE` 八欄請求，真實墓園 `A780h` 的全零／`33h` 會依序到 `A791h COMBAT`。
  Spec 033 又閉合 `ITEM3.DAX/33h`：五筆 63-byte record 是四張兩法術牧師卷軸與一把
  `Two-Handed Sword +1 +3 vs. Undead`，typed loader 保留原始 record 與順序。
  Spec 034 已再閉合 overlay-05 的戰利品主選單、`Take → Items`、`+2Ah` 物品鏈移除與
  `FreeMem(3Fh)`；Spec 035 進一步閉合 overlay-06/19/25 的 `OverLoaded`、16 格、重量與
  力量容量，schema 3 inventory 與原子 Take 已接妥。Spec 036 亦證明墓園的 `COMBAT`
  在零 encounter／服務旗標時分派 overlay-05 post-combat，不是假戰鬥。真實 block8
  `A780h → A791h` 位元組測試已進入五件物品服務；下一步是以正常 clerk/commission
  狀態從地圖抵達此服務並驗收退出 continuation；另有逐槽
  producer 條件／上限、其他
  reward／commission 矩陣與 `AD29h` Skullcrusher 離隊事件。訓練／升級完成時還須把
  一級 immutable roster snapshot 換成有版本的 live derived-stat seam。
  Spec 037 已另修正戰役持久化：schema 5（schema 4 campaign 契約加七貨幣）與
  engine `142b245` 保存穩定玩家邊界的
  GEO position／完整不透明 ECL session，Load 直接回到 adventure；對話／服務／戰鬥
  中途續點仍待把前端 service state 一併版本化。
  Spec 038 的全 ECL 引用清冊再證明墓園正常 gate 依賴 `4AC1h >= 4`。Spec 039 已證明
  `4A39h..4A3Fh` 是墓園七種戰利品累積量，不是七項任務。Spec 040 又以 overlay-21
  的 TPOV entry 與 IDA Pro 9.4 bytes 閉合 Copper／Silver／Electrum／Gold／Platinum／
  Gems／Jewelry 七槽、角色 wallet、Pool／Share／Take、容量 gate 與餘額語意；remake
  已接正常寶物選單、schema 5 七槽存檔及 schema 1..4 Gold 遷移，全專案測試與 vet
  通過。Spec 041 已進一步證明 `9D63h` 掃描 `4AA6h..4ABFh` 26 槽完成通知表，只有
  十個分支增加 `4AC1h`；可重生矩陣及全 ECL 直接引用清冊已有缺證據負對照。下一步
  逐槽追真正的 `FEh` producer，並由至少四條正常 commission 完成鏈抵達墓園服務。
  `4A96h` 是接受後重入
  閘門；`4AB1h` 與吸血鬼事件相連但語意尚未足夠，不可猜名。
  Spec 025 已先修正進門 lifecycle：`NEWECL 8` 後依原版跑 entry `0→4`，正常按鍵抵達
  `(4,4,2)`、script block 8，但依 `LOAD FILES 0` 正確保留 GEO3/block0；sentinel
  `LOAD PIECES` 已消費，非 sentinel 選圖仍失敗即關閉。Spec 026 已把同一正常路徑
  接到 `(4,5)` clerk office 外與 `(5,5)` clerk 第一頁；Spec 027 又依 Pool 原始
  handler 在共用 engine 接妥 `35h SAVE TABLE`；Spec 029 又讓正常按鍵穿過 `9C9Eh`，
  依序顯示三項預設委託與列舉結尾，再由 `AF7Ch EXIT` 回到移動。下一個窄切片是真正
  `4AC1h` 進度 producer 與正常墓園 reward 路徑，不是重做已完成的七貨幣服務、
  City Hall 入口，或把預設分支冒充完整 reward／commission 服務。
- [ ] 將 CoAB 已驗證但仍夾有作品常數的 ECL runtime 分批泛化到共用 engine；目前已抽
  operand 求值、控制流、算術、SAVE／GETTABLE、選單 continuation與 Pool 前端的真實
  Rolf VM boundary consumer（Spec 013 已 CONFORMED）；跨 block session、deterministic
  RANDOM、VM Clone、同-session entry 切換、作品中立 `CLEARMONSTERS` 訊號，以及
  opcode `0Ah`／`20h`／`27h`／`35h` 都已有 Pool consumer；`27h` 目前只完成 raw
  request，ITEM payload 與戰利品 UI 則由 Pool game pack／adapter 依 Spec 033～036
  接線，不污染共用 engine。現行 engine `142b245`（含 `91801a5` 的 TREASURE 契約與
  Spec 037 session snapshot）的 Pool 全套與 CoAB
  核心相容回歸均已通過。下一步依玩家路徑
  閉合 City Hall reward／commission 迴圈與其後尚未接妥的服務規則。
- [ ] 建立最小 game pack 與 adapter；不複製 CoAB 的地名、位址或劇情資料。
- [ ] 從標題以正常按鍵完成建隊、進圖、事件、戰鬥、存檔與讀檔抽樣。

## 發行決策（等待使用者）

- [ ] 公開前決定程式碼授權、repository visibility 與原版素材 deny-list。
  現階段 GitHub repository 採 private，避免在決策前公開來源或錯誤聲明。
