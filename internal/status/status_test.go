package status

import (
	"encoding/json"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

// wantTags is the exact JSON-tag set the old cmd/status.go StatusOutput exposed.
var wantTags = []string{
	"running", "pid", "owner", "supervisorPID", "uptime", "uptimeSeconds",
	"socksOK", "httpOK", "socksAddr", "httpAddr", "profile", "xrayVersion",
	"exitIP", "pidFile", "configPath",
}

func jsonTags(t *testing.T, v any) []string {
	t.Helper()
	rt := reflect.TypeOf(v)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	tags := make([]string, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		if tag == "" {
			t.Fatalf("field %s has no json tag", rt.Field(i).Name)
		}
		for j := 0; j < len(tag); j++ {
			if tag[j] == ',' {
				tag = tag[:j]
				break
			}
		}
		tags = append(tags, tag)
	}
	return tags
}

func TestSnapshotTagSetUnchanged(t *testing.T) {
	got := jsonTags(t, Snapshot{})
	want := append([]string(nil), wantTags...)
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Snapshot json tag set drifted\n got: %v\nwant: %v", got, want)
	}
}

func TestSnapshotMarshalsAllExpectedKeys(t *testing.T) {
	s := Snapshot{
		Running: true, PID: 111, Owner: "daemon", SupervisorPID: 222,
		Uptime: "1m", UptimeSeconds: 60, SocksOK: true, HTTPOK: true,
		SocksAddr: "127.0.0.1:1080", HTTPAddr: "127.0.0.1:8080", Profile: "p",
		XrayVersion: "v1", ExitIP: "203.0.113.7", PIDFile: "/x/pid", ConfigPath: "/x/config",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, tag := range wantTags {
		if _, ok := m[tag]; !ok {
			t.Errorf("marshaled JSON missing key %q", tag)
		}
	}
}

func TestGetReturnsPopulatedStaticFields(t *testing.T) {
	snap, err := Get()
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if snap == nil {
		t.Fatal("Get() returned nil snapshot")
	}
	if snap.PIDFile == "" {
		t.Error("PIDFile should always be populated")
	}
	if snap.ConfigPath == "" {
		t.Error("ConfigPath should always be populated")
	}
	if snap.XrayVersion == "" {
		t.Error("XrayVersion should always be populated (e.g. \"not installed\")")
	}
	_ = runtime.GOOS
}
