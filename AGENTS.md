# Codex working agreement — Pool of Radiance DOS 繁中化／Remake

本檔是 `/home/anr2/cht/golden_box/pool_of_radiance` 的第一閱讀入口與
agent 單一作業權威。目標是以 SSI《Pool of Radiance》DOS 版為主要
oracle，建立可從開場玩到結局的繁體中文 remake，不是畫面 demo、
單一 vertical slice 或只能讀取原版資料的 viewer。

## 1. 語言與主機衛生

- 面向使用者與新撰文件預設使用繁體中文；程式識別字、檔名、
  API、工具與原版文字在翻譯會降低精確度時保留原文。
- 分析、批次搜尋、解壓、轉檔、建置、測試、DOSBox、Xvfb、Wine、
  IDA、擷圖、音訊與 GUI 自動化一律在 Docker 內執行。主機只作
  `docker`、`git`、工作樹狀態與儲存庫檔案編輯。
- 一次性容器使用 `--rm`、合理的 memory／CPU／pids 上限與外層逾時；
  預設 `--network none`。原版輸入唯讀掛載，可寫容器明確使用
  目前 UID／GID，不得製造 root-owned 專案檔案。
- 每批 Docker 工作後檢查並清理本專案容器；只清理已確認屬於本專案
  的停止容器與被取代映像，不全域清理其他專案。
- **格式化一律走 `tools/go.sh fmt`**，它只自動排新檔，既有檔案一個字都不動。
  這份程式碼從來沒有整批格式化過，而 `go fmt` 的操作單位是 package——
  一跑就把 struct 欄位對齊、`var` 區塊對齊與中文註解的 `//（` 全部改掉，
  在 `cmd/pool-game` 一次產生幾百行與當次工作無關的變更；那種 diff
  沒有人審得動，真正的修改會被埋掉。
  這一條以前寫的是「不要對整個目錄跑 `go fmt`」，然後還是被踩了好幾次
  ——**記得不要做某件事擋不住那件事**。所以 2026-09-10 改成由 `tools/go.sh`
  攔下來，傳給 `fmt` 的 package 參數一律忽略並警告。

  範圍縮到「只排改動過的檔案」**還是不夠**：既有檔案本身就不合 gofmt，
  碰到整個檔案就會把沒動過的行一起重排（`main.go` 一個檔案就從 16 行變成
  68 行）。所以現在的行為是**既有檔案一個字都不動**，只自動排新檔——新檔
  沒有既有格式要保。要看自己改的那幾行有沒有問題，
  `tools/go.sh gofmt -d <檔案>` 讀 diff，只手改屬於自己的部分——
  **`gofmt -w` 會被擋下來**（它一樣重排整個檔案，只是繞過了 `fmt` 那道），
  `-d`／`-l` 放行。真的要整包重排是 `tools/go.sh fmt-all <package>`，
  那條路會先講清楚它會動到什麼。

## 2. 原版 oracle 與平台優先順序

- 主要 oracle 是 DOS 版。第一步盤點 `Pool of Radiance (1988).zip`
  內容、可執行檔、版本字串、存檔、資料檔與 SHA-256；在此之前不宣稱
  精確版本或資料格式。
- `珍009-光芒之池.rar` 可作在地化歷史與對照線索，不自動成為
  原版行為 oracle；先記錄來源、雜湊、字碼、修改範圍與可重生性。
- `amiga/` 現有檔名與 `.d64` 容器後綴不足以證明平台；應由 magic、
  目錄與可執行格式重新辨識。非 DOS 版只作交叉驗證或翻譯參考，
  不可在沒有 DOS 證據時靜默覆蓋 DOS 行為。
- 原版手冊、Adventurer's Journal 與提示書是內容與玩家知識的權威來源；
  規則、執行順序與亂數消耗仍以 DOS runtime／bytes 為優先。

## 3. 不可縮減的產品目標

- 正常入口完成「啟動 → 建立人物與隊伍 → Phlan 城市與地城 →
  事件與戰鬥 → 存檔／讀檔 → 主線結局」，不依賴傳送、forced-win、
  賜予資源或測試專用強化。
