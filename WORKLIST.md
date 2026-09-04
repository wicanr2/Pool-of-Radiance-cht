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
- [x] 上冊轉錄的結構清點（2026-09-04 機械複驗）：`## p.N` 涵蓋 p.1–54 無斷號，
  `### 線索報導 N` 1–58 連續、`**傳言 N**` 1–23 連續、`**公告字號 …**` 18 則
  （LIX 起至 CCXIV），附錄七節到齊——第七節「武器一覽表」因為跨到 p.54 用的是
  頁層級標題，不是 `#### 7.`，內容沒有漏。
- [x] **兩冊第二輪覆核**（2026-09-04）：111 頁全部回對過 `workplace/manual-scan`
  的掃描原圖，機械檢查在 `internal/journal/manual_pages_test.go` 與
  `manual_tables_test.go`。
  - **覆蓋**：68 張掃描共 136 個半頁，111 個被某一頁認領，其餘 25 個逐一列在
    `nonContentHalves`（外盒、封面、兩冊目錄、磁片標籤、譯碼轉盤、原版英文廣告頁）。
    差集必須為空，清單也不能留過期條目。
  - **結構**：兩冊印刷目錄的 20 條章節起始頁全部對上；線索報導 1–58、酒店傳言
    1–23、附錄七節連續不重複；議會公告 18 則、羅馬字號遞增。
  - **兩份抄本互比**：轉錄與 `internal/journal/zh-TW.json` 的 99 條逐字相同，
    連頁碼與掃描半頁都對。第一輪就抓到 `掏→搯` 只改了轉錄。
  - **對原版資料**：附錄 4 的經驗值門檻與法術格數對 `START.EXE` 的表；附錄 7 的
    武器傷害對 `ITEMS` 的物品型別表（39 種近戰武器，37 種逐格一致）；附錄 3 的
    重量、AC、移動對 `ITEM` 記錄與 `ArmourMovementRate`。
  - **逐頁回對修正的**：上冊 p.36 地圖表漏了 `IRONFANG KEEP` 與
    `DRAGONSPINE MTNS. 龍脊山脈`；p.39 人名 `阿拉曼`→`阿拉旻`；p.44 `Sokal`→
    `So Kal`（p.10 印的才是 `Sokal`）；p.53 `one hand`→`one han'd`。下冊 p.19
    補齊隊伍名單的 AC／HP；p.22 補上畫面第一行的人名；p.6／p.7 補回原書排出的
    三種路徑標線條；另有八處原書錯字（`及之`、`快復`、`跟製`、`洋`、`遍重`、
    `施展特`、`止法術`、`營加`）被前一輪靜靜改掉，已照原印補回並加註。
  - 附錄 3 的欄名照書印作「價值（單位：黃金）」，但那一欄的數字是**重量**不是
    價目，已加譯註並由測試釘住。
- [x] **問密碼時把答案附在問句後面**（2026-09-04）。答案取自原版資料而不是
  寫死的表：`eclInputAnswer` 從 `INPUT STRING` 往後掃，兩種原版寫法都認得——
  `03h COMPARE <位址> <內嵌字面>`（ecl7/23 的 `NOKNOK`），以及
  `09h SAVE <字面> → 變數` 之後 `COMPARE <位址> <變數>`（ecl4/21）。
  同一個變數被寫過兩次就兩個候選都列，玩家當下需要哪一個靜態分不出。
  實拍：ecl7/23 顯示 `（NOKNOK）`；ecl4/21 的 (12,6)／(10,7)／(12,10)／(12,11)
  顯示 `（SAMOSUD／SHESTNI）`，(8,10)／(7,10) 顯示 `（LUX）`。
  **這是 remake 的擴充，原版沒有這個括號。**

- [x] 治具改成照著畫面上的提示打（2026-09-04）。先前它在索寇要塞硬送 `LUX`，
  而那四格要的是 `SAMOSUD`／`SHESTNI`——密碼提示上線後才看得出來。現在
  `answerHints` 從問句尾巴那個括號解出候選（那是 `eclInputAnswer` 由原版資料
  算的，玩家也看得到），治具照著打，多個候選就輪流試；解不出來才退回舊的
  `knownECLPasswords`。實測：那四格送 `SAMOSUD`／`SHESTNI`，(8,10)／(7,10)
  送 `LUX`，`TestSokalKeepOpensTheOtherBoatRoutes` 的兩個旗標條件仍成立。
- [x] 跨冊譯名取捨已定案，見 `docs/reference/manual/glossary.md` 的「定案譯名」：
  四條依序套用的原則加 14 組決定（Phlan＝菲蘭、Valjevo＝瓦傑渥、Thentia＝珊提亞、
  Magic-User＝魔法師等），並分開 Yarash（亞拉斯）與 Yulash（尤拉斯）、固定
  ROUND／TURN 的定義。轉錄正文維持原書用字不動，本表只約束 game pack 與 UI。
- [x] **怪物名全部定名並接上畫面**（2026-09-04）。先前只處理說明書那 43 種的
  「定名」，而**沒有翻譯管線**——`MONnCHA` 的名字直接印在遭遇與戰鬥畫面上，
  中文畫面會冒出 `SPECTRE ×2`。掃完八個 archive 得到 **103 種**名字（說明書那
  43 種之外還有職業等級變體、NPC 與專名），逐條定名後放進
  `internal/gametext/monsters.zh-TW.json`，由 `gametext.MonsterCatalogue` 載入，
  遭遇標籤與 NPC 入隊都改走它。

  來源分四類，逐條標 `basis`／`note`：說明書怪物一覽表（glossary 的最終譯名，
  含對齊 CoAB 那九條）、遊戲內文字已用過的譯法（30 條，例如 `SKULLCRUSHER`＝
  碎顱者、`TYRANITHRAXUS`＝泰倫斯拉克斯）、等級＋職業的規則（`4TH LVL FIGHTER`
  ＝四級戰士）、由已定名成分組出的複合詞（`ORC LEADER`＝半獸人首領）。
  **exact 46 條、strong inference 57 條，未定 0 條。**

  三個舊的待證項也解掉了：`MONnCHA` 證實 Spectre（`mon2/17`、`mon4/17`）、
  Wight（`mon4/20`）、Wraith（`mon4/21`）都是可遭遇的怪物，戰鬥畫面會顯示
  名字，不能停在待證。Spectre ＝幽魂由遊戲內文字證實；Wight ＝屍妖、
  Wraith ＝幽鬼採 AD&D 繁中通行譯名並標 `strong inference`。glossary 已同步。

  三則測試把關：每個原版名字都有譯名、表裡沒有對不到原版的多餘條目、每一條
  都寫得出來源。
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
  一句不漏。另加結局過場的 13 行（overlay-18 內嵌短字串，不在 ECL 盤點裡，
  spec 108），對照表共 1,744 條。每一批都通過「原文存在於盤點檔」的測試，專名依 glossary，說明書沒收的
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
- [x] **compiler、linker 與 overlay 邊界都閉合了**（2026-09-04，
  `docs/re/dos-toolchain-baseline.md`，`TestDOSToolchainFingerprint` 逐條重生）。
  - **進入點就是證據**：`0000:0006` 起是 **49 個連續遠呼叫**，其中 **38 個的
    位移正好是 `0020h`**（overlay stub 的 entry 0 ＝ 單元初始化），而
    `GAME.OVR` 正好 **38 顆 overlay**。那是 Turbo Pascal 產生的單元初始化鏈。
  - **RTL 指紋**：`"Runtime error \0" " at \0" ".\r\n\0"` 三個連著的 ASCIIZ
    （沒有逐項訊息，錯誤碼用數字印），前面接著用 DOS 功能 `06h` 印 nibble 的
    輔助函式；常駐 RTL 的 segment 是 `05BBh`。
  - **版本是 5.x 家族（強推論）**：TP 4.0 沒有 overlay 支援，overlay 單元是
    5.0（1988）加回來的，而本作是 1988 年的；3.x 是另一套機制。
    **不宣稱小版本**——這幾條位元組分不出 5.0 與 5.5。
  - overlay 邊界：38 顆／774 個進入點由 `cmd/pool-ovr-manifest` 從 ZIP 重生，
    stub segment 對照表在 spec 109。
  **還沒做**（另立）：逐顆 overlay 命名與全模組函式清冊。青色枷的 PC-98 符號表
  可以當對照，但**不是全域對應**——Pool 的 ECL 直譯器在 `overlay-03`，
  青色枷的 `INTERPET` 在 `overlay-02`，要一顆一顆用本作自己的證據認。
