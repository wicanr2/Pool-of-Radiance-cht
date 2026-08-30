# Spec 006：DOS 建角 portrait archive 形狀

狀態：READY（archive shape、CHA selector、循環、descriptor 與合成幾何）
日期：2026-08-31

## 輸入與工具

- DOS ZIP SHA-256：`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。
- `cmd/pool-portrait-audit` 只接受名稱恰為 `HEAD1..8.DAX`／`BODY1..8.DAX` 的
  16 份 archive；每份先由 engine `dax.Parse` 解 block，再由 engine
  `graphics.ParsePicture(block.Data,false,0)` 解圖片。無 fallback 或略過壞檔。
- 可重生結果：`docs/audit/dos-portrait-shapes.json`。

## 已證實形狀

16／16 archives、109／109 blocks 全部解碼，失敗 0；每 block 恰有一張 picture：

| 類別 | archive | blocks／pictures | 固定尺寸 | 固定 payload bytes |
|---|---:|---:|---:|---:|
| HEAD | 8 | 63 | 88×40 | 1,777 |
| BODY | 8 | 46 | 88×48 | 2,129 |

block ID 不是連續索引；例如 HEAD3 有 21 個稀疏 ID，而 BODY1 只有 `3／7／25`。
runtime／exporter 必須保存 `(source archive, block ID)`，不可把排序後位置冒充原版
selector。40＋48 恰為 88；實際合成方式由下方原版像素比對閉合。

## 已證實 selector 欄位與循環

輸入 `overlay-16.bin` 的 SHA-256 為
`a142d8a8f3b3c46a7755cf231228e7981c5b9f77721313ce79ab6c0aec105b94`；使用
IDA Pro 9.4，以 overlay-local 位址空間分析。portrait editor 位於
`33A9h..34A0h`：

- `33B9h..33C1h` 從目前角色 CHA `+BBh` 讀出 HEAD selector；
- `33C4h..33CCh` 從 CHA `+BCh` 讀出 BODY selector；
- `3417h..3437h` 處理 `H`：HEAD 小於 `0Eh` 時加一，否則回到 `1`，所以原版
  可保存範圍是 `1..14`；
- `343Bh..345Bh` 處理 `B`：BODY 小於 `0Ch` 時加一，否則回到 `1`，所以原版
  可保存範圍是 `1..12`；
- `345Fh..3493h` 的 `K` 接受目前兩值並離開。這段函式未觀察到取消分支，不把
  remake 的取消操作冒稱原版行為。

這兩欄是 portrait selector，不是戰鬥 icon：後者由同一 overlay 的
`37F2h..3F01h` 編輯 CHA `+BDh/+BEh`。以上位址均為 overlay-local，不是 START
resident 位址或原始 ZIP file offset。

## 已證實 archive／block descriptor

overlay-16 `014Eh..` 在建角初始化把 DS `52D4h` 設為 `3`。portrait renderer 的
far call 經 START overlay stub `012Bh:0043h` 映射到 overlay-29 entry 7、local
`04C4h`；overlay-29 SHA-256 為
`f085c0ab8b22d74153e7bf2a6318eefbeb28acaf959eea5fada77a88f9cf95f0`。
renderer 的 `05A4h` 使用 Pascal 字串 `HEAD`／`BODY` 加上 DS `52D4h`，因此建角
選擇器讀取 `HEAD3.DAX`／`BODY3.DAX`，不是在 16 個 archive 間任意搜尋。

`04C4h` 以 selector 查 DS `2883h`／`2891h` 後的 byte；START.EXE 由 IDA DOS
loader 載入基址 `10000h`，資料段基址 `17400h`，對應 bytes 如下：

| selector | HEAD3 block ID | BODY3 block ID |
|---:|---:|---:|
| 1 | `00h` | `01h` |
| 2 | `08h` | `02h` |
| 3 | `09h` | `03h` |
| 4 | `0Dh` | `04h` |
| 5 | `10h` | `07h` |
| 6 | `12h` | `08h` |
| 7 | `16h` | `12h` |
| 8 | `22h` | `18h` |
| 9 | `2Dh` | `1Ah` |
| 10 | `33h` | `21h` |
| 11 | `35h` | `23h` |
| 12 | `39h` | `25h` |
| 13 | `43h` | — |
| 14 | `44h` | — |

這 14／12 個 ID 分別全部存在已稽核的 `HEAD3.DAX`／`BODY3.DAX`。DS `2883h`
同時是其他流程使用的 scratch record 區；本結論只適用於建角 portrait renderer
呼叫時的狀態，不能把該地址全域命名成唯讀肖像表。

## 已證實合成幾何

原版 `TEST.CHA` 的 portrait selectors 為 `1／1`。將同一 descriptor 解出的
`HEAD3:00h` 與 `BODY3:01h` 以 EGA palette 對照既有 640×400 DOS runtime capture
`workplace/oracle/character-debug/colors/00-base.png`：

- HEAD 3,520／3,520 像素完全相同，位於螢幕 `(448,16)`、2× 顯示；
- BODY 4,224／4,224 像素完全相同，位於螢幕 `(448,96)`、2× 顯示；
- 換算 320×200 原始座標為 HEAD `(224,8)`、BODY `(224,48)`；BODY y offset
  正好等於 HEAD 高度 40，因此沒有間隙或重疊；
- 兩張 unmasked picture 的全部像素皆為 opaque，不能把 palette index 0 當透明色。

所以建角 portrait 是保留每個 4-bit palette index 的 88×40／88×48 垂直串接，
結果固定為 88×88。這不是 combat icon 的 masked／OR merge 規則。

## 實作閘門

portrait editor 可使用上述 14×12 原版 selector 組合；不得把其他 archive 的 109 張
圖片混入玩家建角選單。合成必須保留 index 0，不能重用 combat icon 的透明／OR merge。
