# Spec 009：DOS GEO 地圖盤點與 Phlan 入口

狀態：CONFORMED（GEO archive／block shape 與 Pool typed catalog）；READY（正常新遊戲初始 map identity／座標／朝向）；DRAFT（移動變體、第一事件與地名）
日期：2026-08-31

## 可重生結構盤點

- 輸入 DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `cmd/pool-geo-audit` 只讀 `GEO1.DAX..GEO8.DAX`，先由 engine `dax.Parse` 解 archive，
  再由 engine `geometry.Parse` fail-closed 解 16×16 四平面。
- 報告：`docs/audit/dos-geo-inventory.json`。

八份 archive 合計 29 blocks；29／29 payload 皆為 `0x402` bytes（兩-byte prefix＋
四個 `0x100` planes），全部成功解成 16×16 cells，失敗 0。報告逐 block 保存：
原始 block ID、prefix、terrain histogram、四向 wall／detail 統計，以及 bounded、wrapped、
dungeon-door 三種 directed move edge 數。這些數字只證明資料與 engine contract 相容；
wrapped 與 dungeon-door 哪一個適用，必須由該 ECL／area consumer 決定。

驗收命令在 `coab-go-test:20260729` 的無網路 Xvfb 容器執行；使用暫存
`go.work` 將本專案連到唯讀的同層 engine，未在 `go.mod` 提交本機 `replace`。
`go test ./...` 全數通過；重新執行 `cmd/pool-geo-audit` 的 JSON 與版控報表相同，
且機械斷言確認 `(archives, blocks, decoded, failed) = (8, 29, 29, 0)`、每列
`bytes = 1026` 且沒有 `error`。

## 正常新遊戲初始 map（READY）

固定 DOS ZIP、overlay hashes、IDA 版本與位址空間同下節。下列垂直鏈已由原始 producer
與 consumer 閉合；不依賴尚未恢復的 DOSBox 選單自動輸入：

1. 全域初始化 `overlay-11:002Dh..06E5h` 先在 `026Dh..027Ah` 將 `DS:4933`
   指向的 0x800-byte party record 清為 0；因此新 session 的 `party +01E4h`
   （目前 ECL block ID）是 0。`0329h..0337h` 再設定位置／朝向
   `DS:6A0B=15`、`6A0C=1`、`6A0D=6`。
2. Party Creation Menu `overlay-16:0155h` 設 archive `DS:52D4=3`；Begin 分支
   `0426h..0445h` 只將 mode 設為 4，不改上述 ECL block 或位置欄位。
3. adventure controller `overlay-03:377Fh..3991h` 在 `37A8h..37B1h` 從
   `party +01E4h` 讀取 ECL block ID 0，`37E0h..37E4h` 交給 overlay-07 ECL loader。
   該 loader `0353h..0420h` 以 `DS:52D4` 選 ECL archive 3。
4. VM 初始化從 ECL3/block 0 的第五 command-set entry `9AF2h` 執行；依修正後
   `9900h` payload mapping base，這是 raw payload offset `01F2h`（498）。順序執行
   到 offset `020Ah`（522）的 opcode `21h LOAD FILES 0,0,0`。
5. `LOAD FILES` handler `overlay-03:0D80h..0ED4h` 於 `0DDBh..0DF1h` 將第一個
   operand 0 寫入 `party +018Ah`，並呼叫 GEO loader；GEO loader
   `overlay-30:10EAh..1225h` 以 `DS:52D4=3` 開啟 `GEO3.DAX`、載入 block 0。
6. 第五 entry 在首次 `LOAD FILES` 前後的直接 `SAVE` destinations，以及其條件式
   `GOSUB 9A86h`（raw offset `0186h`），均未寫 `C04Bh..C04Dh` 的位置 special
   variables；menu 與兩個 loader 亦不寫 `6A0B..6A0D`。因此首次 map frame 保留
   初始化值 `(x=15, y=1, facing=6)`。

結論等級為 `exact`：正常新隊伍的初始資料 identity 是
`(GEO archive 3, block 0, x 15, y 1, facing 6)`。此 READY 範圍授權 game-pack
建立 typed spawn 並由 `B` 進入該 map；它**不授權**把 block 0 命名為 New Phlan、
不決定 bounded／wrapped／dungeon-door 移動語意，也不宣稱第一事件已接回。

## 尚未閉合

- `GEO1` 不因編號最小就自動命名為 New Phlan 或 Slums。
- 第一個畫面事件與城內／地城 wrap 規則必須沿同一正常路徑驗證；direct-entry 只能縮小
  問題，不能作完成證據。
- terrain ID、wall art selector 與地名是不同資料層；本 inventory 不替它們猜名稱。

因此本規格已授權初始 spawn 與 map identity，但目前畫面只能標成 typed geometry
preview；在 WALLDEF／第一人稱 renderer 與正常 DOSBox 同狀態畫面閉合前，不得把它
宣稱為原版視覺 parity。

## 正常 Begin 路徑的新增 RE 證據（仍為 DRAFT）

輸入與工具：

- `overlay-16.bin` SHA-256：
  `a142d8a8f3b3c46a7755cf231228e7981c5b9f77721313ce79ab6c0aec105b94`；
- `overlay-25.bin` SHA-256：
  `9fede24be1e64821c62ab2421783b004a06fa9d305aa57919bd15a9577b50c0e`；
