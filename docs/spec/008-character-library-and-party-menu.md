# Spec 008：建角完成、角色庫與 Party Creation Menu

狀態：READY（原版畫面順序與 remake 自有保存契約）；DOS `.CHA/.SPC` exporter 仍為 DRAFT
日期：2026-08-31

## 原版證據

- 原版正常 UI 的 Party Creation Menu 截圖 SHA-256：
  `22bcb0126e52be62b397a18b64f06542ddfc346aa229d1efefc1503bf4932c71`。
  畫面依序列出 `CREATE NEW CHARACTER`、`ADD CHARACTER TO PARTY`、
  `LOAD SAVED GAME`、`EXIT TO DOS`，底部提示 `CHOOSE A FUNCTION`。
- 原版 Rule Book 的「Creating A Party Of Characters」明寫最多 6 名 player characters，
  遊戲內總共可控制 8 名，另兩格保留給途中加入的 NPC：
  <https://www.mocagh.org/ssi/pool-manual.pdf>。目前 remake `party` 只保存玩家角色，
  因此 fail-closed 上限是 6；NPC slots 待事件／招募垂直鏈另行實作。
- Spec 003 的 runtime 路徑已觀察：icon 最終確認 `YES` 後產生同名 `.CHA/.SPC`，
  再回到上述選單。建角不等於自動加入隊伍。
- overlay-16 local `1E79h` 呼叫 portrait editor、`1E7Dh` 呼叫 icon editor；返回後
  `1E81h..1E96h` 組字串並呼叫外部 exporter，最後釋放 285-byte 暫存 record。
- 33 份原版 UI 角色檔全部是 285 bytes。`.SPC` 不是每角色固定 blob：本機 corpus
  有 9／18／36 bytes，且每 9 bytes 是一個鏈節點；同一 DOS session 的多個角色可
  共享相同首節點。不能把 `BASE.SPC` 複製給新角色。

## Remake 保存契約

在 DOS exporter 的角色主體全部衍生欄位，以及 `.SPC` session 鏈 key／生命週期尚未
閉合前，remake 使用作品專屬、版本化 JSON：

- 本規格實作當時 schema 固定為 `pool-remake-state/1`；現行 schema 4 與 1→2→3→4
  遷移由 Spec 018／035／037 管理，未知版本仍 fail-closed。
- `character_library` 保存已完成但尚未入隊的角色；`party` 保存已加入角色。
- 每個角色保存姓名、race／gender／class／alignment IDs、擲值結果、portrait selectors、
  combat icon selectors、size 與六部位雙色。
- 寫檔採同目錄 temporary file → rename，避免 F10 自動保存留下半份 JSON。
- 建角確認後先寫入角色庫，保存成功才回 Party Creation Menu；保存失敗留在確認頁並
  顯示錯誤，不得假裝角色已建立。
- F10 在可保存狀態先保存再離開。`LOAD SAVED GAME` 只接受現行或明訂可遷移的 schema；它不是 DOS
  `.CHA/.SPC` importer，也不得標示成原版相容存檔。

## DOS exporter 停止線

目前只能安全寫回已證實欄位；AC、THAC0、damage、class-level slots、唯一識別與三條
鏈檔尚未完整閉合。禁止以 Dwarf/Fighter 的 `BASE.CHA` 作所有種族職業模板，也禁止
建立固定 9-byte `.SPC` 冒充完成。待這些 consumer／producer 皆 READY 後，再新增明確
標示的 DOS export 功能；不阻塞 remake 自有角色庫與正常玩家路徑。
