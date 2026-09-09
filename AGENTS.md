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
- **不要對整個目錄跑 `go fmt`。** 這份程式碼從來沒有整批格式化過，
  一跑就把 struct 欄位對齊、`var` 區塊對齊與中文註解的 `//（` 全部改掉，
  在 `cmd/pool-game` 一次產生幾百行與當次工作無關的變更；那種 diff
  沒有人審得動，真正的修改會被埋掉。要檢查自己改的那幾行，
  用 `gofmt -d <檔案>` 看就好。

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

## 7. 原版對拍與截圖契約

- **[HARD] 基準畫面一律由 dosgolem 產，對拍一律走
  `tools/appimage-dos-parity.sh`。** DOSBox 只能從外面看畫面：送鍵靠
  xdotool、等畫面靠 sleep、判斷靠像素，而「猜對」與「猜錯」在截圖上長得
  一樣。dosgolem 走位靠「程式讀走了幾個鍵」與「畫面連續多少道指令沒動」，
  而且直接吐 320×200 的色號陣列——對拍本來就該在色號空間做，不是在 PNG 上。
  `tools/dosgolem-reference.sh` 產基準時會寫下
  `workplace/dosgolem-ref/provenance.json`（generator、dosgolem 的 commit、
  原版 `start.exe` 的 SHA-256、鍵序、幀數）；**對拍腳本檢查這一份，
  generator 不是 dosgolem 就拒絕跑**。報告裡那條 `first-person-vs-dosbox`
  是 repo 裡早就存著的一張 DOSBox 圖，只作第二個意見，**不是基準，
  也不重跑 DOSBox**。
- **[HARD] 動到任何玩家看得到的畫面就要跑對拍，不能只看截圖。** 截圖證明
  「這一版長這樣」，對拍才證明「與原版的距離變近還是變遠」。數字（含變好、
  變差與不變的那幾項）寫進 commit message。
- **對拍對的是發行包，不是原始碼建置。** 「測試綠」與「我們寄出去的那個檔案
  畫得對」是兩件事，中間隔著打包、資源內嵌、視窗縮放與實際的繪圖路徑，
  所以流程是 `tools/package-release.sh <版本>` 之後才對拍。
- **同一個座標不要寫死兩份。** 對拍腳本裡 remake 的視野原點曾經在版面比對與
  DOSBox 交叉核對各寫一次；只改一份的症狀是交叉核對從 100% 掉到 59.89%，
  而那看起來像 remake 退步了。現在收斂成 `REMAKE_VIEW_TOP`。
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

首次工作必須建立「目前狀態表」，分開：

1. remake 已完成；
2. DOS oracle 未知；
3. 可選 polish。

綠色單元測試只證明 remake 內部自洽，不自動證明 DOS parity。新發現要先問
「是否改變玩家體驗或目前交付 gate」；否則留最小定位與證據等級，
不讓無關 helper 把收尾變成無限反組譯。

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