- `overlay-27.bin` SHA-256：
  `8629cfbb045deb8e56724ef4737930e3ddb680a4669c9980d655a9d9ace083ad`；
- `overlay-11.bin` SHA-256：
  `d1e1c62ef11ddf26607bca6bcb183fc70218a4629fde664e38fabd76966e4e0a`；
- `overlay-17.bin` SHA-256：
  `f92fed1bcf009daa8638ba0ab1f8f9a680c0ad3b2b43b799faa9383b02f7d1e4`；
- `overlay-30.bin` SHA-256：
  `9a29e0fe6624f69fd0a5c5789c81c98409380ea36112884862720b0bf473da91`；
- IDA Pro 9.4、`ida-pro-9.4-idapython:locked-v1`；位址均為各 overlay local offset。
  可重生匯出器為 `tools/ida-export-overlay-functions.py`，保留 entry seed、原始 bytes、
  operand、SHA-256、IDA 版本與 16-bit segment，不以推測性改名取代定位。

已證實的控制流：

1. Party Creation Menu 是 overlay-16 entry 1，`014Eh..04DDh`；`02C2h` 引用
   `Choose a function`。`0426h` 比對輸入 `B`，通過 enable／party pointer 檢查後，
   `043Ah..0440h` 把 local 初始值 `4` 寫入 resident `DS:4954h` 並返回。
2. overlay-25 `280Fh..2927h` 是該模式的 consumer；`2815h` 讀 `DS:4954h`，
   `28AAh` 比對 `4`，其分支呼叫 resident relative `0124:0025` 後顯示 party。
   raw opcode census 同時看到其他模組把 `DS:4954h` 寫成 1、2、3、5、6、7，故它是
   功能／模式欄位，**不是 GEO archive 或 block ID**。
3. MZ header 是 `0x3B0` bytes；`0124:0025` 的 relative linear `0x1265` 加 header
   對應 START file offset `0x1615`，落在 overlay-27 descriptor（file `0x15F0`）的
   stub `0x25`，即 overlay-27 entry 1、code `0181h`。該函式讀 `DS:6A0B..6A0D`
   交給視圖 routines，但不寫出生狀態；目前只能把三欄標為位置／朝向候選。
4. overlay-11 entry 1 是全域初始化鏈；`0329h..0337h` 明確寫
   `DS:6A0B=15`、`6A0C=1`、`6A0D=6`。這證明 DOS session 的 default 初始化值，
   但尚未證明 Begin 前後沒有 ECL／load path 覆寫，所以不可先寫成 Phlan 出生點。
5. Party Creation Menu 本身在 overlay-16 `0155h` 把 `DS:52D4` 寫成 `3`；Begin
   分支 `0426h..0445h` 只檢查隊伍、把 mode 寫成 `4` 並返回，分支內沒有改寫
   `DS:52D4` 或呼叫地圖 loader。因此「新遊戲進入 Adventure 時的目前檔集為 3」
   已由同一路徑證實；這仍不等於已知 GEO block。
6. overlay-30 SHA-256
   `9a29e0fe6624f69fd0a5c5789c81c98409380ea36112884862720b0bf473da91`；entry 10
   `10EAh..1225h` 以 `DS:52D4` 組成 `GEO%d.dax`，用唯一 byte 參數取 block，嚴格
   要求解碼長度 `0x402`，略過兩-byte prefix 後各複製四段 `0x100`，最後在
   `1214h..121Dh` 把該 block ID 寫入 party record `+018Ah`。這閉合
   `DS:52D4 = archive`、entry argument `= GEO block` 與存檔 consumer 的方向。
7. overlay-17 SHA-256
   `f92fed1bcf009daa8638ba0ab1f8f9a680c0ad3b2b43b799faa9383b02f7d1e4`；讀檔流程
   `1B56h..1B73h` 從 party-save record `+0624h` 恢復 `DS:52D4`，在 mode < 2 時
   從 party record `+018Ah` 取 GEO block 並呼叫 loader。這是讀檔 producer，不能
   反推尚未存檔的新隊伍 block。

本節原先記錄的最小缺口已由上方「正常新遊戲初始 map」閉合。保留下列勘誤：先前
Spec 002 把 `9914h` 誤作 ECL mapping base，導致 ECL3/block 0 的第五 entry 無法可靠
走到 `LOAD FILES`；修正為 `9900h` 後，`9AF2h - 9900h = 01F2h`，上述控制流成立。

## Typed adapter

`internal/gamepack.ReadDOSGeometryCatalog` 已實作本規格授權的結構範圍：

- 以 `(GEO archive 1..8, original block ID)` 作唯一 `MapKey`，不讓不同 archive 的
  相同 block ID 互相覆蓋；
- 保留兩-byte prefix，地圖內容使用 engine `geometry.Grid`；
- 嚴格要求八份 archive、29 張 map，缺檔、重複 archive／block、DAX 或 GEO 格式錯誤
  均失敗即關閉；
- `Map` 回傳值副本，門鎖等 runtime mutation 不會污染原始 catalog；
- 真實 DOS ZIP 測試固定全部 29 組 archive／block identity、`GEO1/block 18` 的
  `[0,4]` prefix anchor，並證明不存在的 `GEO1/block 0` 不會被補造。

這使結構與 typed adapter 範圍升為 `CONFORMED`；入口與故事語意仍維持 `DRAFT`。
