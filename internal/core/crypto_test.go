package core

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
)

func TestExportImportEncrypted_Roundtrip(t *testing.T) {
	profiles := []config.Profile{
		{
			Name: "Test Server 1",
			Server: config.ServerConfig{
				Address:    "1.2.3.4",
				Port:       443,
				UUID:       "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				Encryption: "none",
				Protocol:   "vless",
				Transport:  config.TransportConfig{Network: "xhttp", Path: "/test"},
				Security:   config.SecurityConfig{Type: "reality", SNI: "example.com"},
			},
		},
		{
			Name: "Test Server 2",
			Server: config.ServerConfig{
				Address:   "5.6.7.8",
				Port:      8443,
				UUID:      "testpass",
				Protocol:  "trojan",
				Transport: config.TransportConfig{Network: "ws"},
				Security:  config.SecurityConfig{Type: "tls", SNI: "test.com"},
			},
		},
	}

	password := "mysecretpassword123"

	encrypted, err := ExportEncrypted(profiles, password)
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}

	if !IsEncryptedExport(encrypted) {
		t.Error("IsEncryptedExport should return true")
	}

	decrypted, err := ImportEncrypted(encrypted, password)
	if err != nil {
		t.Fatalf("ImportEncrypted() error = %v", err)
	}

	if len(decrypted) != 2 {
		t.Fatalf("got %d profiles, want 2", len(decrypted))
	}

	if decrypted[0].Name != "Test Server 1" {
		t.Errorf("profiles[0].Name = %q", decrypted[0].Name)
	}
	if decrypted[0].Server.Address != "1.2.3.4" {
		t.Errorf("profiles[0].Address = %q", decrypted[0].Server.Address)
	}
	if decrypted[1].Server.Protocol != "trojan" {
		t.Errorf("profiles[1].Protocol = %q", decrypted[1].Server.Protocol)
	}
}

func TestImportEncrypted_WrongPassword(t *testing.T) {
	profiles := []config.Profile{
		{
			Name:   "Test",
			Server: config.ServerConfig{Address: "1.2.3.4", Port: 443, UUID: "uuid"},
		},
	}

	encrypted, err := ExportEncrypted(profiles, "correctpass")
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}

	_, err = ImportEncrypted(encrypted, "wrongpass")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestExportEncrypted_EmptyProfiles(t *testing.T) {
	_, err := ExportEncrypted(nil, "pass")
	if err == nil {
		t.Fatal("expected error for empty profiles")
	}
}

