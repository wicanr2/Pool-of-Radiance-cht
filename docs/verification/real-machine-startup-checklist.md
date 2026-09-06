# 實機啟動驗收清單（Windows／macOS）

**在這裡驗得到的**：Linux 版由 `tools/linux-release-smoke.sh`、Windows 版由
`tools/windows-release-smoke.sh` 在 Docker／Xvfb（Windows 那支再加 Wine）裡
實際啟動並截圖，兩支都會擋掉「視窗開得起來但畫面全黑」。

**在這裡驗不到的**：macOS——沒有機器，而 Ebitengine 的 GLFW 在套件 init 就
初始化視窗系統，無頭環境連 `-h` 都會 panic（spec 066），CI runner 也到不了
「畫得出第一個畫面」。**真的 Windows 也還沒跑過**：Wine 不是 Windows，
驅動、字型後備與 DPI 縮放都不同，Wine 過只代表「不是連跑都跑不起來」。

所以這一份清單要在一台實體 Windows 與一台實體 macOS 上各跑一次。

使用者 2026-09-05 決定由自己在實機執行，這一份就是那時照著跑的清單。

## 先準備

1. 用 `tools/package-release.sh <版本>` 產出 `dist-all/<版本>/`，把
   `pool-of-radiance-remake-<版本>-windows-amd64.zip` 與
   `-macos-universal.zip` 各拷到對應的機器。
2. 兩樣要自己準備的東西（`packaging/README-發行包.md`）：
   - 原版資料 `Pool of Radiance (1988).zip`，放在執行檔同一層。
   - 倚天字型 `stdfont.15`（要跑中文介面才需要）。

## 每一台都跑這七步

每一步都記「過／不過」，不過的話把畫面與訊息一起留下來。

| # | 做什麼 | 過的樣子 |
|---:|---|---|
| 1 | 直接雙擊執行檔（Windows）／開 `.app`（macOS）| 開得起來，不是「無法驗證開發者」擋掉就結束 |
| 2 | 看標題畫面 | 出現原版標題圖，不是全黑也不是純色 |
| 3 | 按空白鍵到密碼題，再按 Enter | 進到人物管理選擇項，四項都在 |
| 4 | `C` 建一個角色走到底（含肖像與戰鬥圖示編輯）| 每一頁都畫得出來，鍵盤有反應 |
| 5 | `A` 加入隊伍、`B` 開始冒險 | 出現 Rolf 的半身像與第一頁台詞 |
| 6 | 一路 Enter 走完導覽 | 走到自由移動，右邊的隊伍面板有名字／AC／HP |
| 7 | `F10` 離開，再開一次並 `L` 載入 | 回到離開時的位置 |

**中文介面另外跑一次第 1..3 步**：加 `-lang zh -eten-font <stdfont.15>`，
確認中文字畫得出來（缺字型時程式會直接結束並說明原因，那是預期行為，
不是當掉）。

## 每一台要留下的東西

- 第 2、5、6 三步的截圖各一張。
- 作業系統版本與機器型號（macOS 另記是 Intel 還是 Apple Silicon——
  發行包是 universal，兩種都要能開）。
- 不過的那幾步：完整的錯誤訊息或當掉時的畫面。

把結果寫回 `WORKLIST.md` 的那一項，截圖放在 `docs/screenshots/`。

## 這份清單驗不到什麼

只驗「裝得起來、開得起來、走得到第一段內容、存讀得回來」。玩法正確性由
本機的測試與對拍負責，不靠這七步。
