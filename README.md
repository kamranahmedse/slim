# localname

Map custom `.local` domains to local dev server ports with HTTPS, mDNS for LAN access, and WebSocket passthrough for HMR.

```
myapp.local    → localhost:3000
api.local      → localhost:8080
dashboard.local → localhost:5173
```

## Install

```bash
go install github.com/kamrify/localname@latest
```

Or build from source:

```bash
make build
make install
```

## Quick Start

```bash
# One-time setup: generate CA, trust it, configure port forwarding
localname setup

# Add a domain mapping
localname add myapp --port 3000

# Start the proxy
localname up

# Open https://myapp.local in your browser
```

## Usage

### Managing domains

```bash
localname add myapp --port 3000    # myapp.local → localhost:3000
localname add api -p 8080          # api.local → localhost:8080
localname list                     # Show all mappings
localname remove myapp             # Remove a mapping
```

### Running the proxy

```bash
localname up              # Run in foreground (Ctrl+C to stop)
localname up --detach     # Run in background
localname status          # Show running state + upstream health
localname down            # Stop the background proxy
```

Domains can be added or removed while the proxy is running — routes update automatically.

### Service management

```bash
localname service install     # Start on login (macOS launchd)
localname service uninstall   # Remove login service
```

### Uninstall

```bash
localname uninstall   # Remove everything: CA, certs, config, hosts entries, pfctl rules
```

## How It Works

- **HTTPS**: A root CA is generated during setup and trusted in the macOS keychain. Per-domain leaf certificates are created on demand and served via SNI.
- **Reverse proxy**: Go's `httputil.ReverseProxy` handles HTTP, HTTPS, and WebSocket upgrades natively — HMR for Next.js, Vite, etc. works out of the box.
- **Local resolution**: `/etc/hosts` entries are managed automatically when adding/removing domains.
- **LAN access**: mDNS (Bonjour) advertises domains so other devices on the network can reach them.
- **Port forwarding**: macOS `pfctl` redirects ports 80/443 to unprivileged 10080/10443 so the proxy doesn't need root.

## Configuration

Config lives at `~/.localname/config.yaml`:

```yaml
domains:
  - name: myapp
    port: 3000
  - name: api
    port: 8080
```

Certificates are stored in `~/.localname/certs/` and the root CA in `~/.localname/ca/`.

## Platform Support

- **macOS**: Full support (keychain trust, pfctl, launchd)
- **Linux**: Planned — stubs are in place

## License

***REMOVED***
