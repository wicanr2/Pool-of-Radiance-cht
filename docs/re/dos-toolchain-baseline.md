# DOS executable／overlay 工具鏈基線

狀態：READY（compiler family、linker／格式、overlay 邊界、RTL 指紋，
每一條都由原版位元組重生）；DRAFT（精確的 Turbo Pascal 版本號）。
日期：2026-09-04（原 2026-08-31）。

`TestDOSToolchainFingerprint`（`internal/gamepack`）把下面每一項從
原版 ZIP 重生一次，資料換版就會紅。

## 固定輸入

來源 ZIP：`Pool of Radiance (1988).zip`，SHA-256
`1a7386c3842d3c6b0d02d9e607af249b2b452adb396a17f0971d09b2346b1633`。

| 檔案 | bytes | SHA-256 |
|---|---:|---|
| `poolrad/start.exe` | 47,936 | `12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f` |
| `poolrad/game.ovr` | 232,379 | `bc4e3c32daf04b87db0c0a9508bb67b94d9f138aa39ed1dae7e32911b3171638` |

## START.EXE MZ header（exact）

- magic `4D 5A`；pages 94；last-page bytes 320
- relocations 228；**header paragraphs 59**（碼段從檔案位移 `3B0h` 起）
- 進入點 `CS:IP = 0000:0006`；初始 `SS:SP = 0F92:2000`
- relocation table 在 `001Ch`；overlay number 0

## 進入點是 Turbo Pascal 的單元初始化鏈（exact）

`0000:0006` 起是 **49 個連續的 `9Ah` 遠呼叫**，一個接一個，中間沒有別的指令：

```
0006  call 05BB:0000     ; 常駐單元（RTL）
000B  call 0512:0000
0010  call 0503:0000
0015  call 0502:0000
001A  call 04DE:022B
...
0033  call 0198:0020     ; ← overlay 的 stub，位移固定 0020h
0038  call 018E:0020
003D  call 018A:0020
...
```

其中 **38 個的位移正好是 `0020h`**，而 `GAME.OVR` 正好有 **38 顆 overlay**——
`0020h` 是 stub 表的第一筆，也就是 entry 0，Turbo Pascal 放單元初始化的地方
（見 `knowledge-base/retro/borland-tpov-overlay-re.md`）。剩下 11 個呼叫進的是
常駐單元。

**這是最強的一條**：編譯器產生的程式進入點就是「逐一初始化每個單元」，
而 overlay 化的單元透過 stub 初始化。數量對得起來（38 = 38）不是巧合。

## RTL 指紋（exact）

錯誤處理的三個 ASCIIZ 連在一起（檔案位移 `614Fh` 起）：

```
"Runtime error \0" " at \0" ".\r\n\0"
```

**沒有** `Divide by zero`／`Heap overflow` 這類逐項訊息——錯誤碼用數字印，
不查表。緊接在前面的是把一個 nibble 印出去的 RTL 輔助函式：

```
58        pop  ax
24 0F     and  al, 0Fh
04 30     add  al, '0'
3C 3A     cmp  al, ':'
72 02     jb   +2
04 07     add  al, 7        ; 'A'..'F'
8A D0     mov  dl, al
B4 06     mov  ah, 6
CD 21     int  21h          ; DOS 直接主控台輸出
C3        ret
```

用 DOS 功能 `06h` 直接輸出，不走檔案系統——錯誤發生時那一層可能已經壞了。

常駐 RTL 的 segment 是 **`05BBh`**（進入點第一個呼叫的對象）。遊戲碼對它的
呼叫在各處都看得到，例如結局那一段用 `05BB:0634h` 組字串、`05BB:06C1h` 接字串、
`05BB:0E3Ah` 組檔名。

## GAME.OVR：Borland overlay 容器（exact）

檔頭 `54 50 4F 56` ＝ `TPOV`。控制記錄在 `START.EXE` 裡、以 `CD 3F`（INT 3Fh）
起頭，串成一條鏈；`cmd/pool-ovr-manifest` 直接從 ZIP 重生出
**38 顆 overlay、774 個進入點**，overlay ID 固定 0..37。
stub segment 對照表見 [spec 109](../spec/109-overlay-stub-segments.md)。

## 結論與推論等級

| 對象 | 結論 | 等級 |
|---|---|---|
| 執行檔格式 | DOS MZ | 已證實 |
| Overlay 容器 | Borland `TPOV` ＋ INT 3Fh overlay manager | 已證實 |
| Compiler family | **Turbo Pascal** | 已證實（單元初始化鏈 ＋ runtime error 訊息組 ＋ TPOV）|
| 版本 | **5.x 家族** | 強推論 |
| Linker | Turbo Pascal 內建（沒有第三方 linker 字串）| 強推論 |

版本那一條的理由：**Turbo Pascal 4.0 沒有 overlay 支援**，overlay 單元是
5.0（1988）加回來的；而 Pool of Radiance 是 1988 年的作品。3.x 的 overlay 是
另一套機制、runtime 也不同。**不宣稱精確的小版本**——這幾條位元組分不出
5.0 與 5.5，而精確版本要靠 version string、release-specific bytes 或同一套工具
重現輸出才算數。

## 還沒做

- 逐一替 38 顆 overlay 命名。青色枷的 PC-98 版帶 Borland 除錯符號
  （`docs/audit/overlay-module-names.md`），兩作的編號在 `overlay-18`（結局）、
  `overlay-19`（角色表）、`overlay-20`（休息）、`overlay-22`（法術）、
  `overlay-23`（效果）、`overlay-25`（訓練）等處對得上，但**不是全域對應**：
  Pool 的 ECL 直譯器在 `overlay-03`，青色枷的 `INTERPET` 在 `overlay-02`。
  要一顆一顆用本作自己的證據認，不能整批照抄。
- 全模組函式清冊。青色枷那邊是用 IDA 對每一段種 entry point 做的（見
  `knowledge-base/retro/borland-tpov-overlay-re.md`），Pool 這邊還沒建。
