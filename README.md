<p align="center">
  <img src="docs/public/favicon.png" alt="Slim logo" width="64" height="64" />
</p>

<h1 align="center">Slim</h1>

<p align="center">
  One command to map any local app to a clean HTTPS domain
</p>

<p align="center">
  <a href="https://slim.sh"><img src="https://img.shields.io/badge/website-slim.sh-0f172a?style=flat-square" alt="Website"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-PolyForm%20Shield-16a34a?style=flat-square" alt="PolyForm Shield License 1.0.0"></a>
  <img src="https://img.shields.io/badge/go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-111827?style=flat-square" alt="Platform">
</p>

```
myapp.local     → localhost:3000
api.local       → localhost:8080
dashboard.local → localhost:5173
```

## Install

```bash
curl -sL https://slim.sh/install.sh | sh
```

Or build from source:

```bash
git clone https://github.com/kamranahmedse/slim.git
cd slim
make build
make install
```

Requires Go 1.25 or later to build from source.

## Quick Start

```bash
# Start your dev server, then:
slim start myapp --port 3000

# That's it. Open https://myapp.local
```

First run handles all setup automatically (CA generation, keychain trust, port forwarding).

## Usage

```bash
slim start myapp --port 3000    # start proxying a domain
slim start api -p 8080          # add another
slim start myapp -p 3000 --log-mode minimal  # access logs: full|minimal|off
slim start myapp -p 3000 --wait --timeout 30s # wait for upstream readiness
slim list                       # see what's running + health
slim list --json                # JSON output
slim logs                       # tail request logs
slim logs -f myapp              # follow logs for one domain
slim logs --flush               # clear access logs
slim stop myapp                 # stop one domain
slim stop                       # stop everything
slim version                    # print version
```

### Uninstall

```bash
slim uninstall   # removes everything: CA, certs, hosts entries, port-forward rules, config
```

## How It Works

- **HTTPS**: A root CA is generated on first use and trusted in the system trust store (macOS Keychain or Linux CA store). Per-domain leaf certificates are created on demand and served via SNI.
- **Reverse proxy**: Go's `httputil.ReverseProxy` handles HTTP/2, WebSocket upgrades, and CORS natively — HMR for Next.js, Vite, etc. works out of the box.
- **Local resolution**: `/etc/hosts` entries are managed automatically.
- **LAN discovery**: Optional mDNS (Bonjour/Avahi) service announcements can help discover running apps on the local network.
- **Port forwarding**: macOS `pfctl` or Linux `iptables` redirects ports 80/443 to unprivileged 10080/10443 so the proxy doesn't need root.
- **Daemon**: The proxy runs in the background. `start` launches it automatically, `stop` shuts it down.

## Configuration

Config lives at `~/.slim/config.yaml`. Certificates in `~/.slim/certs/`, root CA in `~/.slim/ca/`, logs in `~/.slim/access.log`.

Set access logging mode globally (persisted in config) with:

```bash
slim start myapp --port 3000 --log-mode full     # default
slim start myapp --port 3000 --log-mode minimal
slim start myapp --port 3000 --log-mode off
```

## Requirements

First-time setup requires `sudo` for CA trust, port forwarding, and `/etc/hosts` management. macOS and Linux supported.

## License

***REMOVED***
