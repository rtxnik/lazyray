package cmd

import (
	"strings"
	"testing"
)

func TestExportEncryptRejectsPositional(t *testing.T) {
	exportEncrypt = true
	exportPassphraseFile = ""
	t.Cleanup(func() { exportEncrypt = false })
	err := exportCmd.RunE(exportCmd, []string{"leftover-passphrase"})
	if err == nil || !strings.Contains(err.Error(), "no longer takes a value") {
		t.Fatalf("want migration error, got %v", err)
	}
}