- [x] 建立 DOSBox 正常啟動 oracle 與未縮放標題／主選單截圖。
  驗收：Docker/Xvfb 有界重播，輸入序列、畫面與 metadata 齊全。
- [x] 完成 `TITLE.DAX` typed consumer 與 PNG／總覽圖匯出；block 1 放大 2× 後
  與原版標題逐像素 AE=`0`，規格見 `docs/spec/001-dos-title-picture.md`。
- [x] **Pool ECL record format 閉合了**（2026-09-04）：**29／29 個 block 全部
  靜態走得完**，16,031 條可達指令（先前 26／29、14,724 條）。三筆卡住的是
  兩種原因，都不是「真的缺 opcode」：
  1. **控制流 fallthrough**（`ECL5/7`、`ECL7/22`）：走訪器走到 `20h NEWECL`
     之後**繼續往下讀**，而 NEWECL 換掉整個程式碼段（`Machine.SwitchBlock`），
     後面放的是資料。把 NEWECL 當成終止點就解決了；被 `IF` 守衛的那一條
     仍然走得到，因為 `guarded` 早就把 `Next` 排進佇列。
  2. **運算元個數**（`ECL7/17`）：`34h ECL CLOCK` 的序言取**一個**運算元，
     共用 engine 那張二手表寫兩個。差一個運算元，`9D37h` 之後的
     `GOSUB 9DA7h` 就被吃掉，PC 停在指令中間（spec 093 已訂正）。
     **CoAB 的 `34h` 確實是兩個**，所以這是各作品直譯器 build 的性質——
     解法是讓作品傳自己的指令表（engine 的
     `TraceGraphAtBaseWithCommands`／`Machine.SetCommands`），
     Pool 這一側是 `gamepack.PoolCommandTable()`，只覆蓋 `34h` 這一條。
  `TestEveryECLBlockTracesEndToEnd` 與
  `TestPoolCommandTableMatchesTheMeasuredChain` 釘住這兩件事。
  ⚠ engine 那兩個改動還沒 push（見「平台驗收的 workflow」那一項的授權說明）。

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
  核心相容回歸均已通過。
  **City Hall 那條迴圈已經閉合**（見下面「貧民窟那條委任的完整迴圈通了」：
  真的打 25 場、走回城區、拿到報酬並收下戰利品）。**旅店也接上了**
  （2026-09-04，spec 081）：`PROGRAM 9` 全遊戲只有旅店那一處在推，
  而它的派發鏈第一支是 overlay-15（記憶法術），所以那是「選法術＋休息」
  而不是隊伍管理——先前玩家付了一枚白金什麼都沒得到。
  剩下的服務規則：紮營的時間與被打斷（`Stop Resting?`）、賣東西、
  神殿的其餘服務。
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
  25 格（護邪、防護、隱形、解鎖、致盲、詛咒…）。加上逐支讀的三十四支處理常式
  （37 個編號），六十七格裡接得出來的有 **62 格**
  （`TestImplementedSpellCount` 釘住這個數字）。要比對整個版型不能只看「有沒有呼叫
  `08BCh`」——會算傷害的那幾支也呼叫它，只看呼叫會把傷害弄不見。
  **`+9Fh` 與 `+6Ch` 解出來了**（2026-09-03）：前者是生物種類（0 人類、
  1 類人、2 巨人、4 不死、0Eh 蛇與蠍…），後者低位是體型、位元 7 標大塊頭。
  四個互相獨立的比較點指同一個讀法（死靈術只認 0、迷蛇術只認 0Eh、
  overlay-12 對 4 設旗標、魅惑與定身要求兩個都不大於 1）。魅惑人類、
  定身術兩個編號與迷蛇術因此接上了；被迷住之後的行為仍是近似（只讓它不再
  行動，沒有倒戈），因為那一段在還沒讀完的敵方 AI 裡。
  **`0100h:007Ah` 也解出來了**：它是 overlay-24 entry 18，把力量往上調
  （只升不降），記錄 `+10h` 是力量、`+16h` 是特殊力量百分位。由此抓到一個
  **實作錯誤**：變大術寫進 `DS:47A6h` 的 `12h` 是十進位的 18（要設成的力量），
  不是效果碼；真正的效果碼是參數表與 `1331h` 都指到的 `0Ch`。縮小術與編號 59
  因此也接上了——縮小術過了三關之後**原版就只印一句話**，不掛效果也不解掉
  `0Ch`，照碼接。
  編號 60（`2F02h`）也接上了：`287Ch` 是「打一格」（豁免類別 4、規則 2
  減半、傷害種類 `0Ch`，走 overlay-24 entry 19），`2919h` 是「由施法者穿過
  目標拉一條射線再逐格打」；傷害 `Roll(1, 6) + 20`，處理常式寫死的豁免值
  與參數表 `+8`／`+9` 相同。
  **力量那一組三支真的會改到角色了**：`1158h` 的「只往上調」寫成
  `RaiseStrength`、`1F16h` 的骰子（看目標職業；超過 18 只有戰士拿得到
  百分位，每超一點加 10、上限 100）寫成 `StrengthSpellResult`，
  變大術、力量術與編號 59 共用同一條套用路徑，改的是 `Abilities[0]` 與
  既有的 `ExceptionalStrength` 欄位，不必動存檔 schema。
  還缺的五支各自卡在一個 remake 還沒有的子系統，不是「再接一支就好」
  （2026-09-03 讀完前兩支、盤點清楚）：

  | 法術 | 原版做什麼 | 缺什麼 |
  |---|---|---|
  | 迷霧術（34，`1AF6h`）| 在盤面生一個五格的雲團物件，掛在 `DS:6CAFh` 的串列上，地形 `1Eh` 會與別人的雲合併 | 留在盤面上、每回合結算、彼此會合併的物件 |
  | 死靈術（36，`2043h`）| 只對 `+9Fh == 0` 的人類屍體 | 戰鬥中新增戰鬥員 |
  | 解除魔法（41／46，`2356h`）| 沿目標 `+7Fh` 的效果節點串列逐個擲：成功率 50 為基準，施法者高一級 +5、低一級 −2，節點 `+3` 低四位是下效果者的等級、`0FFh` 解不掉 | 戰鬥中的效果節點串列（現在只有 `Asleep`／`HeldRounds`／`Charmed` 旗標）|
  | 恢復術（56，`2C01h`）| 規則已由 spec 097 讀出 | 等級吸取的追蹤 |

  規則本身都寫進 spec 098 了，接的時候不必再反組譯一次。
  （共用機制已解出並寫成 spec 098：擲骰是 `Roll(顆數, 面數)`、
  施法者等級由 overlay-25 `26F8h` 取而且戰術地圖外一律當 6、處理常式把傷害
  用四個覆寫參數傳給 `08BCh`。**卡在一個未解的點**：Magic Missile 的發數是
  `等級 ÷ 2`，而說明書寫的是「每昇兩級多一發，第 3 或 4 級 2 發」＝
  `(等級+1) ÷ 2`；第 1 級照碼算出來是 0 發 0 傷害。照碼接了（一手資料贏
  二手），測試把那個 0 釘住，驗出不同結果會是紅的。
  **法術書接上了**（2026-09-04，spec 110）：「這個人會不會這一條」是記錄
  `+32h + 1-based 編號` 的一格一 byte 陣列，1 是會。位置的證據是二十個預設
  人物檔零反例（施法者只放自己那一類、法師 4 級只有第 1..2 級）加上四處
  寫入碼的 `add di, ax` 之後 `es:[di+32h]`；`+32h` 本身是最大生命值，所以
  直接搜 `+33h` 會落空。寫入規則讀出兩條：建角時牧師會第 1 級全部神術、
  法師會寫死的四條（偵測魔法、閱讀魔法、護盾術、催眠術），昇級時 overlay-23
  `0132h` 把牧師「那一級有格子」的神術全部補上。記憶畫面（overlay-15
  `0C33h`）書上沒有的連列都不列，remake 照接——一級法師記不起火球術了。
  **還缺**：法師昇級挑法術的 `00C9:005Ch` 沒讀出來（remake 先讓玩家在法術
  一覽按 `L` 挑，介面是 remake 的）、卷軸抄寫的成功率、以及預設法師的條數
  比「四條加每級一條」多出來的差額從哪來。
  **決定性的實驗還沒做**：原版拿第 1 級法師施魔法飛彈看傷害是不是 0）、
  原版紮營畫面挑時間那一段、
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
  要往下推得靠**有目的地的走法**，那已經做了：
  `TestWorldTourReachesTheAreasBehindTheHarbour` 把碼頭的目的地直接寫進
  `DS:4AC4h`，34 趟不同種子量到 **12 張地圖、12 個 ECL 區塊**
  （2026-09-04）。它證明的是「那些區域的腳本在完整的前端底下跑得動」，
  仍然不是「玩家走得到」。
  `TestRandomWalkReachesKnownContentWithoutFailing` 把隨機走路那個下限釘住，
  走得到的地方變少就會紅。
  **2026-09-05 重量了四種走法的聯集**：隨機走路 4 個 `[0 8 11 20]`、
  有目的地的探索 5 個 `[0 8 11 20 21]`、索寇要塞那一條 7 個
  `[0 8 11 20 21 26 27]`、世界巡迴 12 個
  `[0 8 11 14 18 20 21 24 25 26 27 29]`；聯集仍是 **12 個**，所以「29 個有
  文字的區塊裡還有 17 個沒被任何一種走法碰過」沒有變。
  變的是**證據等級**：26 與 27（野外／樞紐）現在是探索器**自己走到的**，
  不再只有世界巡迴用 `DS:4AC4h` 覆寫目的地才碰得到。那兩塊因此從
  「腳本跑得動」升成「玩家走得到」。
  **還缺**：29 個有文字的區塊裡還有 17 個沒被任何一種走法碰過。
