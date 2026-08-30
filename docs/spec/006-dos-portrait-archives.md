# Spec 006：DOS 建角 portrait archive 形狀

狀態：READY（archive shape、CHA selector、循環與 archive/block descriptor）；組合位置仍是 DRAFT
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
selector。40＋48 恰為 88 只支持可組成 88×88 的 shape；在尚未從 overlay 或同源
runtime 差分閉合疊合 y、透明規則與可選集合前，不宣稱直接上下拼接就是原版。

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

## 下一個證據閘門

1. 對 default、HEAD next、BODY next 各做組合畫面抽樣；閉合透明色與 body y offset。
2. 完成後才可把 portrait editor 接到 `cmd/pool-game`；descriptor adapter 可以先接，
   但不能用全 109 張任意笛卡兒積或猜測合成幾何。
