# Pool of Radiance 繁體中文 Remake

本專案以 SSI《Pool of Radiance》DOS 版為主要行為 oracle，建立可跨平台、
可從開場玩到結局的繁體中文 remake。目前已能從標題正常完成建角／建隊、
進入 Phlan 初始地圖、走過 Rolf 導覽與早期城市事件，並抵達第一場真實 Slums
遭遇的失敗即關閉（fail-closed）戰鬥 staging；完整戰術戰鬥與主線仍未完成，
因此尚不可宣稱可玩版或與原版 parity。

以「完整繁中、正常主線可破關、三平台可發行」為分母，目前保守工程估計為
**20～25%**，其餘約 **75～80%**。這是依玩家垂直鏈與交付硬門檻
估算，不是用規格文件數量換算；細節與未知項以 [CONTEXT.md](CONTEXT.md) 為準。

## 現況

- 已固定 DOS ZIP、歷史中文 RAR 與八張 D64 輸入雜湊。
- 已盤點 DOS ZIP 168 個檔案；包含 `START.EXE`、`GAME.OVR`、八組
  ECL／GEO／WALLDEF／PIC／SPRIT DAX 與角色存檔樣本。
- engine `dax` 已成功解析 113／113 個 DAX、合計 1,245 blocks，Pool 已成為
  該 codec 的第二個真實作品 consumer；payload 語意仍待逐項驗證。
- 已用未修改 DOS 程式建立穩定標題與主選單 oracle；`TITLE.DAX` block 1 經
  typed adapter 匯出、2× 最近鄰呈現後，與原版標題逐像素 AE=`0`。
- ECL payload 映射基準已由原版 loader／resolver 閉合為 `9900h`；`9914h` 是五個
  command-set headers 之後的第一條指令位址。勘誤後 26／29 blocks 可完整走圖、
  14,724 條 reachable instructions 可解；其餘 3 筆仍待分類。正式玩家路徑已使用
  同一 VM session 跨 ECL archive 執行目前需要的 blocks，不能把全 corpus 的三個
  graph 缺口誤寫成「production VM 尚未接」。
- 已有第一支可執行的 Ebitengine `cmd/pool-game`：讀取本機原版 ZIP 的 `TITLE.DAX`，
  可由標題以正常按鍵進入主選單並走完 Race→Gender→Class→Alignment→角色資料頁
  →姓名→原版 HEAD／BODY／KEEP 肖像編輯器→READY／ACTION 戰鬥圖示編輯器；
  F1 Help、F2 theme、ESC 返回、F10 離開與視窗拉伸已接通。戰鬥圖示使用原版
  CHEAD／CBODY，支援 Head、Weapon、Size、六部位雙色及 READY／ACTION 同時預覽。
  完成確認後會寫入版本化 remake 角色庫、回到原版順序的 Party Creation Menu；
  Add、六名玩家角色上限，以及穩定玩家邊界的 schema 6 F10／Load campaign round-trip
  已接通；對話／服務／戰鬥中途續點尚未完成。DOS 285-byte CHA／SPC export
  尚未完成。正常 `B` 已依原版 producer chain 接到 `GEO3/block 0, (15,1), facing 6`
  的 typed geometry 與原版 `WALLDEF3/8X8D3` 第一人稱素材。正常 Begin 隨即依
  ECL3/block 0 進入 Rolf 導覽第一頁：首次旗標、`(15,1), facing 3`、monster 12、
  原版 ZIP packed text 與 Return 閘門已達 exact。Spec 011 另由四張原始 byte table
  閉合並實作完整 34-step scripted tour：六個停靠 selector 顯示七頁原版文字，最後在
  `(0,4), facing 3` 走到 ECL `EXIT`。每步 delay 是可重現的約 150ms approximation；
  導覽結束後已可用方向鍵轉向，並以原始 GEO wall／door data 在 cardinal 朝向前進；
  每格 ECL dispatch 已從 Sune 神殿、City Hall 公告／委託走到跨 archive 的 Slums；
  鎖門互動、全部地圖事件與完整 Pool 移動政策仍未閉合。視錐 traversal 仍是
  跨作品共用引擎的 strong inference。
- 第一場 Slums 遭遇已從真實 ECL 載入 `MON2CHA.DAX` 的 `ORC ×1／ORC ×3`，並將
  ECL PC 停在戰後 continuation 前，不接受 Enter 假造勝利。Spec 048～053 已閉合
  285-byte 怪物戰鬥欄位、基礎命中／傷害、雙攻擊槽、先攻、移動預算、八方向步進與
  66×4 戰術格位表；戰術部署、玩家／AI 回合、反應攻擊完整 gate、死亡／全滅、勝敗
  與戰後續行仍待完成，因此目前只有測試收據，沒有冒充正常可達性的戰鬥截圖。
- 原版建角已走通 portrait 與 OLD／NEW READY／ACTION combat icon；六部位雙色、
  Head、Weapon、Size 的 285-byte CHA offsets 已由 UI 單變因差分閉合。六種族的
  原版職業清單已進 typed catalog，並有 Race→Gender→Class→Alignment＋ESC 狀態機。
