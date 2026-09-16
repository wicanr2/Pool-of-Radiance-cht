# Goal：城裡兩道收錢的門對回原版——訓練所的 1000 金與職業門、武具店的白金付款（GitHub #29、#30；spec 097／116）

主台帳：[issue #29](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/29)（訓練所）、
[issue #30](https://github.com/wicanr2/Pool-of-Radiance-cht/issues/30)（武具店）。
上一個 goal 的結案位置：[`issue-31-norris-thac0.md`](issue-31-norris-thac0.md)
（#31／#36 關：怪物 THAC0 讀 `+2Dh`，樣板 `+110h` 是殘值）。

兩條放同一輪：都是「隊伍的錢怎麼被扣」，同一組資料（`Money[7]`，spec 116 的幣別欄）、
同一條玩家策略層（#22：升級要錢、獎賞是白金）；兩條各自小，合起來一輪。

## 現況（2026-09-16，`ba3ba64` 已推送）

- **訓練所**：說明書 p.9 訓練一級 1000 金；ECL3/11 四道門各寫自己的職業遮罩進 `6DA8h`
  （spec 097，`training_gate.go`）。remake `trainMember` 把遮罩傳 0、不扣錢——探針一趟進牧師門
  把六個人全升完（補六）。overlay-16 訓練常式 `2A25h..2C60h` 的三道閘門裡沒有錢，收費點還沒讀到
  （可能在 overlay-16 別處，或在 ECL3/11 的腳本裡用 `SUB` 扣 `Money`——先問腳本再掃 overlay）。
- **武具店**：`shop.go` 的 `buy` 只看 `Money[Gold]`（「其他六種貨幣的換算還沒閉合」）；委任獎賞
  主要是白金（spec 137 獎賞表），500 白金買不了板甲。原版商店的付款順序與找零還沒讀
  （overlay-21 money services，`docs/audit/ida-overlay21-money-services.json`；spec 116 讀了估價
  與賣出那一半）。
- 工具：`cmd/pool-ecl-memory-audit`／`cmd/pool-text-inventory`（先看 ECL3/11 與商店腳本動不動
  `Money`）、`tools/ida.sh`（overlay-16／21）、dosgolem（城區走得到訓練所與武具店：
  `tools/dosgolem-reference.sh` 的 ref-shop 鍵序有到商店，`-peek` 讀隊員記錄的錢欄）。
- remake 讀點：`cmd/pool-game/training.go`（`trainMember`、檔頭「仍然不限制」）、
  `cmd/pool-game/shop.go`（`buy`，第 94 行「換算還沒閉合」）。鏡像的 verify 就是這兩句還在。

## 2026-09-16 收在哪

訓練常式 `2997h` 從頭讀完：清醒、1000 金（overlay-19 entry 11 四捨五入）、職業門、Y 確認、
`42C1h` 逐種硬幣扣款找零（spec 097〈收費與門〉）；武具店 `034Fh` 角色先於 pool、entry 15
重鑄（spec 116〈付款〉）。dosgolem 五筆收據（`dosgolem-training-fee.json` 三種錢包、
`dosgolem-shop-payment.json` 兩種）逐欄相同。remake：`treasure` 五支付款 primitive、
`trainMember` 四關＋`confirmTraining`、`shop.buy`；測試從 `Update()` 走（錢不夠、門不對、
白金、pool）。探針的 outfit 段預算改成金幣等值，但探針在古托井就全滅（#22），獎賞買板甲那
一段走不到——開 #37 調查（與攻略比對觸發、編成、獎賞與順序）。#29／#30 關。

## 提示詞（可直接貼給 `/goal`）

> 目標：訓練所收 1000 金、只放對職業的門進來；武具店能用白金（與其他幣）付款、找零照原版；
> 主台帳 #29、#30。
>
> 1. **先讀原版，不改 remake。** 兩件事：(a) 訓練所收費點——先用 `pool-ecl-memory-audit`／
>    `pool-text-inventory` 看 ECL3/11 有沒有扣 `Money` 或印「1000」的句子，沒有再掃 overlay-16
>    `2A25h` 前後與它的呼叫端（正對照：spec 097 已讀出的三道閘門要落在同一支裡）；扣的是哪一欄、
>    不夠時的訊息、門的遮罩在哪一步比對。(b) 武具店付款——overlay-21 money services 那一份匯出
>    裡找「總價 → 逐欄扣」的順序（白金 5 金、金 1、琥珀金 ？、銀 1/10、銅 1/100…以 spec 116 的
>    幣值表為準）、找零回哪一欄、錢不夠的訊息。每一條標位址、bytes、證據等級，寫進 spec 097
>    （訓練）與 spec 116（商店，或新開一份「商店付款」spec）。
> 2. **拿原版當裁判。** dosgolem：(a) 隊伍帶 1000 金以下進訓練所看訊息、帶夠了升一級後讀記錄
>    的錢欄；(b) 隊員只帶白金進武具店買一件，讀買前買後的七個錢欄。收據進 `docs/audit/`。
> 3. **改 remake。** `trainMember` 扣錢、用門的遮罩限制職業；`buy` 照原版的順序扣幣與找零。
>    測試各從 `Update()` 送鍵：錢不夠、門不對、白金買板甲三條；主線探針的 outfit 段用獎賞買到
>    板甲（補六重跑那一段）。`training.go` 檔頭與 `shop.go` 那兩句拿掉。
> 4. **不可越線：** 不改物品價格表、不給錢、不動 engine／CoAB；動到畫面（訊息列）才跑對拍。
> 5. **每輪開跑前寫「這輪跟上輪差在哪」**；先一筆（一個人、一件物品）再上探針。
> 6. **台帳：** 每輪在 #29／#30 留言；spec 097／116 狀態行；#22 若探針因此多過一段就留言；
>    收尾 `docs/worklist.json` → `go run ./cmd/pool-worklist -mode render -write WORKLIST.md` →
>    `-mode verify` → `gh issue list --state open`。
> 7. **停止線：** 訓練所收費點 ECL 與 overlay-16 都找不到（那要另開 issue 擴掃描面）；商店付款跨
>    overlay-21 以外兩顆還讀不完；dosgolem 走不到訓練所或武具店（先修驅動）。

## 已知風險與待決

- 訓練費可能是腳本扣的（ECL `SUB` 對 `Money`），那就不在 overlay 裡；先查腳本再掃 overlay，
  別在 overlay-16 裡找一整天。
- 幣值換算若原版是「只找零成金幣」之類的簡化，照原版，不套 AD&D 說明書的匯率。
