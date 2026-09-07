# Pool of Radiance remake 工作清單

只列尚未完成且有驗收條件的工作。做完的留在原處打勾，因為每一條後面都掛著
當初的證據與踩過的坑；**要看「還剩什麼」看下面這一節就好**。

## 還沒完成的（2026-09-07 盤點）

這四項就是把 README 自評壓在 60～70% 的那四項。清單其餘部分幾乎都打勾了，
而分數只有 60～70%——兩者不矛盾：打勾的是「規則解出來、接進去、有測試」，
沒打勾的是「拿原版當裁判實際驗過」與「玩家真的從頭走到尾」。

- [ ] **戰鬥數值對原版。** 敵方 AI 已換成原版的接近規則，命中、傷害、豁免與
      存活**沒有在同一場戰鬥裡對過原版**。
      **驗收**：同一組人物、同一場遭遇、固定骰子種子，逐回合比 THAC0、
      傷害骰結果與雙方 HP 變化；不一致的逐條記進 spec，不留「大致相符」。
- [ ] **主線從開場到結局連續跑過一次。** 破關那一場打得贏、旗標也立得起來
      （2026-09-03，`ECL5/7`），但那是單點驗證，不是一趟。
      **驗收**：一次不中斷的實跑，每一個必經 block 都留下經過紀錄；
      中途卡住的地方寫成可重跑的測試。
- [ ] **Windows 與 macOS 的真機啟動結果回填。** 逐步清單已經寫好交接出去
      （[`docs/verification/real-machine-startup-checklist.md`](docs/verification/real-machine-startup-checklist.md)），
      **結果還沒寫回來**。Wine 與 Docker 證得了「不是連跑都跑不起來」，
      證不了真機。
      **驗收**：把七步的結果與每台三張截圖寫回那份清單。
- [ ] **說明書下冊附錄的三張規則表接進 UI**（金錢換算、法術表、武器表）。
      轉錄與校對都做完了（spec 054／064），只是還沒接到畫面上。
      **驗收**：遊戲裡查得到，且與 `internal/journal` 的同一份資料同源。

對拍那兩項在〈發行〉一節底下。

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

- [x] 反組譯並寫 READY 建角／建隊 spec，包含戰鬥 sprite、調色與角色檔。
  Spec 003（建角流程）、004（擲值欄位）、006（肖像）、007（戰鬥圖示）、
  008（角色庫與人物管理選擇項）全部 READY，`cmd/pool-game` 由標題以正常按鍵
  走得到每一步。最後三項殘留在 2026-09-05 收掉：
  - **DOS 285-byte CHA＋多條鏈 export**：`internal/character/export.go` 疊在一份
    base 上只寫有出處的欄位，`internal/gamepack/record_recompute.go` 補上
    overlay-25 `0E36h` 的整份重算，建角完成時寫出 `<NAME>.CHA`／`.ITM`／`.SPC`。
    七名原版預設人物讀進 remake 的角色模型再寫回去，285 bytes 一個位元組都不差。
  - **完整 Party Creation Menu 功能**：說明書 p.8..p.10 的十一個指令全部接上，
    並照「隊伍是空的時候只有四項」的可見規則收；那四項正好是 spec 008 那張
    原版截圖上的四項。
  - **theme 下 sprite／tileset 同步切換**：F2 換的是 `graphics.Picture.RGBA` 的
    色盤，肖像、戰鬥圖示、標題圖、牆面圖章與第一人稱背景一起換。
  - **原版 nested icon menu**：PARTS／COLOR-1／COLOR-2／SIZE／EXIT 的巢狀選單
    照 spec 003 第 7..10 步實作，取代原本的扁平熱鍵。
