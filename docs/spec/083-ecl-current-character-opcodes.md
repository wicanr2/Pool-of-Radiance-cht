# Spec 083：`39h WHO` 與 `36h ADD NPC`，以及它們共用的「目前角色」槽

狀態：DRAFT（兩條 handler 的骨架與呼叫目標已解出；被呼叫的挑人常式與
NPC 載入常式還沒讀，`+84h` 的語意未閉合）。日期：2026-09-03。

## `DS:5CF0h` 是「目前這個角色」

三條 opcode 都圍著同一個 far pointer 轉：

- `38h PROGRAM`（spec 081）在 `3174h` 把 `[4394h]` 搬進 `5CF0h`。
- `39h WHO` 把玩家挑到的角色**寫進** `5CF0h`（傳的是 `DS:5CF0h` 這個位址，
  不是它的內容——出參）。
- `36h ADD NPC` 只**讀** `5CF0h`，把欄位寫進它指到的記錄。

所以 remake 這一側要有同一個概念：一個「目前角色」的指標，由 `39h` 設定、
被後面的 opcode 讀。

## `39h WHO`（overlay-03 `2E90h`，1 個運算元）

```
2E97h  取運算元 1（overlay-07 entry 2）
2EA5h  overlay-37 entry 14，參數 (1, 11h, 26h, 16h)   ; 清一塊框
2EB0h  從 DS:6E8Eh 讀一個字串到區域緩衝區（上限 FFh）
2ECDh  overlay-25 entry 42（code 2C81h），參數是 &DS:5CF0h
```

`0198h:0066h`（overlay-37 entry 14）也被 `3Dh CLEAR BOX` 用，參數換成
`(1, 1, 0Fh, 0Fh)`，兩組都是「左、上、右、下」，所以那支是清一塊矩形。
`39h` 清的框比 `3Dh` 的低（上緣 11h 對 1），是畫面下半的提示區。

`DS:6E8Eh` **在 START.EXE 的檔案映像之外**（檔案 47,936 bytes，該位址算出來
是 58,942），也就是 BSS，執行時才填。所以提示字串靜態讀不到，只能從
runtime 或呼叫端反推。

## `36h ADD NPC`（overlay-03 `2EDBh`，2 個運算元）

```
2EE1h  取兩個運算元
2EE9h  a = 運算元 1 的值
2EF4h  overlay-17 entry 9（code 1216h），參數 a          ; 依編號載入 NPC
2EFDh  b = 運算元 2 的值
2F08h  b = b / 2（有號）
2F19h  b |= 80h
2F21h  [5CF0h] 指到的記錄 `+84h` = b
2F2Ah  a = 18h 時 `+10Eh` = 1，否則 0                     ; 陣營（spec 057）
2F46h  overlay-25 entry 7（code 0BBEh）
2F53h  overlay-25 entry 2（code 0762h）                   ; 重算衍生值
```

最後那支就是 spec 079／080 那條重算鏈的入口，所以 NPC 一加進來就會照
一般角色的規則重算 AC、負重與移動力。

`18h` 那個特例只有一個編號吃得到——它是唯一會站到另一邊的 NPC。

## 還沒讀

- overlay-25 entry 42（`2C81h`）：挑人的 UI 與它挑的範圍（全隊？含 NPC？）。
- overlay-17 entry 9（`1216h`）：NPC 記錄從哪個封存檔來、編號怎麼對。
- 記錄 `+84h` 的語意。`(值 / 2) | 80h` 這個形狀說明它是個帶旗標的欄位，
  但讀取端還沒找到。
- overlay-25 entry 7（`0BBEh`）在重算之前先做了什麼。

## 為什麼先記下來

兩條都在起始地圖走得到（`39h` 在 ECL3／block 0 兩處、block 11 一處；
`36h` 在兩個 block 各一處），是目前擋在探索前面的兩條。骨架讀出來了，
但挑人 UI 與 NPC 載入都還沒讀——**在讀完之前維持硬失敗**，因為
「跳過一個加入隊伍的 NPC」與「正確處理它」在報表上分不出來。
