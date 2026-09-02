package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
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

// 只用按鍵在隊伍管理畫面把一個角色訓練上去（spec 097）。
func TestTrainingFromThePartyManagementScreen(t *testing.T) {
	zipPath := filepath.Join("..", "..", "Pool of Radiance (1988).zip")
	application, err := newApp(zipPath, filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Skipf("original DOS ZIP is intentionally not tracked: %v", err)
	}
	fighter := poolsave.Character{Name: "A", RaceID: "dwarf", GenderID: "male",
		ClassID: "fighter", AlignmentID: "lawful-good",
		Abilities: [6]int{18, 10, 10, 10, 10, 10}, MaxHP: 10, CurrentHP: 6,
		PortraitHead: 1, PortraitBody: 1, IconSize: 1, Experience: 2001}
	application.state = poolsave.State{Schema: poolsave.Schema,
		CharacterLibrary: []poolsave.Character{fighter}, Party: []poolsave.Character{fighter}}
	application.mode, application.programManaging = modeMenu, true
	// 1 挑第一個人，T 訓練。
	if err := press(application, ebiten.KeyDigit1); err != nil {
		t.Fatal(err)
	}
	if err := press(application, ebiten.KeyT); err != nil {
		t.Fatal(err)
	}
	trained := application.state.Party[0]
	if len(trained.ClassLevels) == 0 || trained.ClassLevels[2] != 2 {
		t.Fatalf("戰士應該升到第 2 級，職業等級是 %v；狀態列 %q",
			trained.ClassLevels, application.statusLine)
	}
	// 受傷量要保留：原本 10/6 差 4 點。
	if trained.MaxHP-trained.CurrentHP != 4 {
		t.Errorf("升級不該治好傷，變成 %d/%d", trained.MaxHP, trained.CurrentHP)
	}
	if trained.MaxHP <= 10 {
		t.Errorf("最大 HP 應該變多，還是 %d", trained.MaxHP)
	}
	// 經驗值被砍到第 3 級門檻減一。
	if trained.Experience != 2001 && trained.Experience != 4000 {
		t.Errorf("經驗值應該留著或被砍到 4000，變成 %d", trained.Experience)
	}
	// 角色庫也要同步，不然回主選單看到的是舊的。
	if application.state.CharacterLibrary[0].MaxHP != trained.MaxHP {
		t.Errorf("角色庫沒同步：%d vs %d",
			application.state.CharacterLibrary[0].MaxHP, trained.MaxHP)
	}
	// 再按一次不該再升：經驗值已經不夠下一級了。
	if err := press(application, ebiten.KeyT); err != nil {
		t.Fatal(err)
	}
	if application.state.Party[0].ClassLevels[2] != 2 {
		t.Errorf("經驗值不夠卻又升了一級，到了第 %d 級",
			application.state.Party[0].ClassLevels[2])
	}
}

// 升級要真的傳到戰鬥數值上。THAC0 是「內部值」，越大越好（60 減去實際的
// THAC0），所以高等級的戰士內部值要比第 1 級大。
func TestTrainedLevelsReachCombatStats(t *testing.T) {
	first := poolsave.Character{Name: "A", ClassID: "fighter",
		Abilities: [6]int{18, 10, 10, 10, 10, 10}}
	trained := first
	levels := memberClassLevels(first)
	levels[2] = 6
	trained.ClassLevels = levels[:]
	firstThac0, _, _, err := partyCombatStats(first)
	if err != nil {
		t.Fatal(err)
	}
	trainedThac0, _, _, err := partyCombatStats(trained)
	if err != nil {
		t.Fatal(err)
	}
	if trainedThac0 <= firstThac0 {
		t.Fatalf("第 6 級戰士的 THAC0 內部值 %d 沒有比第 1 級的 %d 好——"+
			"等級沒有傳到戰鬥數值", trainedThac0, firstThac0)
	}
}