- [x] 解出第一張地圖的移動遮罩、第一人稱背景／視錐與第一個玩家事件。
  **標題那三件事 2026-09-06 都閉合了**：移動遮罩（門的政策 spec 122、
  邊界與繞回 spec 125）、第一人稱背景／視錐（兩張基準圖逐格 100%，spec 126）、
  第一個玩家事件（Rolf，spec 010／011／117）。地名也解出來了（spec 124）。
  底下曾經掛著的下游線——市議會、神殿、戰利品、訓練——**也都各自收掉了**
  （見下面那幾項）。**2026-09-06 這一項因此結案。**

  結案時仍記著的殘留，全部是「已經照接、只差那個值叫什麼名字」的推論缺口，
  各自寫在規格裡，不再掛在這一項底下：

  | 殘留 | 記在哪 |
  |---|---|
  | 狀態列時鐘沒有走路推進的那一段（原版剛開場也是 `00:00`）| 本項內文 |
  | 8×8 遮罩點陣圖由誰在載入時填 | spec 126 |
  | 每一家訓練所**收哪幾類**（remake 只看遮罩非不非零）| spec 097 |
  | ECL `49E6h` 那個旗標叫什麼（兩條分支都照接了）| spec 074 |
  | 紮營那兩個欄位的非零來源 | spec 114 |
  正常 Begin
  的 ECL3/block 0 → `LOAD FILES 0,0,0` 已閉合 `GEO3/block 0, (15,1), facing 6`，
  初始 `LOAD PIECES` 又閉合 `WALLDEF3 block 0` 與 `8X8D3 blocks 101/102/103`，
  並由正常 `B` 畫面解析 42 個可見原版 wall stamps。GEO／wall material identity 是
  exact；Spec 010 已閉合首次旗標、Rolf 第一頁、事件位置／朝向、monster 12 與 Return
  閘門。Spec 011 又閉合並實作四張 34-byte table、34-step scripted movement、六個停靠
  selector／七頁文字與最終 ECL `EXIT`；正常按鍵已走到 Tyr 停靠畫面。
  **自由移動交接已經量過了**（2026-09-05）：原版導覽按完停在 `(0,4)` 朝西、
  底下換成指令列（基準圖 `06`），remake 同一格同一朝向、**視野逐格 100%**
  （spec 126），再往西一步兩邊都換到貧民窟 `(15,4)`。
  **地名解出來了**（2026-09-06，spec 124）：不能用關鍵字（市議會派任務會把
  所有地名唸一遍，spec 055），站得住的形狀是「同一條控制流上先印一句話、
  緊接著 `NEWECL <區塊>`」。`cmd/pool-map-names` 沿反向邊回走收前置文字，
  73 個換圖點、28 個目的地、0 個追不完的區塊。**第一張圖（ECL 區塊 0 ＝
  GEO3/0）原版自己叫它 `THE CIVILIZED AREA OF PHLAN`**——ecl5/7 與 ecl4/21
  兩個不相干的封存檔各說了一次。順帶命名了區塊 6／19／25／28。
  **坑**：共用 engine 的 `Graph.Edges` 只有分支邊沒有循序邊，只用它回走
  73 個 NEWECL 一段文字都收不到——而那看起來像「原版沒有提示語」。
  **bounded／wrapped 定案了：是 wrapped**（2026-09-06，spec 125）。
  提交一步的是 ECL 的 `2Dh CALL C01Eh` ＝ overlay-07 entry 27（`1A17h`），
  內容逐朝向寫死「沒到邊就加減一，到邊了跳到對邊」——標準的 16×16 繞回，
  與 remake 的 `WrapCoordinate` 逐條相同。
  上鍵那一支（overlay-14 `06AEh`）看起來像在夾座標，其實**四個寫入一律等於
  原值**（X 是 0 往西算出 −1、夾回 0），真正的作用是把 `@6DD5` 立成 1
  ——**這也解掉了 spec 100 的 OPEN**（誰寫、寫什麼值）。
  牆的判定也確認是繞的：`0131h:0039h` 自己把座標繞回 0..15 才查，
  那個慣用法在 overlay-30 出現六次、別處零次。
  過程中照 `06AEh` 改成夾試過一次，野外三張圖的貼圖立刻壞掉——
  **那不是野外的特例，是把前置當成了提交**，已還原並寫進 spec 125。
  **開門／上鎖的規則讀完並接上了**（2026-09-05，spec 122）：撞上一道鎖住的門
  原版出 `Bash`／`Pick`／`Knock`／`Exit` 的選單（overlay-14 `0DB7h`）。
  找它的方法是掃字串——三十八支 overlay 裡和門有關的字面只有那一組。
  門的狀態由 `0131h:0039h`（overlay-30 entry 5）用**目前格子與朝向**問出來，
  1 沒鎖、2 鎖住、3 閂住；那三個參數就是 `DS:6A0Bh`..`6A0Dh`，
  **spec 012 掛著的「`6A0Bh`／`6A0Ch` 用途」順帶解掉了**。
  `Bash` 是 AD&D 一版力量表的 Open Doors 欄（狀態 3 用括號裡的數字），
  整隊每人各試一次；`Pick` 是賊的開鎖百分比（記錄 `+78h`）而且**一扇門只能試
  一次**；`Knock` 一定成功但吃掉一格記憶。成功之後兩邊的旗標一起改
  ——共用 engine 的 `UnlockDoorWrapped` 做的就是這件事。
  remake 已照接（`internal/gamepack/door.go`＋`cmd/pool-game/door.go`），
  兩張力量表逐格有測試。
  **鎖住的門有多少，量過了**（2026-09-05，`workplace/doorscan`）：
  `CanMoveDungeonWrapped` 對門旗標 1（未鎖）放行，2（鎖住可撬）與 3（撬不開）
  擋住。GEO2/9 有 29 道、GEO4/2 19 道、GEO2/20 11 道、GEO1/18 2 道；
  GEO3/0、GEO4/21、GEO6/25、GEO8/27、GEO8/29 各 0 道。
  **這些門不是覆蓋率的瓶頸**：全部打開對世界巡迴的區塊數沒有影響（見下面
  「走得到的內容量」那一條），所以這一項是玩法完整度，不是進度閘門。
  **同狀態 DOS 畫面已經對拍了**（2026-09-05）：`tools/capture-dos-adventure.sh`
  把原版一路開到第一人稱畫面，五張基準圖與逐項差異寫在
  `docs/reference/original-dos/adventure/README.md`。**Rolf 初次 APPROACH
  的圖像也在那裡**（不再是待證明）。
  對拍結果是**部分一致**。對上的是事件鏈與兩個背景色索引：
  Rolf 事件在 `15, 1 W`、導覽第二段在 `11, 2 S`，remake 的 `FACING 3`／`FACING 2`
  換算後與原版狀態列逐項相同，台詞逐字相同；內框幾何量出來就是
  `StageInset{24,24,88,88}`，本來就對；天空 EGA 11 青、地面 EGA 6 棕已照原版改
  （先前的 EGA 1 藍／EGA 8 深灰是接手時帶進來的佔位值，spec 047 已訂正）。
  APPROACH 的半身像也接上了：spec 117 讀出 `SETUP MONSTER a,b,c` 的三個 operand
  （`SPRIT` 區塊／接近距離／`BODY` 區塊）與 overlay-29 的 `HEAD`＋`BODY` 疊法，
  Rolf 是 `HEAD3` 區塊 8 疊 `BODY3` 區塊 9，畫在 `(24,24)` 起的 88×88，
  與原版畫面逐格相同；導覽開始之後那一框變回視野，也與原版一致。
  順帶修掉一個一直在的版面錯：對話框原本畫在 `198`，壓掉視野下面 64 個像素
  （只畫牆片時看起來像「牆片本來就矮」，換成半身像才明顯），現在移到視野下方。
  **「視野的透視尺度不對」量過之後不成立**：現在畫面與對拍走同一份
  `composeFirstPersonInset`（88×88 索引圖，放大兩倍貼上去），拿 `(14,1)` 朝西
  對原版逐格比是 **7744 格裡 7442 格相同（96.1%）**，並由
  `TestFirstPersonInsetMatchesTheDOSShot` 釘住地板。三段背景的邊界也改成照
  共用 engine `BuildBackground` 解出來的原版版面（中間兩列黑）。
  剩下的 302 格集中在正前方那道遠牆：牆頂多出一條洋紅（`y=40..42`）、
  門洞沒有填黑（`y=43..55`, `x=32..55`）。兩處都在共用 engine 的
  `BuildWallLayout` 選片與 `TraverseWallViewWrapped` 上，動它要先問過使用者。
  右半版面也接上了：原版的隊伍面板（`NAME` 靠左、`AC`／`HP` 靠右）加狀態列
  `14, 1 W 00:00`（朝向字母由 spec 076 換算，時鐘還沒有走路推進的那一段，
  原版剛開場也是 `00:00`）；出處與現況那幾列移到 F1 說明頁，資訊沒有少。
  **2026-09-05 再往下走一段**：把 DOSBox 的鍵序接上二十二個 Return 按完導覽，
  再送方向鍵，量到自由移動之後的三張基準圖（`06`／`07`／`08`）。讀到兩件事：
  狀態列的時鐘**一步一分**（`00:00 → 00:01 → 00:02`；轉向、被牆擋下來、
  `(0,4)` 往西進貧民窟那一步都不加），以及自由移動時畫面最下面是指令列
  `AREA CAST VIEW ENCAMP SEARCH LOOK`。時鐘已照這條接上並存進存檔
  （spec 118，六十步進位到 `01:00` 同時是進位上限表的正對照）。
  **指令列也接上了**（spec 119）：原版有兩條字串，`DS:04CAh` 是
  `Area Cast View Encamp Search Look`、`DS:04F3h` 是**少了 `Area`** 的野外版，
  分派在 overlay-14 `09CAh`。`A` 切換平面全圖（照 GEO 的牆位元組畫格線加
  一個指著朝向的箭頭）、`S` 翻 `[4937h]+594h` 第 0 位並讓狀態列多一段
  `SEARCH`、`L` 設第 1 位並對這一格重跑 ECL entry 1（`ds:4946h` 就是
  spec 016 的 SearchLocation）。指令列只在自由移動時出現，導覽還在跑時
  原版那一列是「按 Return 繼續」。
  **順帶挖出並修掉一個大缺口**（spec 120）：拿自由移動那一格 `(0,4)` 朝西
  對拍原本只有 **60.4%**——那一格正前方是城門，而 remake 整道都沒畫。
  原因不是「WALLDEF 是空的」（倒出來看，那一段的形狀就是一道有拱的門面），
  是**取圖的方式**：原版的 `Put8x8Symbol`（overlay-35 `015Dh`）按**符號編號
  自己落在哪一帶**決定去哪一組取圖（五帶 `1`／`2Eh`／`74h`／`BAh`／`100h`，
  表在 `DS:2736h`），而共用 engine 的 `BuildWallLayout` 是按 WALLDEF 記錄取，
  低號帶那些編號一律算成負索引被丟掉。
  第 0 帶不是 `LOAD PIECES` 換進來的，是開機時 overlay-11 `04AAh` 從
  **`8X8D1.DAX` 區塊 203** 載一次（區塊 203 剛好 45 個 item，與 `01h..2Dh`
  一樣寬，這個自洽就是驗算）。改成按帶取圖之後：
  `(14,1)` **96.1% → 98.6%**、`(0,4)` **60.4% → 97.0%**。
  `TestFirstPersonInsetMatchesTheDOSShot` 的地板跟著調到 98.0／96.0。
  **外框美術接上了**（2026-09-06，spec 123）：那圈紅色繩索不是畫出來的線，
  是全域第 4 帶（`8X8D1.DAX` 區塊 202）的三個符號拼的——`114h` 四個角、
  `115h` 左右兩欄、`116h` 上下兩列。找法是先從截圖量出一格花紋（PNG 的調色盤
  是縮減過的，要先換算回 EGA），再掃**所有** DAX 的每一個區塊逐格比對，
  整包只命中一處。位置是 40×25 tile 網格的第 0／23 列與第 0／39 欄
  ——**最後一列是空的**。五張截圖逐格相同，所以它是整個遊戲共用的畫面框。
  第一人稱那一框也是同一圈（外框 native `(16,16)..(119,119)`、內部 88×88 從
  `(24,24)` 起），已照同樣的關係圍上去；**絕對位置仍與原版不同**，那屬於整體
  版面。頁尾與指令列的基線跟著從 386 移到 366 讓開下緣。
  **對拍表清空了：兩張都逐格 100%**（2026-09-06，spec 126）。剩下的那
  百分之二三是兩件事，都在消費端，engine 的輸出沒變：
  ① `PostWall` **只該蓋黑的**——它存在的理由是「斜邊素材的黑底會蓋住頂部
  角落」，整條蓋會把真的牆片洗掉（城門左上角那個 120 格的灰色三角），
  97.0% → 97.6%；② 符號色號 **`0Dh` 不畫**，讓背景透出來——把它畫出來時
  兩張圖的差異**全部**是同一形狀「remake 是 D、原版是那一區的背景」
  （地平線下棕 6、上方天空 B、城門那邊灰 7），而原版截圖的調色盤裡根本
  沒有洋紅。跳過之後 **7744/7744**，地板改成 100 不留餘裕。
  原版**用什麼機制**不畫 `0Dh` 2026-09-06 讀出來了（spec 126）：
  **是另一張 AND 遮罩點陣圖，不是顏色鍵**。overlay-36 entry 8 的 `0A7Ah` 起
  逐位元組做「目的地 AND 遮罩、再 OR 來源」，遮罩由圖片記錄 `+13h`／`+15h`
  的遠指標指過去；沒有遮罩時走 `0AECh` 直接 `Move` 整列。
  **所以「哪個顏色算透明」這個問題問錯了**——overlay-36 全檔沒有對 `0Dh` 的
  立即數比較，正是因為根本沒有顏色特判。remake 跳過 `0Dh` 的逐格 100% 仍然
  成立，只是那是用顏色把「素材遮罩留白的格子」反推出來。剩下的一格是那張
  遮罩由誰在載入時填。
  共用 engine 那邊已把 `PostWall` 的套用規則寫進型別註解
  （分支 `postwall-black-only-note`，行為沒動，CoAB 不受影響）。
  **副產品**：那一輪對拍抓到隊伍非空的選單只有九項，沒有 `LOAD SAVED GAME`
  也沒有 `TRAIN CHARACTER`。兩項的可見規則 2026-09-06 都照原版接完了
  （spec 008）：`L)OAD` 只在空隊伍出現，`T)RAIN` 只在訓練所裡出現
  ——那兩張截圖沒有它，是因為兩張都不在訓練所裡。
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
  改由武器決定。裝備之後的 AC 重算鏈已由 Spec 080（`sub_281` 的五個累加器）與
  Spec 079（`sub_39F` 的負重分段）分別閉合，Spec 063 又把整支 `0E36h` 接起來：
  九個衍生欄位對七名預設人物逐格重算得出來。`+2Eh` 也在 2026-09-05 閉合——
  它就是種族碼（spec 003 的 Elf=`2`），四個加值武器型別是長劍、短劍與一族弓，
  即 AD&D 的精靈武器加值。`+0AAh` 的 **producer** 也在同一天閉合：
  全遊戲只有 overlay-16 `1C02h` 一處寫它（建角，常數 1），怪物那一側是資料
  帶的（172 筆裡 123 筆是 0）。**語意仍未命名**——與生物種類、力量是否非零、
  有沒有職業等級三個假說都不成立。
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
- [x] 將 CoAB 已驗證但仍夾有作品常數的 ECL runtime 分批泛化到共用 engine；目前已抽
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
  **紮營的時間接上了**（2026-09-05，[spec 114](docs/spec/114-camp-rest-time.md)）：
  時間是一個逐位進位的數（`DS:6CB6h` 七個 word，上限表在 `DS:35D4h`
  ＝ 10／10／6／24／30／12／256），紮營挑的是一段長度所以月會被折回天、
  天夾在 99。Y／H／M 選欄、I／D 增減、R 開始休息，分鐘一次五分。
  休息的效果照原版與說明書：**每二十四小時每人回一點生命力**（`0830h` 的
  288 刻 × 5 分 ＝ 24 小時，說明書 p.29 同）、法術要休息夠「各法術等級的
  總和」小時才記得完（`0B63h` 每小時把 `+2Ch` 減一）。
  旅店因此不再自動回滿——說明書 p.31 給旅店的保證是「絕對安全而且不會有人
  中途打擾」與「你高興休息到什麼時候就待到什麼時候」，不是免費回滿。
  **打斷的判定也讀出來了**：逐區兩個欄位——`[DS:4937h]+5A4h` 是每幾刻檢查
  一次（0 ＝ 這一區不會被打擾）、`+5A6h` 是 1d100 的門檻。規則實作成
  `gamepack.SimulateRest`。但**那兩個欄位的非零來源還沒找到**：36 顆 overlay
  掃過所有 disp16 形狀，寫它們的只有 overlay-07 `0244h`／`024Fh` 把它們清成
  0，所以非零值一定是某個整塊複製帶進來的。在找到之前 remake 一律用 0／0，
  那正是原版初始化之後的狀態。
  **神殿的其餘六項也讀出來了**（2026-09-05，
  [spec 115](docs/spec/115-temple-services.md)）：治療失明 1000（效果 `21h`）、
  治療疾病 1000（`DS:0112h..0117h` 六個代碼）、起死回生 5500、解毒 1000
  （拿掉 `37h`／`16h`／`0Fh`）、解除詛咒 3500、石化解除 2000。三種傷藥的
  100／350／600 與既有的表一致。前提分兩類：失明／疾病／中毒／詛咒問效果
  串列，起死回生與石化解除看 `+10Ch`（6 死亡、7 石化——與
  `combat.DeadState` 對得上）。付款一律是「先個人、不夠才整隊公款出全額，
  兩邊不合併」。
  **起死回生要付一點體質**，而且生命力上限跟著少掉一個生命骰的體質加成
  （`0607h` 減體質、`060Fh` 起用扣完的體質重算權重；體質 17 以上的戰士不扣）。
  規則實作在 `internal/temple`，九項逐項釘住。
  **賣東西也讀出來了**（2026-09-05，
  [spec 116](docs/spec/116-appraise-and-sell.md)）：商店與神殿共用
  overlay-21 entry 19。寶石與珠寶是兩個計數（記錄 `+92h`／`+94h`，金幣
  `+90h`），估價各有一張 1d100 的表——寶石是固定價 10／50／100／500／1000／
  5000，珠寶是「選一段再 `Random(範圍) + 底價`」七段。
  **賣掉是換成白金付，不是折價**：估價的單位是金幣（畫面印 ` gp.`），而錢加
  進的是記錄 `+90h`——依 spec 040 的七欄版面（`+88h + 2 × 索引`）那是白金。
  AD&D 一版 1 白金 ＝ 5 金，所以 `1F0Dh` 的 `div 5` 是幣別換算。同一支減的
  `+92h`／`+94h` 也落在那張版面上（寶石、珠寶），三個欄位一起自洽。
  按 K）eep 則變成一件物品（`+2Eh = 46h`、`+31h = 65h`、價值寫在 `+3Ah`），
  但背包滿 16 件時「留著」那一項根本不會出現。規則實作在 `internal/treasure`。
  **界面也接上了**：神殿的 H）EAL 九項全部可用（先前只有三種傷藥，其餘六項
  停在 fail-closed），選單名稱直接從 `temple.Services` 取所以兩份表不會漂開；
  A）ppraise 進兩層——挑寶石或珠寶、再對估好價的那一件選 S）ell／K）eep，
  留著時建出來的 63 bytes 記錄直接放進 `poolsave.Item.Raw`。
  **商店那一側也接了**：`G`／`J` 估價、`S` 賣、`K` 留，共用 `internal/treasure`
  那一份規則。

  **這一項的三個服務規則（紮營的時間與被打斷、賣東西、神殿的其餘服務）到此
  都結清了。** 各規格自己還留著的開放項不屬於這一項的驗收：spec 114 的
  「休息打斷那兩個逐區欄位的非零來源」、spec 115 的 `+11Bh`、
  spec 116 的「留下來那件物品的其餘欄位」。
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
- [x] 法術的記憶與施展。容器都就位了（spec 069 效果串列、spec 070 記憶陣列、
  spec 072 格數）、派發表、參數表與豁免判定也閉合了（spec 073／074／075）。
  **六十七支全部接得出來了**（2026-09-05，最後一支是臭雲術）。這一項仍然
  **瞄準那一層 2026-09-06 整支讀完並接上**（spec 127），這一項因此收掉。
  停在那裡的是「迴圈頭那一個旗標的極性靜態讀不出來，而且兩種讀法都會矛盾」，
  **而那個矛盾本身就是讀錯的訊號**：`2E31h` 呼叫的 `05BB:08D4h` 不是 `Pos`
  而是 Turbo Pascal 的**集合成員測試**（結果留在旗標裡），`cs:2D80h` 那 32 個
  位元組是位元圖不是字串，解出來是 `{0, Return, 'E', 'T'}`——正好四個離開鍵。
  外層迴圈頭 `35F2h` 還有另一個集合 `cs:350Ch` ＝ `{0, 'E'}`，那才是 `E`（Exit）
  真正的出口。
  **`Center`（`3714h`）也讀完了**：它拿目前目標那一格去呼叫
  `013D:0061h(X, Y, 0, 8)`，而模式那一格的意思由 `07D4h` 定下來——
  **模式就是「離視窗中心多遠才捲」**。原版的戰術畫面一次只有 **6×6 格**
  （`0912h` 的重畫迴圈兩層都數到 6），中心是 record `+2`／`+3` 加 3；
  Manual 游標用餘裕 3（走到邊緣才捲）、`Center` 與火球術的雲心用 0
  （一定捲到正中央）、`0FFh` 是「不看框直接重畫」。這一併收掉了先前記著的
  「`07D4h` 的模式值代表什麼」。
  remake 已接：`internal/combat.RecentreViewport` 是 `07D4h` 的規則（含
  `3..2Eh`／`3..15h` 的夾制），`cmd/pool-game/manual_aim.go` 的 `M` 進游標、
  `C` 置中、八個方向鍵與戰術移動同一組、`Return`／`T` 選定、`E`／`Esc` 取消。
  **remake 目前整張盤面都畫得出來，沒有 6×6 的捲動視窗**，所以視窗原點只出現
  在瞄準列尾端；原點的初值原版從哪裡來還沒讀。
  範圍法術的收人預算（參數表 `+0Fh`）與火球術的 `DS:4933h → +1CCh` 分支
  都在 2026-09-06 讀完並接上了，九個鄰近查詢呼叫點的預算也逐支對完
  （spec 074）。
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
  **挑天／時／分那一段接上了**（2026-09-05，spec 114）：進位規則、五分鐘的
  級距、99 天的夾限與 Y／H／M／I／D／R 的按鍵都照 overlay-20 接。
  休息被打斷的規則也接了（spec 114），只差那兩個逐區欄位的非零來源。
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
  **範圍法術的預算讀完了**（2026-09-06，spec 074）：它是參數表 `+0Fh`
  ——把三十八支 overlay 掃一遍找 `lcall 0138h:003Eh`，9 個呼叫點裡有兩個
  （overlay-09 `0272h`、overlay-13 `1F56h`）推的是 `[編號×16 + 31A3h]`。
  六十七格裡非零的只有七格，**正好都是範圍法術**：睡眠、沉默 15 呎、臭雲、
  解除魔法兩個編號各 1，火球與 64 號各 3。已照接（`SpellParameters.AreaBudget`），
  不再是「先收整邊」。
  **`DS:4933h → +1CCh` 那個分支也讀了**：它是 ECL 位址 `49E6h`（class 0 的
  換算，spec 106），為零時火球術自己收一次人（`2675h` 的常數 2），非零時沿用
  既有的目標清單。全域初始化設成 1，野外那幾個區塊（ECL6/19、6/25、7/26、8/27）
  有 16 個 `SAVE`（10 個寫 1、6 個寫 0）在開關它——**它代表什麼還沒讀**
  （兩條分支都照接了，缺的是那個旗標的名字；記在 spec 074，不擋這一項）。
  **那 9 個呼叫點的預算全部讀完了**（spec 074 的表）：兩支用參數表 `+0Fh`、
  一支用 `+6 & 7`、一支 `(+6 & 0Fh) >> 4` **恆為 0**（原版兩步一起必定是 0，
  照碼記著不改）、一支常數 `7Fh`、一支常數 `0FFh`、一支常數 2（火球自己那支）、
  兩支由呼叫端給。**`352Ch` 那一支 2026-09-06 整支接完**（Manual 與 Center）。
  **另外二十五支是純泛型的**（2026-09-03）：只推「編號 ＋ 四個零」給
  `08BCh`，射程／持續／豁免／效果碼全部來自參數表，所以認得出版型就等於
  一次接完一整批——`ParseGenericSpellHandlers` 逐位元組比對整個版型認出
  25 格（護邪、防護、隱形、解鎖、致盲、詛咒…）。加上逐支讀的三十四支處理常式
  （37 個編號），六十七格裡接得出來的有 **67 格**——2026-09-05 補上最後一支
  臭雲術之後全數接完（`TestImplementedSpellCount` 釘住 42／25／67）。要比對整個版型不能只看「有沒有呼叫
  `08BCh`」——會算傷害的那幾支也呼叫它，只看呼叫會把傷害弄不見。
  **`+9Fh` 與 `+6Ch` 解出來了**（2026-09-03）：前者是生物種類（0 人類、
  1 類人、2 巨人、4 不死、0Eh 蛇與蠍…），後者低位是體型、位元 7 標大塊頭。
  四個互相獨立的比較點指同一個讀法（死靈術只認 0、迷蛇術只認 0Eh、
  overlay-12 對 4 設旗標、魅惑與定身要求兩個都不大於 1）。魅惑人類、
  定身術兩個編號與迷蛇術因此接上了。
  **被迷住之後會怎樣也讀出來了**（2026-09-05，spec 112）：效果代碼 `0Bh`
  把記錄 `+10Eh` 改成施法者那一邊（**倒戈**）、`+10Fh` 設 1 交給 AI 分派、
  士氣設成 `0B3h`；解除時從效果節點 `+3` 的位元 6 還原原本的陣營。
  **倒戈已經接上**（2026-09-05）：效果串列接好之後，魅惑會把陣營改成施法者
  那一邊、控制權交給 AI（原版記錄的 `+10Fh`，remake 的 `AIDriven`），
  原陣營記在節點 `+3` 的位元 6，解除時還原。倒戈之後它照樣行動、只是換一邊
  打——「誰由 AI 走」因此不能再拿陣營來判，那是這一輪換掉的判準。
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
  **效果節點串列接上了**（2026-09-05）：`gamepack.EffectList` 就是原版角色
  記錄 `+7Fh` 的那條單向串列（新的接在尾端、線性搜尋找到最早掛上的那一個），
  節點 `+3` 打包的四件事（等級、已套用、原陣營、施法者陣營）都有存取器。
  戰鬥狀態把 `Asleep`／`HeldRounds`／`Charmed` 三個旗標換成同一條串列——
  兩份真相會讓解除魔法解到空的。**解除魔法（41／46）因此接上了**，
  成功率照 `2356h` 的不對稱公式（高一級 +5、低一級只扣 2），`+3` 是 `0FFh`
  的解不掉。接得出來的法術從 62 格變成 64 格。
  **死靈術也接上了**（2026-09-05）：先前把它列成「卡在戰鬥中新增戰鬥員」，
  讀完 `2043h` 才知道**那個診斷是錯的**——它不生新的戰鬥員，是把已經死掉
  （狀態 6，瀕死不算）的**人類**屍體叫起來，換到施法者那一邊、生物種類改成
  不死、基礎移動設 6、生命補到 `+32h`、狀態改 1，額度是施法者等級。
  接得出來的法術因此變成 65 格。
  **恢復術也接上了**（2026-09-05）：欠帳那對欄位（記錄 `+74h`／`+75h`）
  進了存檔模型（`DrainedLevels`／`DrainedHitPoints`，`omitempty`），
  施法時呼叫既有的 `gamepack.Restore` 還一級。remake 還沒有吸取的來源，
  所以現在一律沒欠帳——**那正是原版沒欠帳時的行為**（`2C16h` 直接返回），
  不是佔位。接得出來的法術變成 66 格。
  **2026-09-05 補上最後一支臭雲術，六十七格全數接完。** 下面那一段記的是
  它當時卡在哪：

  **2026-09-05：盤面物件那一半接完了**（spec 121）。`0DCE` 之後那一段讀完了
  ——它是**收雲**：掃 `DS:6634h`（筆數 `DS:6673h`、每筆 7 bytes）看那一格上有沒有
  別的物件，有就把地形寫 `1Fh`、沒有就從節點 `+8+i` 取回原地形，摘掉節點釋放，
  **再走一遍剩下的雲把它們的四格重新蓋成 `1Eh`**——那就是重疊的處理方式
  （共用的格子被還原掉了，不重蓋會在另一團中間開一個洞）。
  順帶訂正兩件事：節點的 `+8..+0Bh` 是原地形、`+0Ch..+0Fh` 才是「蓋得住」旗標；
  **雲是四格的 2×2**（雲心、東、東南、南），先前寫的「五格多一格西南」是把
  不在迴圈窗內的 `DS:28A7h[5]` 也算進去了。
  `gamepack.CloudList` 已經照這條接好（蓋／收／還原／重疊重蓋，四格座標用真的
  方向位移表算），四條測試釘住。

  | 法術 | 還缺什麼 |
  |---|---|
  | 臭雲術（`Stinking Cloud`，編號 34 ＝ `22h`，`1AF6h`）| **接完了**（spec 121）。整條路：法術參數表 `+0Ah` ＝ `1Eh`（overlay-22 `0A2Bh` 取，audit JSON 的 `effect_code` 早就算好）→ `0100h:0052h` 掛上 → 群組 11（命中判定，overlay-13 `1595h`／overlay-22 `09B2h`）與群組 15（輪到這個人行動，overlay-08 `0289h`／`065Eh`）每次問到 → 處理常式 overlay-12 entry 29 `0A76h`：印「<名字> **is coughing**」、把 `+108h` 的 `+1`／`+2`（這一回合的行動權）清成 0、AC 每次變差 2 點（下限 AC 10）。腳下算不算站在雲裡由 `013D:007F`（overlay-32 entry 19 `0CB9h`）判，把佔的四格合成一個地形碼，**雲一票否決**。同組的 `15h` 是 `is silenced`，逐行同形，可當正對照。**先前寫的「迷霧術（34）／臭雲術（`22h`）」是同一支法術**——34 = `22h`，而 Pool 的 67 支法術裡根本沒有迷霧術。**remake 也接上了**（`cmd/pool-game/cloud.go`）：施法在盤上蓋一團 2×2、對當下站在裡面的人掛 `1Eh`，之後每次輪到他行動就印「咳個不停」、收掉這一回合、把內部 AC 減 2（下限 `32h`）。結算要求「身上有效果」與「腳下還是雲」**兩個都成立**——走進別人放的雲會不會自己中招，原版哪一段掛的還沒讀，所以取保守讀法 |

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
  **`00C9:005Ch` 讀出來了**（2026-09-05，spec 110）：它是 overlay-19 entry 12
  （`297Bh`），不是「昇級專用」而是全遊戲共用的法術清單畫面。七種清單的字尾
  逐字取自 `2918h..2971h`（`in Memory`、`in Spell Book`、`on Scroll`、
  `on Scrolls`、`to choose from`、`to be memorized`、`to be scribed`），
  收哪些由 overlay-22 entry 2（`0588h`）逐種類決定。動作名在 overlay-22
  `0000h..001Ah`：`Cast`／`Memorize`／`Scribe`／**`Learn`**。
  **昇級挑法術＝種類 4 配動作 4**，候選規則是「那一級有格子
  （`+0B1h + 組×3 + 等級 > 0`）而且法術書 `+32h + 編號` 是 0」——**不看職業組**。
  remake 的 `L` 已改成照這一條（先前多加的「只給法師」是沒有出處的收窄）。
  **卷軸抄寫也讀了**（2026-09-05，spec 110）：overlay-20 `09A2h` 過了
  「這件是卷軸」（型別表類別 `0Bh..0Dh`）與「那一格的第 7 位設起來」兩道閘
  就直接寫 `+32h`，**整支沒有任何擲骰**——原版的抄寫不會失敗，「成功率」
  這件事在 Pool of Radiance 裡不存在。卷軸的三格法術在物品記錄
  `+3Ch..+3Eh`。抄寫與記憶共用節奏計數 `DS:6CC3h + 隊員序號`，歸零時做一條、
  再設成下一條的等級 × 3。
  **還缺**：overlay-22 entry 1（`008Ah`）的逐鍵行為與那兩個沒解的區域變數
  （`[bp-5Eh]` 的 `0Fh`／`16h`、`[bp-60h]`）、`0AC0h` 的呼叫端（節奏計數
  一格是多久）、以及預設法師的條數比「四條加每級一條」多出來的差額從哪來。
  **決定性的實驗還沒做**：原版拿第 1 級法師施魔法飛彈看傷害是不是 0、
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
  **`T` 的啟用旗標 2026-09-06 讀完並接上了**（spec 008）：`DS:06D4h` 由
  overlay-16 `01BFh` 的兩道閘門決定。第一道是這一區的訓練所遮罩
  `[4937h]+550h`——依 spec 106 的 class 1 換算（**要 mod 10000h**）就是
  ECL 位址 `6DA8h`，全遊戲只有 **ECL3 區塊 11**（訓練所／競技場）的四道門
  寫它：`71h` 法師、`72h` 牧師、`74h` 賊、`78h` 戰士。第二道是 `ds:466Eh`
  ——**原版的除錯碼**：隊伍畫面按 `J` 輸入 `STING`，遊戲回
  `I Understand, master...`，`T` 就無條件出現，訓練所裡另外六道閘門也一起略過。
  兩道都不成立時原版**連寫都不寫**，旗標保留上一次的值；開場空隊伍那一輪
  已經寫成 0，所以預設看不到 `T`——**spec 008 那兩張原版截圖沒有 `T`，
  是因為兩張都不在訓練所裡**，不是「原版沒有這一項」。
  remake 照接了（`cmd/pool-game/training_gate.go`），先前那條「一律顯示」的
  已知偏差因此收掉。
  還缺：種族／能力值的等級上限那一段（Pool of Radiance 走得到的四個職業
  沒有那個限制，目前不影響）、每一家訓練所**收哪幾類**（remake 只看遮罩
  非不非零，原版是拿它逐人 `and` 職業分類）、昇上法師等級之後的
  學新法術流程、訓練要不要收費。
