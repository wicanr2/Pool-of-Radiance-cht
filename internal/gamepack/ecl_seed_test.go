package gamepack

import "testing"

// seedFixtureBlock 只有五個命令集 header 與 EXIT；它讓 seed API 的測試
// 不依賴未入版控的原版 ZIP。
func seedFixtureBlock() []byte {
	payload := make([]byte, 21)
	entry := uint16(0x9914)
	for index := 0; index < 5; index++ {
		offset := index * 4
		payload[offset+1] = 2
		payload[offset+2] = byte(entry)
		payload[offset+3] = byte(entry >> 8)
	}
	return append([]byte{0, 0}, payload...)
}

func TestECLSessionConstructorsKeepFixedDefaultAndAcceptExplicitSeed(t *testing.T) {
	block := seedFixtureBlock()
	event := InitialEvent{HandlerAddress: 0x9914, ScriptBlock: block}
	fixed, err := NewInitialEventSession(event)
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := NewInitialEventSessionWithSeed(event, 8675309)
	if err != nil {
		t.Fatal(err)
	}
	fixedSnapshot, err := fixed.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	explicitSnapshot, err := explicit.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if fixedSnapshot.Machine.Random.Seed != deterministicECLSeed {
		t.Fatalf("default seed=%d, want %d", fixedSnapshot.Machine.Random.Seed, deterministicECLSeed)
	}
	if explicitSnapshot.Machine.Random.Seed != 8675309 {
		t.Fatalf("explicit seed=%d, want 8675309", explicitSnapshot.Machine.Random.Seed)
	}

	archive := ECLArchive{Number: 1, Blocks: map[uint16][]byte{0: block}}
	archiveSession, err := NewDOSECLArchiveSessionWithSeed(archive, 0, 0x9914, 42)
	if err != nil {
		t.Fatal(err)
	}
	archiveSnapshot, err := archiveSession.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if archiveSnapshot.Machine.Random.Seed != 42 {
		t.Fatalf("archive seed=%d, want 42", archiveSnapshot.Machine.Random.Seed)
	}
}
