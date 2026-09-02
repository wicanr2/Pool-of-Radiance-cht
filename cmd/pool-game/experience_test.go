package main

import (
	"path/filepath"
	"testing"

	"github.com/wicanr2/Pool-of-Radiance-cht/internal/gamepack"
	poolsave "github.com/wicanr2/Pool-of-Radiance-cht/internal/save"
	"github.com/wicanr2/golden-box-remake-engine/eclvm"
)

// 打贏一場之後隊伍要真的拿到經驗值，而且數字對得上原版的算法：
// 總額 ÷ 分的人數，再各自套職業規則。
func TestVictoryAwardsExperience(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	orc, err := gamepack.ReadDOSMonsterRecord(zipPath, 2, 4)
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	// ORC：基礎 10 加每點生命值 1，樣板 5 點 → 一隻 15。四隻共 60。
	if value := orc.ExperienceValue(int(orc.MaxHitPoints())); value != 15 {
		t.Fatalf("ORC 一隻應該值 15，算出 %d", value)
	}
	application := &app{combatMonsters: []stagedMonster{
		{Spawn: eclvm.MonsterSpawn{Count: 4}, Record: orc},
	}}
	// 力量 16 的純戰士多拿十分之一；力量 15 的剛好不夠。
	strong := poolsave.Character{Name: "A", ClassID: "fighter", Abilities: [6]int{16, 10, 10, 10, 10, 10}}
	plain := poolsave.Character{Name: "B", ClassID: "fighter", Abilities: [6]int{15, 10, 10, 10, 10, 10}}
	dual := poolsave.Character{Name: "C", ClassID: "fighter-thief", Abilities: [6]int{18, 10, 10, 18, 10, 10}}
	fourth := poolsave.Character{Name: "D", ClassID: "cleric", Abilities: [6]int{10, 10, 10, 10, 10, 10}}
	application.state = poolsave.State{Party: []poolsave.Character{strong, plain, dual, fourth}}
	application.awardCombatExperience()
	// 60 ÷ 4 個人 ＝ 每人 15。
	for index, want := range []uint32{16, 15, 7, 15} {
		if got := application.state.Party[index].Experience; got != want {
			t.Errorf("%s 應該拿 %d，拿到 %d",
				application.state.Party[index].Name, want, got)
		}
	}
	// 再打一場要累加，不是覆蓋。
	application.awardCombatExperience()
	if got := application.state.Party[0].Experience; got != 32 {
		t.Errorf("第二場之後應該累到 32，拿到 %d", got)
	}
}

// 沒有敵人就不發，也不該把隊伍的經驗值清掉。
func TestNoMonstersAwardsNothing(t *testing.T) {
	application := &app{state: poolsave.State{Party: []poolsave.Character{
		{Name: "A", ClassID: "fighter", Experience: 500},
	}}}
	application.awardCombatExperience()
	if got := application.state.Party[0].Experience; got != 500 {
		t.Errorf("沒有敵人不該動到經驗值，變成 %d", got)
	}
}
