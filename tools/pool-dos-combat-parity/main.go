// Command pool-dos-combat-parity 由使用者提供的 DOS Pool of Radiance
// 檔案重生一份戰鬥數值收據。輸出只含型別化輸入與結果，不保存原版記憶體或美術。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/wicanr2/dosgolem/oracle"
)

const (
	wantEXESHA256  = "12811cbc8166a9e753283e972a7396db566e37e81ff1b272e34833d99b810d9f"
	randomIDA      = 0x16844
	applyDamageIDA = 0x1114c
)

var diceSignature = []byte{
	0x55, 0x89, 0xe5, 0x83, 0xec, 0x04, 0xc6, 0x46,
	0xfd, 0x00, 0x80, 0x7e, 0x08, 0x00, 0x76, 0x2c,
}

type receipt struct {
	Schema                 string `json:"schema"`
	Generator              string `json:"generator"`
	GeneratorRevision      string `json:"generator_revision"`
	OriginalEXESHA256      string `json:"original_exe_sha256"`
	Path                   string `json:"path"`
	ControlledRandomMethod string `json:"controlled_random_method"`
	RemakeRandomMethod     string `json:"remake_random_method"`
	PreRolls               int    `json:"pre_rolls"`
	Scope                  string `json:"scope"`
	Attacker               string `json:"attacker"`
	Victim                 string `json:"victim"`
	AttackerTHAC0Internal  int    `json:"attacker_thac0_internal"`
	AttackerTHAC0Display   int    `json:"attacker_thac0_display"`
	EffectiveACInternal    int    `json:"effective_ac_internal"`
	EffectiveACDisplay     int    `json:"effective_ac_display"`
	D20                    int    `json:"d20"`
	Hit                    bool   `json:"hit"`
	DamageDice             string `json:"damage_dice"`
	DamageRoll             int    `json:"damage_roll"`
	Damage                 int    `json:"damage"`
	AttackerHPBefore       int    `json:"attacker_hp_before"`
	AttackerHPAfter        int    `json:"attacker_hp_after"`
	VictimHPBefore         int    `json:"victim_hp_before"`
	VictimHPAfter          int    `json:"victim_hp_after"`
}

type callFrame struct {
	args []uint16
	data any
}

func linearAddr(linear uint32) oracle.Addr {
	return oracle.Far(uint16(linear>>4), uint16(linear&0xf))
}

func hookFar(o *oracle.Oracle, entry oracle.Addr, argc int,
	enter func(*oracle.Oracle, []uint16) any,
	leave func(*oracle.Oracle, callFrame, uint16)) {
	queues := map[uint32][]callFrame{}
	registered := map[uint32]bool{}
	o.OnCall(entry, func(o *oracle.Oracle) {
		ret := o.Caller()
		args := make([]uint16, argc)
		for index := range args {
			args[index] = o.Arg(index)
		}
		frame := callFrame{args: args}
		if enter != nil {
			frame.data = enter(o, args)
		}
		queues[ret.Linear()] = append(queues[ret.Linear()], frame)
		if registered[ret.Linear()] {
			return
		}
		registered[ret.Linear()] = true
		o.OnCall(ret, func(o *oracle.Oracle) {
			queue := queues[ret.Linear()]
			if len(queue) == 0 {
				return
			}
			frame := queue[0]
			queues[ret.Linear()] = queue[1:]
			leave(o, frame, o.AX())
		})
	})
}

func settleEGA(o *oracle.Oracle, label string) error {
	last := append([]byte(nil), o.IndexedEGA()...)
	lastChange := o.Steps()
	deadline := o.Steps() + 150_000_000
	for o.Steps() < deadline {
		if err := o.Run(100_000); err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		frame := o.IndexedEGA()
		if !bytes.Equal(last, frame) {
			last = append(last[:0], frame...)
			lastChange = o.Steps()
			continue
		}
		if o.Steps()-lastChange >= 3_000_000 {
			return nil
		}
	}
	// 正式截圖產生器會記錄此情況後繼續；建角動畫已知會碰到這個邊界。
	fmt.Fprintf(os.Stderr, "警告：%s 的 EGA 畫面未在預算內靜止，依基準產生器契約續跑\n", label)
	return nil
}

func send(o *oracle.Oracle, key string) error {
	if len(key) == 1 {
		if err := o.TypeKeys(key); err != nil {
			return err
		}
	} else if err := o.SendKeys(key); err != nil {
		return err
	}
	return settleEGA(o, key)
}

func normalPath(o *oracle.Oracle) error {
	if err := settleEGA(o, "boot"); err != nil {
		return err
	}
	var keys []string
	repeat := func(count int, key string) {
		for range count {
			keys = append(keys, key)
		}
	}
	repeat(9, "Space")
	keys = append(keys, "Return", "Return", "c")
	repeat(6, "Return")
	keys = append(keys, "y", "H", "E", "R", "O", "Return", "k", "e", "y", "a", "a", "e", "b")
	repeat(14, "Return")
	keys = append(keys, "Up", "Up", "Up")
	repeat(10, "Return")
	keys = append(keys, "Up", "Up", "Up", "c")
	for index, key := range keys {
		if err := send(o, key); err != nil {
			return fmt.Errorf("normal path key %d (%s): %w", index+1, key, err)
		}
	}
	return nil
}

func byteAt(o *oracle.Oracle, record oracle.Addr, offset uint32) uint8 {
	return o.Bytes(linearAddr(record.Linear()+offset), 1)[0]
}

