package main

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
)

// "Attack Ally:"（#118，spec 162）。玩家要打自己這一邊的人時，原版先問一句；答 Y 就把
// 隊伍那一側的 NPC 與怪物全部翻成敵方，再照樣打下去。
//
// overlay-13 entry 20（`2977h`，far call `0096h:0084h`）收（目標, 攻擊者）：
//
//	2983  010Ah:00B6h(目標) == 攻擊者 +10Eh → 回 1   ; 目標在對面，直接打
//	2995  攻擊者 +10Fh != 0 → 回 1                     ; AI 在走的不問
//	29A9  印 "Attack Ally: "（cs:2969h），011Dh:003Eh 讀一個鍵
//	29C1  不是 'Y' → 回 0
//	29CF  DS:6CD6h = 1
//	29D4  [4937h]+666h = 1                             ; ECL @6E33
//	29DF  沿 DS:5CF4h：+10Ch == 0 而且 +84h > 7Fh 的
//	2A0C    +10Eh = 1（敵方）；runtime（+108h）的 +0Ah／+0Ch（追的目標）清成 NULL
//	2A36  010Ah:00BBh                                  ; overlay-25 entry 31，兩邊人數
//
// 呼叫它的有兩處：overlay-08 `0D76h`（走進有人的那一格，`0BF1h` 叫 `0D30h`）與 overlay-13
// `2C3Fh`（瞄準列的 Target，只在 `352Ch` 的 `[bp+10h] == 1` 時——overlay-08 A）IM 的
// `03D1h` 推 1，施法的 `1E09h` 推 0，所以**施法不問**）。回 0 時前者什麼也不做、留在移動
// 裡；後者把旗標清 0、留在瞄準列（`376Dh`）。
//
// `+84h > 7Fh` 是「要判士氣」的那一群（spec 091）：怪物、ADD NPC 加進來的 NPC、競技場的
// 複製品。玩家建的角色是 0，所以打自己的隊員不會讓隊員倒戈，倒戈的是跟著的 NPC 與怪物。
// 被死靈術叫起來的狀態是 1（`21C0h`），也不在 `+10Ch == 0` 裡。

const (
	msgAttackAllyPrompt messageID = iota + 5500
)

func init() {
	for id, key := range map[messageID]string{
		msgAttackAllyPrompt: "ui.attackAllyPrompt",
	} {
		if existing, ok := messageKeys[id]; ok {
			panic(fmt.Sprintf("message id %d is already %q", id, existing))
		}
		messageKeys[id] = key
	}
}

// attackedAllyAddress 是 ECL `@6E33`：class 1 位移 `(2A00h + 6E33h × 2) mod 10000h = 666h`。
// 寫入端只有 overlay-10 `1F76h`（開打歸零）與 overlay-13 `29D8h`（答 Y 立 1）；讀的是
// ECL7 block 17 `9ABAh`（兩場 `COMBAT` 之後的 `GOSUB`，非 0 就改 @4A7C 的位元）。
const attackedAllyAddress = 0x6E33

// allyPrompt 是等著回答的那一問。aimed 為真是瞄準列的 Target，否則是走進去撞到。
type allyPrompt struct {
	target uint8
	aimed  bool
}

// needsAllyPrompt 是 `2977h` 的前兩道：目標在同一邊、而攻擊者不是 AI 在走。
func (state *tacticalState) needsAllyPrompt(mover, target uint8) bool {
	same, err := state.sameSide(mover, target)
	if err != nil || !same {
		return false
	}
	return int(mover) >= len(state.AIDriven) || !state.AIDriven[mover]
}

// askAttackAlly 印那一句並等玩家回答。
func (state *tacticalState) askAttackAlly(target uint8, aimed bool) {
	state.AllyPrompt = &allyPrompt{target: target, aimed: aimed}
	state.Status = state.say(msgAttackAllyPrompt)
}

// allyPromptInput 是 `011Dh:003Eh` 讀到的那一個鍵：Y 就打，其他任何鍵都是「不打」。
func (a *app) allyPromptInput(state *tacticalState) error {
	prompt := state.AllyPrompt
	switch {
	case a.justPressed(ebiten.KeyY):
		state.AllyPrompt, state.Status = nil, ""
		a.turnAlliesHostile(state)
		if prompt.aimed {
			return a.strikeAimedTarget(state, prompt.target)
		}
		return a.bumpAttack(state, prompt.target)
	case a.anyKeyJustPressed():
		state.AllyPrompt, state.Status = nil, ""
		if prompt.aimed {
			// `2C46h` 把旗標清 0，`376Dh` 看到 0 就留在瞄準列。
			a.castTargeting, a.castTargetingAttack = true, true
		}
	}
	return nil
}

// clearAttackedAlly 是 overlay-10 `1F76h`：開打時 @6E33 歸零。
func (a *app) clearAttackedAlly() {
	if a.eventMachine != nil {
		a.eventMachine.Memory[attackedAllyAddress] = 0
	}
}

// turnAlliesHostile 是 `29CFh..2A36h`。
//
// DS:6CD6h 不做：全部 overlay、`game.ovr` 與 `start.exe` 裡 `D6 6C` 只出現在這一條寫入
// （`29BFh`），沒有讀的一方（spec 162）。
func (a *app) turnAlliesHostile(state *tacticalState) {
	if a.eventMachine != nil {
		a.eventMachine.Memory[attackedAllyAddress] = 1
	}
	for index := 1; index < len(state.Roster) && index < len(state.Friendly); index++ {
		if index >= len(state.States) || state.States[index] != 0 {
			continue
		}
		if state.Morale.Raw[index] <= moraleHighBit {
			continue
		}
		state.Friendly[index] = false
		state.setFoeTarget(uint8(index), 0)
		// `+10Eh` 寫在記錄上。隊伍裡的 NPC 在戰後像怪物那樣結算（entry 2 的經驗值、錢與
		// 物品），接著 `1164h` 把它從隊伍摘掉（opposing_members.go，spec 167）；怪物那一份
		// 同樣在戰後的經驗值裡改算（overlay-05 entry 2 `0068h` 只跳過 `+10Eh != 1` 的）。
		if index < len(state.PartySlot) && state.PartySlot[index] >= 0 {
			if slot := state.PartySlot[index]; slot < len(a.state.Party) {
				member := &a.state.Party[slot]
				member.Side = 1
				if len(member.Record) > npcSideOffset {
					member.Record[npcSideOffset] = 1
				}
				syncTrainedLibraryCharacter(&a.state, *member)
			}
			continue
		}
		if group, ok := a.combatMonsterGroup(index, state); ok {
			a.combatMonsters[group].Record.Raw[gamepack.RecordSideOffset] = 1
		}
	}
}

// combatMonsterGroup 是第 index 格屬於 combatMonsters 的哪一條：數它前面有幾格不是隊員，
// 再照每一條的數量展開（與 stagedMonsterCopy 同一種數法）。
func (a *app) combatMonsterGroup(index int, state *tacticalState) (int, bool) {
	position := 0
	for slot := 1; slot < index && slot < len(state.PartySlot); slot++ {
		if state.PartySlot[slot] < 0 {
			position++
		}
	}
	for group, monster := range a.combatMonsters {
		if position < int(monster.Spawn.Count) {
			return group, true
		}
		position -= int(monster.Spawn.Count)
	}
	return 0, false
}
