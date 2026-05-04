package acl

import (
	"context"
	"strings"
	"testing"
)

func TestInitOpenFGAClient_DNSTarget(t *testing.T) {
	_, conn := InitOpenFGAClient(context.Background(), "openfga-headless", 8081, 100)
	defer func() { _ = conn.Close() }()

	target := conn.Target()
	if !strings.HasPrefix(target, "dns-refresh:///") {
		t.Fatalf("expected dns-refresh:/// prefix, got target=%q", target)
	}
}
