package grpc

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc/resolver"
)

const (
	// DefaultDNSRefreshInterval is the interval at which the periodic DNS
	// resolver re-resolves hostnames. The gRPC built-in DNS resolver only
	// re-resolves on subchannel failure; when HPA adds new pods behind a
	// headless K8s service the existing client never discovers them. This
	// resolver periodically looks up DNS A records and pushes updated
	// addresses to the balancer, so round_robin distributes RPCs across
	// newly scaled pods within one refresh cycle.
	DefaultDNSRefreshInterval = 30 * time.Second
)

// DNSRefreshScheme is the resolver scheme for the periodic DNS resolver.
const DNSRefreshScheme = "dns-refresh"

// PeriodicDNSResolverBuilder returns a resolver.Builder that periodically
// re-resolves DNS A records for the target host and pushes updated
// addresses to the gRPC balancer.
//
// Usage:
//
//	grpc.NewClient("dns-refresh:///my-headless-svc:8091",
//	    grpc.WithResolvers(xgrpc.PeriodicDNSResolverBuilder(30*time.Second)),
//	    grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`),
//	)
func PeriodicDNSResolverBuilder(interval time.Duration) resolver.Builder {
	if interval <= 0 {
		interval = DefaultDNSRefreshInterval
	}
	return &periodicDNSBuilder{interval: interval}
}

type periodicDNSBuilder struct {
	interval time.Duration
}

func (b *periodicDNSBuilder) Scheme() string { return DNSRefreshScheme }

func (b *periodicDNSBuilder) Build(target resolver.Target, cc resolver.ClientConn, _ resolver.BuildOptions) (resolver.Resolver, error) {
	host, port, err := parseHostPort(target.Endpoint())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &periodicDNSResolver{
		host:     host,
		port:     port,
		cc:       cc,
		interval: b.interval,
		ctx:      ctx,
		cancel:   cancel,
		netRes:   net.DefaultResolver,
	}

	r.resolve()
	go r.loop()
	return r, nil
}

type periodicDNSResolver struct {
	host     string
	port     string
	cc       resolver.ClientConn
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	closeOnce sync.Once
	netRes   *net.Resolver
}

func (r *periodicDNSResolver) resolve() {
	lookupCtx, lookupCancel := context.WithTimeout(r.ctx, 5*time.Second)
	defer lookupCancel()

	addrs, err := r.netRes.LookupHost(lookupCtx, r.host)
	if err != nil {
		r.cc.ReportError(err)
		return
	}

	var endpoints []resolver.Address
	for _, a := range addrs {
		endpoints = append(endpoints, resolver.Address{Addr: net.JoinHostPort(a, r.port)})
	}
	if len(endpoints) > 0 {
		_ = r.cc.UpdateState(resolver.State{Addresses: endpoints})
	}
}

func (r *periodicDNSResolver) loop() {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.resolve()
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *periodicDNSResolver) ResolveNow(resolver.ResolveNowOptions) {
	r.resolve()
}

func (r *periodicDNSResolver) Close() {
	r.closeOnce.Do(func() { r.cancel() })
}

func parseHostPort(endpoint string) (string, string, error) {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		host = endpoint
		port = "443"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return "", "", err
	}
	return host, port, nil
}