- [x] **經驗值發下去了**（2026-09-03，spec 097）。一隻怪物值
  `+0B8h + +0BAh × 生命值`（與 AD&D 一版逐筆相同：KOBOLD 8、ORC 15、
  OGRE 195、SPECTRE 2030），總額除以分的人數，再由每個人依複合職業碼調整
  （純職業主屬性 > 15 多拿十分之一，兩職業除以 2、三職業除以 3）。
  `save.Character.Experience` 存得下來，欄位是 `omitempty` 所以舊存檔照讀。
  索寇要塞第一場實測：12 隻怪物，六人隊每人 64 點。
  還缺：原版「有資格分」的判準（`+10Dh` 為 0 與狀態 1）沒讀完，
  目前全隊都分。
- [x] **最小 game pack 與 adapter 建好了**（2026-09-05，
  [spec 113](docs/spec/113-game-pack.md)）。pack 拆成三個分檔放在
  `internal/gamepack/pack/`，依檔名排序合併：`00-core.json` 是 header
  （`id = pool-of-radiance.phlan`）、`presentation`（原生 320×200 放大兩倍，
  與前端的 `logicalWidth`／`logicalHeight` 一致）與 `events`；
  `20-locale.en.json`／`20-locale.zh-TW.json` 是兩個語言各 152 條的字串表。
  載入走共用 engine 的 `LoadPackPartsFS`，Pool 這一側只加
  `gamepack.Pack()`／`LocaleTable()` 與前端的 `packMessage`——
  **介面字串已經真的從 pack 讀**，不是擺著好看的鏡像。
  `messageID` 與它的出處註解（說明書頁碼、原版字串的位址）留在 Go 那一側，
  因為 JSON 沒有註解，搬進去會掉。
  **不抄 CoAB 有機械檢查**：`TestGamePackCarriesNoAzureBondsContent` 掃整份
  序列化後的 pack，出現 CoAB 專有識別字就紅，並配一條正對照確認掃描面沒有洞。
  **刻意不放進 pack 的**：建角規則（從原版 `START.EXE` 資料段重生的，搬進
  JSON 會把「逐位元組對得上原版」降級成「有人抄了一份數字」）、事件與地圖
  （直接跑原版 ECL／GEO，抄進 pack 會變成第二份真相，而抄錯一格的症狀是
  「測試綠、玩家走不到」）、`search`（分鐘數還沒從原版讀出來，沒有證據不宣告）。
- [x] 隊伍朝向的座標系：原版是 0 北、1 東、2 南、3 西，不是共用 engine 的
  0/2/4/6。導覽 34 步裡有位移的 20 步逐次與「上一步的朝向」相符、零例外，
  35 個原版位置也從沒出現 4..7。混用造成三個症狀：開場結束後隊伍**完全走不
  動**（轉向 45 度一步，從奇數朝向永遠轉不到偶數）、`WallWrapped` 對奇數朝向
  靜默回「沒有牆」（35 個位置裡有 23 個是奇數）、存檔驗證把 1 與 3 判成非法。
  換算改成留在 Pool 這一側（`Spawn.Direction()`），存檔升到 schema 7 並把舊值
  除以 2。契約見 spec 076。