- [x] **貧民窟那條委任的完整迴圈通了**（2026-09-03）：真的打 25 場、
  走回城區、進市政廳、走到職員面前拿到報酬。
  `TestTwentyFiveRealSlumsWinsEarnTheCityHallReward` 逐場檢查
  `DS:4ABBh` 一場加一、第 25 場精確變成 `FEh`，接著用原版的路走出貧民窟
  （入口 0 在 `6DD5h` 非零時依朝向挑鄰居，面向東 → archive 3 → `NEWECL 0`）、
  從 `(3,4)` 往東進市政廳（區塊換成 8）、走到 `(5,5)`，最後拿到
  `Gold 250 / Platinum 50 / Jewelry 1`，`4AC1h` 由 0 變 1。
  **金額有正對照**：ECL3/8 的四張獎賞表（`B5EDh`／`B604h`／`B61Bh`／`B632h`）
  在槽 21 的原始位元組是 `FA`、`32`、`00`、`01` ＝ 250、50、0、1，由
  `9F28h TREASURE` 的第 4..7 欄（金、白金、寶石、首飾）送出——四項逐一對上
  顯示的字，所以那不是碰巧含有 Gold 的句子。
  **負對照**`TestCityHallPaysNothingBeforeTheCommissionIsDone`：一場都不打
  去交差，文字是空的、`4AC1h` 不動——沒有它，「拿到 Gold」證不了因果。
  **獎賞也收下來了**，整條鏈跑到底：ECL3/8 的通知走完之後是

  ```
  9ec2  查四張獎賞表（B5EDh／B604h／B61Bh／B632h）依槽索引取四個數
  9eea  PRINT "HERE IS YOUR REWARD.'"
  9efe  HORIZONTAL MENU          ← 玩家要選一個
  9f28  TREASURE 0,0,0,四個數
  9f3e  COMBAT                   ← 走戰後戰利品服務（spec 036）
  9f5a  SAVE TABLE FF → 4AA6[槽]  ← 這裡才把槽清成 FFh
  ```

  測試在獎賞服務開起來時先驗待分的錢是 `[0 0 0 250 50 0 1]`（四張表的原值），
  再挑 Share 把錢分給隊伍、挑 Exit 讓腳本往下跑：`4ABBh` 變成 `FFh`、
  角色錢包收到金 250 白金 50 首飾 1。**這條委任從接到交差完整走得完。**
