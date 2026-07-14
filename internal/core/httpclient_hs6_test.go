package core

import (
	"net/http"
	"testing"
	"time"
)

func TestDirectClientIgnoresEnvProxy(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:9")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9")
	c := directClient(time.Second)
	tr := c.Transport.(userAgentTransport).base.(*http.Transport)
	if tr.Proxy != nil {
		t.Fatal("directClient transport still carries a Proxy func; want nil")
	}
}
