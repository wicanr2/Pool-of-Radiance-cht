# Spec 005：第一支 remake executable

狀態：CONFORMED（只涵蓋標題 → 建角角色資料頁）  
日期：2026-08-31

## 證據與範圍

- 標題素材與逐像素 oracle 依 Spec 001。
- DOS 主選單 `C`、Home／End 與建角順序依 Spec 003 及
  `docs/playtest/dos-title-main-menu.md`。
- Race／Gender／Class／Alignment typed catalog 依 Spec 003；年齡、能力、Gold、HP
  與亂數消耗依 Spec 004。
- F1 Help、F2 original／modern presentation、ESC cancel／back、F10 quit、視窗拉伸是
  remake 操作層，不宣稱為 DOS parity；不能改動 character rules state。

## 可執行契約

`cmd/pool-game` 必須從使用者指定的 DOS ZIP 透過 typed `assets.ReadTitlePictures`
取得標題；不得提交原版 ZIP 或把 title PNG 編進公開 patch。標題 Enter／Space 到
主選單；`C` 或 Enter 只進入目前已實作的 Create 流程。Add／Load 等尚未有完整
玩家鏈的項目只能標 pending，不得導向假畫面。

建角以正常輸入依序走 Race → Gender → Class → Alignment → Roll。Roll 頁第一次
進入即由注入式 dice roller 產生 Spec 004 結果；`R` 重擲，ESC 回到 Alignment。
Enter／Y 目前只記錄「接受」，不得在姓名、portrait、icon 與 CHA writer 尚未 READY
時宣稱角色已建立。

## F2 的兩套外觀

原版只有 EGA 十六色。「現代」那一套**不是另一份美術素材**，是同一份索引像素
換一張色盤——`graphics.Picture.RGBA` 本來就收色盤參數，這是共用 engine 既有的
接縫，換皮不必複製圖。

契約三條：

1. **原版那一套的色盤必須逐格等於 `graphics.EGA16`。** 換掉任何一格就不是
   原版的畫面了。
2. **兩套都要同步涵蓋 sprite 與 tileset。** 肖像、戰鬥圖示、標題圖、牆面圖章
   與第一人稱背景色塊全部走 `app.artPalette()`。只換文字顏色而美術留在
   EGA，畫面上會變成兩套配色打架。
3. **現代色盤的第 0 格與第 15 格等於介面的背景與前景。** 其餘十四格保持
   EGA 的色相與明度順序，只收斂過飽和度——換順序會讓原版美術的明暗關係垮掉。

每格重算的素材（牆面圖章、背景色塊）下一影格自然跟上；算過一次就存起來的
三張（標題、肖像、戰鬥圖示）由 `app.switchTheme` 重畫。
`TestClassicThemeKeepsTheOriginalPalette`、
`TestModernThemeTiesArtToTheInterfaceColours` 與
`TestSwitchThemeRedrawsTheCachedArt` 釘住這三條。

## 驗收收據

- `TestKeysDriveTitleToOriginalCharacterSheet` 由 title 的 Enter 開始，經 `C` 與四次
  Enter 到真實 roll page；不是 direct-entry。
- `TestGlobalHelpThemeAndQuitKeys` 覆蓋 F1／F2／F10 的輸入接縫。
- Xvfb 960×600、逐鍵間隔 0.4 秒實跑並保存：
  `docs/screenshots/pool-remake-title.png`、`pool-remake-main-menu.png`、
  `pool-remake-pick-race.png`、`pool-remake-character-sheet.png`。
- 本 spec 不證明 portrait、combat icon、party、Phlan、ECL、戰鬥或存讀檔完成。