- [x] **破關那一場打得贏，旗標也立得起來**（2026-09-03）。`ECL5/7` 的
  `A7DCh` 起是最後一戰：`LOAD MONSTER 42h` ＝ `mon5/66 TYRANITHRAXUS`
  （HP 80、AC 0、THAC0 10、體型 `84h` 佔 2×2），打贏之後 `A815h` 把
  `4ABAh` 設成 `FEh`——那正是市政廳槽 20 的
  `CONGRATULATIONS! YOUR QUEST IS OVER!`。
  `TestDefeatingTyranthraxusSetsTheVictoryFlag` 釘住這一條，並驗結局腳本
  把隊伍送回 `ecl3/0`（`6E12 = 3` ＋ `NEWECL 0`）。
  ECL5/7 是三個靜態展開解不開的區塊之一，位址是用線性掃描讀出來的。

  **順帶修掉一個擋在結局前面的缺口**：`finishCombat` 以前只把戰後腳本的
  文字套上去，不分派邊界，所以戰鬥後面接的 `PROGRAM`、`TREASURE`、換區塊
  一條都不會跑——結局就是卡在 `A82Ah PROGRAM 08`。改成走跟走進一格時
  同一條 `consumeInitialSearch` 之後，結局文字與回菲蘭的 `NEWECL 0` 都跑了。

- [ ] **結局過場的圖還沒畫**（[spec 108](docs/spec/108-ending-cutscene.md)）。
  `PROGRAM 8` 是 overlay-18 entry 1（`02A1h`）。**台詞與中譯都接上了**：
  三頁（3／6／4 行，分頁由列號回到第一列算出來），一頁一個 ENTER，翻完讓
  ECL 往下跑；十三行進了對照表，`TestEndingCutscenePagesAreTranslated` 擋
  「翻了但沒接上」。
  **缺的是圖**：`FINAL5.DAX` 的區塊 1／3／4／5／6 解得出 120×120、120×120、
  56×32、16×48、16×48。兩支常式已由 [spec 109](docs/spec/109-overlay-stub-segments.md)
  查出來——`018Eh` 是 overlay-36，載入 entry 5（`011Bh`）、繪製 entry 9
  （`0D9Fh`）。**繪製的引數版面已經讀出來**（2026-09-04，spec 108）：
  `(X, Y, ?, 旗標, 圖片遠指標, 目標緩衝遠指標)`，X 會與圖片記錄 `+2`（寬）
  比大小，旗標的位元 0／2 各控一件事；**五張圖全部是 (0,0)、旗標 0**，
  所以位置不在引數裡。`[4937h]+67Ch` 是逐格推進的計數，決定後三張畫不畫。
  **`DS:4980h` 靜態讀不到**（2026-09-04）：三十八顆 overlay 加 `START.EXE`
  裡沒有任何一處按名字寫它（正對照：同一組樣式在 `4670h` 找得到寫入），
  因為它在 **BSS**——`DS` 換算成 `START.EXE` 檔案位移要加 30640
  （由 spec 068 的法術名稱表釘住），而 `4980h` 換出來是 49,456，
  超出 47,936 bytes 的檔案。那個緩衝是執行時配置的。
  **往下只有兩條路**：拿 DOSBox 當 oracle 抓執行時的值，或直接抓原版結局
  畫面量位置。在那之前不接圖。
  **先前 spec 081 寫「值 8 全遊戲沒有呼叫點」——那是用走得到的碼掃出來的，
  而唯一的呼叫點就在掃不進去的那個區塊裡。** ECL5/7 現在靜態走得完了
  （2026-09-04），`A82Ah PROGRAM 8` 直接掃得到，不必再靠線性掃描。
- [x] 從標題以正常按鍵完成建隊、進圖、事件、戰鬥、存檔與讀檔抽樣。
  `TestNormalKeysReachTheFirstDungeonStep`：只用按鍵從標題走到建角、
  加入隊伍、Begin、推完開場與 34 步導覽，在地圖上走出一步，F10 存檔後
  重開一份按 L 讀回來並繼續走。戰鬥那一段見下面兩條。

  戰鬥中與戰鬥後的存讀檔抽樣由 `TestSavingIsRefusedDuringCombatAndWorksAfterIt`
  補上（2026-09-04），順帶修掉兩個缺陷：

  1. **戰鬥中存得了檔**。`stateForSave` 的閘只擋對話與服務，沒擋戰鬥，而
     `tactical` 與 `combatMonsters` 都不在 `Campaign` 裡——存了讀回來怪物
     整批消失，等於免費脫離戰鬥，而且 ECL session 停在「戰鬥進行中」那一點。
     原版的 SAVE 在營地，戰鬥畫面沒有那個入口，所以照樣擋掉。
     這一條要排在對話那一條**前面**：戰術地圖開著時 `cellEventPending` 仍是
     true（戰鬥掛在格子事件底下），排後面玩家會看到「先把對話讀完」，
     而畫面上根本沒有對話。
  2. **存檔失敗會把視窗收掉**。`Update` 回傳非 `Termination` 的 error 時
     ebiten 直接結束——玩家在對話中按一下 F10 遊戲就沒了。改成把理由寫進
     狀態列。
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
- [x] **裝備走過玩家自己的路了**。`TestNormalKeysBuyAndEquipFromTheWeaponShop`
  只用按鍵走完「建角拿到金幣 → 從開場結束的 (0,4) 走到武具店 (8,11) → 買盾
  15 → 按 I 裝上」，最後量 AC 由 10 變 9，所以「裝上了」不只是翻了一個旗標。
  路線是照原始資料規劃的：城區每格的事件索引就是地形碼低七位（spec 102），
  武具店是索引 22 的五格。規劃時**不走環繞**——開場的起點在 (0,4)，往西一步
  會繞到 (15,4) 直接離開菲蘭，用原版的環繞規則找路會走出城。
  買來的東西也真的會影響戰鬥：`enterCombatStaging` 在建角值之上套
  `memberDefenceStats`（AC 與腳程）與 `readiedWeapon`（THAC0 與傷害）。
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
  這條當時擋的是「靠不過來」；選路本身已經在 2026-09-03 換成原版的五方向
  規則（見上面那一條與 spec 096），BFS 的步數表現在只剩繞路備案在用。
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
- [x] **傷害骰那兩格是兩種攻擊形態，不是備位**（2026-09-04，spec 051 更新）。
  原版不挑一格：攻擊區段由形態 2 倒數到形態 1，每一形態揮幾下由
  `AttacksThisPhase(編碼, 相位)` 給。巨魔的 `04 02` 加 `1d4+4`／`2d6` 正是
  AD&D 一版的爪／爪／咬，先前只打得出一次爪。
  **順手抓到一個更大的問題**：傷害骰的來源欄位是 `+0A2h/+0A4h/+0A6h`，
  `+114h..+11Ah` 是原版排怪時抄過去的**執行期副本**（overlay-25 `0DF4h`），
  而 `MONnCHA.DAX` 的樣板記錄裡那一段沒有初始化——172 筆有 53 筆殘留著
  別的東西，讀出來是 QUICKLINGS 65d68+67、THRI-KREEN 1d4+102。
  正負對照分得很開：用來源欄位算沒有一筆不合理，用執行期那一段算 48 處不合理。
  已改讀來源欄位，`TestMonsterDamageDiceAreAllPlausible` 同時量兩邊。
  **還缺**：玩家角色的攻擊次數（原版由職業等級表給），現在一律一回合一次。
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
- [x] **探索器走得到野外了**（2026-09-04）。
  `TestSokalKeepOpensTheOtherBoatRoutes` 從 **3 張地圖／5 個 ECL block** 變成
  **7 張／7 個**，其中 GEO7/26 與 GEO8/27 就是野外圖（spec 105 的 block 25／26／27）。
  下限已經釘進測試，少於這個數會紅。

  原本擋著的是兩件事，兩件都量得出來：

  1. **主線鎖是看位置的**。`mustSettleSokalGhost` 只在 GEO3/0 與 GEO4/21 回
     true，同一組記憶體值走進貧民窟就回 false——鎖一熄 `heldBack` 整批放回去，
     回城區又亮。加上「鎖的評估只在手上沒有計畫時做」，隊伍就帶著鎖亮之前
     規劃好的路走出城區，回城路上踩到競技場 (7,2)，`9CACh` 把 `4A01` 寫回 1
     （spec 102），港務長再也不開口。十個種子實測，`4A01 255→1` 十次全在 (7,2)。

     修法三處：鎖改成只看記憶體；評估提到每一步，但只在由暗轉亮那一刻丟一次
     計畫（每一步都丟試過，更差——走到的 block 從 5 掉到 4）；鎖住時城區的
     事件格整批不去，只留港務長門口 (11,1)／(11,2)。之後十個種子的
     `4A01 255→1` 十次全在 **(11,1)——港務長本人**，「找港務長」每個種子
     **只呼叫一次**（先前那次失敗的嘗試是 112 次）。

  2. **拿到票之後沒有人走去碼頭**。碼頭 (15,1) 是換圖點，第一次用完就進了
     `avoid`，規劃器再也不挑它。補上「`4A01 == 1` 且 `4AA7 == 254` 時主動
     規劃一條繞開其他事件格的路走到 (15,1)」之後，五個目的地的航線選單就開
     （`[SOKAL EAST WEST BAY NONE]`），隊伍搭船出去落到野外。

  `TestDirectedExplorationReachesMaps` 仍是 3 張／5 個，那是**預期的**：
  它不推主線（`flags == nil`），`4AA7` 永遠到不了 254，碼頭那一步不會觸發。