- 完整繁中化玩家可見文字：主線、選單、戰鬥、物品、法術、
  Journal、Tavern Tales、訓練、商店、神殿、錯誤與存讀檔。
- 將原作要求查閱手冊或 Journal 的內容以合法、可追溯方式整合進遊戲，
  仍保留繁中手冊與攻略文件。
- 原版忠實主題是驗收基準；現代主題、攻略 overlay、多語與輔助功能
  必須是可切換的表現層，不得改變原版規則狀態。

## 4. Game pack 與共用 engine 邊界

共用 engine 固定位於
`/home/anr2/cht/golden_box/golden-box-remake-engine`，是獨立 Git repository。
Pool of Radiance 專案不得複製 engine source、不得將其加成 gitlink，
正式建置應鎖定已推送的 module version，本機開發才用可重生的
proxy／workspace 流程。

- engine：作品中立的 DAX／ECL／GEO codec、VM／runtime、AD&D 共用規則、
  renderer contract、存檔基礎、亂數流、幾何、世界／區域地圖與 schema。
- game pack：Pool of Radiance 的段號、block、旗標、地圖、入口、劇情、
  NPC、怪物編成、物品／法術資料、原文比對、譯文、手冊與美術分派。
- engine 的非測試 Go 程式碼與正式 example 不得出現 Phlan、Pool of Radiance、
  本作 NPC、任務旗標、文字或座標。可重用知識可注明證據來自某作，
  但必須把作品常數與範例 payload 留在 game repo。
- game pack 無法描述某行為時，先擴充 engine schema／runtime，再由 JSON／typed
  adapter 宣告；不為趕進度在 engine 寫 `if title == "pool-of-radiance"`，
  也不在 UI 用地名、block 或座標硬編碼劇情。

### 已完成的 engine 起始契約

CoAB 收尾已完成下列分離，Pool 不得重做或繞過：

1. DOS／PC-98 DAX block container 與 RLE decoder 位於 engine `dax` package；
   Pool 直接消費該 API，再以本作真實檔案補第二作品的格式／consumer 驗證。
2. ECL operand、instruction framing、變長 record、branch table 與 control-flow
   graph 位於 engine `ecl` package；Pool 只實作自己的 memory map、文字 rule、
   段交接、事件效果與 UI adapter。
3. engine `examples/` 只留 synthetic fixture；Pool 的 Phlan 劇情 payload、
   位址、文字、地圖與實機 fixture 一律留在本 repository。
4. engine 固定使用同層 `/home/anr2/cht/golden_box/golden-box-remake-engine`；
   本 repository 內不得建立 nested clone、gitlink 或 source 副本。

現行已驗證的 engine 基準為 `0819c64`。Pool 的第二作品證據已把 ECL code
address base 從 engine 移回 title adapter：本作 DOS raw payload 映射基準為
`0x9900`，五個 command-set headers 後的第一條指令才是 `0x9914`；不得沿用
CoAB 的 `0x8000`，也不得再把 `0x9914` 寫成 mapping base。正式相依鎖定該 commit 或更新後已
推送的 module version；不得提交本機 `replace`。

Pool 的 ECL 玩家路徑必須保留同一個 VM session：Rolf 導覽結束後不能另建一台
只帶座標的新 VM；後續地圖 entry 必須沿用既有 memory、旗標與 continuation 狀態，
同步 `C04B..C04F` 後再切換入口。引擎的 `RunUntilEvent` 只負責在第一個玩家可見
事件、選單或 `EXIT` 邊界停下；它不替 game pack 解釋作品位址，也不得跨過戰鬥等
事件後才回報。前端必須明確續跑或處理該 boundary，不能把 pending 當成完成。

初始地圖每次成功前進目前依五入口 lifecycle ABI 執行 ECL3/block0 entry 0
`9914h`（per-turn），正常 `EXIT` 後再執行 entry 1 `99EBh`（SearchLocation）。入口角色
已有 Pool block shape／dataflow 與 CoAB executable 證據，Pool executable 呼叫順序仍是
`strong inference`，不可寫成 Pool exact。掃描每格時可從已完成 Rolf 的基準狀態複製
VM 以隔離樣本，但靜態 sweep 只證明「給定狀態下執行入口」；不能證明玩家從入口
走得到該格，也不能取代正常按鍵路徑。

