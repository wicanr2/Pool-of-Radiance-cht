# Pool of Radiance remake 工作清單

只列尚未完成且有驗收條件的工作。

## P0：可重現研究基線

- [ ] 在 Docker 內解出 DOS ZIP 到唯讀來源快照，記錄每檔 SHA-256 與 manifest。
  驗收：輸入 ZIP 不被修改；manifest 可由單一命令重生。
- [x] 以 engine `dax` 掃描全部 DAX：113／113 成功，合計 1,245 blocks；結果在
  `docs/audit/dos-dax-inventory.json`。下一階段仍須依 payload consumer 分格式驗證。
- [ ] 完成 `START.EXE`／`GAME.OVR` 的 compiler、linker 與 overlay 邊界。
  MZ／`TPOV!` 與 family 強推論已記於 `docs/re/dos-toolchain-baseline.md`；仍須以
  startup code、RTL helper bytes、overlay directory 與 IDA 位址空間交叉驗證。
- [x] 建立 DOSBox 正常啟動 oracle 與未縮放標題／主選單截圖。
  驗收：Docker/Xvfb 有界重播，輸入序列、畫面與 metadata 齊全。
- [x] 完成 `TITLE.DAX` typed consumer 與 PNG／總覽圖匯出；block 1 放大 2× 後
  與原版標題逐像素 AE=`0`，規格見 `docs/spec/001-dos-title-picture.md`。
- [ ] 閉合 Pool ECL record format：既有 decoder 僅完整走過 3／29 blocks，
  其餘 26 筆不得以 opcode 猜測補洞；先依 Spec 002 追 caller／bytes。

## P1：第一條玩家垂直鏈

- [ ] 反組譯並寫 READY 建角／建隊 spec，包含戰鬥 sprite、調色與角色檔。
- [ ] 解出 Phlan 第一個地圖、入口、移動遮罩與第一個玩家事件。
- [ ] 建立最小 game pack 與 adapter；不複製 CoAB 的地名、位址或劇情資料。
- [ ] 從標題以正常按鍵完成建隊、進圖、事件、戰鬥、存檔與讀檔抽樣。

## 發行決策（等待使用者）

- [ ] 公開前決定程式碼授權、repository visibility 與原版素材 deny-list。
  現階段 GitHub repository 採 private，避免在決策前公開來源或錯誤聲明。
