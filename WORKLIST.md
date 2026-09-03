# Pool of Radiance remake 工作清單

只列尚未完成且有驗收條件的工作。

## 下一輪優先：軟體世界說明書／Journal 繁中化

說明書本身即軟體世界代理時的官方繁中譯本，因此這批是轉錄與校對，不是重新翻譯；
契約見 [spec 054](docs/spec/054-softworld-manual-journal-corpus.md)。

- [x] 上冊「探險者手冊」（Adventurers Journal）p.1–54 全文轉錄，未轉錄數 0：
  線索報導 1–58、酒店傳言 1–23、議會公告 18 則、七節附錄；來源清冊
  `docs/audit/manual-scan-manifest.json` 固定 68 張掃描的 SHA-256 與尺寸；
  譯名對照表 `docs/reference/manual/glossary.md` 記錄異名與撞名陷阱。
- [x] 下冊「操作手冊」p.1–57 全文轉錄，未轉錄數 0：六章全部涵蓋，含安裝流程、
  密碼盤用法、人物管理選單、戰鬥造形編輯、時間制、命中公式與逐條法術；
  術語已併入同一份對照表。
- [ ] 兩冊第二輪覆核：逐條回對掃描原頁，確認沒有轉錄漏字，並解掉目前標記的
  字形疑義（上冊 p.23 的「掏」）與畫面數值判讀（下冊 p.16／p.22 的 THAC0、HP）。
- [x] 跨冊譯名取捨已定案，見 `docs/reference/manual/glossary.md` 的「定案譯名」：
  四條依序套用的原則加 14 組決定（Phlan＝菲蘭、Valjevo＝瓦傑渥、Thentia＝珊提亞、
  Magic-User＝魔法師等），並分開 Yarash（亞拉斯）與 Yulash（尤拉斯）、固定
  ROUND／TURN 的定義。轉錄正文維持原書用字不動，本表只約束 game pack 與 UI。
- [ ] 怪物一覽表 42 種的中文譯名：說明書只給英文。遊戲內文字翻完之後，凡是在
  遊戲文字裡出現過的都已定名，並依 glossary 的「與《青色枷的詛咒》對齊怪物名稱」
  逐條決定；沒在遊戲文字或 `MON*CHA` 記錄裡出現過的仍待證，不發明。
- [x] 用原版資料反查定案譯名：`cmd/pool-name-audit` 掃全部 1,245 個 DAX block 的
  明碼與 6-bit packed 字串，結果在 `docs/audit/dos-original-name-strings.json`。
  九個專名由遊戲文字證實（Phlan、Bishop Braccio、Valjevo Castle、Lord Urslingen、
  Sokal Keep、Kobold、Magic Users、Thief、Yarash）；另外八個只在說明書出現，
  遊戲畫面不顯示，譯名不受遊戲文字約束。兩處訂正已寫進 glossary 的「原版資料反查」：
  `Sokal Keep` 拼法確定、遊戲寫 `MAGIC USERS` 而非 `Magic-User`。同一則公告出現
  `'ROGUES'`，說明書沒有這個詞，中譯待定。
- [x] 上冊三章（線索報導 58、酒店傳言 23、議會公告 18，共 99 條）已接進遊戲內手冊：
  `J` 開啟，打編號跳條目。遊戲引用的每一個 `ENTRY N` 與市政廳實跑派發的九個公告
  字號都有測試逐條核對得到。契約見 spec 064。下冊附錄的規則表尚未接進 UI。
- [x] UI 第一段接線：標題提示、人物管理選擇項、建角四個選單、人物資料頁與姓名輸入
  已用說明書用詞顯示中文，字型走 `internal/etenfont`，`-lang zh` 缺字型失敗即關閉。
  驗收：`tools/capture-chinese-menu.sh` 的四張實拍圖與 manifest。
- [x] 戰術畫面版面重排並中文化：盤面上移到 y=58（25 列畫到 307），四行基線
  322／338／354／370、功能鍵列 386，16 像素行距在 ascent 14 的漢字字型下不相疊。
  狀態列訊息改走可翻譯的 `say`，沒接語言時退回英文。
  驗收：`docs/screenshots/pool-remake-chinese-tactical.png`。
- [x] 遊戲內敘述文字（ECL 6-bit packed）的中文化管線：`internal/gametext` 以原文
  整句為鍵、原版 block 不修改，顯示端已接上導覽、事件文字與選單標籤。
  驗收：`docs/screenshots/pool-remake-chinese-tour.png` 是羅夫導覽的中文畫面。
