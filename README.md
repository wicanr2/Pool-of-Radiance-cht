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