- [x] **走得到的內容量到了**（2026-09-03 起；2026-09-05 收在 **29／29**）。

  **結論先寫**：29 個有文字的 ECL 區塊**全部走得到了**，而且是在完整的前端
  底下用按鍵走的。三條最後打通的路各自有專屬測試：
  `TestPayingTheTollAtStojanowGateOpensTheCastle`／
  `TestTheCastleBehindStojanowGateHasContent`（3、4、5、6、7）、
  `TestTheWildernessCaveRerollReachesTheEasternOutpost`（13）、
  `TestTheNomadCampOnTheMiddleWildernessSheet`（17）、
  `TestTheOutpostOnTheWesternWildernessSheet`（28）。
  下面是一路量出來的過程，保留給後續參考。

  原始量測：六個種子各走 6000 步，走到
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
  `[0 8 11 14 18 20 21 24 25 26 27 29]`；聯集是 12 個。
  變的是**證據等級**：26 與 27（野外／樞紐）現在是探索器**自己走到的**，
  不再只有世界巡迴用 `DS:4AC4h` 覆寫目的地才碰得到。那兩塊因此從
  「腳本跑得動」升成「玩家走得到」。

  **2026-09-05 稍晚：世界巡迴改走三種主線狀態，量到 16 個。**
  單一狀態量不到全部——不同的旗標開不同的區域，而且**已完成的區域反而
  不再給內容**：

  | 主線狀態 | 走到的 ECL 區塊 |
  |---|---|
  | 什麼都不設 | 12 個 `[0 8 11 14 18 20 21 24 25 26 27 29]` |
  | 委任進度 9（`DS:4AC1h`）| 14 個，多開 16、17 |
  | 26 個完成槽（`4AA6h..4ABFh`）全設 `FEh` | 只有 10 個，但多開 22、23 |

  三者聯集 **16 個**：`[0 8 11 14 16 17 18 20 21 22 23 24 25 26 27 29]`。

  **再一步：選單輪替改成跨趟共用，變 17 個。** 探索器對同一格的選單是
  「第幾次來就選第幾項」，而巡迴原本每一趟都把那個計數歸零——於是每一趟
  都只選得到第 0 項。樞紐圖的地點是選單選的，不換選項就永遠只進得去同一個
  地點。改成跨趟共用之後多走到 **10（瓦海登墳場）**，聯集是
  `[0 8 10 11 14 16 17 18 20 21 22 23 24 25 26 27 29]`。
  門檻釘在 15（不頂到實測值，見那一條的註解）。

  **還缺 12 個**：`[1 2 3 4 5 6 7 9 13 15 19 28]`。其中 **8 個是 spec 055
  認定的自載區域腳本**（1、2、4、6、9、13、15、28 ——卡德納紡織廠、
  曼多爾圖書館等），那幾個是「玩家該走得到的地方」；
  另外 4 個（3、5、7、19）不是自載腳本，其中 7 與 17、22 是靜態展開就失敗
  的三個（spec 002 的既有缺口）。
  **為什麼走不到，這一輪也量出來了。** 用 `cmd/pool-world-graph` 的
  `NEWECL` 圖對一遍，缺的 13 個裡有 **7 個只差一步**——它們的來源區塊
  已經走到了：

  | 缺的區塊 | 一步之外的來源 | 來源走到了嗎 |
  |---:|---|---|
  | 1、19、28 | ecl6/25 | ✓ |
  | 2、9 | ecl1/18 | ✓ |
  | 10 | ecl7/26 | ✓（**已走到**，就是靠跨趟共用選單輪替）|
  | 13 | ecl8/27 | ✓ |

  所以**不是旗標的問題，也不是走路預算的問題**：重走上限從 1 拉到 3、預算
  加倍實測過，走到的區塊一個都沒多，時間卻多了七成。原因在那幾張圖的形狀
  ——`geo6/25` 只有 62 格可走卻分成 **42 個互不相連的區塊**，`geo8/27`
  是 62 格 39 區、`geo2/9` 是 88 格 30 區。那是**樞紐圖**：地點之間靠選單
  進出，不是靠走格子（巡迴的紀錄裡就有 `[LEAVE CAVE STAY IN]`、
  `[STAY LEAVE]` 這種選單）。探索器目前只會對 YES／NO 答 YES，不會挑地點。

  **2026-09-05 再一輪：挑到一個真的 bug，地圖多一張。**
  巡迴的紀錄裡一直有一條硬失敗
  `continue Pool transition resource 0x21 from ecl7/10: ECL session target
  block 0x1A is unavailable`——`ecl7/10` 這個組合本身就不成立（10 是 ecl4 的
  區塊），所以那是**鏡像跑到 session 前面**：`syncArchiveFromEventMachine`
  在腳本已經把 `6E12h` 寫成新值、`NEWECL` 還沒執行的空檔就更新了
  `eclArchive`，換 catalog 的判斷拿它當基準就會看到「選擇子等於現行值」
  而不換，session 卻還握著上一個 archive 的區塊。
  改成用 `eclSessionArchive`（session 手上真的是哪一份）判斷之後，硬失敗消失，
  走到的地圖從 18 張變 **19 張**（多了 GEO7/26）。ECL 區塊仍是 17 個。
  `TestArchiveSwapFollowsTheSessionNotTheMirror` 釘住這一條，並做過負對照
  （把判斷改回鏡像就會紅，錯誤訊息形狀相同）。

  **選單計數改成「每一格 ＋ 每一組選項」各自一個**：先前同一格上的
  YES／NO、地點選單與密碼輸入共用一個計數，彼此把指標推走。
  **這一步本身沒有讓覆蓋變多**（改前改後都是 17 個 ECL 區塊，A／B 各跑一次
  量過），留著是因為「每一組選項都輪得完」才是這個治具想做的事。

  **2026-09-05 再量：樞紐圖不是靠選單進去的。**
  `cmd/pool-world-cell-sweep` 加了「哪幾格會停在選單上、選項是什麼」之後
  掃一遍整包，看到的是另一回事：

  | 區塊 | 每一格的邊界 |
  |---|---|
  | `ecl6/25`、`ecl7/26`、`ecl8/27`、`ecl8/29` | **1024 格全部發 `2Dh CALL`** |
  | `ecl1/18` | 1024 格全部停在選單上 |

  所以那四張樞紐圖的地點是 **`2Dh CALL` 進去的**，不是選單選的；而 `ecl1/18`
  才是「整張圖都是選單」的那一種。先前寫的「探索器只會對 YES／NO 答 YES，
  不會挑地點」對 `ecl1/18` 成立，對那四張不成立。

  **再往下追一步：12 個缺的區塊沒有一個是被機制擋住的。**
  `cmd/pool-world-graph` 加了「間接 `NEWECL` 的餵值來源」之後，六個原本寫成
  `@6E79`／`@6E7D`／`@6E7F`／`@6E82` 的出邊全部解出來了（表位址、索引變數與
  表頭位元組都在 spec 101）。把它們接回去做廣度優先，**29 個區塊從區塊 0
  出發全部到得了**——原本看起來是孤島的 {3, 4, 5, 7} 掛在 6 底下，6 掛在 9、
  9 掛在 18。

  所以入口全部在；缺的主要是**走法**（區塊 1 另有一道主線閘門，見下面）：

  | 缺的區塊 | 從哪裡進去 | 要做什麼 |
  |---|---|---|
  | 19、1、28 | 區塊 25（西野外）| 走到野外座標 (10,9)、(12,31)、(3,32) |
  | 13 | 區塊 27（東野外）| 走到野外座標 (6,15) |
  | 2 | 區塊 18 或 26 | 走出圖 18 的邊界，或野外 26 的 (11,28) |
  | 15 | 區塊 29 或 2 | 走出圖 29 的北邊界（表 `@AFCA` 的第 2 格）|
  | 9 → 6 → 3 → {4, 5} → 7 | 區塊 18 | 一條五層深的邊界鏈，每一層都要走出上一張圖 |

  野外那四個座標的來源是地點分派表：地點編號進 `ON GOTO` 之後，圖 25 的
  地點 2／5／7 分別 `NEWECL 19`／`1`／`28`（「A LARGE DRAGON… DISAPPEARS INTO
  A CAVE」、「YOU HAVE REACHED THE BUCCANEER BASE」、「THE RIDERS ESCORT YOU
  INTO THE OUTPOST」），圖 27 的地點 2 `NEWECL 13`。完整對照表在 spec 105。

  而圖 25 本身要從圖 26 的西邊界跨過去：`49C3 == 2` 時往西／西北就會被搬到
  `49C3 = 14`、`6E12 = 6`、`NEWECL 25`（spec 105 的拼接表早就寫了這一條）。

  **野外的走法換掉了，最西邊那一張圖走得到了。** 野外位置不在 GEO 格子上，
  所以先前是「輪流轉向再往前走」。但 spec 105 早就證明每一步是「引擎在載入
  的 GEO 上走一格，ECL 同時把野外座標推一格」——**兩者的差因此是常數**，
  野外其實是一張把 16×16 的 GEO 平鋪上去的迷宮，可以直接做 BFS，判定用
  遊戲自己走路的那一支。

  探索器照這個走之後（`cmd/pool-game/wilderness_explore_test.go`）：

  - `TestTheWildernessWalkReachesTheWesternSheet` 從東邊的登陸點往西跨兩次
    圖，站上圖 25 的 (12,31)——原版在那一格印「YOU HAVE REACHED THE
    BUCCANEER BASE」。**亂走在同樣的預算內連圖 26 都跨不過去。**
    負對照：把量到的位移換成 (0,0) 就走不出圖 27。
  - 世界巡迴多走到 GEO5/4（圖 25 的地形）與 **ECL 區塊 2**，區塊數仍是 17
    ——多的是 2，少的是 17。**這一步沒有把數字推上去**，推上去的是「最西邊
    那一張圖進得去」這件事本身。

  四件事會讓「照著方向一直走」失敗，都寫進 spec 105 了：跨圖會重算位移
  （換一組位移就是換一座迷宮；但**不是哪一列決定的**——從圖 27 西緣七個不同
  的列跨到 26，位移一律是 (10,4)，只有落在哪一列不同）、地點格踩到就被腳本帶走、
  路線要整條記住不能每步重算、**踏出邊界那一步一樣會被牆擋**——最後這一條
  是實測抓到的死循環：有一趟站在圖 26 的 (2,23) 對著西邊的牆按了七千多次。
  修法是邊界格先問 `CanMoveDungeonWrapped`，再配一個「連續 8 個 tick 沒動就
  換目標」的看門狗。修完之後七種入口列有三種走得到圖 25（先前只有一種）。

  **每一個野外地點都有兩成以上的位移走得到**（`workplace/wildreach` 把
  16×16 種位移乘上每一種入口列全跑過，圖 25 是 8192 種）。但**一組位移能走到
  的很少**：實測那一趟進圖 25 的位移是 (15,14)，那組位移下走得到的只有
  (13,30)、(13,31)、(13,32) 三格。

  **「跨出去再跨回來換一座迷宮」行不通**，這一輪試過了：往東與往西的推移大小
  相同方向相反，來回剛好抵銷。上限從 6 拉到 24 實測走到的地點一格都沒多。
  真正決定迷宮的是**當初從哪一條航線下船**——三條量到的落點與位移分別是
  EAST 圖 27 (9,29) 位移 (6,4)、WEST 圖 26 (7,29) 位移 (8,4)、
  BAY 圖 26 (13,27) 位移 (2,6)。

  **三條航線的位移鏈量完了**，結論是明確的：

  | 航線 | 位移鏈 | 圖 25 上走得到的地點 |
  |---|---|---|
  | EAST | 27 (6,4) → 26 (10,4) → 25 (15,14)，落在 (12,31) | 五個 |
  | WEST | 26 (8,4)，落在 (7,29) | 走不到圖 25 |
  | BAY | 26 (2,6)，落在 (13,27) | 走不到圖 25 |

  EAST 那五個（(8,24)、(12,30)、(11,31)、(12,31)、(12,32)）**正好等於**
  `workplace/wildreach` 列舉出來的結果，所以探索器在那條航線上已經走到能走的
  全部。**區塊 19（(10,9) 龍穴）與 28（(3,32) 前哨站）在目前三條航線下走不到
  ——這是量出來的，不是還沒試夠。**

  **換口袋靠進出區域，不是靠走路。** `ecl6/25` 在 `A48A..A4C0` 成對寫
  `DS:4A18h`／`4A19h`（四組：(15,3)、(0,2)、(0,9)、(11,15)），接著
  `A4F4 LOAD FILES 25,2,255` 載區域圖 GEO6/25、`A4FBh` 把那一對擺回
  `C04B`／`C04C`，才 `NEWECL 25`。`49C3`／`49C4` 沒有跟著動，**位移因此整個
  換掉**。圖 26 有三組、圖 27 有四組。

  這也解釋了那兩張圖的兩種身分：`LOAD FILES 4,4,0` 載的 GEO5/4 是地形
  （在上面走），`LOAD FILES 25,2,255` 載的 GEO6/25 是區域圖（62 格 42 區，
  靠那四組座標各自進去）——先前把 geo6/25 的形狀當成「樞紐圖」是看錯了層次。

  **選哪一組是擲的，入口是一句原文**：

  ```
  A425  'YOU HAVE FOUND A SMALL DARK CAVE.  WILL YOU ENTER?'
  A44E  HORIZONTAL MENU
  A460  ON GOTO @6E79 → [A46C 進洞, ADC9 不進]
  A46C  SAVE 255 → @4A9E     ; 進區域圖；入口 0 第一行比的就是這個旗標
  A472  RANDOM 3 → @6E7E     ; 四個落點擲一個
  ```

  回野外走另一條（`AB2A LOAD FILES 4,4,0`、`AB31 SAVE 0 → @4A9E`、
  `AB3B NEWECL 25`）。所以**答應進洞再走出來，位移就重擲一次**——這才是換
  口袋的方法，也是 (10,9) 與 (3,32) 唯一的路。

  **順著這條線抓到一個真的 bug。** `moveForward` 原本只看「腳本區塊是不是
  野外」就做野外那一套（把方向寫進 `033Dh`、清 `6DC9h`、事後把 `00FBh`／
  `00FCh` 收回 `49C3`／`49C4`）。在區域圖裡入口 0 整段不跑，那兩格停在上一次
  野外移動的舊值，於是**每走一步都把隊伍的野外位置蓋成舊值**——進洞走一圈再
  出來，人會被丟回進洞前那一格，口袋永遠換不掉。判斷改成
  `inWildernessOverland()`（區塊是野外**而且** `4A9E != 255`）。
  `TestAnAreaMapStepLeavesTheWildernessPositionAlone` 釘住，負對照把判斷改回
  只看區塊，野外座標當場從 (8,29) 變成灌進去的 (99,99)。

  探索器改成「在野外地形上看到 `ENTER`／`ENTER CAVE` 就選它」（只認 `ENTER`，
  所以不會誤選登陸點的 `TAKE BOAT` 回城）。

  **然後把整條鏈接起來了，巡迴從 17 個區塊變 19 個。** EAST 那個口袋裡走得到
  的五格是地點 3／4／4／5／4，其中只有地點 5（(12,31) 海盜基地）帶 `ENTER`
  選單——而那一支開頭就是兩道閘門：

  ```
  9D87  COMPARE @4A8C 255 → IF <> → 跳走
  9D92  COMPARE @4AA9 0   → IF <> → 跳走
  9DA5  'YOU HAVE REACHED THE BUCCANEER BASE…'  選單 [ENTER LEAVE]
  ```

  `4A8C` 是誰寫的也找到了：**`ecl3/8 ACFAh`，印出「THE HEIR TO THE HOUSE OF
  BIVANT MUST BE RESCUED. WE WILL PAY GENEROUSLY FOR HIS SAFE RETURN.」之後
  才寫 255**——也就是在城裡接下「拯救比凡特家繼承人」的委任。閘門是
  `4AA9 == 0`（那個委任還沒推進）。

  巡迴加上第四種主線狀態 `{0x4A8C: 255}` 之後，走到的區塊變成 **19 個**
  `[0 1 2 8 10 11 14 16 18 19 20 21 22 23 24 25 26 27 29]`，地圖 18 張
  （多了 GEO6/1）。多的正是**區塊 1（海盜基地）與區塊 19（龍穴）**——鏈條與
  預測完全吻合：接委任 → (12,31) 給 ENTER → 進海盜基地 → 回來時位移重擲 →
  走到 (10,9) 的龍穴。

  **再一步：走去地點的路上也不能路過 `avoid` 的格子，區塊數 19 → 20。**
  `explorePlan` 已經改成連路過都不行，但「走進地點」那一條還用
  `planToCells`（不看 `avoid`），等於白擋。兩條都改成 `planToCellsAvoiding`
  之後多走到 **區塊 15（GEO2/15）**，地圖 18 → 19 張。時間 153 → 321 秒
  ——那是因為隊伍真的把圖走完了，不是繞路。

  **但這個禁令在主線鎖亮著的時候要關掉。** 第一版沒關，索寇要塞那一段當場
  垮掉：`TestSokalKeepOpensTheOtherBoatRoutes` 從 7 張地圖／7 個區塊掉到
  **2 張／3 個**，隊伍在 GEO4/21 上只踩到 19 格就宣告走不動。原因是鎖亮著
  的期間 `avoid` 裡塞的是**暫時擋起來的換圖點**（`holdMainlineExits`），
  那些是要路過的，不是「踩到就換走」。用 `heldBack` 當例外不夠（同一批格子
  兩邊都在），改成鎖亮著就整個關掉禁令才對。改完之後索寇那一條**反而更好**：
  8 張地圖／9 個區塊。

  **最後一個漏洞：走去出口的那一段也在路過換區格。** 三個規劃入口（逐格探索、
  走進地點、走去出口）裡前兩個補過了，第三個沒有——結果是
  `chooseAreaExit` **挑對了出口也沒用**：實測 GEO1/18 三次都挑中 (4,0) 朝北
  （那正是缺的區塊 9 那一支），三次都在半路被別的格子換走，落點是 GEO8/29 的
  (1,11)。三個都補上之後 **區塊 9 走到了**（GEO2/9 進到地圖清單）。

  巡迴同時多了一種主線狀態的組合 `{4A8C: 255, 4A77: 4}`——`4A77 = 4` 是
  「近塔已清、遠塔還在」，那是區塊 6 那場遭遇的前提。**它到目前為止沒有生效**
  （區塊 6 還沒走到），留著是因為那是文件裡唯一的入口條件。**單獨加成第五種
  狀態反而更差**（區塊數 20 → 19，少掉 19）：34 趟分五種每種只剩七趟，
  稀釋掉的比換到的多，所以是併進既有那一種。

  **再一個：`exitUses` 沒有跨趟共用。** 巡迴每一趟都新建一份，所以
  `chooseAreaExit` 每一趟都從同一個出口開始輪——實測 GEO4/2 六次全挑北邊那
  兩個，而缺的**區塊 15 在東邊**（`ecl4/2 996Dh` 的 `ON GOTO @C04D`：
  0 北→區塊 18、**1 東→區塊 15**、3 西→野外 26 的 (11,28)）。改成跨趟共用
  （與 `menuTurn` 同一個理由）之後 **9 與 15 同時進來**。

  走到 **21 個 ECL 區塊、20 張地圖**：
  `[0 1 2 8 9 10 11 14 15 16 18 19 20 21 22 23 24 25 26 27 29]`。

  **還缺 7 個**：`[3 4 5 6 7 13 28]`——3／4／5／7 全部掛在區塊 6 後面，
  6 掛在區塊 9 的遠塔遭遇；13 在東野外 (6,15)；28 在西野外 (3,32)。
  （**這一段的診斷後來被推翻**：6 不是掛在塔的遭遇，是掛在斯托亞諾夫城門的
  過路費；3、4、5、6、7 現在全部走得到了。見下面 2026-09-05 那一段。）

  **兩個野外的量清楚了，都要換口袋。** 用 `workplace/wildreach` 對實測的位移
  問一次：

  | 航線落點 | 位移 | 那個口袋裡走得到的地點 |
  |---|---|---|
  | 圖 27 的 (9,29)（EAST） | (6,4) | 只有 (11,8) 與 (9,29) —— **(6,15) 走不到** |
  | 圖 25 的 (12,31) | (15,14) | (8,24)、(12,30)、(11,31)、(12,31)、(12,32) —— **(3,32) 走不到** |

  **同一張圖的位移會隨走法變。** 把 `4A8C` 先設成 255 再走同一條路，進圖 25
  的落點變成 (13,25)、位移 **(14,4)**，那個口袋走得到七個地點
  （(12,30)、(13,30)、**(2,31)、(3,31)**、(11,31)、(12,31)、(13,31)）——
  比 (15,14) 那個多，而且已經摸到 (3,32) 隔壁那一排，但還是不含 (3,32) 本身。
  所以 28 不是「永遠到不了」，是還沒擲到含它的那一組。

  所以 13 與 28 都不是走法問題，是**要先換一次口袋**：圖 27 那個口袋裡的
  (11,8) 是廢墟城堡，帶 `ENTER` 選單（`ecl8/27 9BE6h`）→ 區塊 16 → 回來時
  位移重擲。探索器已經會挑 `ENTER`，也走得到 (11,8)，但回來之後還沒走到
  (6,15)。

  試過沒效果的：**位移一變就把放棄過的目標放回去**（理由對：換一座迷宮之後
  走不到的可能就走得到）。區塊數一樣 21、時間 183 → 270 秒，退回。

  **區塊 6 那條追到底了：塔的遭遇格在一個進不去的口袋裡。**

  遭遇掛在哪幾格解出來了——`ecl2/9 9D7Bh` 用 `AND @C04F 1Fh → @6E79` 取地形碼
  再 `9D84h ON GOTO` 十四支，**第 10 與第 11 支都是 `A3FC`**（塔那一段）。
  拿 `cmd/pool-world-graph` 的 `cell_terrain` 對 GEO2/9 掃一遍，地形碼
  `&1Fh ∈ {10,11}` 的**只有兩格：(4,5) 與 (11,5)，都在第 2 連通區**。

  而從區塊 18 的北緣進來，落點是 **(4,15)，第 24 區**。實跑驗過兩件事：

  - 直接規劃到 (4,5)：六步，但**路線繞回 (4,0)**（南邊繞回北邊），
    一踏上去就換到區塊 2 —— 走不到塔。
  - 繞開所有邊界格再規劃：**兩格都是 0 步**，根本沒有路。

  也就是說 (4,5)／(11,5) 只能經過 (4,0)／(11,0) 進去，而那兩格踩到就走人。
  **所以 remake 現況下區塊 6（連同 3、4、5、7）走不到。**

  **再量一次，那一步其實是「一步之內連跳兩次」，而且完全照著原始碼走。**
  站在 (4,15) 朝南踏一步，量到的是：位置 (4,15)→(4,0)、`C04C = 0`、
  `C04D = 2`、`6DD5 = 0`、block 2（GEO4/2）。拆開來是：

  1. 往南踏出 16×16 的框 → 前端設 `6DD5`；
  2. **區塊 9 的入口 0** 跑，`9C7F COMPARE @C04C 0` 比的是**移動前**的 15，
     不是 0，所以走 fall-through：`6E79 = 18`、`6E12 = 1` → `NEWECL 18`；
  3. 落在區塊 18 的 (4,0) 朝南，**區塊 18 的邊界分派**第 2 支
     （`ecl1/18 99FAh`：`6E12 = 4`、`NEWECL 2`）→ 區塊 2。

  `CALL @C01E` 會把 `6DD5` 清掉（spec 104），所以事後看是 0——不是沒觸發。
  (4,0) 的地形碼是 **0**，走的是分派表第 0 支 `A0A1`（`SPRITE OFF`／
  `CALL @2C90`／`EXIT`），**確實沒有換區塊**；換區塊的是邊界那一支。

  **所以區塊 6 的條件很精確：要站在 GEO2/9 的 y = 0 上，然後往北踏出去**
  （`9C7F` 那一支要 `C04C == 0`，而 `C04C` 取的是移動前的位置）。而 y = 0 那
  一排只能從圖內（第 2 連通區）走上去——從 (4,15) 往南繞回去的那一步，
  在 `C04C` 還是 15 的時候就已經被判成離開了。

  結論不變但更硬：**區塊 6 要先進到 GEO2/9 的第 2 連通區，而從區塊 18 進來
  一定落在第 24 區。**

  「有沒有別的路進區塊 9」也查完了：解碼後的 `NEWECL` 圖（spec 101）裡指向
  區塊 9 的只有兩處——`ecl1/18 99ECh`（立即值）與 `ecl5/6 9B95h`（變數，
  立即值 3 與 9），而**區塊 6 本身就是從區塊 9 進去的**，所以那是回頭路，
  不是新入口。區塊 9 的唯一入口就是區塊 18 的北緣。

  而繞回也不是路：從 (4,15) 往南繞回 (4,0) 那一步，`setMapExitFlag` 會設
  `6DD5`，接著區塊 9 的入口 0 比的 `C04C` 是移動前的 15 → 直接送回區塊 18。
  **原版跑的是同一份 ECL，所以這一段在原版也一樣。**

  剩下的可能只有兩個，兩個都要對拍才分得出來：

  1. 原版的 GEO2/9 走起來**不是**分成兩塊（remake 的可通行判定或地圖解讀有
     出入）；
  2. 原版進區塊 9 時隊伍不是落在 (4,15)（`CALL @C01E` 的移動語意有出入）。

  | 缺的 | 路 |
  |---|---|
  | 28 | 同一條野外鏈，還沒擲到含 (3,32) 的那組位移 |
  | 13 | 東野外圖 27 的 (6,15) |
  | 9→6→3→{4,5}→7 | 走出 **GEO1/18 的北邊界**：`99D4 ON GOTO @C04D` 第 0 支是 `6E12 = 2`、`NEWECL 9`（另三支：東→區塊 29、南→區塊 2、西→野外 26 的 (11,28)）|

  最後那一條是六個區塊的鏈，最值得先打。GEO1/18 有 248 格可走、只有 3 個連通
  區，八個邊界出口全在同一區（含 (4,0)N 與 (11,0)N），所以**不是走不到**。

  **查過了：探索器一次都沒試過北向出口。** 巡迴的紀錄裡它在 GEO1/18 上走遍
  內部（(14,8)、(9,6)、(5,6)、(5,10)、(2,7)、(4,8)、(8,9)、(11,11)…），
  但每一趟的換圖點都是 **(15,4) 往東 → GEO8/29**，來回而已。原因是**它是走
  路走出去的，不是挑出口挑出去的**：`chooseAreaExit`（會照 `exitUses` 輪流
  挑八個方向）只在「這一張踩完了」之後才跑，而規劃器逐格踩的時候就先踩到
  東邊那個邊界格、當場換走了。

  **改了：踏上新的一張圖就把它的邊界出口格整批放進 `avoid`**，走路踩不到，
  離開一律走 `chooseAreaExit` 那條路（它與它的走位都不看 `avoid`）。
  區塊數不變（19 個），但**巡迴的時間從 407 秒掉到 183 秒**——先前有一半的
  預算花在「踩到邊界格被換走、再繞回來」。

  **但巡迴還是沒開到區塊 9。** 再量一次換圖點：34 趟裡只有 **2 趟**走到
  GEO1/18，而且兩趟都是從野外（GEO5/5 的 (11,28)）進去的——走完野外預算也
  差不多用完，沒有機會在那張圖上輪出口。卡的不是出口挑法，是**那張圖太晚才
  到得了**。

  **那條路本身通不通，直接跑一次就知道了：通。**
  `TestTheNorthEdgeOfBlockEighteenLeadsToBlockNine` 在完整的前端底下走完
  「東邊登陸 → 跨到野外圖 26 → 走到 (11,28) 城西緣 → 選 NORTH → 區塊 18 →
  從 GEO1/18 的北緣踏出去 → **區塊 9（GEO2/9）**」。所以那六個區塊的鏈第一段
  沒有問題，缺的是巡迴的預算分配。

  **但第二段（9 → 6）走不通，原因量出來了。** 兩條路都試過：

  1. **邊界那條走不到。** `ecl2/9` 入口 0 是「`C04C == 0`（站在 Y=0）就
     `NEWECL 6`，否則回區塊 18」。但從區塊 18 進來的落點是 GEO2/9 的
     **(4,15)，屬於第 24 連通區**，而兩個北緣出口 (4,0)N／(11,0)N 在
     **第 2 區**——走不到。（geo2/9 是 88 格分成 30 個互不相連的區。）
     實跑時「走到 (4,0)」是**繞回去的**：(4,15) 往南繞回 (4,0)，而那一格的
     事件把隊伍送去區塊 2，不是入口 0 的邊界那一支。
  2. **遭遇那條沒觸發。** 區塊 6 的另一個入口是 `ecl2/9 A558h` 的
     `ENCOUNTER MENU`（前一句是 'MONSTERS ARE CHARGING TOWARD YOU FROM THE
     FAR TOWER.'）：`A57E ON GOTO @6E82` **只有兩支**（`A0A1`、`A5A4`），
     索引落在兩支之外就 fall-through 到 `A58E`——擺位置到 GEO (4,15) 朝北，
     再 `A5A0 NEWECL 6`。從落點把走得到的範圍全走過、遭遇選單分別挑
     FLEE／PARLAY／WAIT，三種都沒觸發到那一場。

  **那場遭遇的閘門讀出來了：兩座塔的旗標。**

  ```
  A444  AND @4A77 8 → @6E81 ；IF = ；OR @6E7A 2 → @6E7A
  A457  AND @4A77 4 → @6E82 ；IF = ；OR @6E7A 1 → @6E7A
  A476  ON GOTO @6E7A → [A0A1(0) A4EC(1) A51E(2)]，索引 3 落到 A485
  ```

  也就是 `DS:4A77h` 的第 2 位（值 4）與第 3 位（值 8）各代表一座塔，湊出來的
  `@6E7A` 決定演哪一段：0 什麼都沒有、1「MONSTERS THUNDER DOWN THE STEPS OF
  THE TOWER NEXT TO YOU.」、**2「MONSTERS ARE CHARGING TOWARD YOU FROM THE
  FAR TOWER.」**、3 兩座一起。而**只有 2 那一段的遭遇會通到區塊 6**。
  `4A77` 的十個寫入點全部在 `ecl2/9` 自己裡面（`A2BF`、`A3F2`、`A60E`、
  `A689`、`A7AE`、`AABB`、`AD9D`、`B014`、`B145`、`B14F`），都是
  `OR @4A77 → @4A77`——也就是**在區塊 9 裡清塔才會設起來**。

  所以區塊 6（連同後面的 3、4、5、7）的完整條件是：進區塊 9 → 清掉其中一座塔
  讓 `@6E7A` 變成 2 → 那場遠塔遭遇 → 遭遇結果落在 `A57E ON GOTO` 兩支之外
  （fall-through）→ `NEWECL 6`。下一步是查那十個 `OR` 各自掛在哪一格、以及
  哪幾格在落點那個第 24 連通區裡。

  這一輪還量了三招，**三招都沒有把數字推上去**，記下來免得下一輪重試：

  | 試的 | 結果 |
  |---|---|
  | 把「先回頭踩已知的換圖點」與「挑用得最少的出口」對調順序 | 19 個、154 秒對 183 秒。沒換到覆蓋，卻動到「拿到船票之後回頭再用碼頭那一格」那條刻意的行為 → **退回** |
  | 規劃器連 `avoid` 的格子都不路過（先前只是不當目標）| 19 個、153 秒對 183 秒 → **留著**，換區的格子是「踩到就換走」，路過等於換走 |
  | 每一趟的預算從 20 萬拉到 50 萬 | 19 個、397 秒對 153 秒 → **退回** |
  | 戰術地圖吃掉 6 萬個迴圈之後遭遇選單一律選 FLEE | 19 個、137 秒對 153 秒 → **退回**（省時間但沒換到覆蓋，而且會少跑那些區域的戰鬥）|
  | 規劃器一步都不繞回（理由：繞出 16×16 會被 `setMapExitFlag` 設成離開）| 索寇要塞垮成 2 張圖／3 個區塊、巡迴 21 → 20 → **退回**。**前提是錯的**：繞回確實會設 `6DD5`，但那不等於換圖——要看該區塊的入口 0 對那個朝向怎麼處理，而 GEO4/21 就是**要靠繞回才走得通** |
  | 把「踩到就被換走的格子」跨趟記著，下一趟一開始就繞開 | **19 個掉到 5 個**（只剩 GEO2/20、GEO3/0、GEO4/21）→ **退回**。碼頭與城區↔貧民窟那幾格也在裡面，記著等於把隊伍鎖在城區 |

  **GEO1/18 那條線這一輪追到底了，結論是治具改不動它。** 再量一次：隊伍在
  那張圖上是被**內部格子的事件**拉走的（離開時站在 (0,4)／(1,4)，朝東），
  不是走到邊界——`boundaryExitKeys` 擋得住邊界，擋不住內部。所以刻意挑出口的
  `chooseAreaExit` 在那張圖上**一次都沒輪到**。而把那些格子跨趟記起來會把
  碼頭一起鎖掉（上表最後一列）。要繼續得換角度：`ecl1/18` 是「1024 格全部停
  在選單上」的那一種，先讀出哪幾格會 `NEWECL` 才有得談。

  **預算那一招的失敗說出真正的瓶頸**：多數趟的結束理由是「走完預算」，而走到
  的步數只有 36～2566 步——20 萬個迴圈幾乎全花在**戰術地圖**上（實測一趟
  199476／200000）。拉大預算只是讓同一場架跑更久。要讓探索器走更遠，得先讓
  戰鬥便宜下來，不是給更多預算。

  **剩下的不全是走法。** 圖 25 的 (12,31) 站上去了，但那一支要
  `4A8C == 255` 且 `4AA9 == 0` 才會 `NEWECL 1`（`9D87h`／`9D92h`）——
  **區塊 1 另外還有主線閘門**，不只是走得到。區塊 19（(10,9) 龍穴）與 28
  （(3,32) 前哨站）那兩支沒有這種比對，是純粹的走法問題。下一步是讓跨圖的
  列輪得更廣（位移換掉之後迷宮就換一座），把 (10,9) 與 (3,32) 納進走得到的
  範圍。

  （掃描器記到的選單是**下限**：只有 VM 已經解出選項的邊界才記得到，
  `ecl1/18` 那 1024 格的 `Result.Menus` 是空的。整包只記到兩格——
  `ecl7/23` 的密碼門 `[GIVE THE PASSWORD, FORCE THE GATE OPEN]`。）

  **2026-09-05：區塊 6 的路走通了，靠的是修掉一個真的 bug。**

  先前這一段的診斷（連通區編號、「塔的遭遇是入口」）有錯，正確的機制寫在
  [spec 101](docs/spec/101-world-transition-graph.md)。重點：

  - GEO2/9 是**斯托亞諾夫城門**，區塊 6 是**瓦傑沃城堡**。
  - 這張圖南北兩半在 GEO 上真的不通：不繞邊界重算，(4,15) 與 (11,15) 都只
    走得到 **88 格，全在第 9～15 列**；把「鎖住可撬」的門全開也只有 104 格。
    （先前寫的「落點在第 24 連通區」是錯的，但「走不到」的結論成立。）
  - **塔的遭遇不是入口，是出口**：它掛在 (4,5)／(11,5)，那兩格在北半邊。
  - 真正的入口是**付過路費**：地形 9 的格子拿到商人的馬車——遭遇選單選
    PARLAY 花 250 金買，選 COMBAT 就「YOU QUICKLY KILL THE TRADER AND GRAB
    HIS WAGON.」，兩條都接到 `B014 OR @4A77 64`——再到地形 5 的格子付 15 金
    → `ecl2/9 AC4Bh`
    「THE BUGBEAR TAKES THE MONEY AND THE GATE OPENS. YOU PROCEED THROUGH.」
    → 腳本把 `C04B`／`C04C` 寫成 (8,5)，直接把隊伍放到北半邊。
    路上不能先踩地形 3（熊地精衛兵），那會把 `@4A78` 推到 3，之後索賄那一支
    必定翻臉。

  **卡住的是 `consumeInitialTransitionResources` 把 `Writes` 丟掉。**
  它每吃掉一個資源邊界（`2Dh CALL`、`PICTURE`、`SPRITE OFF`…）就
  `result = next`，前一段的 `Writes` 跟著消失；而城門那三個 `SAVE` 剛好夾在
  `PICTURE 255` 與 `CALL 2C90h` 中間。症狀很難認：**記憶體裡 `C04B/C04C`
  已經是新座標，隊伍卻站在原地**。改成把 `Writes` 累積下去之後，實跑量到
  「(6,11) 付錢 → (8,5) → 走北緣 → **ECL block 3、GEO5/3**」。
  `TestPayingTheTollAtStojanowGateOpensTheCastle` 釘住整條路。

  這個 bug 不只影響城門：**任何在資源邊界之間傳送隊伍的腳本都會被吃掉。**

  **但巡迴還是走不到，卡在 `@4A00`。** 把「先去馬車格、再去索賄格，路上避開
  地形 3」寫進巡迴實測過：區塊數不變（21），而診斷指出原因很精確——
  隊伍踏進 GEO2/9 的時候 **`4A00` 已經是 1**，而馬車那一支的前置是
  `ADBD COMPARE @4A00 0 / IF <> / EXIT`，所以站上 (4,14) 三次都沒有反應，
  最後只能去打守衛（`4A77` 變 48、`4A78` 變 3），整條路就斷了。
  沒有馬車也沒有第二條路：投降那一支（`A7B8`）是被搶光再丟回 y = 14，
  閘門那兩格要 `@4A77 & 1`，而那個位元只能從**北半邊**朝南設起來。
  **治具因此退回**（沒換到覆蓋就不留）。

  `4A00` 是 class 0（`4900h..4CFFh`，spec 106），照那份規格是**跨區塊持續**
  的。問題是同一個位址被好幾個區塊當自己的變數用：`ecl4/10`（瓦海登墳場）
  `9AD6 ADD 1 @4A00`、`ecl3/11 9AE6 ADD 1`、`ecl4/21`／`ecl2/20` `SAVE 255`，
  而 `ecl2/9` 拿它當「馬車出現過沒有」。巡迴走過墳場，`4A00` 就變成 1。

  **算過了，原版也一樣，不是 remake 的偏差。** class 0 的取法
  `[4933h] + 6E00h + addr*2` 在 16 位截斷之下，`4900h` 對到偏移 `0000h`、
  `4CFFh` 對到 `07FEh`——正好鋪滿 `[DS:4933h]` 指到的那塊 **0x800 位元組隊伍
  記錄**，而它是全域初始化清一次的（spec 009 的 `overlay-11 026Dh`），
  不隨換圖重設。所以「先進墳場就關掉城門的馬車」在原版也成立。
  推導寫在 spec 106。

  於是巡迴要走到 3／4／5／6／7 的條件是**先走城門再走墳場**——那是走訪順序，
  不是機制缺口。要推的話得讓巡迴知道「某些地點有先後」，而現在的探索器沒有
  這個概念。

  **城門後面五個區塊全部走到了。** `TestTheCastleBehindStojanowGateHasContent`
  走完城門之後在城堡裡掃：**ECL block `[3 4 5 6 7]`、地圖 GEO5/3、GEO5/4、
  GEO5/6、GEO5/7 四張**。

  關鍵是走法要**逐格加四個朝向**，不能只踩過每一格——城堡這幾張圖的出口是
  朝向敏感的：

  | 在哪 | 朝向 | 去哪 |
  |---|---|---|
  | GEO5/3 (15,4) 或 (15,3) | 東 | 區塊 4（GEO5/4 (0,4)）|
  | GEO5/3 (11,15) 或 (3,15)、(4,15) | 南 | 區塊 6（GEO5/6 (11,0)）|
  | GEO5/3 (4,8) | 北 | 區塊 5（上樓）|
  | 區塊 5 裡「YOU GO UPSTAIRS.」 | — | 區塊 7（GEO5/7 (5,6)）|

  出邊的表是 `ecl5/3` 入口 0 的 `GETTABLE @9AA6[@C04D]`，內容
  `[04 04 06 06]`（北、東、南、西）。四個方向輪替起點各掃一趟再取聯集：
  規劃器先往哪邊走，決定先撞到哪個出口，而出口一走就換圖，後面那段就看不到。

  所以走得到的聯集從 21 個變 **26 個**：
  `[0 1 2 3 4 5 6 7 8 9 10 11 14 15 16 18 19 20 21 22 23 24 25 26 27 29]`，
  **（後來全部走到了，見下面 2026-09-05 那幾段。）還缺 3 個**：`[13 17 28]`——都是野外換口袋的老問題（13 在東野外 (6,15)、
  17 是野外 26 的游牧民營地 (12,11)、28 在西野外 (3,32)），與城堡這條路無關。
  （13 後來走到了，見下面 2026-09-05 晚那一段，聯集因此是 **27 個**。）

  **這兩個又量細了一輪（`workplace/wildoffsets`）。** 把 256 種位移全跑一遍：
  從東野外的登陸點 (9,29) 走得到 (6,15) 的位移有 **121 種**，從西野外的
  (12,31) 走得到 (3,32) 的有 **97 種**——所以不是結構上到不了，是位移不對。

  卡住的地方很具體：

  - 船的登陸點給的位移是 **(6,4)**，那個口袋裡只有 (11,8) 與 (9,29)。
  - 重擲位移的唯一機制是**區域圖**（`ecl8/27 A20Dh`「YOU HAVE FOUND A SMALL,
    DARK CAVE. WILL YOU ENTER?」→ `A255 RANDOM 3` → 四組 `4A18/4A19`：
    (0,3)、(6,15)、(12,15)、(11,0)）。而那個洞穴就是**地點 1** 那一叢
    (6,14)、(7,14)、(5,15)、(7,15)、(6,16)、(7,16)——**全部緊鄰 (6,15)，
    也就是全部在同一個進不去的口袋裡**。
  - 口袋裡另一個地點 (11,8) 是廢墟城堡（區塊 16）。它的出口也是「地形 ×
    朝向」的表（`@9B00`／`@9B0C` 十二組），逐格踩挑不出來——用「逐格 × 四
    朝向」硬掃十分鐘還沒掃完（那張圖的戰鬥很貴），改成靜態解表就秒解，
    見下面。
  - 跨圖來回不換位移：實測 27 是 (6,4)、26 是 (10,4)、25 是 (15,14)，
    出去再回來會抵消（先前已記）。

  **廢墟城堡出得來了，位移確實會換，但換到的那一組還是不夠。**

  出口是靜態解出來的（同一個「地形 × 朝向」的形狀）：`ecl8/16` 的入口 0
  逐組比 `@9B00` 地形 × `@9B0C` 朝向十二組，命中就查 `@9B18` 的群組再走
  `9AEEh ON GOTO` 四支。第 4 組（**地形 15、朝南**）是群組 1 → `9CB9h` →
  `NEWECL 27`，而 GEO 區塊 16 上地形 15 只有 **(8,15)** 一格。
  下層 GEO8/30 沒有那個地形，要先走群組 2／3 的「往上」格
  （地形 17..23：(13,1)N、(10,4)E、(4,5)N、(1,7)N、(4,15)S、(7,10)W、(10,8)N）。

  實跑驗過：進廢墟城堡再從 (8,15) 朝南出來，位移從 **(6,4) 變 (13,7)**，
  而且**每一次都一樣**（不是隨機）。連做 26 次，26 次都是 (13,7)。

  (13,7) 確實打開了新的一片：從 (11,8) 走得到 159 格，含地點 1 的
  (7,14)、(7,15)、(7,16)（(6,4) 那組只有 (11,8) 與 (9,29)）。但**站上地點 1
  什麼事都沒有**——因為那不是洞穴。

  **洞穴是隨機事件，不是地點**：`ecl8/27 9EA7h RANDOM 19` 中 0，再
  `A867h` 裡的 `RANDOM 20` 中 20，才跳 `A20Dh`「YOU HAVE FOUND A SMALL,
  DARK CAVE. WILL YOU ENTER?」。所以重擲位移要靠在野外走到它自己出現。

  試了四萬次按鍵沒等到，但**那不是機率的量測**：隊伍只實際走過 27 格就卡在
  (5,3) 不動了。所以下一步是先弄清楚**野外的亂走為什麼會卡住**（那本身可能
  是 remake 的問題），再回頭量洞穴事件。

  順帶：`workplace/wildoffsets` 算過，從 (7,15) 出發，256 種位移裡有 220 種
  走得到 (6,15)；而洞穴重擲的四組 `4A18/4A19`（(0,3)、(6,15)、(12,15)、(11,0)）
  換算成位移之後有**三組**接得通。所以只要洞穴出現一次，機會很大。

  **2026-09-05 晚：區塊 13 走到了。**
  先前「四萬步等不到洞穴」是治具的問題不是機率的問題——隊伍卡在**打不完的
  戰鬥**上（漏了 `combatActive`／`tacticalPreview`，戰術地圖只按 Enter 全隊
  都不出手）。補上探索器的駕駛之後，洞穴**五百步內就出現兩次**。

  整條路量出來了（`TestTheWildernessCaveRerollReachesTheEasternOutpost`）：

  1. 在野外亂走 → `ecl8/27 9EA7h RANDOM 19` 中 0、再 `A867h` 的 `RANDOM 20`
     中 20 →「YOU HAVE FOUND A SMALL, DARK CAVE.」選 ENTER；
  2. 進去那張區域圖（GEO8/27）**一個邊界出口都沒有**，要在裡面亂走到
     `A7F7h`「YOU FIND A PASSAGEWAY GOING OUT.」選 LEAVE；
  3. 出來時 `A255h RANDOM 3` 從四組 `4A18/4A19` 挑一組寫回位置，**位移就換了**；
  4. 換到走得到的那一組就走過去。

  實測兩次重擲就成：位移 (6,4) → (7,10) → (1,9)，然後走到 **(6,15) → ECL
  block 13、GEO8/13**。整支 6 秒。

  **走得到的變 27 個**（後來 17 與 28 也走到了，是 **29 個，全部**），還缺 `[17 28]`：野外 26 的 (12,11)（游牧民營地，
  `ecl7/26 9C59h`，地點 1，要 `4A0F == 0`）與野外 25 的 (3,32)（前哨站）。
  兩個都是**同一個洞穴重擲機制**。

  這一輪把周邊都清掉了，只剩「在那張圖上待夠久」這件事沒解決：

  - **跨圖不慢**：走到「最西走得到的那一格」再往西踏一步（spec 105 的
    `49C3 == 2`），27 → 26 只要 **1.2 秒**（先前用通用 walker 要 134 秒）。
    26 → 25 在 (2,3) 跨不過去，換一列還沒試。
  - **亂走會誤入城裡**：野外的地點裡有「進城」「上船」，踩到就離開野外，
    回來的路很長。洞穴**不是地點**，所以亂走時可以把所有地點都繞開
    （`wildernessPlaceSet`）。
  - 改成「在兩個非地點的鄰格之間來回踏」也可以（洞穴是每一步都擲），
    但實測九分鐘還沒擲到——野外 26 的隨機表可能與 27 不同，還沒讀。

  **讀過了，三張圖的觸發條件完全一樣**：
  `RANDOM 19` 中 0 → `GOSUB` 一支 `RANDOM 20` → `COMPARE AND @6E79 20 @4A15 0`
  才跳洞穴（27 在 `9EA7h`、26 在 `A0E7h`、25 在 `A0B3h`）。進到那一支的條件也
  一樣：每步處理先用「列 → 該列的地點數」兩張表找地點，**找不到就跳過去**
  （27 是 `9AB3h COMPARE @6E79 5`，26 是 `9B16h COMPARE @6E79 9`）。

  **所以機率不是原因。而實測的結果是：野外 26 走 14032 步，一次洞穴都沒有。**

  野外 26 的每步腳本**是有在跑的**：兩百步裡 199 步有事件，文字是
  「YOU SURPRISE A GROUP OF」「YOU ARE SURPRISED BY A GROUP OF」
  「YOU SPOT A GROUP OF」——遭遇正常。閘門也都放行
  （`4A9E=0`、`4AB3=0`、`4AA2=0`），`49C3`／`49C4` 跟著走。

  剩下的差別只在洞穴那一支：兩張圖都是
  `RANDOM 20 → 6E79`，`COMPARE AND @6E79 20 @4A15 0` 成立才回傳「命中」
  （27 在 `A867h`、26 在 `AB15h`，兩段位元組結構相同）。
  野外 27 上五百步內跳兩次，野外 26 上一萬四千步零次——**這個落差還沒有解釋**。

  **解了，而且是我自己的治具在拒絕它。** 取樣 `@6E79`（`RANDOM` 的目的地）
  之後看得很清楚：野外 26 走 4000 步，第一擲中 0 的有 **198 次**（1/20，正確），
  `RANDOM n` 也確認是 `Intn(maximum + 1)`＝**回 0..n 含端點**。

  真正的原因是**每一張圖的隨機區域名字不一樣**：

  | 野外 | 問句 | 位址 |
  |---|---|---|
  | 25 | YOU HAVE FOUND A SMALL DARK CAVE. WILL YOU ENTER? | `A425h` |
  | 26 | YOU HAVE FOUND A SMALL WOODED GROVE. WILL YOU ENTER? | `A45Dh` |
  | 26 | YOU HAVE FOUND SOME RUINED HUTS. WILL YOU INVESTIGATE? | `A4A6h` |
  | 27 | YOU HAVE FOUND A SMALL, DARK CAVE. WILL YOU ENTER? | `A20Dh` |

  治具只認 `DARK CAVE`，所以在野外 26 上把 198 次機會全部答成 LEAVE。
  改成三種都認（`randomAreaPrompts`）之後，**一次重擲就走到游牧民營地**。

  **區塊 17 走到了**（`TestTheNomadCampOnTheMiddleWildernessSheet`，1.8 秒）：
  跨到野外 26（走到「最西走得到的那一格」再往西踏一步，一秒出頭）→ 位移
  (10,4) → 進小樹林再出來 → 位移 (1,13) → 走到 (12,11) 答 ENTER →
  **ECL block 17、GEO7/17**。

  **走得到的變 28 個**，只剩 `[28]`：西野外 25 的前哨站 (3,32)。

  路線本身沒有問題：跨圖通了（野外 26 的 (2,24) 往西 → 野外 25 的 (12,31)，
  位移 (15,13)），隨機區域的問句也認得了。擋住的是兩件事，兩件都解掉了：

  **一、一場打不完的戰鬥（真正的缺陷）：**

  ```
  卡住：block 25 在 [8 30]｜"Encounter: WILD BOAR ×5"
  戰術：回合 22526 行動者 5 我方 4 敵方 5 狀態 "BLOCKED"
  ```

  雙方都結束得了回合（`foeTurn` 最後一定 `endTurn`），但誰也打不到誰——野豬的
  五個方向都 `MovementBlocked`、繞路備案也找不到路，隊伍這邊探索器的駕駛走滿
  12 步就結束回合。**這一場永遠不會結束**，回合數一路往上加。換種子沒用
  （11／3／7 都停在同一格 (8,30)），選 FLEE 也沒用（那一場沒有遭遇選單，
  直接進戰術地圖）。

  加了**非原版的僵局安全閥**（`tacticalStalemateRounds = 50`）：盤面連續五十
  回合完全沒變（位置、生命值、狀態、倒地計時都一樣）就收場，走
  `finishCombat(CombatOngoing)` 把排好的遭遇清掉。指紋**一定要含生命值**——
  不還手的隊伍站著被打，位置與狀態都不會變，漏了就會把
  `TestPassiveCombatTerminates` 那條（不還手的隊伍要被打死）誤判成僵局。
  性質與同一個檔案裡的繞路備案一樣：原版怎麼收這種場面還沒讀。
  **使用者 2026-09-06 決定留著它**，所以這不是待決事項；原版那條讀出來之後
  再換掉，換的時候要連同這個安全閥一起拿掉，不要兩份真相並存。

  **二、前哨站有主線閘門**：`ecl6/25 9E4Ch COMPARE @4A98 255` 要成立，
  隊伍才是「新費蘭來的外交使節」。沒有它，站上 (3,32) 只會演
  「YOU ARE TRESSPASSING ON PRIVATE LAND」那一段（選單 `[STAY LEAVE]`），
  選哪一項都進不去；有了它才是
  「ARE YOU THE DIPLOMATIC ENVOYS FROM NEW PHLAN?」（選單 `[YES NO]`），
  答 YES 就 `NEWECL 28`。與區塊 1 的 `4A8C == 255` 同一類。

  `TestTheOutpostOnTheWesternWildernessSheet` 走完整條，1.4 秒。

  巡迴早就會報「戰術地圖卡住」的硬失敗，但先前只知道現象；現在那一場有了
  具體的重現，也有了收場的辦法。

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