- [x] 1,731 句、110,473 字元的遊戲內文字全部翻完（覆蓋率 100%），八個 ECL 封存檔
  一句不漏。每一批都通過「原文存在於盤點檔」的測試，專名依 glossary，說明書沒收的
  記在「遊戲內文字新增譯名」表。覆蓋率由 `cmd/pool-text-inventory -coverage` 量出。

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
  Spec 047 已將第一人稱內容填滿內框的三層繪製契約抽到共用 engine
  `viewport.FillBackgroundToStageInset`；Pool 與 CoAB 都只宣告各自的 viewport 幾何，
  共用 engine 負責背景外擴與牆片之後的上緣補層。兩作的整合測試已通過；Pool 的
  同狀態 DOS 畫面對拍仍屬本項其餘驗收，不因共用機制完成而自動升格。
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
  Spec 037／040／044 已另修正戰役持久化：現行 schema 6（schema 5 再加 ECL archive）與
  engine `142b245` 保存穩定玩家邊界的
  GEO position／完整不透明 ECL session，Load 直接回到 adventure；對話／服務／戰鬥
  中途續點仍待把前端 service state 一併版本化。
  Spec 038 的全 ECL 引用清冊再證明墓園正常 gate 依賴 `4AC1h >= 4`。Spec 039 已證明
  `4A39h..4A3Fh` 是墓園七種戰利品累積量，不是七項任務。Spec 040 又以 overlay-21
  的 TPOV entry 與 IDA Pro 9.4 bytes 閉合 Copper／Silver／Electrum／Gold／Platinum／
  Gems／Jewelry 七槽、角色 wallet、Pool／Share／Take、容量 gate 與餘額語意；remake
  已接正常寶物選單；七槽由 schema 5 引入、現由 schema 6 保存，schema 1..4 Gold
  遷移與 schema 5 ECL archive 遷移均有測試。Spec 041 已進一步證明 `9D63h` 掃描
  `4AA6h..4ABFh` 26 槽完成通知表，只有
  十個分支增加 `4AC1h`；可重生矩陣及全 ECL 直接引用清冊已有缺證據負對照。下一步
  逐槽追真正的 `FEh` producer，並由至少四條正常 commission 完成鏈抵達墓園服務。
  Spec 042 已先閉合 Slums 槽 `4ABBh`：ECL2/block20 的共用 helper 每次加一，達 25
  精確寫 `FEh`，`FEh` 以上不再改動；真 bytes 第 24／25 次正對照已通過。正式遊戲
  啟動也已載入八個 archive／29 blocks 的獨立 ECL namespace，不再只持有 ECL3；
  Spec 045 又閉合並實作 `ECL3/block0:9955h` 的正常 controller：
  `LOAD FILES FF,FF,7F → SAVE 2,6E12h → NEWECL 20` 會切到 ECL2，再由 entry 4
  載入 GEO2/block20 與 WALLDEF2 slots 2/4/1；真檔垂直測試由 ECL3 起跑且 ECL2
  存讀回歸通過。尚待把 25 個實際戰鬥完成呼叫與返回 City Hall 接成正常玩家鏈。
  `4A96h` 是接受後重入
  閘門；`4AB1h` 與吸血鬼事件相連但語意尚未足夠，不可猜名。
  Spec 046／048／049 已再接通第一場真實 Slums 遭遇的戰鬥前鏈：共用 VM 保留
  `LOAD MONSTER 13,1,4` 與 `4,3,4`，Pool 依目前 ECL archive 從
  `MON2CHA.DAX` 精確載入兩筆 285-byte `ORC` record，正常前端 staging 顯示
  `ORC ×1 / ORC ×3`。ECL PC 保持在戰後第一條 `9E6Dh`，Enter 不會自動勝利，
  `4ABBh` 也不會預先增加。Pool 原版角色資料頁 consumer 已閉合 285-byte record 的
  max/current HP、AC、THAC0、第一組 damage dice／signed bonus 與 movement；真實兩筆
  ORC 抽樣已固定。Spec 050 又由 overlay-13 attack caller 與 overlay-24 hit／dice
  consumer 閉合並實作 roll 1 miss／roll 20→100 score、internal THAC0／AC 比較及
  NdS＋signed bonus 的純規則。Spec 051 再閉合雙 attack slot 的 `+A1h/+A2h` base
  source、`+113h/+114h` phase count、slot 2→1 消耗順序及交錯 damage offsets；兩筆
  ORC 的 primary/secondary base rate 是 `2/0`。
  phase counter 生命週期亦已閉合：combat setup 清零、每個回合邊界 byte 增一，並有
  `255→0` wrap 測試。Spec 052 已接先攻候選選擇、DEX／casting-time modifier 與 Delay；
  Spec 053 已接移動初始化、effect code 12 的 `27h→2Ah→3Ah` accumulator 及八方向
  2／3 半步成本；START.EXE 的 66×4 戰術格位類別表已有嚴格 typed parser，四欄 raw
  shape、entry threshold、presentation consumer 與八方向 byte-wrap 座標 delta 已閉合。
  速度效果的玩家可見語意已有分級證據，法術 selector 與 effect ID 則保持分離。
  移動後反應攻擊已由 Spec 059 閉合觸發條件與閘門順序：候選是「移動前鄰接、
  移動後不鄰接」的差集（原版暫時移動、查詢、再復原），不是全部鄰接敵人；
  overlay-25 entry 6 是「對手身上有沒有 `DS:2880h` 那四個致能效果碼」，
  entry 27 是效果串列的線性搜尋（頭在 record `+7Fh`、節點 `+5` 是 next）；
  朝向窗是目前朝向的前後兩格；攻擊槽依 `+A1h` 與 `+113h`／`+114h` 選出。
  仍未閉合的是 overlay-13 entry 11 背後的 `DS:677Ch` 否決旗標 producer、
  `+108h` 結構的 `+3`／`+0Fh` 旁路旗標，以及查詢碼 `4Bh`／`4Ah` 的玩家語意。
  地形／碰撞已由 Spec 057／058 閉合（直線追蹤、格位類別表的 PathByte1／
  PathByte2、目的格探測與戰術層 DS 版面）。
  裝備覆寫已由 Spec 065 閉合一半：物品型別表不在 START.EXE 裡（`DS:54E0h` 那段是
  未初始化的，匯出來整片 FF），來源是 ZIP 的 `poolrad/items`（2-byte 檔頭加 128 筆
  × 16 bytes）；索引就是物品記錄的 `+2Eh`，由墓園那把 Two-Handed Sword +1
  （`+2Eh = 26h`，表的第 26h 筆是 1d10／3d6）正對照釘住。三張能力值修正表
  （力量索引、力量命中、力量傷害、敏捷投射）已逐分支照抄，overlay-25 entry 1 的
  十一個步驟已實作，玩家可在遊戲裡按 `I` 裝備武器，戰術戰鬥的 THAC0 與傷害骰
  改由武器決定。仍未閉合：記錄 `+0AAh` 與 `+2Eh` 的 producer、裝備之後的 AC
  重算鏈（`sub_281`／`sub_39F`）。
  下一步不再重做基礎命中／傷害、rate rounding、phase counter、先攻選擇、cell table
  raw parser、純移動預算或地形／碰撞，而是閉合命中 modifier 的其餘來源、
  防禦與死亡分支的先攻值消耗、
  特殊攻擊與 status transition，再接戰術畫面及勝敗／全滅 continuation；在該範圍 READY
  前保持失敗即關閉。
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
  接線，不污染共用 engine。現行 engine `d59f339`（含 `91801a5` 的 TREASURE 契約與
  Spec 037 session snapshot）的 Pool 全套與 CoAB
  核心相容回歸均已通過。下一步依玩家路徑
  閉合 City Hall reward／commission 迴圈與其後尚未接妥的服務規則。
- [x] 商店服務：四家店與墓園戰利品共用 `CLEARMONSTERS → TREASURE → SAVE → COMBAT`
  邊界，判別靠 `6E6Ch=1`／`6EF6h=1`／`6E6Dh=16` 三個旗標（不是 item block 編號）。
  存貨是 `ITEM3.DAX` 的 block `34h`..`37h`，價格在記錄 `+3Ah`。購買扣金幣並沿用
  spec 035 的負重／格數上限。契約見 spec 067；賣出與其餘六種貨幣的換算仍未做。
- [x] 法術名稱表：`START.EXE` 位移 41052 起 56 筆、步長 41，六個分組與說明書
  下冊第六章逐條相符；56 條中譯取自說明書。遊戲裡按 `K` 查閱。契約見 spec 068。
- [x] 預設人物檔的 `.spc` 已解出：它是 spec 059 那條**效果串列**的存檔形式，
  9-byte 節點＝5 bytes 內容＋4 bytes 遠指標。代碼與已裝備的魔法物品一一對應
  （`3Dh` ↔ Ring of Fire Resistance、`26h` ↔ Gauntlets of Ogre Power，
  20 個樣本零反例）。契約見 spec 069；代碼 `59h` 與節點內容仍未解。
- [x] 記憶法術的存放方式：角色記錄 `+1Fh` 起 13 格，0 是空格，其餘是 1-based
  的法術編號（overlay-22 以 `2883h + id×41` 取名，`2883h+41` 正是第一筆 Bless）。
  三名預設施法者逐格解出，牧師只拿到神術、法師只拿到巫術。契約見 spec 070。
- [x] 經驗值門檻與等級上限：`DS:4013h` 起 8 職業 × 14 欄 × 4 bytes，
  門檻與 AD&D 逐項相同，`FFFFFFFF` 是上限哨兵，讀出來是牧師 6／戰士 8／
  法師 6／賊 9。昇級常式在 overlay-22 `2C6Bh`。契約見 spec 071。
