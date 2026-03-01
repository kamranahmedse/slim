package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	godaemon "github.com/sevlyar/go-daemon"

	"github.com/kamranahmedse/slim/internal/config"
	"github.com/kamranahmedse/slim/internal/log"
	"github.com/kamranahmedse/slim/internal/proxy"
)

func IsRunning() bool {
	_, err := os.Stat(config.SocketPath())
	if err != nil {
		return false
	}

	resp, err := SendIPC(Request{Type: MsgStatus})
	if err != nil {
		return false
	}
	return resp.OK
}

func RunDetached() error {
	if err := os.MkdirAll(config.Dir(), 0755); err != nil {
		return err
	}

	daemonCtx := &godaemon.Context{
		PidFileName: "",
		PidFilePerm: 0644,
		LogFileName: "",
		WorkDir:     config.Dir(),
		Umask:       027,
	}

	child, err := daemonCtx.Reborn()
	if err != nil {
		return fmt.Errorf("daemonize: %w", err)
	}

	if child != nil {
		return nil
	}

	defer daemonCtx.Release()
	return run()
}

func WaitForDaemon() error {
	for i := 0; i < 50; i++ {
		if IsRunning() {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("daemon failed to start within 5 seconds")
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := log.SetOutput(config.LogPath(), cfg.EffectiveLogMode()); err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer log.Close()

	srv := proxy.NewServer(cfg)

	responder := newMDNSResponder()
	if err := responder.refreshNetworkInfo(); err != nil {
		log.Error("mDNS: failed to enumerate network interfaces: %v", err)
	}
	for _, d := range cfg.Domains {
		if err := responder.register(d.Name, d.Port); err != nil {
			log.Error("mDNS registration failed for %s: %v", d.Name, err)
		}
	}

	ipc, err := NewIPCServer(func(req Request) Response {
		return handleIPC(req, srv, responder)
	})
	if err != nil {
		return err
	}
	go ipc.Serve()

	if err := os.WriteFile(config.PidPath(), []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			responder.shutdown()
			ipc.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
			_ = os.Remove(config.PidPath())
		})
	}
	defer cleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		<-sigCh
		cleanup()
	}()

	return srv.Start()
}

func handleIPC(req Request, srv *proxy.Server, responder *mdnsResponder) Response {
	switch req.Type {
	case MsgShutdown:
		go func() {
			p, _ := os.FindProcess(os.Getpid())
			p.Signal(syscall.SIGTERM)
		}()
		return Response{OK: true}

	case MsgStatus:
		return handleStatus()

	case MsgReload:
		return handleReload(srv, responder)

	default:
		return Response{OK: false, Error: fmt.Sprintf("unknown message type: %s", req.Type)}
	}
}

func handleStatus() Response {
	cfg, err := config.Load()
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	var allPorts []int
	domains := make([]DomainInfo, len(cfg.Domains))
	for i, d := range cfg.Domains {
		domains[i] = DomainInfo{Name: d.Name, Port: d.Port}
		allPorts = append(allPorts, d.Port)
		for _, r := range d.Routes {
			domains[i].Routes = append(domains[i].Routes, RouteInfo{Path: r.Path, Port: r.Port})
			allPorts = append(allPorts, r.Port)
		}
	}

	health := proxy.CheckUpstreams(allPorts)
	idx := 0
	for i := range domains {
		domains[i].Healthy = health[idx]
		idx++
		for j := range domains[i].Routes {
			domains[i].Routes[j].Healthy = health[idx]
			idx++
		}
	}

	status := StatusData{
		Running: true,
		PID:     os.Getpid(),
		Domains: domains,
	}
	data, err := json.Marshal(status)
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	return Response{OK: true, Data: data}
}

func handleReload(srv *proxy.Server, responder *mdnsResponder) Response {
	cfg, err := srv.ReloadConfig()
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if err := log.SetOutput(config.LogPath(), cfg.EffectiveLogMode()); err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	responder.shutdown()
	if err := responder.refreshNetworkInfo(); err != nil {
		log.Error("mDNS: failed to enumerate network interfaces: %v", err)
	}
	for _, d := range cfg.Domains {
		if err := responder.register(d.Name, d.Port); err != nil {
			log.Error("mDNS registration failed for %s: %v", d.Name, err)
		}
	}
	return Response{OK: true}
}
