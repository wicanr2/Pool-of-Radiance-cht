# Spec 006：DOS 建角 portrait archive 形狀

狀態：READY（archive shape、CHA selector 欄位與循環範圍）；descriptor／組合位置仍是 DRAFT
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

## 下一個證據閘門

1. 追 portrait renderer 使用的 archive／block descriptor table，把 selector
   `1..14`／`1..12` 映射到實際 `(archive, block ID)`。
2. 對 default、HEAD next、BODY next 各做組合畫面抽樣；閉合透明色與 body y offset。
3. 完成後才可把 portrait editor 接到 `cmd/pool-game`，不能用全 109 張任意笛卡兒積。