- [x] 可記憶法術數：牧師表 `DS:4035h`、法師表 `DS:414Dh`，每級三個 byte，
  第 2..6 級與 AD&D 逐項相同；睿智加成在 overlay-23 `01ADh`，逐段各加一次且
  只加在本來就有格子的等級上。算出來與三名預設施法者記錄裡已存的
  `+0B2h..+0B7h` 完全相同。契約見 spec 072。
- [x] 法術效果的派發表：overlay-22 entry 5 在 `0EA3h` 用
  `call dword ptr [di+6A78h]`（`di = 編號 × 4`）一次分岔到 67 支常式，表在
  `DS:6A7Ch`（`6A78h` 是另一個程序變數，兩處直接呼叫時推四個引數，overlay-08
  還會改寫它）。表在 BSS，但填表的 overlay-22 entry 10 是 67 段固定樣式，
  所以對應關係完全靜態解得出來。五組完全同名的法術全部落在同一支常式上，
  沒有例外。表 dump 在 `docs/audit/pool-spell-dispatch.json`，契約見 spec 073。
- [x] 法術的參數表與效果訊息：67 個編號的行為差異全部來自 `DS:3194h` 的
  16-byte 記錄，共用施法常式 `08BCh` 用 `編號 × 16` 取它。已讀出八個欄位：
  職業（牧師 25／法師 31／物品 11，前兩者相加正是名稱表的 56 筆）、法術等級、
  命中旗標（FFh 的正好是五個碰觸法術）、射程與持續回合數（都是固定值加每級
  增量，數字與規則書逐條相符：Fireball 10 加每級 1、Lightning Bolt 4 加每級
  1、Bless 6 格 6 回合）、豁免要不要擲與類別、效果碼（為零的正好是不留狀態
  的那些）。
  41 段效果訊息各自屬於哪一支常式也解出來了，每一段都驗過本體真的引用。
  順帶更正了法術目錄：Restoration 是牧師第 7 級（隊伍碰不到，神殿服務），
  不是法師第 3 級。契約見 spec 074。
- [x] 豁免判定：全遊戲只有 overlay-24 entry 7（`0D61h`）一支。自然 1 必敗、
  自然 20 必成，都在加修正之前返回；其餘是「記錄 `+6Dh` 起第 N 格的目標值
  ≤ 骰值 + `+101h` 修正 + 呼叫端修正」。那五格就是 AD&D 的豁免表，七名預設
  人物涵蓋五個職業等級組合，逐列與規則書相同，欄位順序因此定下來。契約見
  spec 075。
- [x] **豁免接進施法路徑**：參數表 `+8` 的三種處置讀出來了（overlay-24
  entry 19，code `133Ah`）——1 完全無效、2 減半、其餘不動傷害。規則 2 剛好是
  火球術與閃電術、規則 1 剛好全是狀態類，與規則書一致。戰術狀態多了每一格的
  五個豁免目標值（隊員查 `DS:41E6h` 的表、怪物讀記錄 `+6Dh`），傷害法術施展
  時逐目標擲一次。契約見 spec 074。
- [x] **定身術**：規則 1 的第一支。豁免失敗就照參數表的持續回合數定住，
  那一格輪到直接結束回合，回合開始各減一。效果碼 `34h`。
- [ ] 法術的記憶與施展。容器都就位了（spec 069 效果串列、spec 070 記憶陣列、
  spec 072 格數）、派發表、參數表與豁免判定也閉合了（spec 073／074／075）。
  **流程在 overlay-15**（2026-09-03 定位到）：原版存取記憶陣列一律用
  `es:[di+17h]` 加索引，不用 `+1Fh` 當基底——先前拿 `+1Fh` 去搜位元組與
  反組譯文字都落空，是搜錯了基底。overlay-15 有 23 個進入點，整支都在動
  這個陣列，由 overlay-14 entry 2 與 overlay-03 entry 1 叫進來。
  **記憶已經接上**（2026-09-03）：可記憶數在記錄 `+0B1h + 組 × 3 + 等級`，
  那是上限不是遞減的剩餘量，所以「還能記幾個」是上限減掉已經記了幾個。
  依職業等級與睿智算出來的上限與十個預設施法者的記錄**逐格相同**，
  他們的記憶陣列也全部落在上限之內。
  **第 7 位的語意也解出來了**：那是「還沒記完」。選法術時設起來、休息時
  overlay-20 `0945h` 用 `subb $80h` 清掉並印 `has memorized`，時間是各法術
  等級的總和。預設人物一個都沒設，因為他們是休息完的狀態——沒有矛盾。
  **紮營接上了**（按 E）：休息把待記的記完並治好整隊（原版的
  `The Whole Party Is Healed`）；選法術仍在法術一覽上（1-6 挑人、M、F）。
  還缺原版挑天／時／分那一段與休息被打斷（`Stop Resting?`）。
  **施展接上了八支**（2026-09-03）：祝福術、治療輕傷、燃燒之手、魔法飛彈、
  電擊術、催眠術、火球術、閃電束，各自的算法逐條標了 overlay-22 的位址，
  除了魔法飛彈都與說明書逐字相符。催眠術的額度是 4d4 生命骰、依目標的
  `+73h` 分段扣，生命骰 6 以上一律花 20 而額度上限是 16——說明書寫的
  「一定程度以上的敵人完全失效」就是這一格。戰鬥中按 C 開清單、Enter 施出，**只列得出讀過
  處理常式的法術**，沒讀過的不列也不會安靜地什麼都不做。
  **選目標改成照原版的模式分派**（2026-09-03）：參數表 `+6` 的低四位是目標
  模式（overlay-13 `20ADh`），模式 0 作用在施法者自己、0Ah 整邊、8／9／0Bh
  是範圍。分組本身就是證據——模式 0Ah 正好是祝福、詛咒、急速、緩速，
  與逐支讀出來的處理常式對得上。**挑目標那一步也接上了**：模式 4／6／7／0Fh
  會先進選目標（N／P 換人、Enter 確定，預設停在繞得過去的最近敵人），
  與原版 overlay-13 的 `Next Prev Manual` 同一件事。`1E09h` 的輸出結構
  （目標遠指標 ＋ X ＋ Y）與參數表 `+0Eh`（沒得挑時的預設目標）已解出；
  **射程限制接上了**：物品型別表的 `+0Ch` 是
  「射程加一」（近戰存 0），overlay-13 `358Dh` 取它之後 `dec`，0 與 FFh 都
  墊成 1。戰鬥中按 A 用裝備中的武器瞄一個目標打，超出射程不打也不消耗回合。
  **距離的算法讀出來了**：`261Bh` 算的其實是**方向**
  （`0579h` 是 45 度楔形測試，`n` 索引兩張方向表），距離是
  `010Ah:00C5h`（overlay-25 `2591h`）——它用 overlay-31 entry 6（`0912h`）
  建一條路徑，找到目標那一格之後取**路徑成本除以二**。除以二說明成本以
  半格為單位（斜走與直走不同價），與盤面是斜投影相符；同一支 `0912h`
  也是怪物 AI 找落腳格用的（spec 096），移動／射程／AI 共用一個成本模型。
  `0912h` 的成本就是 `0419h`＝**TraceMovement**
  （spec 057，remake 早就有），它的成本以半格為單位，所以 `2591h` 的
  「除以二」是換回格。**距離已經換成這個算法**（`tacticalRange`），
  先前的切比雪夫在斜投影盤面上會低估。`0912h` 還多做兩件事 remake 沒有：
  把兩邊的體型展開成最多四格再兩兩比對（大型怪物佔 2×2）、以及先用方向
  楔形過濾。**範圍也照原版收人了**：`0912h` 把預算原封不動交給
  `0419h` 當上限，所以「在範圍內」＝「那個預算內走得到」，不是半徑比大小；
  火球術推的預算是 2（`2675h`），方向 `FFh`（不限方向）。其他範圍法術的
  預算還沒讀，那些先收整邊。另外還缺 `352Ch` 的格子游標（Manual），
  以及火球術那個 `DS:4933h → +1CCh` 分支（非零時走單一目標那條）。
  **另外二十五支是純泛型的**（2026-09-03）：只推「編號 ＋ 四個零」給
  `08BCh`，射程／持續／豁免／效果碼全部來自參數表，所以認得出版型就等於
  一次接完一整批——`ParseGenericSpellHandlers` 逐位元組比對整個版型認出
  25 格（護邪、防護、隱形、解鎖、致盲、詛咒…）。加上逐支讀的二十七支，
  六十七格裡接得出來的有 **52 格**（`TestImplementedSpellCount` 釘住這個數字）。要比對整個版型不能只看「有沒有呼叫
  `08BCh`」——會算傷害的那幾支也呼叫它，只看呼叫會把傷害弄不見。
  還缺：其餘十三支處理常式（共用機制已解出並寫成 spec 098：擲骰是 `Roll(顆數, 面數)`、
  施法者等級由 overlay-25 `26F8h` 取而且戰術地圖外一律當 6、處理常式把傷害
  用四個覆寫參數傳給 `08BCh`。**卡在一個未解的點**：Magic Missile 的發數是
  `等級 ÷ 2`，而說明書寫的是「每昇兩級多一發，第 3 或 4 級 2 發」＝
  `(等級+1) ÷ 2`；第 1 級照碼算出來是 0 發 0 傷害。照碼接了（一手資料贏
  二手），測試把那個 0 釘住，驗出不同結果會是紅的。
  **決定性的實驗還沒做**：原版拿第 1 級法師施魔法飛彈看傷害是不是 0）、
  原版紮營畫面挑時間那一段、
  法師的法術書（哪些法術記得起來，現在不限制）、
  參數表 `+0Bh`..`+0Fh` 五個欄位、`+8` 的 1..3、
  編號 57..67 這 11 個無名效果從哪裡進來。
