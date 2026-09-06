# Spec 066：三平台發行包

狀態：CONFORMED（Linux AppImage、Windows ZIP、macOS 雙架構 ZIP 可重生，
AppImage 已在容器裡實際啟動並截圖）；DRAFT（Windows 與 macOS 尚未在真機驗收）。
日期：2026-09-03。

## 產出

`tools/package-release.sh <版本>` 由 Docker 工具鏈產生四個檔案，
並寫一份 `dist-all/<版本>/manifest.json` 固定各檔的大小與 SHA-256：

| 檔案 | 內容 |
|---|---|
| `pool-of-radiance-remake-<版本>-x86_64.AppImage` | Linux |
| `pool-of-radiance-remake-<版本>-windows-x86_64.zip` | Windows，含 `啟動遊戲.bat` |
| `pool-of-radiance-remake-<版本>-macos-x86_64.zip` | macOS Intel，`.app` bundle |
| `pool-of-radiance-remake-<版本>-macos-arm64.zip` | macOS Apple Silicon |

macOS 以 osxcross 交叉編譯（`u2cht-osxcross:20260826-r1`），不需要 Mac。
`dist-all/` 由 `.gitignore` 排除，產出物不進版控。

## 發行包不含什麼

**原版遊戲資料**與**倚天點陣字型**都不隨包散布——兩者的權利都不在本專案
（見 `NOTICE.md`），玩家要自己準備。`packaging/README-發行包.md` 說明放哪裡。
沒有字型時 `-lang zh` 直接結束並說明原因，不會默默用英文跑。

圖示是原創的（`tools/build-release-icon.py` 畫的同心圓），不用原版標題畫面——
那是 SSI 的美術，拿它當圖示等於把原版素材放進發行包。

授權條款跟著每一個發行包走：`LICENSE` 與 `NOTICE.md` 複製進四個包的根目錄，
AppImage 另外放一份在 `usr/share/doc/`。拿到 ZIP 的人看不到儲存庫，
包裡沒有這一份，收到的人就不知道自己被授權了什麼。

## AppDir 不夾帶任何 `.so`

這個執行檔只連 libX11／libxcb／libXau／libXdmcp 加 glibc，全部屬於系統圖形
堆疊，本來就該由主機提供。

**從建置 image 複製過去反而會壞。** 那幾顆是對 glibc 2.38 連結的，搬到 glibc
較舊的機器上會噴 `version GLIBC_2.38 not found`——而且是在建置階段完全正常、
到玩家手上才失敗，兩者在報表上分不出來。第一版就是這樣打的，
`tools/linux-release-smoke.sh` 在容器裡啟動時當場攔下：

```
/tmp/run/squashfs-root/usr/bin/pool-game: /lib/x86_64-linux-gnu/libc.so.6:
version `GLIBC_2.38' not found (required by .../usr/lib/libX11.so.6)
```

改成不夾帶之後同一支腳本啟得動，截圖見
`docs/screenshots/pool-release-linux-appimage.png`。

## 存檔位置

存檔預設寫到作業系統的使用者設定目錄（`pool-of-radiance-remake/state.json`），
不是工作目錄：AppImage 與 `.app` 都是唯讀的，寫在旁邊會失敗，
而那個失敗要到玩家按下存檔才會出現。`-save <路徑>` 可以改。

## 驗收

- `tools/linux-release-smoke.sh <版本>` 在 Docker／Xvfb 裡用
  `--appimage-extract` 啟動 AppImage（容器沒有 FUSE），確認視窗開得起來並截圖。
  驗的是包內容與相依，不是 FUSE 掛載本身。
- `tools/windows-release-smoke.sh <版本> [口味]` 在 Docker／Wine／Xvfb 裡跑
  `pool-game.exe`，等它畫出畫面之後從 root 裁下視窗並**擋掉全黑**——視窗開得
  起來但沒畫東西，和啟動失敗一樣糟。兩個坑寫在腳本裡：先跑 `wineboot -u`
  （第一次要建 prefix，不先做等再久都是全黑），截圖要從 root 裁
  （沒有視窗管理員時 `import -window <id>` 拿到的是全黑）。
- macOS 目前只有「建得出來、包得起來」。**Wine 過不等於 Windows 過**：
  驅動、字型後備與 DPI 縮放都不同。兩者的真機啟動驗收見
  [`docs/verification/real-machine-startup-checklist.md`](../verification/real-machine-startup-checklist.md)。

## 無頭環境跑不動這個執行檔

實測：Ebitengine 的 GLFW 在**套件 init** 就初始化，沒有顯示器時連 `-h` 都會
panic——不是我們的錯誤處理沒接上，是 process 根本起不來。

```
PlatformError: X11: The DISPLAY environment variable is missing
panic: NotInitialized: The GLFW library is not initialized
```

因此「啟得動」這件事在無頭環境驗不了，Linux 那支 smoke 測試才要起 Xvfb。
`.github/workflows/platform-smoke.yml` 對 Windows 與 macOS 只驗原生建置與
測試（那兩件事交叉編譯給不了），不嘗試啟動執行檔；真正的啟動驗收仍需要
有桌面工作階段的機器。

該 workflow **只用手動觸發**：repo 目前是 private，Actions 的用量與 log
可見性都跟著 repository visibility 走，而那還沒定案。

## 不做

- 不把原版 ZIP 或倚天字型放進任何可公開散布的包。
- 不用 GitHub Actions 建置：工具鏈（osxcross、appimagetool）都在本機 image 裡，
  搬上 CI 是另一件事，要先決定 repository visibility。
