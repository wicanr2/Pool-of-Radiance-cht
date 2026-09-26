package gamepack

// 戰鬥外施法的收表（spec 098〈營地施法：同一支 `08BCh`，表換成 `0A88h`〉，issue #100）。
//
// 營地與戰鬥走的是同一支 overlay-22 entry 5（`0C14h`）→ 處理常式 → `08BCh`，差別只在
// 收表的那一支：entry 5 的 `0D63h` 是 `call dword ptr ds:6A78h`，開打時 overlay-08 entry 1
// 把它設成 overlay-13 entry 18（`0096h:007Ah`，`20AEh`），收場時 overlay-08 `0060h` 設回
// overlay-22 entry 4（`00E2h:0034h`，`0A88h`）。`0A88h` 讀參數表 `+7`（`[di+319Bh]`）：
//
//	0ACEh  1 → 表 = [施法者]（`DS:5CF0h`）
//	0AD6h  2 → "Cast Spell on whom"（`0A75h`）挑一個隊員，沒挑 → 表空、結果 0
//	0B2Ah  4 → 從 `DS:5CF4h` 沿 `+104h` 走完整隊，全部進表
//	其餘  → 結果 0（entry 5 的 `0C3Fh` 已經先把 0 擋成 "is a combat-only spell"）
//
// 所以戰鬥外那一格的語意是「營地怎麼收目標」，與戰鬥的目標模式（`+6`）無關。

// CampTargetKind 是參數表 `+7` 在戰鬥外的收表方式。
type CampTargetKind uint8

const (
	// CampTargetCombatOnly 是 0：entry 5 `0C3Fh` 問 "Lose it?"，不放。
	CampTargetCombatOnly CampTargetKind = 0
	// CampTargetSelf 是 1：表只有施法者自己（`0ACEh`）。
	CampTargetSelf CampTargetKind = 1
	// CampTargetPick 是 2：挑一個隊員（`0AD6h`，"Cast Spell on whom"）。
	CampTargetPick CampTargetKind = 2
	// CampTargetParty 是 4：整隊（`0B2Ah`）。
	CampTargetParty CampTargetKind = 4
)

// CampTarget 是 overlay-22 entry 4（`0A88h`）的 `0AC8h` `mov al, [di+319Bh]`。
func (p SpellParameters) CampTarget() CampTargetKind {
	return CampTargetKind(p.Raw[spellParameterArea])
}
