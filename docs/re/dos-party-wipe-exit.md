# 原版全滅之後停在哪：THE END 頁等一個鍵，然後結束程式

狀態：READY（呼叫鏈與位元組 exact；`DS:4961h` 的語意 hypothesis）
日期：2026-09-25
相關：issue #53（dosgolem 推不動全滅頁）、#19（全滅之後的去向）；
dosgolem 規格 `189-int16-blocking-read-and-exit`（分支 `fix/pool-wipe-anykey`，
commit `2ab6f65`）；收據 `docs/audit/dos-party-wipe.json`。

## 固定輸入

| 檔案 | SHA-256 |
|---|---|
| `start.exe` | `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f` |
| `game.ovr` | `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638` |

工具：dosgolem `shots -serve`（`2ab6f65`）、`objdump -m i8086`（golang:1.24-bookworm 內）、
`docs/audit/dos-ovr-manifest.json` 的 overlay 邊界。執行期位址 = 連結期段 + `0110h`；
overlay 位移 = 檔案位移 − `control_file_offset`。

路徑：`tools/dosgolem-party-wipe.py`，一級戰士進貧民窟，扣血攔截關閉，
正常按鍵打到全滅。

## 結論

1. 全滅後原版顯示 THE END 頁：`THE END`／`THE MONSTERS REJOICE FOR THE PARTY
   HAS BEEN DESTROYED`／底列 `PRESS ANY KEY TO CONTINUE`（收據 `02-anykey`，
   畫面 SHA-256 `88c28723…`）。此時 `DS:4960h = 1`、`DS:4961h = 0`。
2. 這一頁停在 Turbo Pascal CRT 的 `ReadKey`，執行期 `0622:031A`
   `int 16h AH=00h`（exact，下節）。按任意一個鍵，`consumed = 1`。
3. 按鍵之後主迴圈返回，主程式呼叫結束常式，**程式以離開碼 0 結束回 DOS**
   （`int 21h AX=4C00h`，執行期 `06CB:0192`）。沒有回標題、沒有重新載入存檔。
4. 結束當下 `DS:4960h = 0`（主迴圈返回前清掉）、`DS:4961h = 1`；
   驅動讀的 ECL class 0 `4961h` 從頭到尾是 0。

remake 目前全滅回標題（`returnToTitleAfterGameOver`），與原版不同；要不要照做是
#19 的產品決定，不在本筆記範圍。

## 呼叫鏈（exact）

| 位置 | 原始位元組／指令 | 說明 |
|---|---|---|
| ovr5 `+04B8h`（執行期 `2927:04B8`） | `C6 06 60 49 01` `mov byte [4960h],1` | 全滅時立主迴圈離開旗標（修正前的診斷跑用寫入監看量到寫入者就是這一道） |
| `0622:030C`–`032D`（CRT 單元，連結期段 `0512h`） | `A0 C9 82 / C6 06 C9 82 00 / 0A C0 / 75 12 / 32 E4 / CD 16 / 0A C0 / 75 0A / 88 26 C9 82 / 0A E4 / 75 02 / B0 03` | `ReadKey`：`ScanCode` 為 0 才 `int 16h AH=00h`；回 `AX=0` 時結果是 `#3` |
| ovr5 `+15CFh`（執行期 `2D40:15CF`） | `C6 06 61 49 01` | 按鍵之後把 `DS:4961h` 設 1 |
| ovr3 `+397Fh`–`3991h`（entry 1，stub `0130:0025`，code `377Fh`） | `80 3E 60 49 00 / 75 03 / E9 98 FE / C6 06 60 49 00 / 89 EC / 5D / CB` | 主迴圈：`[4960h]` 非零就清掉並 `retf`（同一趟監看量到清除者 `29A6:3989`） |
| `start.exe` 檔案 `4F2h`（執行期 `0110:0142`） | `9A 25 00 20 00 / 9A 00 00 6B 02 / 89 EC 5D / 31 C0 / 9A D8 00 BB 05` | 主程式本體：呼叫主迴圈 → 呼叫 `026B:0000` → `Halt(0)` |
| 執行期 `037B:0000`（連結期 `026Bh`） | `55 89 E5 A0 D3 52 A2 D2 52 FF 36 26 26 9A .. / 9A AE 00 EE 05 / 31 C0 / 9A D8 00 CB 06` | 結束常式：音效、經 `Dos.Intr` 發 `int 10h AX=0003h`（切回文字模式）、`Halt(0)` |
| 執行期 `06CB:00D8`（System 單元） | `33 C9 / 33 DB / …` | `Halt`：`ExitProc` 為空，還原中斷向量，`int 21h AH=4Ch` |

## dosgolem 為什麼推不動（已修）

- `int 16h AH=00h` 佇列空時回 `AX=0` 不等，`ReadKey` 拿到 `#3` 就往下走：
  THE END 頁一個鍵都沒按就過去，程式接著正常結束。
- 結束之後 `shots` 照樣步進：IRQ0 清掉 `Halted`，CPU 從 `int 21h` 下一道解進
  overlay 區，在錯的 CS 上執行 ovr32 `+126Dh` 的 `0E E8 31 F1`（`push cs; call`），
  跳到 `1FFE:F5C2`，之後在 ECL 資料上空轉。issue #53 看到的
  `cs_ip=1FFE:FDF7`、`consumed=0`、佇列 183，就是這個空轉狀態；
  畫面 `15f24639…` 是結束前最後一張，不是在等鍵。

照 DOSBox-X（`INT16_Handler` 沒鍵就 `reg_ip+=1` 繞回重跑；`DOS_Terminate`
把控制交回父行程）補在 dosgolem，細節與測試見規格 189。

## 還沒解決的

- `DS:4961h` 的語意：唯一的讀取點是 ovr37 `+598h`
  `80 3E 61 49 00 / 74 10 / A0 43 49 / … / 9A 9E 02 12 05`，在逐字輸出迴圈裡
  決定要不要呼叫 `Delay([4943h]×10)`。它像「慢速出字」旗標，不像全滅旗標
  （hypothesis；位元組掃描 `80 3E 61 49` 只找到這一處，指標間接讀寫未排除）。
- 結束前原版切回文字模式，dosgolem 的畫面輸出仍是 EGA 平面的舊內容
  （`03-exited` 那張），不能當「結束後的畫面」使用。
- 要重現按鍵之前那一刻，用 `-load-state` 載入本次跑出來的 `wipe-anykey.state`
  （step 504,600,000，停在 `0622:031A`）；它由 `tools/dosgolem-party-wipe.py`
  寫到 `workplace/dosgolem-cheat/`。