- [x] **昇級接完了**（2026-09-03，spec 097）。訓練掛在隊伍管理畫面的 `T`
  指令上（原版就是這樣：overlay-16 entry 1 的 `036Fh` 比 `T`，選單字串
  `DS:06ABh` 是 "Train Character"），1-6 挑人、T 訓練。角色的八個職業等級
  存進 `save.Character.ClassLevels`，`partyClassLevels` 讀它，所以 THAC0
  與豁免會跟著變——`TestTrainedLevelsReachCombatStats` 擋「升了卻沒傳到
  戰鬥數值」這條。已閉合的規則：經驗值在記錄 `+0ACh`／`+0AEh`（32 bit，
  不是「下一級的門檻」）、三道閘門、訓練會把經驗值砍到「升兩級的門檻減一」、
  職業分類遮罩 `DS:5F2h`、生命骰兩張表（走得到的四個職業與 AD&D 一版相同）、
  第一級的地板「面數 × 2 ÷ 3」、體質加成表與純戰士的額外加成（比的是複合
  職業碼，戰士／賊拿不到）、HP 保留受傷量加上去。
  還缺：種族／能力值的等級上限那一段（Pool of Radiance 走得到的四個職業
  沒有那個限制，目前不影響）、每一家訓練所收哪幾類（`T` 的啟用旗標
  `DS:06D4h` 從哪裡設還沒讀，所以現在每一類都收）、昇上法師等級之後的
  學新法術流程、訓練要不要收費。
- [x] **經驗值發下去了**（2026-09-03，spec 097）。一隻怪物值
  `+0B8h + +0BAh × 生命值`（與 AD&D 一版逐筆相同：KOBOLD 8、ORC 15、
  OGRE 195、SPECTRE 2030），總額除以分的人數，再由每個人依複合職業碼調整
  （純職業主屬性 > 15 多拿十分之一，兩職業除以 2、三職業除以 3）。
  `save.Character.Experience` 存得下來，欄位是 `omitempty` 所以舊存檔照讀。
  索寇要塞第一場實測：12 隻怪物，六人隊每人 64 點。
  還缺：原版「有資格分」的判準（`+10Dh` 為 0 與狀態 1）沒讀完，
  目前全隊都分。
- [ ] 建立最小 game pack 與 adapter；不複製 CoAB 的地名、位址或劇情資料。
- [x] 隊伍朝向的座標系：原版是 0 北、1 東、2 南、3 西，不是共用 engine 的
  0/2/4/6。導覽 34 步裡有位移的 20 步逐次與「上一步的朝向」相符、零例外，
  35 個原版位置也從沒出現 4..7。混用造成三個症狀：開場結束後隊伍**完全走不
  動**（轉向 45 度一步，從奇數朝向永遠轉不到偶數）、`WallWrapped` 對奇數朝向
  靜默回「沒有牆」（35 個位置裡有 23 個是奇數）、存檔驗證把 1 與 3 判成非法。
  換算改成留在 Pool 這一側（`Spawn.Direction()`），存檔升到 schema 7 並把舊值
  除以 2。契約見 spec 076。
- [ ] **走得到的內容量到了**（2026-09-03）：六個種子各走 6000 步，走到
  **2 張地圖**（GEO3/0、GEO4/21）與 **4 個 ECL 區塊**（ecl3 的 0／8／11、
  ecl4 的 21），戰鬥 tick 兩萬多次，**一次硬失敗都沒有**。
  遊戲總共有 **29 個有文字的 ECL 區塊**
  （`docs/audit/dos-ecl-text-inventory.json`，1731 條字串、11 萬字元），
  所以走得到的是其中四個。
  **這不是實作進度，是量測**：隨機走路走不到主線——主線要有目的地才走得到。
  它證明的是「引擎不會走著走著炸掉」，證不了「其餘 25 個區塊會動」。
  要往下推得靠**有目的地的走法**（照攻略的路線走），那是下一步。
  `TestRandomWalkReachesKnownContentWithoutFailing` 把這個下限釘住，
  走得到的地方變少就會紅。
- [ ] 從標題以正常按鍵完成建隊、進圖、事件、戰鬥、存檔與讀檔抽樣。
  已有 `TestNormalKeysReachTheFirstDungeonStep`：只用按鍵從標題走到建角、
  加入隊伍、Begin、推完開場與 34 步導覽，在地圖上走出一步，F10 存檔後
  重開一份按 L 讀回來並繼續走。戰鬥那一段見下面兩條；**還缺**存檔／讀檔
  在戰鬥中與戰鬥後的抽樣。