- [x] **結局過場**（[spec 108](docs/spec/108-ending-cutscene.md)）。台詞、中譯與
  圖都接上了。`PROGRAM 8` 是 overlay-18 entry 1（`02A1h`）；三頁（3／6／4 行），
  一頁一個 ENTER，翻完讓 ECL 往下跑。
  圖是 `FINAL5.DAX` 的五層：區塊 1／3 兩張 120×120 底圖，加上 4（56×32）、
  5（16×48）、6（16×48）三張小圖。**位置在繪製常式的引數裡，X 與 Y 都是
  8 像素為單位**（Y 在 `0F1Fh` 先 `shl 3`），換算成像素是 `(0,88)`、`(104,72)`、
  `(0,72)`——三張都正好落在 120×120 的場景裡，這個自洽同時是單位的驗算。
  後三張畫不畫看 `[4937h] + 67Ch`，那是**隊伍人數**（spec 084／091 早就寫了）。
  `DS:4980h` 也解開了：overlay-11 `00AEh` 把它當出參推給 overlay-36 entry 15
  （`24EDh`）配置成 168×168 的圖片緩衝。
  **先前「靜態讀不到、只剩 DOSBox oracle 一條路」是掃描面有洞**：當時只掃
  `mov ds:4980h, 暫存器／立即值`，而 Pascal 的出參寫法是把位址推進去，
  被寫的變數不會出現在任何 `mov 目的` 的位置上。改掃位元組樣式就看得到。
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

