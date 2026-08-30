# Spec 006：DOS 建角 portrait archive 形狀

狀態：READY（只涵蓋 HEAD／BODY archive shape）；selector／組合位置仍是 DRAFT  
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

## 下一個證據閘門

1. 以同一次原版角色、相同 roll state，只變更 HEAD 一次與 BODY 一次，配對畫面及
   最終 CHA；找出 portrait selector 欄位或證明它不寫 CHA。
2. 追 overlay-16 portrait editor 的 archive／block descriptor table與 H／B wrap。
3. 對 default、HEAD next、BODY next 各做組合畫面抽樣；閉合透明色與 body y offset。
4. 完成後才可把 portrait editor 接到 `cmd/pool-game`，不能用全 109 張任意笛卡兒積。
