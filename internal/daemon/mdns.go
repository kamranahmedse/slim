package daemon

import (
	"net"
	"sync"

	"github.com/grandcat/zeroconf"
	"github.com/kamranahmedse/slim/internal/log"
)

type mdnsResponder struct {
	servers []*zeroconf.Server
	ifaces  []net.Interface
	ips     []string
	mu      sync.Mutex
}

func newMDNSResponder() *mdnsResponder {
	return &mdnsResponder{}
}

func (r *mdnsResponder) register(name string, port int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	hostname := name + ".local."

	if len(r.ifaces) == 0 {
		ifaces, err := getActiveInterfaces()
		if err != nil {
			return err
		}
		r.ifaces = ifaces
	}
	if len(r.ips) == 0 {
		r.ips = getLocalIPs()
	}

	srv, err := zeroconf.RegisterProxy(
		name,
		"_http._tcp",
		"local.",
		port,
		hostname,
		r.ips,
		[]string{
			"slim=true",
			"domain=" + name + ".local",
		},
		r.ifaces,
	)
	if err != nil {
		return err
	}

	r.servers = append(r.servers, srv)
	log.Info("mDNS: advertising %s.local on LAN via %s", name, hostname)
	return nil
}

func (r *mdnsResponder) shutdown() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, srv := range r.servers {
		srv.Shutdown()
	}
	r.servers = nil
}

func (r *mdnsResponder) refreshNetworkInfo() error {
	ifaces, err := getActiveInterfaces()
	if err != nil {
		return err
	}
	ips := getLocalIPs()

	r.mu.Lock()
	r.ifaces = ifaces
	r.ips = ips
	r.mu.Unlock()
	return nil
}

func getLocalIPs() []string {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ipNet.IP.To4() != nil {
			ips = append(ips, ipNet.IP.String())
		}
	}
	return ips
}

func getActiveInterfaces() ([]net.Interface, error) {
	allIfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var active []net.Interface
	for _, iface := range allIfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		active = append(active, iface)
	}
	return active, nil
}
