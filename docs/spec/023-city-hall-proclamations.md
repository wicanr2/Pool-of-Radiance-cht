# Spec 023：City Hall 公告與選單目的位址

狀態：READY／CONFORMED；日期：2026-08-31。

## 範圍與固定證據

本規格只閉合初始 Phlan 地圖中，隊伍未持有 commission 狀態時進入 City Hall 的
玩家可見流程；clerk／commission 的非零分支仍是下一個獨立切片。

- DOS 輸入 ZIP SHA-256：
  `1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `ECL3.DAX` block 0 的可重生 trace 為
  `docs/audit/dos-ecl3-block0-trace.json`；位址空間是 ECL 虛擬位址，raw payload
  基準為 `9900h`。
- `overlay-03.bin` SHA-256：
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`；
  IDA Pro 9.4、16-bit、overlay-local 位址。選單 handler 的非破壞性匯出為
  `docs/audit/ida-overlay03-menu-handlers.json`。
- `overlay-07.bin` SHA-256：
  `a59f9d16a1d186bbd3806865ffe55de4be5b58484287237fb73872295da778ae`；
  相同 IDA 契約下的 classifier／完整 value resolver 匯出為
  `docs/audit/ida-overlay07-operand-resolvers.json`。

## 原版控制流

`ECL3/block 0` 的 City Hall 分支依序為：

1. `ABD2h PRINTCLEAR` 顯示隊伍在 City Hall 外。
2. `AC11h COMPARE AND` 通過後，`AC1Eh GOSUB AF1Ch`。
3. `AF1Ch HORIZONTAL MENU` 只有
   `PRESS <RETURN> OR BUTTON TO CONTINUE`；選擇後 `AF3Fh RETURN`。
4. `AC22h PRINTCLEAR` 顯示公告前言。
5. `AC55h` 比較 `4AC1h` 與零；本切片的預設狀態為零，故 `AC5Ch GOTO AC9Bh`。
6. `AC9Bh PRINT` 顯示公告編號 LXIV、LXXVIII、CIX、LIX，`ACBEh EXIT` 回到移動。

以上文字、位址與分支為 `exact`；正常按鍵測試從 Rolf、Sune、神殿服務一路走到
City Hall，不使用 direct-entry。

## `9801h` 不是 `4AC1h` 的別名

先前曾假設 `AF1Ch` 的 menu destination 是 `4AC1h`；重新解析 record 後已推翻：

- menu header 的目的位址是 `9801h`。
- `overlay-07` classifier `070Bh..0763h` 將 `9700h..98FFh` 分為 bank 2，
  `4900h..4CFFh` 則分為 bank 0。
- 完整 value resolver `0E2Eh..0F81h` 對 bank 2 使用
  `DS:493B` 指向的 word table 與 `address*2-2E00h`；對 bank 0 使用
  `DS:4933` 指向的另一張 word table與 `address*2+6E00h`。兩者不是同一 storage。
- `overlay-03` horizontal-menu handler `11B1h..133Dh` 將 UI 選擇值與解析後的
  destination 交給 far service `0045:0070`；這支持 `9801h` 是選單結果暫存，
  不支持它會改寫 commission 狀態 `4AC1h`。

因此 remake 不得增加 `9801h → 4AC1h` alias。預設 `4AC1h=0` 時直接看公告並離開，
是本切片的正確原版行為。

## 驗收

- trace 必須輸出 `AF1Ch menu_destination=0x9801` 與唯一 Return 選項。
- 正常玩家按鍵依序驗證：外部文字 → Return menu → 公告前言 → 公告編號 →
  回到自由移動。
- 不以「最多 N 次 Return」的寬鬆迴圈代替逐 boundary 斷言。
- clerk／commission 非零狀態未在本規格宣稱完成。

實作不需要新增作品特例；既有共用 VM 的 menu continuation、COMPARE、GOSUB／RETURN、
GOTO 與 PRINT／EXIT 已能忠實走完本分支。本輪只補可重生 trace 欄位及精確正常路徑測試。