func run(exe, root, scratch, revision string) (receipt, error) {
	raw, err := os.ReadFile(exe)
	if err != nil {
		return receipt{}, err
	}
	sum := sha256.Sum256(raw)
	gotSHA := hex.EncodeToString(sum[:])
	if gotSHA != wantEXESHA256 {
		return receipt{}, fmt.Errorf("START.EXE SHA-256 %s, want %s", gotSHA, wantEXESHA256)
	}
	o, err := oracle.Load(exe, root)
	if err != nil {
		return receipt{}, err
	}
	defer o.Close()
	o.SetScratch(scratch)
	if err := normalPath(o); err != nil {
		return receipt{}, err
	}

	matches := o.Search(diceSignature)
	if len(matches) != 1 {
		return receipt{}, fmt.Errorf("dice signature has %d matches, want 1", len(matches))
	}
	diceEntry := linearAddr(matches[0])
	hitEntry := linearAddr(matches[0] - 0x130)
	if !bytes.Equal(o.Bytes(hitEntry, 16), []byte{
		0x55, 0x89, 0xe5, 0x83, 0xec, 0x02, 0xc6, 0x46,
		0xff, 0x00, 0xff, 0x76, 0x0e, 0xff, 0x76, 0x0c,
	}) {
		return receipt{}, fmt.Errorf("hit routine signature mismatch at %s", hitEntry)
	}

	result := receipt{
		Schema:                 "pool-dos-combat-parity/1",
		Generator:              "dosgolem",
		GeneratorRevision:      revision,
		OriginalEXESHA256:      gotSHA,
		Path:                   "normal creation -> Rolf tour -> three steps -> COMBAT -> QUICK",
		ControlledRandomMethod: "restore battle snapshot, then advance Turbo Pascal Random(word) eight times with n=20",
		RemakeRandomMethod:     "fixed roll stream [20, 2] consumed as d20 then 1d2",
		PreRolls:               8,
		Scope:                  "first resolved attack of the first tactical turn",
		Attacker:               "HERO",
		Victim:                 "GOBLIN",
		DamageDice:             "1d2",
	}
	var hitDone, applyDone bool
	var applyErr error
	var attacker, victim oracle.Addr

	hookFar(o, diceEntry, 2, nil, func(_ *oracle.Oracle, frame callFrame, value uint16) {
		if len(frame.args) != 2 {
			return
		}
		if frame.args[0] == 20 && frame.args[1] == 1 && !hitDone {
			result.D20 = int(value)
		}
		if hitDone && frame.args[0] == 2 && frame.args[1] == 1 && result.DamageRoll == 0 {
			result.DamageRoll = int(value)
		}
	})
	hookFar(o, hitEntry, 5, func(o *oracle.Oracle, args []uint16) any {
		victim = oracle.Far(args[2], args[1])
		attacker = oracle.Far(args[4], args[3])
		result.EffectiveACInternal = int(uint8(args[0]))
		result.AttackerTHAC0Internal = int(byteAt(o, attacker, 0x110))
		result.AttackerTHAC0Display = 60 - result.AttackerTHAC0Internal
		result.EffectiveACDisplay = 60 - result.EffectiveACInternal
		result.AttackerHPBefore = int(byteAt(o, attacker, 0x11b))
		result.VictimHPBefore = int(byteAt(o, victim, 0x11b))
		return nil
	}, func(_ *oracle.Oracle, _ callFrame, value uint16) {
		result.Hit = uint8(value) != 0
		hitDone = true
	})
	hookFar(o, o.IDA(applyDamageIDA), 3, func(o *oracle.Oracle, args []uint16) any {
		appliedVictim := oracle.Far(args[2], args[1])
		if appliedVictim.Linear() != victim.Linear() {
			applyErr = fmt.Errorf("damage victim %s, want hit victim %s", appliedVictim, victim)
			return applyErr
		}
		result.Damage = int(uint8(args[0]))
		return nil
	}, func(o *oracle.Oracle, frame callFrame, _ uint16) {
		if callErr, ok := frame.data.(error); ok && callErr != nil {
			return
		}
		result.AttackerHPAfter = int(byteAt(o, attacker, 0x11b))
		result.VictimHPAfter = int(byteAt(o, victim, 0x11b))
		applyDone = true
	})

	state := o.Save()
	o.Restore(state)
	for range result.PreRolls {
		if _, err := o.Call(o.IDA(randomIDA), 20); err != nil {
			return receipt{}, err
		}
	}
	if err := o.TypeKeys("q"); err != nil {
		return receipt{}, err
	}
	if err := o.RunUntil(oracle.NewCond("first HP apply", func(*oracle.Oracle) bool {
		return applyDone || applyErr != nil
	}), oracle.Budget(20_000_000)); err != nil {
		return receipt{}, err
	}
	if applyErr != nil {
		return receipt{}, applyErr
	}

	if result.AttackerTHAC0Internal != 40 || result.EffectiveACInternal != 54 ||
		result.D20 != 20 || !result.Hit || result.DamageRoll != 2 || result.Damage != 2 ||
		result.AttackerHPBefore != 6 || result.AttackerHPAfter != 6 ||
		result.VictimHPBefore != 4 || result.VictimHPAfter != 2 {
		return receipt{}, fmt.Errorf("unexpected combat receipt: %+v", result)
	}
	return result, nil
}

func main() {
	exe := flag.String("exe", "/orig/start.exe", "original START.EXE")
	root := flag.String("root", "/orig", "read-only original data directory")
	scratch := flag.String("scratch", "/scratch", "writable DOS scratch directory")
	out := flag.String("out", "", "receipt path; stdout when empty")
	revision := flag.String("revision", "", "dosgolem Git revision recorded by the wrapper")
	flag.Parse()
	r, err := run(*exe, *root, *scratch, *revision)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	raw = append(raw, '\n')
	if *out == "" {
		_, err = os.Stdout.Write(raw)
	} else {
		err = os.WriteFile(*out, raw, 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
