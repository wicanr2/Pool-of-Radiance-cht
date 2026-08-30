# DOS 標題 → 主選單 oracle

狀態：CONFORMED（只涵蓋啟動、code-wheel gate 與主選單抵達）。

## 環境與輸入

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`
- `START.EXE` SHA-256：`12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f`
- Docker image：`dosbox-run:latest`；DOSBox 0.74-3。
- Xvfb 800×600×24；保存原生 DOS 畫面區域 640×400。
- 解壓來源唯讀；每次複製至 `/run-game` tmpfs，因原版要求 `GAME.OVR` 可寫。

## 固定輸入序列

`START.EXE` → 畫面連續兩張 AE=0 → `Space` ×18（間隔 0.5 秒）→ code-wheel
prompt 輸入空字串 `Return` → 主選單連續兩張 AE=0。

空字串在未修改 executable 的本發行版可通過 gate；這是 runtime 觀察，不外推
到其他 revision。

## 收據

| 畫面 | SHA-256 |
|---|---|
| [`title-intro.png`](../reference/original-dos/title-intro.png) | `d6ccda5996d75e92d1f94549a75f0d1e5c12125e2079576e465bbcbd064fbc10` |
| [`main-menu.png`](../reference/original-dos/main-menu.png) | `b947d12ab7091da6dd57018a0864fdfa9811300912f48301d8afceec8125cf61` |

重生：`tools/capture-dos-oracle.sh`。腳本先等待畫面至少出現十色，再要求連續
兩幀 AE=0；不能把啟動時同樣穩定的灰階 `LOADING` 畫面當成標題。

尚未證明主選單鍵盤映射、建角流程與音訊；本輪為 nosound。