- [x] `29h ENCOUNTER MENU` 已接上。四個選項的選單、五格類型表、兩個移動力
  門檻、以及把結果碼寫進運算元 4 指的 ECL 變數都照 spec 078 實作；十四個
  運算元的型別也量過（運算元 4 是帶位址的記憶體運算元，要用 `WordAddress`
  不是 `NumericValue`；10..12 是打包文字）。移動力由 spec 079 的三段管線算，
  `+72h`／`+102h`／`+11Ch` 三個欄位已進存檔 schema。
  `TestNormalKeysReachTheFirstCombat` 只用按鍵從標題走到索寇要塞，觸發選單、
  選 COMBAT，ECL 依結果碼分支把骷髏 ×6 與殭屍 ×6 擺上場。讀不到證據的組合
  （類型 3 選 ADVANCE、類型 4 以上）回錯誤不猜。
  **還缺**：怪物數怎麼擲、運算元 10..12、以及決定第四項是 PARLAY 還是
  ADVANCE 的那個條件。
- [x] 戰術戰鬥收得了尾。索寇要塞第一場（vs 骷髏 ×6＋殭屍 ×6，只用按鍵）
  現在四十回合內結束，三個各自能讓它永遠打不完的缺陷都修掉了：
  - 走進同伴那一格會揮刀砍同伴。`ProbeDestination` 忠實重現原版，只回報
    「那一格站著誰」——原版的格位表（`DS:5E89h`）沒有陣營，陣營在角色記錄
    的 `+10Eh`，所以判斷是呼叫端的責任。現在同陣營一律當成擋住。
    **待證**：原版撞到同伴是「擋住」還是「換位」。
  - 攻擊不消耗行動，同一個角色可以無限連打。原版的玩家指令迴圈
    （overlay-08 `0307h`）把「這一回合結束了嗎」的旗標位址交給攻擊常式
    （`0096h:0089h`，`03D7h` 那個 `lea -2(bp)`），`05A5h` 的 `cmp` 就是那個
    旗標。現在打完就 `endTurn`，與敵方路徑一致。
    **待證**：戰士的多次攻擊（記錄 `+A1h`，spec 051）接上之後要改成
    「打完所有攻擊次數才結束」。
  - 怪物撞到地形就原地不動。對卡住的那一刻逐方向問 `ResolveDestination`：
    殘敵在 (42,10) 有六個方向走得進去，只有它想去的正西與西南是牆。現在
    先問方向表要的那一格，走不進去才看其餘七個方向，挑走得進去而且離目標
    最近的（保留方向表偏好，否則會拿斜走換直走多花 50% 預算）。
    **繞路的規則不是原版的**。
- [x] 裝備算出來的 AC 與腳程已接進戰鬥。`0281h` 的五個累加器、型別記錄 `+6`
  最高位的開關、有加值的類別 2 盔甲壓掉護符戒指（`0382h` 立旗標、`0F35h`
  清槽）、以及 `10E3h` 的敏捷調整都讀出來了，七名預設人物的 `+111h` 與
  `+112h` 逐一相符。契約見 spec 080。AC 與移動力共用同一條物品鏈；空手的
  角色也要走完，否則會少掉敏捷那一項。
  **待證**：`+112h` 應該是背擊用的 AC（不含敏捷與盾、再減 2，與規則書逐項
  對得上），但還沒讀到戰鬥那邊的取用點，所以只算不用。
- [x] 武器是類別 0 那一件。原版把穿戴中的物品依型別表的類別放進
  `+CCh + 類別 × 4`（overlay-25 `0C76h`；類別 9 另外走 `+F0h`／`+F4h` 兩個
  戒指槽，型別索引 `49h`／`1Ch` 走 `+F8h`／`+FCh`），判斷有沒有武器讀的是
  類別 0 那一格（`0E81h`）。槽位表補進 spec 065。
  拿物品鏈上第一件裝備當武器會挑到戒指，傷害骰 `0d0`——戰鬥照跑，整隊打不
  出傷害，報表上只看得到「打不贏」。
- [x] 裝備好的隊伍打得贏索寇要塞第一場。`TestAnEquippedPartyWinsTheFirstFight`
  固定骰子，六名拿長劍 +4、穿板甲 +2 與盾 +2 的戰士只用按鍵從標題走到那一場，
  五個回合內清光十二隻怪物，答 N 收尾，戰後腳本續跑。
  驅動程式會繞路：八個方向依「走完之後離目標多近」排序逐一試到動了為止——
  只試最好的那一個等於撞牆就放棄，量到的會是驅動程式的極限而不是遊戲的。
- [ ] **裝備還沒走過玩家自己的路**。上面那條是把預設人物的裝備直接塞進隊伍。
  真正的鏈是「建角拿到金幣 → 走到菲蘭的兵器鋪 → 買 → 按 I 裝上 → 出城打」，
  其中兵器鋪由 ECL 的 treasure request 開（`enterShop`），庫存與價目已由真檔
  測試涵蓋（長劍 15、鏈甲 75、盾 15）。**還缺**：只用按鍵從標題走到兵器鋪的
  那一段沒有測過，所以「玩家買得到裝備」目前是推論不是實測。
- [x] `38h PROGRAM` 的值 0 已接上（spec 081）：走到 ECL3／block 11 那一格會
  開起隊伍管理，B 或 ESC 回地圖，ECL 從原地繼續。從地圖進來的那次不走
  「開始冒險」——那會把開場重跑、隊伍丟回起點，而畫面上只看得出「怎麼又在
  講故事」。值 9 仍硬失敗：問句的字串與「讓 block 結束」的路徑都還沒讀出來。
- [x] **ECL opcode 待辦清空**（從 16 條、253 個呼叫點降到 0）。盤點在
  `docs/audit/pool-ecl-opcode-frontier.json`，由 `cmd/pool-ecl-frontier` 重生
  （掃 29 個 block；「已處理」那一半直接讀共用 VM 的 switch 與 Pool 的
  passthrough 清單，不另抄一份會過期的常數）。
  **handler 內部也沒有硬失敗了**：`1Eh` 的 `6BA7h` 模式統計的是記錄 `+79h`，
  那是賊技能的「找／解陷阱」（spec 095，由預設人物與規則書雙向釘住）；
  `38h` 的值 9 先問一句再結束 block，走的是 `00h EXIT` 的同一條收尾路徑
  （spec 081）。兩處問句的原文都在 BSS 讀不到，用的是 remake 自己的字句，
  spec 裡標明了。`1Eh` 是隊伍統計，依運算元 1 的位址挑欄位算最小／最大／平均，
  骨架讀過一半；`34h` 讀運算元編號時讀的是未初始化的堆疊位元組。
  起始地圖上走得到的是 `39h WHO`（ECL3/b0 兩處、b11 一處）與 `36h ADD NPC`
  （各一處），所以那兩條擋在最前面。兩條的骨架已由 spec 083 解出，並解出
  它們共用的「目前角色」槽 `DS:5CF0h`——`39h` 寫、`36h` 讀、`38h` 也會搬。
  **還缺**：挑人的 UI（overlay-25 entry 42）、NPC 記錄從哪個封存檔載入
  （overlay-17 entry 9）、以及記錄 `+84h` 的語意。
  已接：`33h PRINT RETURN`／`3Dh CLEAR BOX`（spec 082）、`2Eh DAMAGE`
  （spec 084）、`32h FIND ITEM`／`22h PARTY SURPRISE`／`23h SURPRISE`
  （spec 085）、`2Ch PARLAY`（spec 086）、`0Fh`／`10h` 輸入（spec 087）、
  `28h ROB`（spec 088）、`3Ch PROTECTION`（spec 089）、`39h WHO`（spec 090）、
  `36h ADD NPC`（spec 091）、`1Eh CHECKPARTY`（spec 092，三種模式接了兩種）、
  `34h ECL CLOCK`（spec 093）、`3Bh SPELL`（spec 094）。