### CoAB 程式沿用與零干擾規則（2026-08-31 使用者決定）

- CoAB 與 Pool 同屬 Gold Box 系列；已由 Pool 真實 corpus 證實的 ECL framing、
  opcode 形狀與作品中立 runtime 機制，優先沿用共用 engine，不另造 Pool-only VM。
- 使用者已授權參考或複製 CoAB 程式以加速，但複製後必須先移除 CoAB 位址、劇情、
  旗標、文字與 UI 假設，泛化後提交到獨立 `golden-box-remake-engine`；Pool 只引用
  engine API，不把 CoAB source 複製進本 repository。
- **不得干擾原有 CoAB remake。** 抽取順序固定為：engine 新增向後相容 API與測試 →
  Pool 先成為新 consumer → 以 CoAB 現行測試做唯讀回歸驗證。未經另行授權，不修改
  CoAB adapter、game pack、玩家行為或既有 API 呼叫。
- CoAB 的 ECL 記憶體 map、external selector、戰鬥服務旗標與 opcode 副作用只能當
  跨作品候選；Pool 必須以自己的 DOS bytes／runtime 證明後才能寫進 Pool adapter。
  「兩作共用 opcode」不等於「每個位址與副作用也相同」。

## 5. 證據、反組譯與 spec 門槛

正式行為依以下單向流程：

```text
DOS bytes／runtime／手冊 → DRAFT spec → 證據審查 → READY
                         → typed 實作 → 同狀態抽樣 → CONFORMED
```

- 每項結論保留輸入檔名與 SHA-256、工具／版本、位址空間、原始位址、
  bytes／xref／trace／截圖、語意、證據等級與是否阻擋玩家路徑。
- 證據等級固定為 `exact`、`strong inference`、`hypothesis`、`unknown`；
  工具內改名只供導覽，不可取代原始位址與證據等級。
- IDA 資料庫、交叉參照與資料流為主，攤平 `.asm` 只作搜尋線索。
  far pointer、段前綴、間接讀寫、overlay entry 與 compiler helper 必須單獨追蹤，
  不得因 direct xref 為零就宣稱沒有 consumer。
- **overlay 掃不到 ECL 腳本的寫入。** 掃 overlay 找 writer 用的是機器碼的定址
  形狀（disp16 modrm 之類），而遊戲資料有一大半是 ECL 的 `SAVE` 寫的——那是
  bytecode，不是機器碼，任何指令層掃描都結構上看不到它。所以**一個位址在
  overlay 裡找不到 writer 時，先問它是不是 ECL 變數，再說「掃描面有洞」**。
  判準是落不落在兩個基底的窗內：class 0 是 `[4933h] + 6E00h + addr × 2`，
  class 1 是 `[4937h] + 2A00h + addr × 2`（皆 mod 10000h，見 spec 008／106）。
  換回 ECL 位址之後用 `cmd/pool-ecl-memory-audit -addresses <hex>` 列引用。
- **反過來也一樣：ECL 位址的字面值掃不到引擎的寫入。** 引擎寫的是
  「基底在暫存器＋位移」（`[4937h] + 5AAh`），位元組序列裡不會出現 ECL 位址的
  兩位元組字面值，所以掃 `D5 6D` 得到零筆不代表沒有 writer。要掃 overlay 就先
  把位址反解成位移：class 1 是 `(2A00h + addr × 2) mod 10000h`，
  class 0 是 `(6E00h + addr × 2) mod 10000h`，拿位移去掃 disp16。
  **兩條合起來的意思是：ECL 位址與引擎位移是同一個東西的兩種寫法，
  一邊查不到就換算到另一邊再查，不要停在「掃描面有洞」。**
  已驗過的換算對：`6DD5h↔5AAh`、`6DD2h↔5A4h`、`6DE1h↔5C2h`、`6DE2h↔5C4h`、
  class 0 的 `49E6h↔1CCh`。
