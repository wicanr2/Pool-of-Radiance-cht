# Pool of Radiance remake 工作清單

只列尚未完成且有驗收條件的工作。

## P0：可重現研究基線

- [x] DOS ZIP 的 168 個檔案已逐檔記錄 size、CRC32 與 SHA-256，固定 ZIP 雜湊、
  `START.EXE` 錨點、檔案數與解壓總長；報表為 `docs/audit/dos-input-manifest.json`，
  可由 `cmd/pool-input-manifest` 在不解壓、不修改來源的情況下重生。DOSBox oracle
  仍將 ZIP 複製到 tmpfs writable overlay，原始來源唯讀掛載。
- [x] 以 engine `dax` 掃描全部 DAX：113／113 成功，合計 1,245 blocks；結果在
  `docs/audit/dos-dax-inventory.json`。下一階段仍須依 payload consumer 分格式驗證。
- [x] 以 engine `geometry.Parse` 掃描 `GEO1..8`：29／29 blocks 皆為 `0x402` bytes
  且成功解成 16×16 四平面；重生報表與結構契約見
  `docs/audit/dos-geo-inventory.json`、Spec 009。Pool typed catalog 已依
  `(archive, block ID)` 接妥並通過真檔／fail-closed 測試；地名、正常入口與移動變體
  不在此完成項。
- [ ] 完成 `START.EXE`／`GAME.OVR` 的 compiler、linker 與 overlay 邊界。
  MZ／`TPOV!` 與 family 強推論已記於 `docs/re/dos-toolchain-baseline.md`；仍須以
  startup code、RTL helper bytes、overlay directory 與 IDA 位址空間交叉驗證。
  TPOV 結構層已可重生解出 38 overlays／774 entries；角色 overlay 的 IDA 9.4
  窄切片已完成，但 compiler／RTL fingerprint 仍未閉合。
- [x] 建立 DOSBox 正常啟動 oracle 與未縮放標題／主選單截圖。
  驗收：Docker/Xvfb 有界重播，輸入序列、畫面與 metadata 齊全。
- [x] 完成 `TITLE.DAX` typed consumer 與 PNG／總覽圖匯出；block 1 放大 2× 後
  與原版標題逐像素 AE=`0`，規格見 `docs/spec/001-dos-title-picture.md`。
- [ ] 閉合 Pool ECL record format：Spec 002 已訂正 payload mapping base
  `9914h → 9900h`，既有 decoder 現可完整走過 26／29 blocks、14,724 條 reachable
  instructions。只剩 ECL5/block 7、ECL7/block 17、ECL7/block 22 三筆；須逐筆判斷
  variable record、控制流 fallthrough 或真 opcode 缺口，不得以 byte 猜命令。

## P1：第一條玩家垂直鏈

- [ ] 反組譯並寫 READY 建角／建隊 spec，包含戰鬥 sprite、調色與角色檔。
  已用正確的 `C:\POOLRAD\` 掛載走通 portrait、READY／ACTION combat icon、Parts、
  雙色六部位、Size 與完成建角；285-byte CHA icon 欄位已有單變因差分，Spec 003
  對這一範圍已 READY。六種族職業清單、引導文字與 Race→Gender→Class→Alignment
  狀態機已實作；Spec 004 已 READY，閉合年齡、能力限制、exceptional STR、Gold、
  hit dice／CON／多職平均公式，並已有注入式 dice roller 與純資料生成器。剩餘驗收是
  `cmd/pool-game` 已由標題以正常按鍵走到完整公式資料頁，R 可重擲，F1／F2／ESC／F10
  已有接縫測試與 Xvfb 截圖。YES 後的 1..15-byte 姓名與 HEAD／BODY／KEEP portrait
  editor 已接正常玩家路徑；Spec 006 已閉合 `+BBh/+BCh`、1..14／1..12 wrap、
  HEAD3／BODY3 稀疏 block descriptor 與 88×88 零間隙不透明合成。Spec 007 又閉合
  CHEAD／CBODY 184 blocks、Head 0..13、Weapon 0..31、READY／ACTION × 大小 family
  與六部位雙色；remake 已接真實素材雙預覽及 Head／Weapon／Size／顏色熱鍵。剩餘驗收是
  原版 nested icon menu 細節仍待 polish；Spec 008 已接版本化 remake 角色庫、原子保存、
  icon 確認後回到 Party Creation Menu、Add／Load 與六名玩家角色上限，正常按鍵抓圖已
  走到 Party 1/6。剩餘驗收是 DOS 285-byte CHA＋多條鏈 export、完整 Party Creation Menu
  功能，以及 theme 下 sprite／tileset 同步切換。
- [ ] 解出第一張地圖的移動遮罩、WALLDEF 第一人稱畫面與第一個玩家事件。正常 Begin
  的 ECL3/block 0 → `LOAD FILES 0,0,0` 已閉合 `GEO3/block 0, (15,1), facing 6`，
  並接到 `B` 的 typed geometry 診斷總覽；地名、bounded／wrapped／door policy、事件
  與同狀態 DOS 畫面仍待證明，不得因初始 identity 已知就稱為 Phlan parity。
- [ ] 建立最小 game pack 與 adapter；不複製 CoAB 的地名、位址或劇情資料。
- [ ] 從標題以正常按鍵完成建隊、進圖、事件、戰鬥、存檔與讀檔抽樣。

## 發行決策（等待使用者）

- [ ] 公開前決定程式碼授權、repository visibility 與原版素材 deny-list。
  現階段 GitHub repository 採 private，避免在決策前公開來源或錯誤聲明。