func TestExportEncrypted_EmptyPassword(t *testing.T) {
	profiles := []config.Profile{{Name: "Test"}}
	_, err := ExportEncrypted(profiles, "")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestIsEncryptedExport(t *testing.T) {
	if IsEncryptedExport("not encrypted") {
		t.Error("should return false for non-encrypted")
	}
	if !IsEncryptedExport("LZRENC1:somedata") {
		t.Error("should return true for encrypted prefix")
	}
}

// --- LZRENC2 container tests ---

// lzrenc1Fixture was produced by the pre-v2 export code (PBKDF2-SHA256,
// 100k iterations) for []config.Profile{{Name: "legacy-fixture", Server:
// config.ServerConfig{Protocol: "vless", Address: "fixture.example.org",
// Port: 443, UUID: "00000000-1111-2222-3333-444444444444"}}} under the
// passphrase "fixture-passphrase". It guards decrypt compatibility forever.
const lzrenc1Fixture = "LZRENC1:eyJzYWx0IjoibTdFT1RjQ25kbDF6dWZwUGt3cHY2UT09Iiwibm9uY2UiOiJwZE5VNmIreHE2dEt5UDhuIiwiY2lwaGVydGV4dCI6ImRMcGJGWjB4VUR3aGRKcjlCU1RNc1B3VVU3RXhDci9DcXhNd0FuK1pjYTlLYlhMcWloV0t0eStJNzVjOEFGVU5lREdCeTFwMXNvWTJiVnpEOENtcWRLS1FIY1psNHBxaDk3OWE4YlhvTm05NGFES1hPcTBWcmIwL0RpdFZ3UXd4WE9FbnNsUDU5YS85L3lTblVIMDl1NTA1OHFKWlM5RitMMHV6WmdlVjVqOGxBTnJMUFFid2kwdUdxRGkzRURUZWJQRnN2WGRVVEhEQVRRa1FZT3JHMVorR1VwcDBVUE1RY3VvcGhka2J4L3RVdmhFUUE0U0diOFhmZkkzdUlFMnhORUUrUG5GamsvcGc1M2ZkQlloeElhRy9LN3pwYXVETGNkcVkyYlhXS1NDU2hiMmZzcEIxZGxwOXdLUllTRmtlVW5EUzVxREQ3bDdWNktYYXN2WHJBMlFtVVN0Y0doREVIcFM5bWFLMDZrNXZsRys2dTYvaEczbmNudG5MM090Y3g1RmR4N2QxaFRZVWNwc2kyTDd4NmV4TGpnZExCWEdib3dPZXZHUEl2TWNNcGZ6a05DOUN6TUVKNi9ESlY0YjFVUndxTzJYNlZxdElabS84Rm1ZWS95VW9MYmY0RmV0ZGRia0JMY1hRRU5FbGU3eXJpa3FnVVgxSE8rUmM4RjRXMGp5YWRydHIrb2tSOXJMOG1icXRyWWtJcmV0SVpzK0dRMXl0MVlwOGprdjcvakhpVGxTOGEvSk44YU5udTVUeGN3OW4xSnlnWGpqWlV5Sjd3TXRxZXJQS2hvbmpVeDYrRnA0TXMraUI2dThFMFVadlA5alBTOC9tY1MvQWFCRmdBRzdxM1RpMERrNnE5ZERPcU81NDBMOUh0eEVZUHBUMHorN2xOS0FleWpjdFNxUGt3RUdxY3VkNHVxdHVoZk1FZy9kWnptZWpFRThrMytEN2NISDlKdlkzbHczbG11VDU1OW93b0VpQlI3cVJQVmtRdW5yWC9GUnJXcURMVDNVRW5NWGE4VXRvZjV4S0NDWHd1a3BjMkVPZ045OWVvcElITzJqVVFDRDJ2T3VJTkdFMWUrZkJOWUhOZms1OE5IdXNPdUJyR2ZBVXZzdk84WXd5Ykl0YzU5UHZadEdpTm5MNGo4YWlkRGQyQ29rcmxkNGcxOXR4ZzNkL0hIM0JQWGg0M2ZzMFFoSGlVV2I3MGNtL2lQOEVHLzNkcEZZbFQ4NVcvY21aaFpwaVp0U0pzZ1R0NWNncUI0Qm1WS2N0ZEsvcXVwRnJDbmVaODFmQ1ZHczVLVWpFU3FNZFJBPT0ifQ=="

func TestEncryptData_EmitsLZRENC2(t *testing.T) {
	blob, err := EncryptData([]byte("payload"), "pw")
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}
	if len(blob) < 8 || blob[:8] != "LZRENC2:" {
		t.Errorf("blob prefix = %q, want LZRENC2:", blob[:min(len(blob), 8)])
	}
}

func TestEncryptDecryptData_Roundtrip(t *testing.T) {
	payload := []byte("arbitrary bytes \x00\x01\xff not just JSON")
	blob, err := EncryptData(payload, "round-trip-pw")
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}
	got, err := DecryptData(blob, "round-trip-pw")
	if err != nil {
		t.Fatalf("DecryptData() error = %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("roundtrip payload mismatch: got %q", got)
	}
}

func TestDecryptData_WrongPassword(t *testing.T) {
	blob, err := EncryptData([]byte("data"), "right")
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}
	if _, err := DecryptData(blob, "wrong"); err == nil {
		t.Fatal("DecryptData with wrong password should fail")
	}
}

func TestDecryptData_LZRENC1Fixture(t *testing.T) {
	got, err := DecryptData(lzrenc1Fixture, "fixture-passphrase")
	if err != nil {
		t.Fatalf("DecryptData(LZRENC1 fixture) error = %v", err)
	}
	var profiles []config.Profile
	if err := json.Unmarshal(got, &profiles); err != nil {
		t.Fatalf("fixture plaintext is not a profile list: %v", err)
	}
	if len(profiles) != 1 || profiles[0].Name != "legacy-fixture" {
		t.Errorf("fixture profiles = %+v, want single legacy-fixture", profiles)
	}
}