- **但數值落在窗內不等於它就是 ECL 變數。** 引擎自己的 DS 段全域變數數值範圍
  和 class 1 的窗重疊，`6CD2h`／`6CD3h`／`6CD4h` 就是——它們一筆 ECL 引用都
  沒有，overlay 用的是 `mod=00 rm=110` 的 `[disp16]` 絕對定址。**判準是
  overlay 怎麼定址它，不是數值範圍**：`[基底+disp16]`（`mod=10`）才是 ECL
  變數的形狀，`[disp16]` 是 DS 全域。兩種一起掃用
  `cmd/pool-disp-scan -addresses <逗號分隔>`，**呼叫時一定要多帶一個已知答案當正
  對照**（`5AAh` 必須掃出 overlay-14 那四條 `mov word [di+5AA], 1`），否則
  「零筆」什麼也證明不了。
- **ECL 的文字也掃不到——它是 6-bit packed 的 `80h` 運算元。** 拿一句原版臺詞去
  grep payload 或 overlay 的 ASCII，**結構上一定零命中**，而那個零長得跟「原版
  沒有這句話」一模一樣。找字串一律走 `cmd/pool-text-inventory`（它跑指令圖、解
  `80h` 運算元與選單記錄，一次吐 1800 句，含每句的 `ecl?.dax#block@位址`）。
  正對照的做法是先確認**整個 ECL payload 連一段像樣的明文都沒有**——
  ECL2 的三個 block 各只有一段長度 ≥ 8 的可印位元組，內容是亂碼。
  掃到一段明文反而表示自己讀錯了層。

- **讀 ECL 內嵌資料表之前先拿已知指令對一次位址基準。** `GETTABLE` 指到的是
  腳本自己 payload 裡的位元組，位址基準偏一格不會報錯，只會安靜地給出一張
  看起來合理的表——而那張表會一路寫進 spec。正對照的做法是拿一條**反組譯器
  已經解出來的指令**去讀同一個位址：`workplace/eclbytes -addr 0x9BCF` 要回
  `2A 01 E3 B6 01 7F 6E 01 C1 6D`，逐位元組對上 `GETTABLE @B6E3 @6E7F → @6DC1`。
  對得上基準才算驗過。（spec 136 的五張表曾整張錯一格。）

- 追 Borland TPOV far call 時保留原始 `segment:offset`，先用 MZ header size 換算
  executable file offset，再與 `docs/audit/dos-ovr-manifest.json` 的
  `executable_file_offset` 精確反查 overlay／entry；每次匯出同時記錄輸入 overlay
  SHA-256、IDA 版本、位址空間與原始 bytes。IDA 自動命名與人工猜測不可取代這條鏈。
- ECL external `CALL` 的位址只能先視為服務邊界。即使它出現在神殿、商店或戰鬥
  流程附近，也必須比較同一呼叫在其他分支／步驟的出現方式，並追到 overlay
  dispatcher 後才能命名；不能把「在某流程看見」直接升格成該服務本體。
- 先辨識 compiler、linker、overlay、runtime、圖形／音訊 driver 與檔案工具，
  再用未知函式數量評估玩法缺口。stack check、RTL、記憶體與檔案 helper
  必須有具體分類理由，不可當成未實作玩法。

## 6. 縱向功能鏈與驗收

每個玩家功能至少追通：

```text
原始資料／bytes → parser 邊界 → typed data → 規則／亂數
                    → UI／動畫／音訊 → 存檔回讀 → 正常玩家路徑
```

- parser 要有 count、offset、stride、bounds、malformed input 失敗即關閉與真檔 anchor；
  可寫格式保留未知 bytes 並做 round-trip。
- 建角與建隊不只測資料表：必須抽測姓名、種族、性別、職業／多職、
  陣營、屬性、HP、法術、戰鬥 sprite／調色、角色檔、加入隊伍與讀檔。
- 戰鬥同時抽測玩家／AI、快速／戰術路徑、近戰／遠程／法術、狀態、
  隊形、大型 footprint、勝利／逃跑／全滅與戰利品。
- 真實全滅是合法遊戲結果；正常強度抽測遇到全滅就停止並記錄，
  不用強化隊伍或 forced-win 強求通關。
