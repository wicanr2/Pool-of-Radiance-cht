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
- ECL 已確認本作 code-address base 為 `9914h`；目前只有 3／29 blocks 可由既有
  CoAB decoder 完整走圖，其餘屬格式差異研究缺口，尚未接入 production VM。
- 已有第一支可執行的 Ebitengine `cmd/pool-game`：讀取本機原版 ZIP 的 `TITLE.DAX`，
  可由標題以正常按鍵進入主選單並走完 Race→Gender→Class→Alignment→角色資料頁
  →姓名→原版 HEAD／BODY／KEEP 肖像編輯器→READY／ACTION 戰鬥圖示編輯器；
  F1 Help、F2 theme、ESC 返回、F10 離開與視窗拉伸已接通。戰鬥圖示使用原版
  CHEAD／CBODY，支援 Head、Weapon、Size、六部位雙色及 READY／ACTION 同時預覽。
  Add／Load、285-byte CHA 完整序列化、建隊與進圖仍未完成。
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

目前以 Docker／Xvfb 做離線測試與煙霧擷取：

```sh
tools/go.sh test ./...
```

`tools/go.sh` 會在測試容器內自行建立有界 Xvfb。可互動封包尚未完成；本階段不把
主機 X11 socket 掛入開發容器，也不把只在背景 Xvfb 執行的入口寫成玩家啟動方式。

這仍是首條玩家垂直鏈；角色檔、建隊與進入 Phlan 尚未完成，不能宣稱
完整建角或可玩版。