- [x] **野外／樞紐地圖（ECL block 25、26、27）**：座標系統、三張圖怎麼接、
  四張地點表、每一步的移動與地點派工都接上了（spec 105）。走得通的證據是
  `TestAWildernessStepMovesBothPositions`（一步同時推 GEO 位置與野外座標，
  牆擋住時兩個都不動）與 `TestAWildernessLocationDispatchesItsScript`
  （踏進圖 27 的 (9,29) 會問「要不要搭船回文明區」，答應就換到 ECL block 20）。
  三張圖的 `NEWECL` 目標合起來是 16 個區塊，所以野外確實是走到其餘區域的入口。
  邊界（北／西／東三張八支表，斜向也算）與 20 支地點腳本的入口與第一句
  也都讀完了，46 個格子共用那 20 支。剩下的只有畫面細節：GEO block 6 的
  112 格與野外 14×26 之間怎麼對應（spec 105 的 OPEN）。
- [x] `WALLDEF selector 3 is not present`：**archive 挑錯了**。`LOAD PIECES`
  是腳本要的資源，要用 ECL 的 archive，不是它載進來的 GEO 的——野外那幾張圖
  就不同號（ecl7/26 載的是 GEO5 的區塊，而 `WALLDEF5.DAX` 只有 1 與 24 兩塊）。
- [x] `LOAD PIECES` 的 selector 解法閉合：**selector 不是編號時，退到比它小的
  最近一塊，取第 (selector − 編號) 筆記錄**。全遊戲 33 處 `LOAD PIECES` 只有
  五處落在編號之外（ecl4/10 的 22、ecl5/3・5/4・5/6 的 25），每一處都正好落在
  前一塊的記錄數之內——`WALLDEF4` 的 21 與 `WALLDEF5` 的 24 都是兩筆連著的
  記錄。這是資料全面對得起來的證據，不是單點推測。
- [x] **`Pool DAMAGE saving throw category 10 is outside 0..4`**：讀反了。
  呼叫端（overlay-03 `2BCEh..2BD6h`）先推 `運算元5 & 7` 再推 `旗標 & 1Fh`，
  而豁免常式（overlay-24 entry 7、`0D61h`）把第一個參數加進 d20、第二個拿去
  索引 `record[+6Dh + 類別]`——所以**低五位是修正、運算元 5 才是類別**。
  10 是 +10 的修正，不是第 10 個類別。spec 084 已更正，
  `TestDamageRequestFlags` 用 `旗標 0Ah／運算元5 0Ch` 釘住這一組。

- [x] **`NEWECL FF` 當成「不換區」**（2026-09-03 修在共用 engine，見
  [spec 107](docs/spec/107-newecl-ff-sentinel.md)）。`eclvm` 加了
  `NoBlockChange` 常數，`switchTo` 的呼叫端碰到它就不換、不回報 transition，
  讓機器從 `NEWECL` 的下一條跑下去。世界巡迴的
  `ECL session target block 0xFF is unavailable` 歸零，**GEO1/31 因此走得進去了**。

  三件事釘住了語意：八個封存檔的區塊編號最大是 29，**沒有 255**；全遊戲五張
  以變數當 `NEWECL` 目標的表裡**只有 `ecl1/24` 那張有 `FF`**，而且它跟同一列
  的 `LOAD FILES` 欄成對（`FF` 在那一欄已證實是「不載地圖」，spec 043）；
  拿原始 GEO 量兩張圖的邊界，走得到的五格裡有四格被腳本明文處理，剩下索引 6
  （GEO31 往南）兩欄都是 `FF`，沒有守衛——所以 `FF` 自己就得是無害的。