func TestImportEncrypted_LZRENC1Fixture(t *testing.T) {
	profiles, err := ImportEncrypted(lzrenc1Fixture, "fixture-passphrase")
	if err != nil {
		t.Fatalf("ImportEncrypted(LZRENC1 fixture) error = %v", err)
	}
	if len(profiles) != 1 || profiles[0].Server.Address != "fixture.example.org" {
		t.Errorf("fixture import = %+v", profiles)
	}
}

func TestExportEncrypted_EmitsLZRENC2(t *testing.T) {
	blob, err := ExportEncrypted([]config.Profile{{Name: "p"}}, "pw")
	if err != nil {
		t.Fatalf("ExportEncrypted() error = %v", err)
	}
	if blob[:8] != "LZRENC2:" {
		t.Errorf("export prefix = %q, want LZRENC2:", blob[:8])
	}
}

func TestDecryptData_TamperedCiphertext(t *testing.T) {
	blob, err := EncryptData([]byte("data"), "pw")
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}
	// Flip one character deep inside the base64 body.
	b := []byte(blob)
	i := len(b) - 10
	if b[i] == 'A' {
		b[i] = 'B'
	} else {
		b[i] = 'A'
	}
	if _, err := DecryptData(string(b), "pw"); err == nil {
		t.Fatal("tampered blob should fail to decrypt")
	}
}

// forgeEnvelope builds an LZRENC2 blob with attacker-chosen KDF params and a
// well-formed salt/nonce/ciphertext so only parameter validation can reject it.
func forgeEnvelope(t *testing.T, kdf string, time, memoryKiB uint32, threads uint8, saltLen int) string {
	t.Helper()
	env := map[string]interface{}{
		"kdf": kdf, "time": time, "memory_kib": memoryKiB, "threads": threads,
		"salt":       base64.StdEncoding.EncodeToString(make([]byte, saltLen)),
		"nonce":      base64.StdEncoding.EncodeToString(make([]byte, 12)),
		"ciphertext": base64.StdEncoding.EncodeToString(make([]byte, 32)),
	}
	j, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return "LZRENC2:" + base64.StdEncoding.EncodeToString(j)
}

func TestDecryptData_EnvelopeParamCaps(t *testing.T) {
	cases := []struct {
		name string
		blob string
	}{
		{"memory over cap", forgeEnvelope(t, "argon2id", 3, 128*1024+1, 4, 16)},
		{"time over cap", forgeEnvelope(t, "argon2id", 9, 64*1024, 4, 16)},
		{"threads over cap", forgeEnvelope(t, "argon2id", 3, 64*1024, 9, 16)},
		{"zero time", forgeEnvelope(t, "argon2id", 0, 64*1024, 4, 16)},
		{"zero memory", forgeEnvelope(t, "argon2id", 3, 0, 4, 16)},
		{"zero threads", forgeEnvelope(t, "argon2id", 3, 64*1024, 0, 16)},
		{"salt too short", forgeEnvelope(t, "argon2id", 3, 64*1024, 4, 7)},
		{"salt too long", forgeEnvelope(t, "argon2id", 3, 64*1024, 4, 65)},
		{"unsupported kdf", forgeEnvelope(t, "scrypt", 3, 64*1024, 4, 16)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptData(tc.blob, "pw")
			if err == nil {
				t.Fatal("over-cap envelope must be rejected")
			}
			// Validation must fail BEFORE any KDF work, with an error
			// distinct from an authentication failure.
			if strings.Contains(err.Error(), "decryption failed") {
				t.Errorf("got decryption-stage error %q, want pre-KDF validation error", err)
			}
			if !strings.Contains(err.Error(), "invalid encrypted data") {
				t.Errorf("error %q should identify invalid envelope", err)
			}
		})
	}
}

func TestIsEncryptedExport_BothPrefixes(t *testing.T) {
	if !IsEncryptedExport("LZRENC1:x") {
		t.Error("LZRENC1 must be recognized")
	}
	if !IsEncryptedExport("LZRENC2:x") {
		t.Error("LZRENC2 must be recognized")
	}
	if IsEncryptedExport("LZRENC3:x") || IsEncryptedExport("plain") {
		t.Error("unknown formats must not be recognized")
	}
}