- direct-entry、座標注入與快照可作診斷，不可取代從標題開始的正常按鍵路徑。
- **測試裡的 `t.Skip` 只給環境前提用**（原版 ZIP 不在版控、字型檔不在這台）。
  「走不到那個狀態」不是環境問題——那是測試沒把狀態設對，要設對，不要跳過。
  跳過的測試在 `go test` 的輸出裡與通過長得一樣，**除非加 `-v` 才分得出來**，
  所以一條會 skip 的測試等於一條沒有人在看的測試。
  > 開場那一格四面都走不動（它是換圖點，換圖那一步不加時間），
  > 於是「走一步效果減一分」那條測試 skip 掉了自己要測的東西；
  > 把隊伍先站到一格走得動的地方，同一條測試就真的跑起來了。

- **[HARD] 未完成項的權威是 [`docs/worklist.json`](docs/worklist.json)，
  不是 WORKLIST.md。** 那一節由 `go run ./cmd/pool-worklist -mode render`
  產生，手改會在下一次 render 被蓋掉。每一項掛一個 `verify`：**跑
  `-mode verify` 為真代表這一條仍然未完成**，為假就是東西做好了而條目沒改
  ——也就是過期斷言。多數條目綁在程式碼的自承註解上，註解一旦被拿掉 verify
  就會開口；沒有機器可判訊號的標 `manual`，它一律回「仍未完成」並標出來，
  **沉默不等於通過**。
  > 清單是假斷言長得最好的地方：東西接上了，而沒有人回頭改那一條。
  > 2026-09-10 抓到臭雲術寫著「派發表六十七格裡只剩這一支」，實際上派發
  > 那一格早就接上，缺的只是盤面上的雲團物件。

## 7. 原版對拍與截圖契約

- **[HARD] 基準畫面一律由 dosgolem 產，對拍一律走
  `tools/appimage-dos-parity.sh`。** DOSBox 只能從外面看畫面：送鍵靠
  xdotool、等畫面靠 sleep、判斷靠像素，而「猜對」與「猜錯」在截圖上長得
  一樣。dosgolem 走位靠「程式讀走了幾個鍵」與「畫面連續多少道指令沒動」，
  而且直接吐 320×200 的色號陣列——對拍本來就該在色號空間做，不是在 PNG 上。
  `tools/dosgolem-reference.sh` 產基準時會寫下
  `workplace/dosgolem-ref/provenance.json`（generator、dosgolem 的 commit、
  原版 `start.exe` 的 SHA-256、鍵序、幀數）；**對拍腳本檢查這一份，
  generator 不是 dosgolem 就拒絕跑**。
- **[HARD] 一個畫面只留一組斷言，來源是 dosgolem。** 報表裡不放第二個
  oracle 的交叉核對：兩組數字擺在一起，讀的人分不出哪一個是誰說的，
  而「兩邊互相同意」在其中一邊悄悄退步時反而不會開口——一組會動的斷言，
  勝過兩組互相背書的。
- **dosbox-x 是 dosgolem 的參考來源，不是第二個基準。** dosgolem 缺功能、
  行為對不上或要追某一段時，可以啟動 dosbox-x 比對、據以擴增 dosgolem
  或替它除錯——那是**擴增 dosgolem 的手段**。它產的畫面不進對拍報表，
  結論一律以 dosgolem 為準。
- **[HARD] 動到任何玩家看得到的畫面就要跑對拍，不能只看截圖。** 截圖證明
  「這一版長這樣」，對拍才證明「與原版的距離變近還是變遠」。數字（含變好、
  變差與不變的那幾項）寫進 commit message。
- **[HARD] 對拍對的是發行包，而且要是當前原始碼建的。** 「測試綠」與「我們
  寄出去的那個檔案畫得對」是兩件事，中間隔著打包、資源內嵌、視窗縮放與實際
  的繪圖路徑，所以流程是 `tools/package-release.sh <版本>` 之後才對拍。
  **對拍腳本檢查 AppImage 有沒有比 `cmd/`、`internal/` 與共用 engine 的 `.go`
  舊，舊了就拒跑**——現成的發行包看不出它是哪一版建的，改完程式碼直接對拍
  只會量到上一版：數字照樣印得出來、看起來也正常，而整份結論是空的，還會把
  別的 commit 的升幅記到這一輪頭上。基準那一側的產地證明擋的是同一件事。
