package grpc

import (
	"net"
	"net/url"
	"sync"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

func TestPeriodicDNSResolverBuilder_Scheme(t *testing.T) {
	c := qt.New(t)
	b := PeriodicDNSResolverBuilder(30 * time.Second)
	c.Assert(b.Scheme(), qt.Equals, DNSRefreshScheme)
}

func TestPeriodicDNSResolverBuilder_DefaultInterval(t *testing.T) {
	c := qt.New(t)
	b := PeriodicDNSResolverBuilder(0).(*periodicDNSBuilder)
	c.Assert(b.interval, qt.Equals, DefaultDNSRefreshInterval)
}

type fakeCC struct {
	mu    sync.Mutex
	state resolver.State
	err   error
	calls int
}

func (f *fakeCC) UpdateState(s resolver.State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = s
	f.calls++
	return nil
}

func (f *fakeCC) ReportError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

func (f *fakeCC) NewAddress([]resolver.Address)                             {}
func (f *fakeCC) ParseServiceConfig(string) *serviceconfig.ParseResult      { return nil }
func (f *fakeCC) NewServiceConfig(string)                                   {}

func (f *fakeCC) getState() (resolver.State, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state, f.calls
}

func buildTarget(host string) resolver.Target {
	u, _ := url.Parse(DNSRefreshScheme + ":///" + host)
	return resolver.Target{URL: *u}
}

func TestPeriodicDNSResolver_ResolvesOnBuild(t *testing.T) {
	c := qt.New(t)

	cc := &fakeCC{}
	b := PeriodicDNSResolverBuilder(1 * time.Hour)
	r, err := b.Build(buildTarget("localhost:12345"), cc, resolver.BuildOptions{})
	c.Assert(err, qt.IsNil)
	defer r.Close()

	state, calls := cc.getState()
	c.Assert(calls, qt.Not(qt.Equals), 0, qt.Commentf("Build should trigger initial resolve"))
	c.Assert(len(state.Addresses) > 0, qt.IsTrue, qt.Commentf("localhost should resolve to at least one address"))

	foundPort := false
	for _, addr := range state.Addresses {
		_, port, _ := net.SplitHostPort(addr.Addr)
		if port == "12345" {
			foundPort = true
			break
		}
	}
	c.Assert(foundPort, qt.IsTrue, qt.Commentf("resolved addresses should use the target port"))
}

func TestPeriodicDNSResolver_ReResolvesOnTick(t *testing.T) {
	c := qt.New(t)

	cc := &fakeCC{}
	b := PeriodicDNSResolverBuilder(50 * time.Millisecond)
	r, err := b.Build(buildTarget("localhost:9999"), cc, resolver.BuildOptions{})
	c.Assert(err, qt.IsNil)
	defer r.Close()

	_, initialCalls := cc.getState()
	c.Assert(initialCalls, qt.Not(qt.Equals), 0)

	time.Sleep(200 * time.Millisecond)

	_, afterCalls := cc.getState()
	c.Assert(afterCalls > initialCalls, qt.IsTrue,
		qt.Commentf("should have re-resolved; initial=%d after=%d", initialCalls, afterCalls))
}

func TestPeriodicDNSResolver_StopsOnClose(t *testing.T) {
	c := qt.New(t)

	cc := &fakeCC{}
	b := PeriodicDNSResolverBuilder(50 * time.Millisecond)
	r, err := b.Build(buildTarget("localhost:9999"), cc, resolver.BuildOptions{})
	c.Assert(err, qt.IsNil)

	time.Sleep(150 * time.Millisecond)
	r.Close()

	_, callsAtClose := cc.getState()
	time.Sleep(200 * time.Millisecond)
	_, callsAfter := cc.getState()
	c.Assert(callsAfter, qt.Equals, callsAtClose,
		qt.Commentf("no re-resolution after Close; atClose=%d after=%d", callsAtClose, callsAfter))
}

func TestParseHostPort(t *testing.T) {
	tests := []struct {
		input    string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"myhost:8091", "myhost", "8091", false},
		{"myhost", "myhost", "443", false},
		{"127.0.0.1:80", "127.0.0.1", "80", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c := qt.New(t)
			host, port, err := parseHostPort(tt.input)
			if tt.wantErr {
				c.Assert(err, qt.Not(qt.IsNil))
				return
			}
			c.Assert(err, qt.IsNil)
			c.Assert(host, qt.Equals, tt.wantHost)
			c.Assert(port, qt.Equals, tt.wantPort)
		})
	}
}
