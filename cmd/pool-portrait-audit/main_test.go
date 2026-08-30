package main

import "testing"

func TestPortraitArchiveNameIsFailClosed(t *testing.T) {
	for _, name := range []string{"HEAD1.DAX", "HEAD8.DAX", "BODY1.DAX", "BODY8.DAX"} {
		if !isPortraitArchive(name) {
			t.Fatalf("rejected %s", name)
		}
	}
	for _, name := range []string{"CHEAD.DAX", "HEAD0.DAX", "HEAD9.DAX", "BODY10.DAX", "HEAD1.PNG"} {
		if isPortraitArchive(name) {
			t.Fatalf("accepted %s", name)
		}
	}
}