- [x] **平台驗收的 workflow 進不了共用 engine——決定不動 CI**（2026-09-03 實跑
  `gh workflow run platform-smoke.yml` 量到）。`build (macos-14)` 與
  `build (windows-latest)` 都掛在 **`check out the shared engine`** 那一步：
  `actions/checkout` 去抓 `wicanr2/golden-box-remake-engine`，而
  **兩個 repo 都是 private、這個 repo 一個 secret 都沒有**，預設的
  `GITHUB_TOKEN` 只有本 repo 的權限，跨 repo 抓 private 一定失敗。
  2026-09-04 使用者定案 **engine 維持 private**，所以只剩加 PAT 這條路。
  解除條件有**兩條路，要使用者選一條**（2026-09-05 補上第二條）：

  1. **PAT secret**：在本 repo 建一個有 engine 讀取權的 PAT secret，
     workflow 的 `actions/checkout` 改成用它。維持「共用 engine 只有一份
     來源」不變，代價是要管一個長期憑證。
  2. **`go mod vendor`**：把 engine 的原始碼 vendor 進本 repo，CI 就完全
     不必抓另一個 repo，一個 secret 都不用。代價是 repo 裡多一份 engine 的
     複本，而且 engine 每次更新都要重新 vendor——`vendor/` 是建置產物不是
     第二份真相，但它會讓「engine 改了、Pool 沒重新 vendor」變成一種
     新的不同步。

  **使用者 2026-09-05 決定：先不要動 CI。** 這一項因此不再是待辦——
  平台驗收維持只靠本地 Docker（`tools/linux-release-smoke.sh`、
  `tools/windows-release-smoke.sh`），`platform-smoke.yml` 留著但驗不到東西。
  要重開這一項，條件是使用者改變上面那個決定並選一條路；在那之前不要
  每一輪重提。

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

