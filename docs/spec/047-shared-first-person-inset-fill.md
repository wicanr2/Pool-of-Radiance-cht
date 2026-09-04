# Spec 047：共用第一人稱內框填滿

狀態：READY。

## 來源與決策

- Pool 與 CoAB 使用相同 Gold Box 三段背景、8×8 wall stamps 與透視合成機制。
- CoAB 的 README／正常遊戲畫面證明：只畫 DOS 88×88 背景時，title-owned stage
  inset 可能在四周留黑帶；牆片合成後，斜邊素材的黑底還會重新蓋住頂部角落。
- 使用者 2026-09-01 指定兩作共用此機制，不能在兩個遊戲各複製一份演算法。
- **Pool 的 inset 幾何與兩個背景色索引已量過原版**（2026-09-05，
  `docs/reference/original-dos/adventure/02-first-person-14-1-west.png`，
  由 `tools/capture-dos-adventure.sh` 在 DOSBox 內走到 GEO3/0 的 (14,1) 朝西）：
  - 那一框的內部在 320×200 座標是 `(24,24)` 起的 88×88，與契約 3 的
    `StageInset` 逐格相同——幾何本來就對。
  - 天空是 EGA 11（`85,255,255` 青），地面是 EGA 6（`170,85,0` 棕）。
    契約 3 原本寫的「保留既有 EGA blue／dark-gray」是接手時帶進來的佔位值，
    不是從原版讀的。

  **這兩個索引仍是「量出來的」，不是「從原版資料讀出來的」**：原版一定是逐區
  決定的（地城不會有青色天空），而那份資料在哪還沒找到。所以它們只對碼頭這一
  張成立；接第二張圖之前不要把它當通用值。

## 契約

1. 共用 engine `viewport.FillBackgroundToStageInset` 接收標準三段 `Background`
   與作品提供的 `StageInset`，輸出先於牆片的 `Backdrop` 及後於牆片的 `PostWall`。
2. engine 不得知道 CoAB／Pool、地名、frame 素材或 palette RGB；作品只提供 native
   inset 幾何及第一個由牆面接管的 row。
3. Pool 的現行 176×176 第一人稱視窗使用 native `(24,24,88,88)`、wall row 40；
   背景用 EGA 11（天空）與 EGA 6（地面），wall stamp 座標照原始。
4. CoAB 使用自己的 `(22,22,98,98)` inset，但消費同一 API。
5. 非法尺寸、wall row 或非三段背景失敗即關閉；輸入不得被修改。

