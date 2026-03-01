<p align="center">
  <img src="docs/public/favicon.png" alt="Slim logo" width="64" height="64" />
</p>

<h1 align="center">Slim</h1>

<p align="center">
  Simple command to get a clean HTTPS <code>.local</code> for your local projects
</p>

<p align="center">
  <a href="https://slim.sh"><img src="https://img.shields.io/badge/website-slim.sh-0f172a?style=flat-square" alt="Website"></a>
  <img src="https://img.shields.io/badge/go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-111827?style=flat-square" alt="Platform">
</p>

```
myapp.local        → localhost:3000
myapp.local/api    → localhost:8080
dashboard.local    → localhost:5173
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

Or define all services in a `.slim.yaml` project config and start them at once:

```bash
slim up
```

First run handles all setup automatically (CA generation, keychain trust, port forwarding).

## Usage

```bash
# Start proxying domains
slim start myapp --port 3000
slim start api --port 8080

# Route different paths to different upstream ports
slim start myapp --port 3000 --route /api=8080 --route /ws=9000

# Start all services from .slim.yaml
slim up
slim up --config /path/to/.slim.yaml

# Optional start flags
# Access logs: full | minimal | off
slim start myapp --port 3000 --log-mode minimal

# Wait for upstream readiness before returning
slim start myapp --port 3000 --wait --timeout 30s

# Inspect running domains
slim list
slim list --json

# Access logs with or without tail
slim logs
slim logs --follow myapp
slim logs --flush

# Run diagnostic checks
slim doctor

# Stop proxying one or all
slim stop myapp
slim stop

# Stop all project services from .slim.yaml
slim down
slim down --config /path/to/.slim.yaml

# Version
slim version
```

### Path-based Routing

Route different URL paths to different upstream ports on a single domain:

```bash
slim start myapp --port 3000 --route /api=8080 --route /ws=9000
```

Routes use longest-prefix matching — `/api/users` matches `/api` before `/`. The `--port` flag sets the default upstream for unmatched paths.

### Project Config (`.slim.yaml`)

Define all services for a project in a `.slim.yaml` file at the project root:

```yaml
services:
  - domain: myapp
    port: 3000
    routes:
      - path: /api
        port: 8080
  - domain: dashboard
    port: 5173
log_mode: minimal
```

```bash
slim up                              # start all services from .slim.yaml
slim up --config /path/to/.slim.yaml # or specify a config path
slim down                            # stop all project services
```

### Doctor

Run diagnostic checks to verify your setup:

```
$ slim doctor
  ✓  CA certificate        valid, expires 2035-02-28
  ✓  CA trust              trusted by OS
  ✓  Port forwarding       active (80→10080, 443→10443)
  ✓  Hosts: myapp.local    present in /etc/hosts
  !  Daemon                not running
  ✓  Cert: myapp.local     valid, expires 2027-06-03
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

## License

[PolyForm Shield 1.0.0](./LICENSE) © [Kamran Ahmed](https://x.com/kamrify)
