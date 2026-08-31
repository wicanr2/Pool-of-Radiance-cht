# Pool of Radiance 繁體中文 Remake

本專案以 SSI《Pool of Radiance》DOS 版為主要行為 oracle，建立可跨平台、
可從開場玩到結局的繁體中文 remake。現在是證據盤點與第二作品 engine
接線階段，尚不可宣稱可玩或與原版 parity。

## 現況

- 已固定 DOS ZIP、歷史中文 RAR 與八張 D64 輸入雜湊。
- 已盤點 DOS ZIP 169 筆內容；包含 `START.EXE`、`GAME.OVR`、八組
  ECL／GEO／WALLDEF／PIC／SPRIT DAX 與角色存檔樣本。
- engine `dax` 已成功解析 113／113 個 DAX、合計 1,245 blocks，Pool 已成為
  該 codec 的第二個真實作品 consumer；payload 語意仍待逐項驗證。
- 已用未修改 DOS 程式建立穩定標題與主選單 oracle；`TITLE.DAX` block 1 經
  typed adapter 匯出、2× 最近鄰呈現後，與原版標題逐像素 AE=`0`。
- ECL payload 映射基準已由原版 loader／resolver 閉合為 `9900h`；`9914h` 是五個
  command-set headers 之後的第一條指令位址。勘誤後 26／29 blocks 可完整走圖、
  14,724 條 reachable instructions 可解；其餘 3 筆仍待分類，尚未接 production VM。
- 已有第一支可執行的 Ebitengine `cmd/pool-game`：讀取本機原版 ZIP 的 `TITLE.DAX`，
  可由標題以正常按鍵進入主選單並走完 Race→Gender→Class→Alignment→角色資料頁
  →姓名→原版 HEAD／BODY／KEEP 肖像編輯器→READY／ACTION 戰鬥圖示編輯器；
  F1 Help、F2 theme、ESC 返回、F10 離開與視窗拉伸已接通。戰鬥圖示使用原版
  CHEAD／CBODY，支援 Head、Weapon、Size、六部位雙色及 READY／ACTION 同時預覽。
  完成確認後會寫入版本化 remake 角色庫、回到原版順序的 Party Creation Menu；
  Add、Load、六名玩家角色上限及 F10 原子保存已接通。DOS 285-byte CHA／SPC export
  尚未完成。正常 `B` 已依原版 producer chain 接到 `GEO3/block 0, (15,1), facing 6`
  的 typed geometry 與原版 `WALLDEF3/8X8D3` 第一人稱素材。正常 Begin 隨即依
  ECL3/block 0 進入 Rolf 導覽第一頁：首次旗標、`(15,1), facing 3`、monster 12、
  原版 ZIP packed text 與 Return 閘門已達 exact。Spec 011 另由四張原始 byte table
  閉合並實作完整 34-step scripted tour：六個停靠 selector 顯示七頁原版文字，最後在
  `(0,4), facing 3` 走到 ECL `EXIT`。每步 delay 是可重現的約 150ms approximation；
  Pool 專屬自由移動政策尚未閉合，因此導覽結束後仍不開放移動。視錐 traversal 仍是
  跨作品共用引擎的 strong inference。
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
[WORKLIST.md](WORKLIST.md)；輸入盤點收據見
[docs/audit/input-inventory.md](docs/audit/input-inventory.md)。

標題格式與驗收見 [Spec 001](docs/spec/001-dos-title-picture.md)；原版啟動收據見
[DOS 標題／主選單 oracle](docs/playtest/dos-title-main-menu.md)。

## 目前 remake 畫面

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

| 建隊後正常按 `B` 進入原版 Rolf 導覽第一頁（後續導覽／移動尚未開放） |
|---|
| ![Pool remake 初始 Rolf 導覽事件](docs/screenshots/pool-remake-initial-rolf-event.png) |

| Rolf 34-step 導覽的 Tyr 停靠點（正常 Return 路徑） |
|---|
| ![Pool remake Rolf 導覽 Tyr 停靠點](docs/screenshots/pool-remake-rolf-tour-tyr.png) |

目前以 Docker／Xvfb 做離線測試與煙霧擷取：

```sh
tools/go.sh test ./...
```

`tools/go.sh` 會在測試容器內自行建立有界 Xvfb。可互動封包尚未完成；本階段不把
主機 X11 socket 掛入開發容器，也不把只在背景 Xvfb 執行的入口寫成玩家啟動方式。

這仍是首條玩家垂直鏈；remake 自有角色庫、建隊與初始 map identity 已接通，但 DOS
相容角色檔、完整 Party Creation Menu、第一人稱背景／視錐的 Pool 專屬 oracle、
Rolf 初次 APPROACH 圖像、自由移動、戰鬥與存讀檔垂直鏈仍未完成，不能宣稱可玩版，
也尚未證實該 block 的地名。