- **同一個座標不要寫死兩份。** remake 的視野原點在對拍腳本裡只有
  `REMAKE_VIEW_TOP` 一處。寫死兩份時，只改一份的症狀是比對率整段掉下來，
  而那看起來像 remake 退步了——查的人會去翻繪圖程式碼，不會想到是量尺。
- 對拍前固定 DOS 輸入雜湊、存檔、地圖、座標、朝向、隊伍、旗標、
  RNG／動畫相位、畫布、色盤與輸入序列。狀態不同時只可標為
  `nearby`、`material-exact/layout-reconstructed` 或 `layout-only`。
- DOS 擷圖每個目標連拍到可變區域兩張一致才收；拍完後再執行會改畫面的
  AREA／狀態核對。每個批次使用獨立可寫目錄，不共用 SAVE 工作區。
- 單張目標圖變好不算修正；同時保留已逐格一致的控制組，且查表的
  驗證範圍必須與作用域一樣大。
- 公開圖像可支持布局／視覺，但要記錄平台、來源 URL、取得日期與狀態限制。

## 8. 繁中化、字型、資產與音訊

- 建立集中術語表，記錄原文、繁中、人名／地名音譯、AD&D 規則譯名、
  禁用異體與證據。不把 CoAB 的專有名詞清單當成 Pool 翻譯事實。
- 每個文字欄位定義安全矩形、字型、字號、粗體、基線、最大行數、
  換行／截斷／省略策略；長繁中與英文都做真實字形截圖。
- **[HARD] 點陣字不靠膨脹字身加粗。** 倚天的字模是「筆畫一格、空隙一格」，
  往右膨脹一格會同時吃掉字與字之間的留白**和字內的縫隙**，筆畫密的字糊成
  一塊（「鈕」「繼」那種）。要厚度就用同色系暗一階畫外圈（`ShadowFace`），
  字身維持原樣；全形半形走同一套，否則同一行裡的重量差比兩套字更刺眼。
- **[HARD] 視窗尺寸是邏輯畫布的整數倍。** 非整數倍會把某些像素行複製、
  某些丟掉，點陣字看起來像糊掉，而截圖降取樣取到的也不是同一組邏輯像素。
- 原版 archive 只作 export 來源；remake runtime 使用獨立、可盤點的 PNG
  sprite／tileset／picture／atlas，並生成總覽圖、來源索引與資產數量測試。
- remake 音樂、語音與音效正式資產預設使用 OGG。技術解碼、觸發順序、
  loop／duration 與人耳確認分開驗收；轉檔不會改變原版音樂的權利狀態。
- 索引前先驗 archive count／格式／尺寸；版本參數不是資產形狀證明，
  越界必須 fail closed 或有明確安全 fallback 與測試。

## 9. 文件、完成狀態與避免 loop

開工建立並只保留一套現行角色：

- `README.md`：用途、啟動、截圖、下載、目前完成度與已知限制。
- `CONTEXT.md`：唯一目前狀態、定案、被推翻斷言與現行驗證策略。
- `WORKLIST.md`：只放未完成、可執行、有驗收條件的項目。
- `docs/spec/`：行為契約；`docs/re/`：查證方法；`docs/playtest/`：實跑證據；
  `docs/audit/`：可重生盤點與發行收據。

**要找「某個東西的規格／實作／測試在哪」先看
[`docs/spec/000-index.md`](docs/spec/000-index.md)。** 那份由
`cmd/pool-doc-index` 產生，別手改。對應關係的主鍵是 spec 編號——程式碼註解
裡的 `spec NNN` 就是那條線，索引只是把它反過來收攏，所以新增註解後重跑一次
就對了。反過來查也一樣快：`grep -rl 'spec 122' --include='*.go' cmd internal`。

