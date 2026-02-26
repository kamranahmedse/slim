package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	godaemon "github.com/sevlyar/go-daemon"

	"github.com/kamranahmedse/localname/internal/cert"
	"github.com/kamranahmedse/localname/internal/config"
	"github.com/kamranahmedse/localname/internal/log"
	"github.com/kamranahmedse/localname/internal/mdns"
	"github.com/kamranahmedse/localname/internal/proxy"
)

func IsRunning() bool {
	_, err := os.Stat(SocketPath())
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

	cntxt := &godaemon.Context{
		PidFileName: PidPath(),
		PidFilePerm: 0644,
		LogFileName: "",
		WorkDir:     config.Dir(),
		Umask:       027,
	}

	child, err := cntxt.Reborn()
	if err != nil {
		return fmt.Errorf("daemonize: %w", err)
	}

	if child != nil {
		return nil
	}

	defer cntxt.Release()
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
	if err := log.SetOutput(config.LogPath()); err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	defer log.Close()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	srv := proxy.NewServer(cfg, proxy.HTTPPort, proxy.HTTPSPort)

	responder := mdns.New()
	for _, d := range cfg.Domains {
		responder.Register(d.Name, d.Port)
	}

	ipc, err := NewIPCServer(func(req Request) Response {
		return handleIPC(req, srv, responder)
	})
	if err != nil {
		return err
	}
	go ipc.Serve()

	if err := os.WriteFile(PidPath(), []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		responder.Shutdown(ctx)
		ipc.Close()
		cancel()
		srv.Shutdown(ctx)
		os.Remove(PidPath())
	}()

	return srv.Start()
}

func handleIPC(req Request, srv *proxy.Server, responder *mdns.Responder) Response {
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

	case MsgAddDomain:
		return handleAddDomain(req, srv, responder)

	case MsgRemoveDomain:
		return handleRemoveDomain(req, srv)

	default:
		return Response{OK: false, Error: fmt.Sprintf("unknown message type: %s", req.Type)}
	}
}

func handleStatus() Response {
	cfg, err := config.Load()
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	var domains []DomainInfo
	for _, d := range cfg.Domains {
		domains = append(domains, DomainInfo{
			Name:    d.Name,
			Port:    d.Port,
			Healthy: proxy.CheckUpstream(d.Port),
		})
	}

	status := StatusData{
		Running: true,
		PID:     os.Getpid(),
		Domains: domains,
	}
	data, _ := json.Marshal(status)
	return Response{OK: true, Data: data}
}

func handleReload(srv *proxy.Server, responder *mdns.Responder) Response {
	if err := srv.ReloadConfig(); err != nil {
		return Response{OK: false, Error: err.Error()}
	}

	cfg, _ := config.Load()
	responder.Shutdown(context.Background())
	for _, d := range cfg.Domains {
		responder.Register(d.Name, d.Port)
	}
	return Response{OK: true}
}

func handleAddDomain(req Request, srv *proxy.Server, responder *mdns.Responder) Response {
	var dd DomainData
	if err := json.Unmarshal(req.Data, &dd); err != nil {
		return Response{OK: false, Error: fmt.Sprintf("invalid request data: %v", err)}
	}

	cfg, err := config.Load()
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if err := cfg.AddDomain(dd.Name, dd.Port); err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if err := cert.EnsureLeafCert(dd.Name); err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	srv.ReloadConfig()
	responder.Register(dd.Name, dd.Port)
	return Response{OK: true}
}

func handleRemoveDomain(req Request, srv *proxy.Server) Response {
	var dd DomainData
	if err := json.Unmarshal(req.Data, &dd); err != nil {
		return Response{OK: false, Error: fmt.Sprintf("invalid request data: %v", err)}
	}

	cfg, err := config.Load()
	if err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	if err := cfg.RemoveDomain(dd.Name); err != nil {
		return Response{OK: false, Error: err.Error()}
	}
	srv.ReloadConfig()
	return Response{OK: true}
}