- [x] **GEO1/31 的戰術地圖卡住**（2026-09-03 修，spec 062 契約 7）。
  根因是**離場的 combatant 每回合又拿到先攻**：死亡當場兩個入口都把體型與分數
  歸零了，但 `startRound` 對名冊上每一格重擲，死者於是復活成行動者。接下來
  整條連鎖都是無聲的——體型 0 的 mover 在目的格探測裡取不到佔格偏移，出界與
  地形檢查那一整段被略過（那是原版行為，spec 058 契約 6），它走出盤面，
  `RequiredFacing` 對出界座標九個候選全不成立而回錯誤，錯誤被 `Update` 收進
  狀態列，每個影格重試一次。
  修的是 `startRound`：離場者分數一律 0，骰子照擲、結果丟掉（`+3` 的每回合
  來源還沒讀到，沒有證據前不動亂數流）。`RequiredFacing` 改成出界當場失敗，
  順帶訂正它「DirectionAny 恆真，搜尋一定會停」那句——界限檢查對 `DirectionAny`
  一樣生效，出界時九個候選一個都不成立。
  **不是敵方 AI 改版引入的**：`state.Scores[index] = score` 只由 `e6fe452`
  （2026-09-02 的回合迴圈）引入，之後沒有任何 commit 改過，三個敵方 AI commit
  也沒碰過 `startRound`／`selectActor`。要走得進這一區、而且架打到二十幾回合
  有人死，才看得到。
  驗收：`TestStartRoundKeepsTheFallenOutOfInitiative`（帶正對照，修正前紅、
  修正後綠）、`TestRequiredFacingRejectsCellsOffTheBoard`，世界巡迴 22 趟的
  硬失敗從 3 筆歸零，整包測試綠。

- [x] **GEO7/23 (1,1) 的格子選單卡住**（2026-09-04 查完並修好）。
  **不是遊戲的缺陷，是治具永遠答 NO。** ecl7/23 的密碼門控制流讀完了：

      A4A1 INPUT STRING #6 → 6E79h   輸入六個字
      A4B9 PRINT 6E79h               把玩家打的字印出來
      A4C1 GOSUB 9982h               [YES NO] 確認框
      A4C5 IF <> → A4C6 GOTO A3FAh   答 NO：跳回去重新輸入
      A4CA COMPARE 6E79h #5
      A4D5 IF = → A4D6 GOTO 9BE3h    答對：SPRITE OFF、CALL 2C90h、門開
      A4DA…A559 DAMAGE #224 #1 #1 #200 #0
      A564 OR 4A51h #64 4A51h        答錯：扣血、設旗標
      A56D EXIT

  入口 `A3E4 AND 4A51h #64` 檢查同一個 bit，設了就 `A3F9 EXIT`。所以原版
  **三條路都會結束**：答對開門、答錯把這一格永久關掉、答 NO 才回頭重打。
  無限迴圈只有一種走法會發生——**每一次都答 NO**，而那正是治具的行為。

  根因在治具：密碼輸入與選單**共用同一個 `menuTurn[key]`**，兩者交替各
  `++` 一次，於是選單永遠落在奇數（`% 2 == 1` ＝ NO）、密碼永遠落在偶數
  （`% 4` 只取得到 `SAMOSUD` 與 `SHESTNI`），`NOKNOK` 這個正解永遠輪不到。
  兩個週期咬死，看起來就像遊戲卡住。

  修了兩件事：剛送出密碼的格子下一個選單一律選第 0 項（確認框答 NO 只會
  跳回去重打）；已知密碼的門直接說對的字（`knownECLPasswords`），因為
  原版只給一次機會，輪流試等於把它用掉。修完那一趟的迴圈分布從
  「輸入字串 2668、格子選單 4045」變成「輸入字串 2、格子選單 49」，
  並走到門後的 `[MOVE QUICKLY AWAY DESTROY THE EQUIPMENT]`。
  回歸測試 `TestCellMenuDoesNotStallAtTheGEO7PasswordDoor` 單獨跑那一趟；
  世界巡迴 22 趟仍是零硬失敗。

  提示只剩 `?` 的那一項也查完了，成因與 `33h` 無關：**`11h PRINT` 被當成
  取代**。原版的確認框是 `A4A7 PRINTCLEAR "DO YOU REALLY MEAN"` ＋
  `A4B9 PRINT <玩家打的字>` ＋ `A4BD PRINT "?"`，三段拼成一句，中間沒有
  等待玩家的指令；把 `11h` 也當成取代就只剩最後那個問號。市政廳
  `AC22 PRINTCLEAR …YOU NOTE` 接 `AC9B PRINT PROCLAMATIONS LXIV…` 是同一個
  模式，先前被拆成兩頁，第一頁以 `YOU NOTE` 結尾。兩處都改由
  `joinPrintedText` 接起來，契約寫進 spec 082。

- [ ] **平台驗收的 workflow 進不了共用 engine**（2026-09-03 實跑
  `gh workflow run platform-smoke.yml` 量到）。`build (macos-14)` 與
  `build (windows-latest)` 都掛在 **`check out the shared engine`** 那一步：
  `actions/checkout` 去抓 `wicanr2/golden-box-remake-engine`，而
  **兩個 repo 都是 private、這個 repo 一個 secret 都沒有**，預設的
  `GITHUB_TOKEN` 只有本 repo 的權限，跨 repo 抓 private 一定失敗。
  2026-09-04 使用者定案 **engine 維持 private**，所以只剩加 PAT 這條路。
  解除條件：在本 repo 建一個有 engine 讀取權的 PAT secret，並把 workflow 的
  `actions/checkout` 換成用它。在那之前這個 workflow 驗不到任何東西——
  Linux 與 Wine 的驗收改走本地 Docker（`tools/linux-release-smoke.sh`、
  `tools/windows-release-smoke.sh`），已經在做，不受影響。

  動共用 engine 之前要先取得使用者同意——那是另一個 repo，本專案的 push 授權
  不涵蓋它。該 repo 的 repo-local `user.email` 已經是 `wicanr2@gmail.com`
  （全域仍是公司位址，靠 repo-local 蓋掉），進去工作時照例再複查一次。
- [x] **敵方回合換成原版的接近規則**（2026-09-03，spec 096）。overlay-09 整條
  讀完：**entry 5（`0B3Ch`）是接近迴圈**、**entry 13（`07E8h`）是走一步**。
  原版的順序是「先問武器搆得到誰（射程取自武器型別的 `+0Ch` 減一），搆得到
  就打，搆不到才走」；走哪一步的判準是**進不進得去**——照戰術模式那一列的
  五個相對方向依序試，`方向 = (基準方向 + DS:[02ACh + 模式 × 5 + 步]) mod 8`，
  **步從 1 數到 5**，不比較距離。基準方向是目標的方位（overlay-13 `261Bh`，
  spec 098 早就解過、已經實作成 `combat.RequiredFacing`）。
  卡住的處置也讀出來了：`DS:439Eh` 記上一步的方向、`439Fh` 記卡住次數，
  走回反方向第一次照走、第二次忘掉目標、第三次這一輪不再動。
  模式的每回合處置精確化：1..4 有四分之三沿用，重擲時 `1d8` 只用來分支，
  擲到 8 才取 `1d2+4`（`gamepack.RollTacticMode`）。
  順手把 overlay-31 `0579h` 的八個分支逐條重讀一次，與既有的
  `FacingArcContains` 逐條相符——等於替它補了一次獨立對照。
  三處仍是近似，spec 096 逐項列了：挑哪個目標、五個方向多一道「要離目標
  更近」的閘門、全不合用時的繞路備案。後兩項擋的是同一個坑（貼牆的怪物
  挑到 ±2 往旁邊走、下一步又走回來）；加上之後 `TestPassiveCombatTerminates`
  從 2.49 秒打不完變成 0.54 秒收尾，擁擠盤面那一條從 24.9 秒降到 2.5 秒。