- [x] **敵方 AI 讀完了**（spec 096）。
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
  **士氣那一支的 `0100h:0066h` 也讀了**：那是 overlay-24 entry 14（`103Bh`），
  把 `DS:0C38h`／`0C39h` 兩個代碼（`4Ah`、`4Bh`——139 個代碼裡唯二處理常式不
  在 overlay-12 的）交給 overlay-24 entry 2 **移除**。entry 2 順帶把效果串列的
  版面補齊：節點 9 bytes，`+0` 代碼、`+4`「有收尾常式」旗標、`+5`／`+7` 下一個
  的遠指標，串列頭佔記錄 `+7Fh`／`+81h`，釋放走 RTL 的 FreeMem；`+4` 非 0 時
  摘除前會以**模式 1** 叫一次處理常式（套用是模式 0）。spec 059 已補上。
  **順帶對上一條跨規格的線**：士氣讀的 `entity[+10h]` 就是轉變不死生物成功時
  寫的那一個（spec 111）——原版的「轉變」實作就是讓那一隻進入被嚇跑的狀態，
  不是另一套機制。
  **2026-09-05 群組編號有名字了**：把呼叫端逐個讀出來之後，
  **效果系統就是原版套用戰鬥修正的機制**——群組 `0Ah`（10）是攻擊者身上的
  命中修正、`10h`（16）是目標身上的、`0Ch`（12）是豁免的修正、
  `12h`（18）是每回合初始化調整攻擊次數。三個累加格也定了：`DS:6780h` 是
  命中骰（1d20，自然 1 直接失手、自然 20 改寫成 100），`DS:6774h` 是豁免骰，
  `DS:6788h` 是目前的豁免類別。所以「代碼 `47h` 讓 `6780h` 減 4」就是
  「這個效果讓對它出手的攻擊 −4」。**spec 050 那條「`02E2h` 的 effect codes
  `0Ah`／`10h` 來源」的開放項因此關掉**，spec 075 也補上群組 12 的內容。
  **115 支處理常式已有機器盤點**（`docs/audit/dos-effect-handlers.json`）：
  138 個 overlay-12 的代碼對到 115 支相異常式，另外 `49h`／`4Ah` 在
  overlay-13、`58h` 由 overlay-22 填；每支記了大小、寫哪些全域、碰哪些欄位、
  呼叫誰。其中**十二個代碼指到 9–12 bytes 的空常式**，永遠不改變任何東西。
  **`DS:6776h` 與 `6777h` 也定了**：前者是這一次要套的**傷害量**——
  overlay-24 entry 19（`133Ah`）先讓**群組 6** 調整它，**才**套豁免
  （規則 1 全免、規則 2 減半），所以「減半」減的是效果調整過的值；
  後者是傷害訊息的旗標（低三位選句型、位元 3 另一個），由施法的處理常式設。
  **士氣的後半段也讀完了**：`+84h` 的位元 7 是「這一隻做不做士氣判定」，
  低七位 × 2 才是士氣值（放在 `DS:6783h`，超過 102 當作不會逃），群組 17
  可以調整它。判定分兩關：先比自己掉了多少血
  （`100 − +11Bh × 100 ÷ +32h`），過不了再換 `DS:6D22h` 這個基準比隊伍的
  `[4937h]+58Ch`。
  **順帶解出 `+11Bh` ＝ 目前生命值**（`+32h` 是上限），那個百分比算式就是
  證據。神殿的起死回生與石化解除都把它寫成 1——**那兩項是「活過來但只剩
  一點」，不是回滿**，remake 原本漏了這一條，已經修掉（spec 115）。
  **四十八個代碼的修正算式已經抽出來了**：處理常式對累加格的改動是
  `ds:<格>, <值>` 這個固定形狀，機械抽得出來。例如 `21h`（失明）與
  `24h`（詛咒）都是「命中 −4、豁免 −4」——**與 spec 115 各自解出來的
  「神殿治得好的失明與詛咒」吻合**，兩份獨立來源對上；`03h` 是命中 +2 傷害
  +2、`11h` 讓傷害歸零、`6Fh`／`7Dh` 讓豁免自動成功、`25h`／`59h` 讓攻擊
  自動落空。整張表在 spec 112。
  **效果的生命週期也閉合了**：overlay-24 entry 10（`0E54h`）掛上一個 9 bytes
  的節點並接在 `+7Fh` 串列**尾端**，entry 2（`0028h`）摘掉並釋放。節點欄位
  由 entry 10 的參數填齊——`+0` 代碼、`+1`..`+2` 持續、`+3` 下效果者的等級、
  `+4`「摘掉時要叫收尾」、`+5`..`+8` 下一個。spec 059 已補上整張版面。
  不改修正的 88 個代碼也分完類：**21 個替自己掛節點、15 個把自己摘掉、
  25 個只印一句話、12 個指到空常式**，其餘寫的是記錄欄位
  （`+3h` 先攻、`+4h`、`+10h`／`+16h` 力量、`+15h` 魅力、`+11Bh`／`+32h`
  生命值）。
  **`DS:6776h` 的另一個讀取端也對上了**：overlay-13 `01CFh` 是**近戰傷害骰**
  ——骰完寫進去、背刺倍數在這之後乘、再由**群組 4** 調整。順帶關掉 spec 050
  的半個開放項：背刺倍數的 level byte 是 **`+9Ch`（賊等級）**，所以倍數就是
  AD&D 的級距（1–4 級 ×2、5–8 ×3、9–12 ×4）；predicate（`2558h`）仍未讀。
  **背刺的 predicate（`2558h`）也讀完了**：賊等級 > 0、副手若有東西其種類要
  是 `32h`、主手是空手或種類 7／8／23h..25h、runtime `+0Fh` > 1，最後一條是
  **攻擊者指向目標的方向等於目標自己的朝向**——那就是「從背後打」。
  spec 050 的 backstab 判定因此從 DRAFT 移除。
  **會改記錄欄位的 25 個代碼也抽出來了**，其中最有價值的一條是
  **`0Bh` ＝ 魅惑**（overlay-12 entry 14 `040Ah`）：套用時把記錄 `+10Eh`
  改成施法者那一邊（**倒戈**）、`+10Fh` 設 1 交給 AI 分派、士氣設成 `0B3h`；
  解除時從節點 `+3` 的位元 6 還原原本的陣營。節點 `+3` 因此打包了四件事：
  低四位是下效果者等級、位元 5 是「套過了」、位元 6 是原陣營、位元 7 是施法者
  陣營——**與 Charm Person 推的 `(施法者 +10Eh << 7) + 施法者等級` 逐位元
  吻合**（spec 098）。其餘如 `0Ch`／`26h` 改力量與特殊力量百分位、
  `0Eh` 改魅力（與法術表的 Friends 對得上）、`3Ah` 把腳程歸零、
  `21h`（失明）連 `+111h`／`+112h` 也各減 4、`62h` 每次回 3 點生命。
  **overlay-13 `1726h` 也對上了**：它只是近戰那一條在 `0191h` 骰完傷害之後
  把 `DS:6776h` 讀出來往下傳，同一個累加格沒有第二種用法。
  **型別表也結案了**：`DS:31A3h + 型別 × 16` 只有 `+0`（體型類別）有讀取端
  ——全 36 顆 overlay 加 `START.EXE` 只存取這張表四次，四處都是
  `[di + 31A3h]` 而 `di` 正好是 `型別 << 4`。16 bytes 的間距是真的，但其餘
  15 個 byte 在這一版沒有人讀。
  **entry 3 的物品過濾也解得差不多了**：`00E2h:003Eh` 是 overlay-22 entry 6
  （`31F6h`），判物品型別表的類別是不是落在 `0Bh..0Dh`——怪物不動那一類；
  `+34h` 非零代表**已裝備**，那個早就寫在 spec 065／067 裡（又一次沒先 grep
  自己的文件）；`+3Dh` 是物品上的編號，大於 `38h` 時減 `17h` 換成法術編號。
  **`[4933h]+1CAh` 也結案了**：只有 overlay-07 `025Bh` 寫它一次（寫 0），
  四處讀它而且都只跟 0 比——那幾個閘門在這一版永遠不成立，與 `DS:43A0h`
  同一個形狀。

  **這一項到此結清。** AI 分派鏈上的每一支都讀過了：entry 1 的分派、
  entry 5 的接近迴圈、entry 13 的單步移動、entry 7 的玩家打斷、entry 8 的
  士氣兩關、entry 3 的用物品與四個過濾、entry 2 的轉變不死生物、`1087h` 的
  「能不能打」與它背後整套效果系統。spec 096 自己還留著三項小的
  （115 支效果常式逐支的行為、`DS:6674h` 那個結構的 `+6` 有沒有別的用途、
  物品 `+3Eh`），都寫在那份規格的〈還沒讀〉裡——它們是欄位語意的細節，
  不是「AI 還沒讀」。
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
  **「收集／旗標」與「計數」那兩類也跑過了**（2026-09-06，
  `TestFlagAndCountCommissionsNeedTheirCondition`）：曼多爾六本書、斯托揚諾河、
  遊牧營地、蜥蜴人 40 場、卡德納兩支、提前完成的第一段，每一條都從掃描讀出來
  的條件判斷起跑，**而且每一條都帶負對照**——條件不成立時那一槽不能變成
  `FEh`。蜥蜴人那一條把 40 與 39 對調再跑一次，兩個方向都失敗，門檻確實是 40。
  槽 14 的兩段式單獨驗（`TestTheEarlyCompletionCommissionTakesTwoStages`）：
  `ecl6/28` 寫 `FDh`、`ecl6/25` 只在讀到 `FDh` 時升成 `FEh`。

  **交差那一段也逐槽走過了**（2026-09-06，`TestEveryCommissionHandsInAtCityHall`）：
  二十六槽各走一次「走回城區 → 進市政廳 → 走到職員面前」，檢查演出的是這一槽
  的通知、`4AC1h` 只在該加的十槽加一、有獎賞的收得到、該槽從 `FEh` 變成 `FFh`。
  負對照 `TestCityHallSaysNothingWithNoCommissionDone`：一條都沒完成就去交差，
  26 條通知一條都不該出現。把期望文字整體位移三槽再跑，26 條裡 23 條失敗。

  **中間那一段也量得到了**（`TestPlayingTheWorldCompletesCommissionsOnItsOwn`）：
  完全不給主線旗標，讓探索器從開場自己走，第二階段再從「交差完」的狀態走。
  現在走得完**五條**（0 諾里斯、1 索寇要塞、11 瓦海登墳場、18 卡德納的付款、
  23 巴恩神殿），順帶 20 張地圖、19 個 ECL 區塊，硬失敗 0 次。

  **這一支抓到兩個「測試綠、玩家卡關」的缺陷**：`36h ADD NPC` 沒把職業從記錄
  帶出來（一條→三條）；治具在雙向梯子上一律答 YES，在井的兩層之間上上下下
  出不來（三條→五條，**遊戲沒壞，是治具的答題策略壞了**）。第二個是靠新加的
  `eclvm.Machine.Trace` 做指令級追蹤找到的，只花六個位址。

  剩下的槽需要密碼、湊物品或挑特定選單，那是「玩家知道要做什麼」，不是
  「走得到那一格」。門檻是量到的下限，不是目標。