- 共用 engine 固定使用同層
  `/home/anr2/cht/golden_box/golden-box-remake-engine`，本 repository 不複製
  engine source。
- 第一支 `cmd/pool-inventory` 只做唯讀 ZIP／DAX 形狀盤點，不解讀劇情語意。

## 本機盤點

原版檔案不進 Git。將 `Pool of Radiance (1988).zip` 放在 repository 根目錄後，
以專案 Docker 工具鏈執行：

```sh
tools/go.sh run ./cmd/pool-inventory -zip "Pool of Radiance (1988).zip"
```

目前真相來源與下一步分別見 [CONTEXT.md](CONTEXT.md) 與
[WORKLIST.md](WORKLIST.md)；逐輪變更見 [WORKLOG.md](WORKLOG.md)；輸入盤點收據見
[docs/audit/input-inventory.md](docs/audit/input-inventory.md)。

標題格式與驗收見 [Spec 001](docs/spec/001-dos-title-picture.md)；原版啟動收據見
[DOS 標題／主選單 oracle](docs/playtest/dos-title-main-menu.md)。

## 目前 remake 畫面

姓名輸入至 Rolf 導覽的七張操作畫面已從真實 Ebitengine 視窗，透過 Docker／Xvfb
逐鍵重拍；標題與角色資料頁沿用較早的同一路徑收據。重拍後七張 PNG 與版控內容
逐 byte 相同；來源提交、日期、雜湊、尺寸與狀態標籤見
[截圖 manifest](docs/audit/remake-screenshot-manifest.json)。這些圖片只證明各圖所列的
正常路徑與目前畫面，不代表完整遊戲 parity。

| 標題（原版素材 typed decode） | 建角角色資料頁（Spec 003／004） |
|---|---|
| ![Pool remake 標題](docs/screenshots/pool-remake-title.png) | ![Pool remake 角色資料頁](docs/screenshots/pool-remake-character-sheet.png) |

| 姓名輸入（原版順序） | 肖像編輯器（原版 HEAD3／BODY3 素材） |
|---|---|
| ![Pool remake 姓名輸入](docs/screenshots/pool-remake-character-name.png) | ![Pool remake 肖像編輯器](docs/screenshots/pool-remake-portrait-editor.png) |

| 戰鬥圖示編輯器（原版 READY／ACTION 素材） |
|---|
| ![Pool remake 戰鬥圖示編輯器](docs/screenshots/pool-remake-combat-icon-editor.png) |

| 最終確認 | 返回建隊選單並加入角色 |
|---|---|
| ![Pool remake icon 最終確認](docs/screenshots/pool-remake-icon-confirm.png) | ![Pool remake Party Creation Menu](docs/screenshots/pool-remake-party-menu.png) |

| 建隊後正常按 `B` 進入原版 Rolf 導覽第一頁 |
|---|
| ![Pool remake 初始 Rolf 導覽事件](docs/screenshots/pool-remake-initial-rolf-event.png) |

| 冒險畫面按 `F5` 的戰術地圖預覽（Spec 060） |
|---|
| ![Pool remake 戰術地圖預覽](docs/screenshots/pool-remake-tactical-preview.png) |

這張圖是由目前地城座標與真實 GEO 牆面資料現場生成的 50×25 戰術格，1250 格全部
有內容；牆呈斜線是投影本身的形狀（`X = 21 + 6dx + 5dy + subB`）。它只呈現地形，
還沒有 combatant、輸入與回合流程。GEO 的牆面樣式編號與戰術層的 0／1／3 尚未對上，
所以門目前畫成實牆——這一點直接寫在畫面上。來源提交與雜湊見
[戰術預覽截圖 manifest](docs/audit/remake-tactical-screenshot-manifest.json)。

| Rolf 34-step 導覽的 Tyr 停靠點（正常 Return 路徑） |
|---|
| ![Pool remake Rolf 導覽 Tyr 停靠點](docs/screenshots/pool-remake-rolf-tour-tyr.png) |

目前以 Docker／Xvfb 做離線測試與煙霧擷取：

```sh
tools/go.sh test ./...
```

`tools/go.sh` 會在測試容器內自行建立有界 Xvfb。可互動封包尚未完成；本階段不把
主機 X11 socket 掛入開發容器，也不把只在背景 Xvfb 執行的入口寫成玩家啟動方式。

這仍是首條玩家垂直鏈。remake 自有角色庫、建隊、穩定邊界存讀檔、初始 map identity、
Rolf 導覽、Sune／City Hall 早期事件與 Slums 戰鬥前 staging 已接通；DOS 相容角色檔、
完整 Party Creation Menu、Pool 專屬視錐 oracle、Rolf 初次 APPROACH 圖像、全部鎖門／
地圖事件、完整戰術戰鬥、對話／服務／戰鬥中途續點及主線破關仍未完成。