- [x] 「人多的戰鬥會卡死」查清楚了：**卡的是測試驅動程式，不是遊戲**。
  先前那支驅動程式挑方向用的是直線距離，而戰場是斜投影又多牆（spec 060），
  直線距離會把人帶進死角然後在那裡來回。改成從目標做一次寬度優先、依實際
  步數挑方向之後，**六個種子全部走完 4000 步沒有硬失敗**——先前卡住的兩個
  （29 與 41）分別在第 915 與第 687 回合停住，現在都打得完——人類玩家本來
  就會繞路。
  排除掉的成因（量過，不要再查）：不是收尾規則漏接（原版同樣沒有「兩邊都
  還在」的出口，spec 062）、不是戰場產生錯了（斜向的牆是原版的斜投影，
  寫入與讀取都用 `TacticalRowStride = 32h`）、也不是方向常數混用
  （`WallDirection*` 是 0／2／6，`geometry.Grid.Wall` 收的正是 0／2／4／6）。
- [x] **敵方靠不過來**（2026-09-03 修）。敵方選方向改用**繞得過去的實際步數**
  （對目標格做一次 BFS，牆與不可進入的格子不展開），不再用直線距離。
  對照量測：同一場 41 隻對 6 人、隊伍完全不還手，直線距離版打到第 852 回合
  仍是 `FOE 31 CLOSED 0 STEPS ON 6`、我方滿血；步數版第 3 回合接觸、
  第 5 回合隊伍被打死。`TestFoesCloseOnACrowdedBoard` 顧這條，門檻訂在
  「不還手的隊伍會被打死」而不是「有人靠過來」。
  **這仍不是原版的選路**：原版（overlay-31 entry 6，spec 096）是先列出目標
  周圍可站的格子再挑，換成原版規則仍在下一條。
- [x] **有目的地走完整個世界**（2026-09-03 新增量測）。隨機走路量的是「不會
  炸掉」，走不到的地方它分不出來是「原作沒有」還是「remake 走不進去」。
  `TestDirectedExplorationReachesMaps` 改成**廣度優先逐格踩**：每次挑最近
  一格沒踩過的走過去，換圖就在新圖上繼續，路徑用的是 GEO 自己的
  `CanMoveDungeonWrapped`，不會宣告出資料裡沒有的通路。
  量到的數字：**448 格、2 張圖、4 個 ECL block（0、8、11、21）**，
  而且**兩張圖都踩滿了**（GEO3/0 226 格、GEO4/21 222 格，前者與
  `cmd/pool-initial-cell-sweep` 算出來的幾何連通分量 226 完全相同）。
  兩張圖是**訓練所那一區**與**索寇要塞**——碼頭那一格的文字寫著
  「唯一的船是去索寇要塞的」，走過去就上船。
  全遊戲有 29 個有文字的 block，**其餘 25 個走不到的原因是主線進度沒推進**
  （spec 099：碼頭的航線被 `DS:4AA7h` 擋著，只有清掉索寇要塞才會開）。
  量到這個數字踩過三個坑，三個都會讓量測低報：
  1. 第一趟在起點附近就上船，之後困在要塞回不來，起始圖只走了 31 格。
     → 改成**逐趟走**，把已知的換圖點記下來，下一趟先擋掉它。
  2. 換圖**不是在按下前進那一 tick 發生的**（格子事件先跑，LOAD FILES 在
     事件那一段做掉），所以「按完前進比對 spawn.Map」一次都不會成立，
     上一條的機制整個是空的。→ 跨 tick 比對，記換圖前站的那一格。
  3. **牆是單向的**：走得過去不代表走得回來。廣度優先固定從同一個方向
     展開，每一趟都會走進同一個死角。→ 每一趟換一個展開順序。
     光是這一條就從 393 格變成 448 格（起始圖從 171 補到 226 全滿）。
  這一輪順手挖出三個會讓遊戲停住的東西：
  1. **雙方被地形分在兩塊不相連的區域**：GEO4 block 21 那一場，隊伍站
     x=17..22、敵方站 x=36..38，中間走不過去，回合數加到兩百多還在跑。
     暫時部署的敵方候選格現在要求**與隊伍連通**（spec 061）。
  2. **打輸之後同一場架無限重來**：`finishCombat` 的戰敗那一支沒有清掉排好
     的遭遇，而戰鬥的生命值是另一份陣列、沒有寫回隊伍，所以隊伍又是滿血。
     實測那個迴圈跑滿兩百萬 tick 都沒停。現在戰敗一併清掉遭遇
     （spec 046 契約 5 的「停止該玩家路徑」）。
  3. 寶物選單的 Exit 後面還有一句 Yes／No 確認——**這一條是測試自己的問題**，
     不是遊戲的：探索器原本一律選最後一項，最後一項是 No，於是選單原地重開。
  **還沒補的**：戰鬥的生命值不會寫回隊伍，所以打完架隊伍永遠是滿血。
  原版打完之後的生命值處理還沒讀。
