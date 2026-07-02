// Package hysteria2_e2e contains the hysteria2 end-to-end test harness. The
// harness and its tests are gated behind the `e2e` build tag so they never run
// in the normal unit suite; this untagged file keeps the package valid for
// `go build ./...` / `go test ./...`. Run the harness with:
//
//	go test -tags e2e ./test/e2e/hysteria2/
package hysteria2_e2e
