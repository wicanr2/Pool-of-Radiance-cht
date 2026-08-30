# DOS executable／overlay 工具鏈基線

狀態：DRAFT；日期：2026-08-31。

## 固定輸入

來源 ZIP：`Pool of Radiance (1988).zip`，SHA-256
`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。

| 檔案 | bytes | SHA-256 |
|---|---:|---|
| `poolrad/start.exe` | 47,936 | `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f` |
| `poolrad/game.ovr` | 232,379 | `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638` |

## START.EXE MZ header（exact bytes）

- magic：`4D 5A`
- pages：94；last-page bytes：320
- relocations：228；header paragraphs：59
- initial `CS:IP = 0000:0006`
- initial `SS:SP = 0F92:2000`
- relocation table offset：`001Ch`
- overlay number：0

ASCII corpus 可見 `Overlay error, program abort!`、`Please insert overlay disk.` 與
`Runtime error `。這些是 overlay/runtime 候選線索，不足以單獨證明 compiler
精確版本。

## GAME.OVR（strong inference）

檔頭前 16 bytes：

```text
54 50 4F 56 21 54 69 6D 65 20 74 6F 20 73 61 76
T  P  O  V  !  T  i  m  e     t  o     s  a  v
```

`TPOV!`、START 的 overlay 錯誤字串與 CoAB 已知工具鏈共同支持
「Borland／Turbo Pascal overlay family」為強推論；尚未以 startup code、RTL helper
bytes、overlay directory 與版本字串交叉驗證，因此不可寫成已證實的精確 compiler
版本。下一步以 IDA 9.4 對固定雜湊建立非破壞性資料庫，保留原始位址與推論等級。

## 重生方式

以 Python 3.12 Docker 標準庫唯讀開啟 ZIP，對兩檔計算 SHA-256、解析 MZ 14 words，
並掃描最少六字元 ASCII。未將原始 executable／overlay 寫入 repository。