改一份規格的某一節之前**先讀它的檔頭**。狀態行與檔頭的引言會寫明這一份已經
閉合到哪、被哪一份接手；跳過檔頭直接改中段，會把早就有答案的問題當成待解。

首次工作必須建立「目前狀態表」，分開：

1. remake 已完成；
2. DOS oracle 未知；
3. 可選 polish。

綠色單元測試只證明 remake 內部自洽，不自動證明 DOS parity。新發現要先問
「是否改變玩家體驗或目前交付 gate」；否則留最小定位與證據等級，
不讓無關 helper 把收尾變成無限反組譯。

**解碼用的表要跟著走完整條路徑。** 運算元個數是各作品直譯器 build 的性質，
不是格式的性質（`gamepack.PoolCommandTable()`）。只要有一個地方退回預設的二手表，
出錯的就不是那一條指令，而是**它之後的每一條**——症狀延後發作，報出來的位置離
成因很遠，而且往往只有某一條路徑才踩得到，於是表現成「偶爾撞一次，重跑就消失」。
2026-09-11 一次找到三處：VM 的 passthrough 分支算長度沒帶表、`RestoreSnapshot`
重建 machine 漏搬 `commands`、`NewDOSECLArchiveSession` 沒有 `SetCommands`。
**新增任何建 machine／換 machine 的路徑時，這張表要一起帶。**

**治具裡「一直按到某個狀態成立」的迴圈一律帶 guard。** 沒有終點的迴圈報出來
的是 `panic: test timed out`，而逾時不會說是哪一個條件永遠不成立——尤其當那條
測試本來就是套件裡最慢的一條時，第一眼會判成「變慢」而不是「卡住」。分辨的
方法是看 `running tests:` 那一行的耗時：差的是倍率還是「沒有終點」。
2026-09-11 把門選單改成 modal 之後，九處 `for spawn.Facing != 目標 { press(右) }`
一起變成死迴圈——方向鍵歸了門，`spawn.Facing` 一個都不會動。

**guard 的用意是讓迴圈有終點，不是限制次數，所以要給得寬。** 那九處看起來
「正常最多轉三下」，但它們也會在有格子事件擋著的時候被呼叫——那時右鍵不轉向，
要先把事件按過去，次數沒有上限可推。給 8 的那一版讓城堡那條測試走不到 ECL
區塊 7，而**沒有任何一條斷言指出朝向錯了**：症狀是覆蓋少了一塊，看起來像
「那個區塊本來就走不到」。

**測試治具模擬的輸入，行為要與真實的輸入層一致。** `scriptedKeys.JustPressed`
原本查一次就 `delete`，而 ebiten 的 `IsKeyJustPressed` 不消費——於是「前面某一段
查過這個鍵、但沒有處理它」的路徑會把鍵吃掉，而真實遊戲裡不會。症狀完全像產品
缺陷（2026-09-11：門的游標移得動、ENTER 沒反應），因為吃掉它的那一段只檢查
`Enter`／`Space`、不檢查方向鍵。**治具與真實行為的每一個差異，都會在某天變成
一個查不到成因的假缺陷。**

**改完腳本編輯要驗新內容，不是只驗錨點。** `assert count == 1` 只證明錨點唯一，
不證明它在你以為的那個函式裡——中間插進一個 `var (` 區塊就足以讓錨點落到別處。
改完 `grep` 一次新加的字串，沒中就是沒改到。2026-09-11 為此多追了兩輪：診斷
用的 panic 兩次都沒進到目標函式，而我照著「panic 沒觸發」繼續推論。

**用最小重現換掉整條慢測試。** 同一個 bug，整合測試要 80～1200 秒還只給得出
「卡住了」，最小重現 0.2 秒就把變因收斂到一個布林（`cellTextSticky`）。追了四輪
猜測都錯之後才想到寫它——**症狀隨參數改變方向的時候**（guard 給小是走不到、
給大是逾時，兩者指向相反的修法），那正是該停下來寫最小重現的訊號。

