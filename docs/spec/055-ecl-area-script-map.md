# Spec 055：ECL 區域腳本對照

狀態：READY（自載 `LOAD FILES` 清單、八個城內區域的腳本定位與識別字串）；
DRAFT（各區事件點數量、GEO 逐格內容、非自載 block 的角色）。
日期：2026-09-02。

## 原始證據

`workplace/bin/pool-ecl-trace` 對 29 個 ECL block 逐一靜態展開，26 個成功；
失敗的三個（`ECL5/block 7`、`ECL7/block 17`、`ECL7/block 22`）與
`docs/audit/dos-ecl-coverage.json` 的 `failed_blocks: 3` 一致，屬 spec 002 的既有缺口。

### 自載 `LOAD FILES` 是區域腳本的判準

spec 042 認定 Slums 腳本時用的是 `9A8Bh LOAD FILES 20,2,FFh` ——第一參數等於自身
block ID。把這個判準套到全部 26 份 trace，命中 **16 個 block**：

`ECL1/18`、`ECL2/9`、`ECL2/15`、`ECL2/20`、`ECL3/14`、`ECL4/2`、`ECL4/10`、
`ECL4/21`、`ECL5/4`、`ECL5/6`、`ECL6/1`、`ECL6/28`、`ECL7/23`、`ECL8/13`、
`ECL8/16`、`ECL8/29`

這 16 個是「載入自己那張地圖」的腳本。相對地，`ECL3/block 8` 的
`LOAD FILES 0,0,0` 屬 spec 025 已閉合的 transition，不是區域腳本——雖然它的文字
提到幾乎所有地名（市議會派發任務時會列出各區），**不可因為關鍵字命中就當成該區腳本**。

## 城內八區的定位

命名依據是各 block 內的原始英文字串，逐條可回查：

| 區域 | ECL block | 自載 GEO | 識別字串（節錄）|
|---|---|---:|---|
| 新菲蘭城／碼頭 | `ECL3/0` | 0（`LOAD FILES 0,0,0`）| `THE HARBOR MASTER TELLS YOU BOATS LEAVE FOR THE WEST, THE EAST, SOKAL KEEP, AND THE NORTH SIDE OF THE BAY.` |
| 貧民區 | `ECL2/20` | 20 | spec 042 已閉合 |
| 索卡爾城堡 | `ECL4/21` | 21 | `THE SAGE MENDOR WORKED HARD TO GATHER RECORD OF ALL THESE THINGS, BUT THEY ARE LOST NOW, HIS LIBRARY OVERRUN.` |
| 古托井 | `ECL8/29` | 29 | `BEFORE YOU LIES KUTO'S WELL.  IT IS A SOURCE OF WATER IN ALL SEASONS.` |
| 曼多爾圖書館 | `ECL2/15` | 15 | `THESE ARE THE LIBRARY STACKS. OLD AND MOLDERING BOOKS ARE STORED ON SHELVES.` |
| 波多廣場 | `ECL1/18` | 18 | 含 `AUCTION` 與 `THE PLAZA` |
| 卡德納紡織廠 | `ECL4/2` | 2 | `YOU FIND AN IRON BOX, ACROSS THE LOCK IS THE SEAL OF THE FAMILY OF CADORNA.` |
| 瓦海登墳場 | `ECL4/10` | 10 | 含 `VAMPIRE` 與 `CRYPT` |

`ECL8/29` 另外發出 `LOAD FILES 32,2,FFh`，指向自身以外的地圖；古托井有地下墓穴，
這是最可能的解釋，但 **32 超出目前 GEO 盤點的 29 個 block**（`dos-geo-inventory.json`），
所以在解出該來源前不得宣稱它就是地下墓穴。

## MENDOR 與 MANTOR 的一手證據

`MENDOR` 在全部 26 份 trace 中只出現於 `ECL4/21`；**`MANTOR` 一次都沒有出現**。
遊戲原始資料只認 Mendor，`Mantor's Library` 是桌上模組 Ruins of Adventure 的名稱。
《軟體世界》創刊號攻略內文寫 `Mantor`，係沿用模組那一系，不採。

有意思的是這個字串出現在**索卡爾城堡**而不是圖書館本身——圖書館現場的腳本是
`ECL2/15`，它的字串是 `LIBRARY STACKS`。這與二手資料所述「模組在索卡爾城堡的遭遇
提到 the sage Mendor」相符，但本項結論由原始 ECL 字串獨立證成，不依賴該二手說法。

## READY 契約

1. 區域腳本的認定一律要有「自載 `LOAD FILES`」加「該 block 內的原始字串」兩項證據；
   只有關鍵字命中不足以認定，因為 transition 與任務派發腳本會列舉所有地名。
2. GEO block 只能由 `LOAD FILES` 的第一參數取得，不得由 ECL block 編號推定
   （spec 009 的既有禁令）。本表的「自載 GEO」欄即該參數本身。
3. 未命中自載判準的 block（如 `ECL6/19`）在找到其角色的證據前，不得歸給任何區域。

## 待閉合

- 各區的事件點數量。Slums 已完成：14 個 `GOSUB B69Ch` 計數呼叫點對上完成門檻 25
  與攻略的「約 15 群隨機怪物」（見 spec 042 與
  [`docs/reference/walkthrough-notes-softworld-001.md`](../reference/walkthrough-notes-softworld-001.md)）。
  其餘七區要各自找出對應的計數 helper 後才能比對攻略的 24／10／9／15／4／9。
- `ECL8/29` 的 `LOAD FILES 32` 來源。
- 其餘八個自載 block（`ECL2/9`、`ECL3/14`、`ECL5/4`、`ECL5/6`、`ECL6/1`、`ECL6/28`、
  `ECL7/23`、`ECL8/13`、`ECL8/16`）對應哪些地圖；這些多半是城外與後期區域，
  攻略「城內篇」不涵蓋。
