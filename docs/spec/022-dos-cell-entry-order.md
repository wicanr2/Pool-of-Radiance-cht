# Spec 022：DOS 自由移動的 ECL 入口順序

狀態：CONFORMED；日期：2026-08-31。

## 原版證據

- 輸入：`overlay-03.bin`，SHA-256
  `5a3a18bd061c5b75bfad2fe6f9cff27ed5443aa4c2d59b026e4a05e2806ea68f`。
- 工具：IDA Pro 9.4；16-bit raw overlay、base 0；非破壞性匯出保留位址、bytes 與
  operands。收據：`docs/audit/ida-overlay03-lifecycle-controller.json`。
- Spec 002 已由 overlay-07 VM 初始化 `0207h..02C7h` 精確證明五個 header 依序寫到
  `DS:4944..494C`。因此 `DS:4944=entry 0`、`DS:4946=entry 1`、
  `DS:494C=entry 4`，不是名稱推測。

overlay-03 自由移動 controller `377Fh..3992h` 的原始順序如下：

1. `38DDh` push `DS:4944`，`38E2h` 呼叫 VM runner `35DBh`；若 `DS:4391` 表示
   玩家事件，`38EDh` 進 event continuation 並跳到迴圈尾，不套用移動。
2. 無事件時，`38FDh..390Eh` 把當前 X／Y 保存到 party `+1E0h/+1E2h`。
3. `3917h` 呼叫 movement service `A75h`，`391Ch` 重畫；`3921h..3957h` 比較
   移動前後 X／Y。
4. `396Ch` push `DS:4946`，`3971h` 再呼叫同一 VM runner；若有事件，`397Ch`
   進 continuation。

結論為 `exact`：一般自由移動的玩家可見合約是
`entry 0（舊座標） → 套用移動 → entry 1（新座標）`。舊 remake 先改座標才執行
entry 0，雖然初始地圖的 entry 0 全部直接 EXIT，仍屬真實順序缺口，必須修正。

## READY 實作契約

1. 先以目前 `Spawn` 投影 `C04B..C04F` 並執行 entry 0。
2. entry 0 若產生文字、選單、外部事件或未正常 EXIT，保持舊座標並暫停。
3. entry 0 正常 EXIT 才提交 GEO 移動。
4. 以新 `Spawn` 重新投影 `C04B..C04F`，再執行 entry 1；不得沿用 entry 0 的舊格
   registers。
5. 轉向不觸發這一對入口；牆／門擋下的移動也不觸發。

## 驗收

- game-pack 的兩個入口都明確投影自己的座標，而非靠呼叫端殘留 memory。
- 正常按鍵路徑仍可由 Rolf 結束走到 `(1,3)` 的 Sune 文字。
- 正常按鍵由 Sune `(1,3)` 回到 `(1,4)` 再走到 City Hall `(3,4)`；entry 1 顯示
  `YOU ARE OUTSIDE THE CITY HALL...`。單角色隊伍經兩個 Return boundary 後續到
  `PROCLAMATIONS ARE POSTED ON THE WALLS...`，證明 continuation 沒有重啟 entry。
- Pool 受影響測試與 CoAB 共用 engine 回歸通過；Xvfb 以明確 trap 管理，不把
  `xvfb-run` 清理停滯誤報為產品失敗。