**玩家按得到的東西，測試要從 `Update()` 送按鍵進去。** 直接呼叫處理函式
（`resolveDoorMenu`、`selectXxxOption`）證明的是規則對，不是玩家按得動——
中間還隔著一層輸入分派，而那一層出錯時兩者在報表上長得一樣。2026-09-11 的
門選單就是這樣：四條測試全綠，而鎖住的門在遊戲裡按方向鍵與 ENTER 都沒反應，
因為那一段分派埋在 `cellEventPending` 裡面，撞門的當下那個旗標是 false。

連帶的一條：**一份狀態同時表示「記憶」與「正在等輸入」時，輸入分派會用錯它**。
`a.door` 要跨多次撞門留著（接續 BASH／PICK 額度），而選單只在最上層時該吃鍵；
兩件事共用 `a.door != nil` 這一個條件，放在別的選單前面就把它們的按鍵吃掉，
放在後面就在該作用的時候走不到。要分的是條件，不是順序。

**核實缺口清單要從總覽往程式走，不是反過來。** 2026-09-11 一次抓到四條
README 上早就做完卻還掛著的缺口（休息被打斷、遠程武器的射程、結局的圖、
說明書附錄的規則表），加上 `camp.go` 檔頭一條「還沒接」。**五條同一個形狀**：
缺口是在別的檔案、別的函式裡補完的，補的人改了那一支，沒有人回頭看總覽段落。
從程式往外看是看不到的——那一支已經是對的。所以核實的方向是拿總覽上的每一條
去程式裡找反證，而且要找到**實際生效的呼叫點**，不是找到同名的函式就算。

過期的缺口條目會直接壓低自評：清單上多掛一條做完的，自評就比事實低一截，
而那看起來像「保守」，不像錯誤。

**用腳本改 Markdown 一律先驗錨點再取代**：`assert s.count(old) == 1`，
然後才 `s.replace(old, new)`。錨點對不上時 `str.replace` 不會報錯，
空字串錨點更會把新內容插進**每一個字元之間**——檔案照樣是合法 Markdown、
`head` 與 `tail` 看起來都正常，只有大小會從幾十 KB 變成幾十 MB。
同一批 commit 因此可能一路把它推上去而沒人察覺。改完看一眼 `wc -c` 或
`git diff --stat` 的行數是第二道關。

## 10. 打包、授權與推廣片

- 所有封包、smoke、推廣片、影音 metadata 與 SHA-256 集中在
  `dist-all/<版本>/`；中間目錄不得成為文件下載入口。
- 公開 patch／engine 包與含合法自備原版資料的 `full-local` 完全分離；
  原版 ZIP／資料檔、轉檔 PNG／OGG、手冊、字型與第三方依賴各自建立
  可公開／僅本機／權利未明拒絕清單。
- 授權採 RRSAL-1.0（復古重製 source-available 授權條款 1.0），與 CoAB 及共用
  engine 一致，由使用者於 2026-09-04 指定。標準授權文必須保持原文；`LICENSE`
  與 `NOTICE.md` 要同時存在於 repo 根目錄與每一個發行包。
- 推廣片由目前封包重新擷取，至少包含 DOS／remake 明確標示的流程對照、
  實際遊玩、建角／建隊、地圖、戰鬥、語言與設定。對照圖必須標出
  same-state／nearby／layout-only，不用不同存檔冒充 exact。
- 影片完成後以 `ffprobe`、`volumedetect`、`silencedetect`、`blackdetect`、
  逐幕抽幀與 contact sheet 驗收；原版音畫權利未釐清前只供內部。

## 11. 停止線

- DOS PCM／DAC／PIT／DMA／Sound Blaster／OPL 的標準硬體時序優先採公開規格與
  成熟模擬器契約。remake 只需可重現 duration、播放完成閘門與合理人耳同步；
  標為 `hardware-spec approximation`，不冒稱逐週期 exact。
- 一條玩家功能已有原始輸入、typed model、規則、UI、存檔與正常路徑收據後，
  就停止逐行翻譯無關 executable。檔案服務、初始化樣板、compiler runtime
  與無玩家 consumer 的 helper 只保留最小定位與不阻擋理由。
- 完成聲明以正常玩家路徑、預先登記的抽樣與交付硬門槛為準；
  不用文件數、函式改名數、單張截圖或只有 remake 內部的綠色測試代替。
