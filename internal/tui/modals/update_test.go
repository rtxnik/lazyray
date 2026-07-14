package modals

import (
	"errors"
	"testing"

	"github.com/rtxnik/lazyray/internal/config"
	"github.com/rtxnik/lazyray/internal/core"
)

type fakeApplier struct {
	called bool
	ret    error
}

func (f *fakeApplier) ApplyXrayUpdate(_ *core.XrayProcess, _ *core.ReleaseInfo, _ string,
	_ *config.Settings, _, _ bool) error {
	f.called = true
	return f.ret
}

func TestUpdateModal_ApplyRoutesThroughService(t *testing.T) {
	fa := &fakeApplier{ret: errors.New("refused")}
	m := NewUpdateModal(core.NewXrayProcess(), config.DefaultSettings(), 80, 24, fa)
	// applyUpdate resolves FindAssetURL BEFORE the service call, so the release
	// needs a matching platform asset or the flow short-circuits before routing.
	m.release = &core.ReleaseInfo{
		TagName: "v1.2.3",
		Assets:  []core.Asset{{Name: core.AssetName(), BrowserDownloadURL: "https://example/test"}},
	}

	msg := m.applyUpdate()() // execute the tea.Cmd
	if !fa.called {
		t.Error("modal did not route apply through the service")
	}
	if am, ok := msg.(updateApplyMsg); !ok || am.err == nil {
		t.Errorf("expected updateApplyMsg with error, got %#v", msg)
	}
}