- [x] **`DS:6DD5h`：走出這一區**（2026-09-03 接上，spec 100）。貧民窟
  （`ecl3` block 0 `993Ah` → archive 2、`NEWECL 20`）與索寇要塞的回程船
  （`ecl4` block 21 `9918h`）**掛在同一個變數上**，`DS:6DD5h` 不是 0 才會
  發生。remake 從來沒寫過它，所以兩條路都是死的。
  第三個使用點是貧民窟（`ecl2` block 20 `9934h`），而且**那一支在
  `DS:6DD5h` 非零之後用朝向 `C04Dh` 挑鄰居**——面向東回城區，其他方向去
  archive 8 的 block 29。只有「往哪一邊走出去」需要這個分支。
  一次性實驗也印證了：移動前把「這一步會讓座標越界」寫進 `DS:6DD5h`，
  遊戲立刻照它自己的文字走（21 → 0 → 20 → 29 → 18 → 26）；不設就一條都
  不會發生。語意訂為 **strong inference**，寫入點仍是 OPEN。
  **整套接起來試過一次**（沒有留在程式碼裡）：加上「看到 `2Dh CALL C01Eh`
  就清掉旗標」之後換區不再失控，走得到的地圖從 2 張變 **11 張**、ECL block
  從 4 個變 **8 個**、archive 從兩個變成六個。
  **還不能落地的兩個原因**：八個既有的逐鍵重現測試會紅（起點 (0,4) 就在
  西邊界上，往西一步現在是離開而不是移動，而那一步該不該離開沒有對過原版）；
  以及走出去之後很快撞上 `ecl7` block 26 的
  `LOAD FILES 5,5,0`——當下 archive 是 7，而 GEO7 沒有 block 5。
  順帶排除掉一條：`LOAD FILES` 載地圖前看的 party `+1CCh` 閘門不是答案，
  全遊戲只有 overlay-07 `02D2h` 一處寫它而且只寫 1。
  **語意已經不必再查**：攻略寫得很直接——菲蘭分成幾區，區與區之間靠邊界上
  的城門相接，走過去就到隔壁區（來源記在 spec 100）。
  `LOAD FILES` 那個硬失敗也解掉了（見下一條）。
  **`CALL C01Eh` 已經讀清楚並落地**：`2Dh` 的處理常式（overlay-03 `3026h`）
  拿運算元的字減 `7FFFh` 分派，`C01Eh` 那一支轉呼叫 overlay-07 `1A17h`
  ——依朝向走一格、邊界繞回、重算地形與牆的暫存。走出這一區的那一步是
  ECL 自己叫這一支完成的。`applyMapExitCommit` 就是它；因為還沒有人寫
  `DS:6DD5h`，那三個分支目前走不到，所以它不會觸發。
  **接上去之後會走到貧民窟**：從起點 (0,4) 往西一步落在 GEO2 block 20 的
  (14,4)，正是攻略說的貧民窟，落點就是西邊界的另一側。
  量到的世界：**2 張圖 → 3 張**（訓練所那一區、**貧民窟**、索寇要塞）、
  **4 個 ECL block → 5 個**（0、8、11、**20**、21）。貧民窟是主線的第二站。
  （先前試接時報過「12 張圖 8 個 block」，那個數字是虛的：當時
  `LOAD FILES` 還在追會落後的 archive，會生出 GEO1/21 這種不存在的組合，
  而且旗標沒清乾淨會一路換區換不停。）
  **九個逐鍵重現的測試也帶上來了**：不重挑種子，改成**逐格走**
  （`walkThisAreaUntil`，廣度優先找最近一格沒踩過的），並補上新走得到的畫面
  的處理（隊伍管理按 B、商店按 ESC、選單有 Exit 就走過去、清光敵人之後那句
  「還要不要繼續打」要答 N）。幾個原本釘死「是哪一場架」的斷言改成釘管線
  本身——第一場架現在是索寇要塞的毒蛙，那是路線決定的，不是規則決定的。
- [x] **打不死人的怪物**（2026-09-03 修）。原因不是命中判定，是**傷害骰讀錯格**。
  怪物記錄的傷害骰有兩格：`+115h`／`+117h` 與 `+116h`／`+118h`
  （overlay-13 有兩處寫入分別寫這兩格，所以「兩格都存在」是量出來的）。
  168 份記錄裡 162 份填第一格，**剩六份只填第二格**，而且都是帶特殊攻擊的
  那幾隻：POISONOUS FROG、MEDUSA、GIANT SNAKE（兩份）、DRIDER、PHASE SPIDER。
  只讀第一格的話牠們每一擊都是 0 點——實測索寇要塞那一場四隻毒蛙對完全不
  還手的隊伍打了 **10792 次、一次都沒扣到血**，那一場永遠打不完。
  現在第一格空的就退到第二格；填了第一格的那 162 份第二格都是 0，不受影響。
  **原版依什麼挑格還沒讀出來**（角色表的顯示常式 overlay-19 `06F0h` 讀第一格），
  所以這是推論，兩個方向的測試都釘住了。
  查法值得記一筆：先量「命中幾次、落空幾次」（0 比 10792），再量骰值分布
  （只出現 1..10，因為命中的那一半不會印出骰值），才看出問題不在命中判定。
- [ ] **原版依什麼挑傷害骰的那一格**：目前是「第一格空就退第二格」。
  **貧民窟的隨機遭遇**：`ecl2` block 20 `B1B0h` 起的 `LOAD MONSTER` 三個運算元
  都是記憶體參照。追下去發現不是新機制——`0x9808` 是 `1Dh PARTYSTRENGTH`
  的結果再 `÷3 ×2 +5`（另一處 `+10`），`0x6E80` 是
  `GETTABLE(B6E6h, 6E7Fh, 6E80h)` 查出來的怪物編號，兩者都在**入口 4**
  （`9A5Eh` 起）設定。所以要查的是「換區之後入口 4 有沒有跑到」，
  不是再讀一段原版。
- [x] **`LOAD FILES` 用區塊編號查地圖**（2026-09-03，spec 043）。
  區塊編號在八個 GEO 檔裡全域唯一（29 個編號對 29 張圖），而 `21h` 只帶編號
  ——第二欄整支沒有 consumer。所以編號本身就決定了 archive，不必追那個會
  落後的值。`TestGeometryBlockIDsAreGloballyUnique` 釘住那條性質。
  同一輪把 `syncArchiveFromEventMachine` 裡「順手設 `spawn.Map.Archive`」拿掉
  ——ECL 的 archive 與地圖的 archive 是兩回事，混在一起會生出 GEO1/21 這種
  不存在的組合。
- [x] **索寇要塞那一段推得動**（2026-09-03）。
  `TestSokalKeepOpensTheOtherBoatRoutes` 讓探索器拿到裝備、打贏那一場，
  `DS:4A21h` 變成 255；鬼魂說出通關密語 SAMOSUD，`DS:4AA7h` 變成 254，
  碼頭的其他航線就開了。密語要用 `INPUT STRING`（spec 087）打進去，
  測試會照打——那是遊戲裡鬼魂自己說的字。
  同一輪修掉一個會硬失敗的東西：`LOAD FILES` 的第一欄是 `FFh` 時原版**不載
  地圖**（spec 043），remake 原本只認 `{FF,FF,7F}` 這一組，於是要塞的
  `LOAD FILES FFh,2,FFh` 被讀成「載入 block 255」然後炸掉。
- [x] **碼頭的其他航線**：玩家的路徑整條通了，有測試釘著
  （`cmd/pool-game/harbour_walk_test.go`、`internal/gamepack/harbour_master_test.go`）。
  三道閘門缺一不可，前兩道是進度、第三道是功能：
  1. `DS:4AA7h` 要到 254——只有索寇要塞寫得了（ecl4/21 `ADAAh`／`ADD0h`：
     對費蘭說實話，或打贏亡魂那一場）。
  2. `DS:4A01h`（船票）不能是 1——清票的只有要塞裡對費蘭「說謊」
     （`ABBDh` 寫 255）。順序只能先說謊再了結，中間不能回城區，
     否則港務長會把票再發一次（spec 102）。
  3. 船資是**一枚白金**，而白金在 ECL 的 active-character 視窗 `6BC3h`。
     先前 remake 只投影姓名與士氣，那一格永遠是 0，港務長永遠回
     「你的白金不夠」。原版的視窗是指標指到本尊，所以扣錢直接扣在那個人
     身上；remake 用 `CharacterBinding` 做抄進去／抄回來，`39h WHO` 挑完人
     也要投影（spec 090）。
  **走路走不到別的地方，這件事量過了**（spec 101）：29 個 ECL 區塊之間
  幾乎全靠 `NEWECL`，而走出地圖邊界只接得起城區與貧民窟這一對——城區 28 個
  邊界出口裡，站在起點雙向走得到的只有 (0,4)W 一個。接得最廣的是 ecl7/26
  （一個區塊接十一個），而它掛在碼頭選單的 EAST／WEST／BAY 後面。
  圖與工具：`tools/go.sh run ./cmd/pool-world-graph -text`。
  順帶釘死一個容易讀反的東西：起點 (0,4) 的文字寫著
  「YOU ARE BY THE GATEWAY TO THE UNSETTLED AREAS」，而**牆的查詢會把座標
  夾回去**——overlay-30 `0358h` 在查牆之前先把 X／Y 夾回 0..15。走出邊界
  這回事仍然存在，只是由 ECL 自己讀 `DS:6DD5h`、自己叫 `CALL C01Eh` 完成
  （spec 100）。