- [ ] **還沒讀的敵方 AI**（spec 096）。
  **2026-09-05 讀掉一支**：`010Ah:00C0h`（overlay-25 entry 32，code offset
  `246Dh`）就是候選名單的建表與篩選——`0912h`（overlay-31 entry 6，
  spec 057 的 `TraceMovement` 成本模型）以**射程當預算、方向 `FFh` 不限**
  搜一遍，填出 `DS:6676h` 那張每筆 3 bytes 的中繼表（筆數在 `DS:6678h`），
  再篩掉 `+10Eh` 不等於對立陣營值的、原地壓實，最後把每筆的索引抄成
  `DS:6CD7h` 的 1-based byte 陣列。這與 `37B8h` 的讀法逐格吻合
  （`DS:[6CD7h + 號碼]` 為 0 ＝ 已劃掉，非 0 拿去查 `DS:6517h + 索引 × 4`）。
  順帶多一個用例：**移動、射程、範圍法術與 AI 找目標共用 `0912h` 一個成本
  模型**。
  推給 `0912h` 的前五個參數也定下來了：三支 overlay-32 取值（entry 15／16／17）
  先用 entry 18 把記錄換成 combatant 索引，再讀 `DS:5E85h` 那張每筆 4 bytes
  的戰場位置表（`+0` X、`+1` Y、`+3` 體型類別，spec 056 早就讀過），
  所以是「我在哪、走得動多遠、不限方向、我多大」，最後推戰術地圖的遠指標。
  中繼表就是 spec 056 的鄰近查詢結果表（`+0` 索引、`+1` 成本、`+2` 朝向），
  壓實時三個 byte 一起搬，只是 `37B8h` 只用得到索引。
  **2026-09-05 再讀掉一支**：entry 2（`0203h`）**不是施法，是轉變不死生物**
  （[spec 111](docs/spec/111-turn-undead.md)）。三條互相獨立的證據：程式段裡
  的字串 `turns undead...`／`is turned`／`Is destroyed`、牧師等級 1..8 各自
  一列而 9..13 併一列、14 以上再併一列的折疊法，以及 `DS:45Bh` 那張
  10 欄 × 10 列的有號位元組矩陣本身（正數＝ 1d20 門檻，0 或負數＝自動摧毀，
  99 ＝ 無法轉變）。目標由 overlay-13 entry 13（`1352h`）在 `+76h` 小於 13
  的不死生物裡挑最小的一隻，overlay-13 entry 12（`116Ah`）擲骰執行。
  矩陣已重生成 `gamepack.ReadDOSTurnUndeadTable`，100 格全釘住，並與手冊
  附錄五的八個怪物欄位對上（`TestUndeadAppendixMatchesTheTurnTable`）。
  **玩家側走的是同一支**：overlay-08 的戰術指令迴圈在 `0427h` 比到 `T`
  之後直接 `call 0096h:005Ch`（＝ entry 12），再交給 overlay-25 entry 34
  收掉一個行動；沒有另一條玩家專用的實作。全 36 顆 overlay 掃過
  `9A xx xx 96 00` 這個形狀，指向 entry 12 的只有 overlay-08 `0431h` 與
  overlay-09 `0241h` 兩處，指向 entry 13 的只有 overlay-09 `022Eh` 一處。
  **配額那一段也想通了**：`12EBh` 的配額與 `12F4h` 的隻數各減各的，所以
  `[bp-3] = 6` 是一個**下限**——1d12 擲小時，只要接下來每一隻的門檻都是負數，
  額度就一直被補回來，直到配額用完。自動摧毀的情況下至少處理六隻。
  兩處的分界差一格：`1280h` 用 `jle`（門檻 0 算摧毀），`1303h` 用 `jge`
  （門檻 0 補不回額度），只有負數兩邊都成立。
  整條迴圈已實作成 `gamepack.ResolveTurnUndead` 與 `SelectTurnUndeadTarget`，
  七則測試釘住共用一擲、失敗即收工、自動摧毀、配額下限與不重複挑同一隻。
  另有一則負對照 `TestUndeadTurnColumnStaysInsideTheTable`：八個怪物檔 172 筆
  記錄裡 16 隻不死生物，最大欄位剛好是 10——entry 13 的上界 13 讀得到表外，
  但原版沒有那種目標，所以那條路走不到。**spec 111 已無開放項**。
  **2026-09-05 第三支：`02E2h` 那 2051 bytes 也讀完了**
  （[spec 112](docs/spec/112-effect-code-dispatch.md)）。它是一個 20 路的
  `case`，每一路就是一串效果代碼，逐一交給 `014Dh` 問「這個效果影響得到這筆
  記錄嗎」——2051 bytes 幾乎全是 140 次相同形狀的呼叫，沒有任何判斷。
  `014Dh` 有兩條命中路徑：記錄自己身上帶著這個效果，或者站在別人的作用範圍
  裡（只對 `CS:012Dh` 集合裡的 `15h`／`2Dh`／`2Eh`／`31h` 成立，`31h` 半徑 6、
  其餘 1）。命中就把代碼送進 overlay-24 entry 1，由 `DS:6786h` 那張以代碼為
  索引的遠指標表交給該代碼自己的處理常式。
  **`DS:677Ch` 也定位了**：全 36 顆 overlay 加 `START.EXE` 掃過四種存取形狀，
  寫它的是 overlay-12 的 entry 25（`0946h`）、35（`0C52h`）、66（`173Ah`）
  與 120（`2ECAh`），`1087h` 只負責清成 0 再讀。所以「不能打」是效果處理
  常式否決的。
  **正對照**：群組 7 是 `33h 34h 35h 1Fh`，與 `DS:2880h..2883h` 那四個反應
  攻擊否決代碼逐位元組相同、順序也一樣，而呼叫群組 7 的正是 overlay-08
  `021Ah`——兩份各自解出來的東西對上。
  順帶多一支通用工具 `tools/ida-export-overlay-span-lenient.py`：嚴格版碰到
  解不開的位元組就丟例外，而 headless 的例外看起來就是「沒有輸出檔」；
  寬鬆版把那些 byte 記成 data 再往下走。`02E2h` 整段 931 條指令裡只有
  `0AB9h` 一個 byte 解不開，用嚴格版卻是整段拿不到。
  **記錄 `+2Fh` 其實早就有答案**：它是複合職業碼，跟玩家角色同一個欄位
  （`gamepack.ClassCodeOffset`、spec 085、spec 097）。條目一直掛在「還沒讀」
  是因為沒去 grep 自己的 docs。5 是純法師，所以 `07E8h` `087a` 那一條是
  「不逃跑的純法師不走路，交給 entry 6」。怪物用的是同一套編碼：172 筆記錄
  只出現七個值，全部是建角目錄挑得到的合法碼
  （`TestMonsterClassCodeIsTheCharacterClassCode`）。
  **`DS:43A0h` 沒有讀取端**：36 顆 overlay 加 `START.EXE` 掃過絕對定址與所有
  disp16 的 modrm 形狀，只有 overlay-09 `0B4Ch` 那一次清零。旁邊的 `439Eh`
  （上一步的方向）與 `439Fh`（卡住次數）都讀得到，所以那是三個純量旗標而不是
  陣列——`43A0h` 在這一版就是清掉之後沒人用的格子。
  **runtime `+3` 也是自己文件裡早就有的**：那是先攻分數（spec 052／062），
  spec 096 的條目同樣是舊的。順手把 spec 062 那個「還沒閉合：runtime `+3` 的
  維護鏈」關掉——來源是 overlay-13 entry 1（`0000h`）：`0084h` 看記錄 `+10Dh`，
  為 0 就直接跳到 `00FFh` 寫 0，**連 `骰(1,6)` 都不擲**；活著的才走
  `009Fh` 的擲骰、墊到 1、突襲扣 6、超出 0..20 歸零那一串（逐條就是
  `combat.ResolveInitiativeScore`）。回合開頭的 overlay-25 entry 31（`2419h`）
  把 `DS:6772h`／`6773h` 清成 0 再沿串列重數，所以兩個存活數是每回合數出來
  的；`DS:5CF4h` 串列不摘人，離場者留在上面只是 `+10Dh` 為 0。
  **remake 跟著修**：`startRound` 以前對離場者照擲骰再丟掉（當時沒有證據，
  刻意不動亂數流），現在照原版跳過。
  順帶修掉一個量測上的坑：`TestSokalKeepOpensTheOtherBoatRoutes` 的種子迴圈
  只要旗標湊齊就 break，而 `maps`／`blocks` 是跨種子累加的——於是「走得到幾
  張圖」其實是「第幾個種子先湊齊旗標」的函數。改成**斷言要的東西全部湊齊**
  才提早收工。
  **`DS:6786h` 那張表也解出來了**：`6786h` 是 Turbo Pascal 的**偏移基底**，
  不是第一格——表宣告成 `array[1..N]`，實際落在 `678Ah`，基底是
  `678Ah − 4 × 1`。填表的是 overlay-12 entry 1（`31B2h`）一長串直線寫入，
  138 格（代碼 1..139），漏掉的 `58h` 由 overlay-22 `3760h` 另外填；處理常式
  136 支在 overlay-12、2 支在 overlay-13。查表之後「能不能打」完全展開：
  群組 1 的 `19h`／`25h`／`47h` 會立起 `677Ch`，`3Fh` 指到一支空常式
  （overlay-12 entry 126，`31A9h`，`retf 0Ah` 什麼都不做）所以不會；群組 0
  的 `7Eh` 走的是自己當下目標的物品串列——`1087h` 先把 `+0Ah`／`+0Ch` 換成
  候選正是為了它。
  **還缺**：139 支處理常式只讀了五支；`DS:6780h` 這個共用累加格的語意。
  入口仍是 overlay-08 entry 3（`01E4h`）依 `+10Fh` 分派——非零走
  `0058h:0025h`（overlay-09 entry 1），零則走 overlay-08 `0307h` 的玩家
  指令迴圈（指令字串 `Move `／`View Aim `／`Use `／`Cast `／`Turn `／
  `Quick Done` 在 overlay-08 `05E1h` 起）。