## 發行

### 對原版的抽樣對拍（2026-09-07 起）

`v1.1.1-20260907` 的第一輪結果與方法在
[`docs/audit/dos-parity-sample.md`](docs/audit/dos-parity-sample.md)：
標題整張 99.79%（差的是 remake 自己加的按鍵提示），第一人稱框 88×88 100%。
**原版那一側改用 [dosgolem](https://github.com/wicanr2/dosgolem) 跑真的
`START.EXE`，不再用 DOSBox**（使用者 2026-09-07 指定；DOSBox 只作 dosgolem
的參考）。重生：`tools/dosgolem-reference.sh` → `tools/appimage-dos-parity.sh`。

- [x] **第一人稱那一框已經改成對 dosgolem 比**（2026-09-07 當天收掉）。
      原本卡在 dosgolem 少了 EGA 圖形控制器（`3CE`／`3CF`），第一人稱框在它
      那邊畫成黑底白線框、只有 18.93%。它的 master 併進其他分支的平面式寫入
      模式之後就畫得出來了：**7744/7744**。
      **交叉核對那一項留著沒刪**——兩個獨立 oracle 現在互相同意，
      「兩個來源說同一件事」比任何一個單獨的數字都強，哪一邊之後退步了
      它會先開口。
- [ ] **抽樣擴到戰鬥畫面。** dosgolem 目前走到導覽結束的自由移動；戰鬥、商店、
      神殿、結局都還沒走到。**驗收**：`tools/dosgolem-reference.sh` 的鍵序
      走進一場戰鬥並拍到戰術盤面，`docs/audit/dos-parity-sample.json` 多一項。

**dosgolem 放在 `workplace/dosgolem`**（gitignore，使用者 2026-09-07 指定）。
用 `master` 就好：Pool 的那一批（`cmd/shots`、
`docs/spec/008-bios-keyboard-injection.md`、`009-scratch-writes.md`）已經併進去，
`pool-of-radiance-oracle` 分支功成身退。**引用 dosgolem 的規格一律連檔名**
——它十條分支各自從 007 開始編，同一個號碼底下有好幾份（它的
`docs/spec/000-index.md`）。

### `v1.1.1-20260907`（2026-09-07）

介面中文化補完那一版。Release：
<https://github.com/wicanr2/Pool-of-Radiance-cht/releases/tag/v1.1.1-20260907>

發行包由 `tools/package-release.sh v1.1.1-20260907` 從 `f8fc2dc` 重生，
`patch` 口味兩支煙霧測試都過：Linux AppImage 起得來、OGG 0 個；
Windows 在 Wine 下起得來、畫面平均亮度 0.23（擋掉「視窗開了但全黑」）。
對拍數字見 [`docs/audit/dos-parity-sample.md`](docs/audit/dos-parity-sample.md)。

### 第一階段完成（2026-09-07，`v1.1.0-20260907`）

repository 轉 public，release 對外掛四個發行包＋`manifest.json`：
<https://github.com/wicanr2/Pool-of-Radiance-cht/releases/tag/v1.1.0-20260907>

這一版之前修掉兩個「玩不下去」的缺陷，兩個都是**讓治具真的從開場走一遍**
抓到的，不是讀程式碼看出來的：`36h ADD NPC` 沒把職業從記錄帶出來；
「先換地圖再問問題」的腳本因為事件處理提早返回而地圖不換。

公開時的三個決定（2026-09-07 使用者定案）：

- **engine 維持 private**，所以公開的 repository clone 下來建置不起來。
  README 開頭直說這件事，不讓人白試。
- **軟體世界說明書譯文照原樣公開**。`NOTICE.md` 的敘述同步改成與現況一致
  （未取得授權、不因 RRSAL-1.0 重新授權、留移除管道）——留一句與現況相反的
  聲明比沒有聲明更糟。
- **推廣片留在本機不發布**：它嵌 Amiga 原版配樂（Wally Beben），公開等於散布
  他人著作。影片本身已建好（122 秒、15 段字幕）在 `dist-all/<版本>/promo/`。

尚未驗證的仍然是 macOS 與真實 Windows 機器上的啟動（見
[`docs/verification/real-machine-startup-checklist.md`](docs/verification/real-machine-startup-checklist.md)）。


- [x] 授權已定案並落地：RRSAL-1.0（非商業免費含修改再散布，實況與平台分潤
  明示允許，商業另談），`LICENSE` 與 `NOTICE.md` 在 repo 根目錄，也複製進每一個
  發行包。`NOTICE.md` 明列不因此被重新授權的東西：SSI 原版資產、軟體世界的
  說明書譯文、倚天字型、共用 engine 與第三方套件。
- [x] 三平台發行包可重生：`tools/package-release.sh <版本>` 產出 Linux AppImage、
  Windows ZIP 與 macOS 雙架構 ZIP，並寫 `manifest.json` 固定雜湊。
  AppImage 已由 `tools/linux-release-smoke.sh` 在容器裡實際啟動並截圖。
  契約見 spec 066。
- [x] **Windows 版在 Wine 裡實際啟動並截圖**（2026-09-06，
  `tools/windows-release-smoke.sh`）。它在 Docker／Wine／Xvfb 裡跑
  `pool-game.exe -lang zh -eten-font …`，等它畫出畫面之後從 root 裁下視窗，
  **並且擋掉全黑**——視窗開得起來但沒畫東西，和啟動失敗一樣糟。
  結果是原版標題畫面加上中文的「ENTER/空白鍵」提示，截圖在
  `docs/screenshots/pool-release-windows-wine.png`。順帶用同一組判準檢查
  發行包內容（full-local 六個 OGG、patch 包一個都不能有）。
  兩個做法上的坑寫在腳本裡：**先跑 `wineboot -u`**（第一次啟動要建 prefix，
  不先做的話等再久都是全黑，看起來像畫不出來）；**截圖要從 root 裁**
  （沒有視窗管理員時 `import -window <id>` 拿到的是全黑）。
  **這不能取代真機**：Wine 不是 Windows，驅動、字型後備與 DPI 縮放都不同。
  它證的是「不是連跑都跑不起來」。

- [x] **Windows 與 macOS 的真機啟動驗收交接出去了**（2026-09-05 使用者決定
  由自己在實機執行）。這裡做得到的部分做完了：建得出來、包得起來，Linux 走
  `tools/linux-release-smoke.sh`、Windows 走 `tools/windows-release-smoke.sh`
  （兩支都在本地 Docker 裡實際啟動並截圖）。**macOS 在這裡驗不了**
  ——Ebitengine 的 GLFW 在套件 init 就初始化視窗系統，無頭環境連 `-h` 都 panic
  （spec 066），所以 Docker 與 CI runner 都到不了「畫得出第一個畫面」。
  逐步清單寫成 [`docs/verification/real-machine-startup-checklist.md`](docs/verification/real-machine-startup-checklist.md)
  （七步、每台要留哪三張截圖、中文介面另跑一次）。
  **這一項不是「驗過了」，是「交接了」**：實機跑完之後把結果寫回這一條。
- [x] repository visibility 與原版素材 deny-list。原版素材的 deny-list 已經
  落地：`NOTICE.md` 列出不隨發行包散布的東西（原版資料、軟體世界說明書譯文、
  倚天字型），`tools/package-release.sh` 的 patch 封包排除原版 ZIP 與字型，
  公開 Release 只掛 patch。

  **可見性（2026-09-07 使用者定案，改掉 09-04 的「三個都 private」）**：
  Pool 轉為 **public**，CoAB 與共用 engine 維持 **private**。

  代價要講明：`go.mod` 依賴私有的 `golden-box-remake-engine`，所以**公開的
  repository clone 下來建置不起來**——模組抓不到。這是刻意的取捨：遊戲專屬的
  內容公開，可重用的引擎另外授權。README 開頭因此加了〈從原始碼建置需要什麼〉，
  直說「想玩的人下載發行版，不需要建置」，不讓人白試。