- [ ] **探索器到不了野外**。主線那條路本身通了（`harbour_walk_test.go` 從走進
  港務長一路釘到落在野外 (9,29)），但隨機探索器走不出來：說謊清掉船票之後，
  回城區的路上會先踩到競技場 (7,2)，`9CACh` 把 `4A01` 寫回 1，港務長就再也不
  開口（spec 102）。已經加的兩件事不夠——「鎖住時優先去港務長」只在沒有計畫
  時評估，而隊伍回城時手上常常還有上一段的計畫。

  **「每一步都評估主線鎖、鎖一起來就丟掉舊計畫」試過了，更差**：走到的 ECL
  block 從 5 個掉到 4 個，而且「找港務長」一趟觸發 112 次——隊伍在城區與貧民窟
  之間來回，每次進城鎖就重新起來一次。下一次要先回答的是**隊伍為什麼還離得開
  城區**（鎖住時 `chooseAreaExit`／`chooseTransitionCell` 都關了），
  不是再加一個優先級。

- [ ] **野外／樞紐地圖（ECL block 25、26、27）還沒讀**。這是走到其餘區域的
  真正入口：spec 101 量到 ecl7/26 一個區塊接十一個，而 spec 104 把
  「每一格都停在 `2Dh CALL`」修掉之後，隊伍已經走得進去（治具
  `TestWorldTourReachesTheAreasBehindTheHarbour` 走到 block 26 與 27）。
  但**進去之後第一步就被送回城區**：ecl7/26 的入口 0 依 `DS:49C3h`／
  `DS:49C4h` 分派，那兩個值是碼頭那一段設的（`9C04h SAVE 7 → 49C3`、
  `SAVE 29 → 49C4`），數值超過 15，所以它們**不是格子座標**，是野外地圖上的
  位置或區域編號。要接的是：那兩個變數的定義域、野外地圖怎麼投影成格子、
  以及 `9989h`／`99B8h` 兩張 `ON GOTO` 的分支條件。
- [x] `WALLDEF selector 3 is not present`：**archive 挑錯了**。`LOAD PIECES`
  是腳本要的資源，要用 ECL 的 archive，不是它載進來的 GEO 的——野外那幾張圖
  就不同號（ecl7/26 載的是 GEO5 的區塊，而 `WALLDEF5.DAX` 只有 1 與 24 兩塊）。
- [x] `LOAD PIECES` 的 selector 解法閉合：**selector 不是編號時，退到比它小的
  最近一塊，取第 (selector − 編號) 筆記錄**。全遊戲 33 處 `LOAD PIECES` 只有
  五處落在編號之外（ecl4/10 的 22、ecl5/3・5/4・5/6 的 25），每一處都正好落在
  前一塊的記錄數之內——`WALLDEF4` 的 21 與 `WALLDEF5` 的 24 都是兩筆連著的
  記錄。這是資料全面對得起來的證據，不是單點推測。
- [ ] **`Pool DAMAGE saving throw category 10 is outside 0..4`**（治具走到
  野外後面的區域時出現四次）。`2Eh DAMAGE` 的旗標低五位被讀成豁免類別
  （`DamageFlagSaveCategoryMask = 0x1f`），但類別只有五個。要回頭讀原版的
  DAMAGE handler，弄清楚那五位到底是什麼。
- [ ] **`NEWECL FF` 要當成「不換區」**（治具走到 ecl1/24 那一帶時出現十幾次）。
  ecl1/24 的入口 0 是離開這一區的處理（`992Eh` 讀 `DS:6DD5h`），依朝向查兩張
  平行的八格表：`99A8h` 決定 `LOAD FILES` 要載哪一張 GEO、`99B0h` 決定要
  `NEWECL` 到哪一個區塊。

  ```
  索引    0    1    2    3    4    5    6    7
  GEO    FF   1F   FF   FF   0E   1A   FF   18
  ECL    FF   FF   FF   FF   0E   1A   FF   FF
  ```

  索引 1（往東）載 GEO 31 但**留在同一個 ECL 區塊**，索引 7（往西）載回
  GEO 24——block 24 一個腳本管兩張圖。`LOAD FILES` 的 `FF` 已經是「不載地圖」
  （spec 043），同一張表配對的 `NEWECL FF` 只能是「不換區塊」。前面那兩道
  `COMPARE @6E82 1／7` 之所以存在，正是因為那兩格要載圖但不換區塊。

  **修的位置在共用 engine**（`eclvm.BlockSession.switchTo` 目前對不存在的區塊
  直接報錯），而那是另一個 repo，要先取得使用者同意再動。
- [ ] **原版的敵方回合還沒讀**：入口是 overlay-08 entry 3（`01E4h`）依角色
  記錄的 `+10Fh` 分派——非零走 `0058h:0025h`（overlay-09 entry 1，code
  `000Fh`，整個 overlay-09 就是敵方 AI），零則走 overlay-08 `0307h` 的玩家
  指令迴圈（指令字串 `Move `／`View Aim `／`Use `／`Cast `／`Turn `／
  `Quick Done` 在 overlay-08 `05E1h` 起）。目前的目標選擇與繞路都是讓戰鬥
  能結束的權宜作法，要換成原版的選擇規則。

## 發行

- [x] 授權已定案並落地：PolyForm Noncommercial License 1.0.0（非商業免費含修改
  再散布，商業另談），`LICENSE` 與 `NOTICE.md` 在 repo 根目錄，也複製進每一個
  發行包。`NOTICE.md` 明列不因此被重新授權的東西：SSI 原版資產、軟體世界的
  說明書譯文、倚天字型、共用 engine 與第三方套件。
- [x] 三平台發行包可重生：`tools/package-release.sh <版本>` 產出 Linux AppImage、
  Windows ZIP 與 macOS 雙架構 ZIP，並寫 `manifest.json` 固定雜湊。
  AppImage 已由 `tools/linux-release-smoke.sh` 在容器裡實際啟動並截圖。
  契約見 spec 066。
- [ ] Windows 與 macOS 的真機啟動驗收。目前只證明得出「建得出來、包得起來」。
  `.github/workflows/platform-smoke.yml` 已寫好（只手動觸發），在 runner 上驗
  原生建置與測試——那兩件事交叉編譯給不了。**啟動仍驗不了**：實測 Ebitengine
  的 GLFW 在套件 init 就初始化，無頭環境連 `-h` 都 panic（見 spec 066）。
  要啟用該 workflow 得先定 repository visibility。
- [ ] repository visibility 與原版素材 deny-list（等待使用者）。
  現階段 GitHub repository 採 private。
