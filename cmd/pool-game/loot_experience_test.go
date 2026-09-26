package main

// 戰利品折算經驗值與 NPC 分錢（spec 148，issue #94）。戰鬥與交件都從 `Update()` 送鍵。

import (
	"strings"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
)

// 打完兩隻狗頭人首領與法師（一隻逃掉）：經驗總額是怪物那一項，加上公款（怪物身上的錢）
// 與戰利品串列的加值折算，一起除以四個人（overlay-05 entry 2 `0000h..0306h`）。
func TestMonsterLootFoldsIntoExperience(t *testing.T) {
	application, kobold, wizard := finishWithLoot(t, false)
	if !application.treasureActive {
		t.Fatal("the loot menu did not open")
	}
	monsters := kobold.ExperienceValue(int(kobold.MaxHitPoints())) + // 兩隻首領，逃掉一隻
		wizard.ExperienceValue(int(wizard.MaxHitPoints()))
	plus := make([]int8, 0, len(application.treasureItems))
	for _, item := range application.treasureItems {
		plus = append(plus, int8(item.Raw[gamepack.LootItemPlusOffset]))
	}
	loot := gamepack.LootExperience(application.state.PooledMoney, plus)
	if loot == 0 {
		t.Fatal("the fight left nothing to fold; the test would not tell anything")
	}
	want := (monsters + loot) / 4
	for index, member := range application.state.Party {
		if member.Experience != want {
			t.Fatalf("member %d has %d XP, want (%d monsters + %d loot) / 4 = %d",
				index, member.Experience, monsters, loot, want)
		}
	}
	t.Logf("monsters %d, loot %d (pool %v, pluses %v), each %d", monsters, loot,
		application.state.PooledMoney, plus, want)
}

// receiptParty 是 docs/audit/dosgolem-loot-experience.json 那一隊：五個戰士，力量
// 17、16、18、14、15（超過 15 的多拿十分之一，spec 097）。
func receiptParty() []poolsave.Character {
	strengths := []int{17, 16, 18, 14, 15}
	party := make([]poolsave.Character, 0, len(strengths))
	for index, strength := range strengths {
		party = append(party, poolsave.Character{
			Name: string(rune('B' + index)), RaceID: "dwarf", GenderID: "male", ClassID: "fighter",
			AlignmentID: "lawful-good", Abilities: [6]int{strength, 12, 12, 13, 15, 12},
			MaxHP: 40, CurrentHP: 40, PortraitHead: 1, PortraitBody: 1, IconSize: 1,
		})
	}
	return party
}

// 市政廳交貧民窟的件：職員的 `TREASURE → COMBAT` 走同一個 entry 2，公款 250 金、
// 50 白金、1 珠寶折成 2700，五個人各 540，力量超過 15 的 594——與原版收據逐人相同。
func TestCityHallRewardFoldsIntoExperienceLikeTheOriginal(t *testing.T) {
	application, session, _ := slumsCommissionApp(t, 25)
	party := receiptParty()
	application.state.Party, application.state.CharacterLibrary = party, party
	handInAtCityHall(t, application, session)
	if !application.treasureActive {
		t.Fatal("the reward service did not open")
	}
	if want := ([7]uint32{3: 250, 4: 50, 6: 1}); application.state.PooledMoney != want {
		t.Fatalf("reward pool %v, want %v", application.state.PooledMoney, want)
	}
	want := []uint32{594, 594, 594, 540, 540}
	for index, member := range application.state.Party {
		if member.Experience != want[index] {
			t.Fatalf("%s gained %d XP, want %d (dosgolem receipt)", member.Name, member.Experience, want[index])
		}
	}
}

// 隊伍帶著份額 3 的傭兵（SWORDSMAN，MON3/36 `+85h = 03h`）交件：經驗值照六個人分，
// 之後 `1295h` 讓傭兵從公款拿走三份——總份數 5 ＋ 3 ＝ 8，金 250/8 = 31 扣 93、
// 白金 50/8 = 6 扣 18、珠寶 1/8 = 0 不扣。
func TestHiredSwordsmanHidesHisShareOfTheReward(t *testing.T) {
	application, session, _ := slumsCommissionApp(t, 25)
	party := receiptParty()
	application.state.CharacterLibrary = party
	record, err := application.loadMonster(3, 36)
	if err != nil {
		t.Fatal(err)
	}
	npc := poolsave.Character{Name: strings.TrimSpace(record.Name), NPC: true,
		Record: append([]byte(nil), record.Raw[:]...), MaxHP: 10, CurrentHP: 10}
	npc.Record[gamepack.MoraleOffset] = gamepack.NPCMoraleByte(50)
	if npc.Record[gamepack.MoraleOffset+1]&7 != 3 {
		t.Fatalf("SWORDSMAN +85h = %02X, want a share of 3", npc.Record[gamepack.MoraleOffset+1])
	}
	application.state.Party = append(party, npc)
	handInAtCityHall(t, application, session)
	if !application.treasureActive {
		t.Fatal("the reward service did not open")
	}
	if want := ([7]uint32{3: 250 - 93, 4: 50 - 18, 6: 1}); application.state.PooledMoney != want {
		t.Fatalf("pool after the swordsman's share %v, want %v", application.state.PooledMoney, want)
	}
	if !strings.HasSuffix(application.statusLine, "takes and hides his share.") {
		t.Fatalf("status %q does not say who took a share", application.statusLine)
	}
	// 經驗值在分錢之前發，照整筆 2700 除六個人：450，力量超過 15 的 495。
	for index, want := range []uint32{495, 495, 495, 450, 450, 450} {
		if got := application.state.Party[index].Experience; got != want {
			t.Fatalf("member %d gained %d XP, want %d", index, got, want)
		}
	}
}