- [x] **overlay 的 stub segment 對照表**（2026-09-03，
  [spec 109](docs/spec/109-overlay-stub-segments.md)）。跨 overlay 呼叫寫成
  `call <segment>:<offset>`，那個 segment 不是程式碼位址而是 stub 段；
  少了對照表只能猜，而**IDA 以平坦 16-bit 載入單顆 overlay 時會把它解成
  同一顆裡的 `sub_XXXX`**，看起來很像在呼叫自己。
  推法是 `(executable_file_offset − stub_offset − 3B0h) ÷ 16`，同一顆的
  每個進入點都算得出同一個值——那本身就是一道自洽檢查。
  `ReadDOSOverlayStubSegments` 重生整張表，測試釘住六個已知的呼叫與
  「三十八顆各不相同」。

- [x] **二十五槽的完成條件都讀出來了**（2026-09-03，spec 041）。把每個 `FEh`
  寫入點前面八條指令一起印出來，條件就在那裡。三種形狀：**打贏一場架**
  （槽 2、3、11、20、23）、**收集或旗標**（4..9 的六本書、13、17、24）、
  **計數**（21 的 25 場、15 的 40、14 的兩段式——`ecl6/28` 先寫 `FDh`，
  `ecl6/25` 才升成 `FEh`）。所以「所有區域共用同一種完成判定」是錯的假設。
  **「打贏一場架」那一類已經逐條驗過**：
  `TestWinningTheAreaBattlesCompletesTheirCommissions` 對五個槽各從那一場架的
  `LOAD MONSTER` 起跑、擺出遭遇（有的還要先在原版的
  `29h ENCOUNTER MENU` 選 COMBAT）、清光敵人、答 N、把戰後文字翻完，
  再看該槽是不是變成 `FEh`：邪惡神殿旁（2）、巴恩神殿（23）、瓦海登墳場（11）、
  救回孩子（3）、結局（20）。
  剩下的是「收集／旗標」與「計數」那兩類，以及各槽的正常玩家路徑實跑。

## 發行

- [x] 授權已定案並落地：RRSAL-1.0（非商業免費含修改再散布，實況與平台分潤
  明示允許，商業另談），`LICENSE` 與 `NOTICE.md` 在 repo 根目錄，也複製進每一個
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
  解除條件：一台實體 Windows 與一台 Mac，各啟動一次並截圖。
- [x] repository visibility 與原版素材 deny-list（2026-09-04 使用者定案）。
  CoAB、Pool 與共用 engine 三個 repo 都維持 **private**。原版素材的 deny-list
  已經落地：`NOTICE.md` 列出不隨發行包散布的東西（原版資料、軟體世界說明書
  譯文、倚天字型），`tools/package-release.sh` 的 patch 封包排除原版 ZIP 與
  字型，公開 Release 只掛 patch。
