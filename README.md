# localname

Map custom `.local` domains to local dev server ports with HTTPS, mDNS for LAN access, and WebSocket passthrough for HMR.

```
myapp.local     → localhost:3000
api.local       → localhost:8080
dashboard.local → localhost:5173
```

## Install

```bash
curl -sL https://raw.githubusercontent.com/kamranahmedse/localname/main/install.sh | sh
```

Or build from source:

```bash
git clone https://github.com/kamranahmedse/localname.git
cd localname
make build
make install
```

## Quick Start

```bash
# Start your dev server, then:
localname start myapp --port 3000

# That's it. Open https://myapp.local
```

First run handles all setup automatically (CA generation, keychain trust, port forwarding).

## Usage

```bash
localname start myapp --port 3000    # start proxying a domain
localname start api -p 8080          # add another
localname list                       # see what's running + health
localname logs                       # tail request logs
localname logs -f myapp              # follow logs for one domain
localname stop myapp                 # stop one domain
localname stop                       # stop everything
```

### Uninstall

```bash
localname uninstall   # removes everything: CA, certs, hosts entries, pfctl rules, config
```

## How It Works

- **HTTPS**: A root CA is generated on first use and trusted in the macOS keychain. Per-domain leaf certificates are created on demand and served via SNI.
- **Reverse proxy**: Go's `httputil.ReverseProxy` handles HTTP, HTTPS, and WebSocket upgrades natively — HMR for Next.js, Vite, etc. works out of the box.
- **Local resolution**: `/etc/hosts` entries are managed automatically.
- **LAN access**: mDNS (Bonjour) advertises domains so other devices on the network can reach them.
- **Port forwarding**: macOS `pfctl` redirects ports 80/443 to unprivileged 10080/10443 so the proxy doesn't need root.
- **Daemon**: The proxy runs in the background. `start` launches it automatically, `stop` shuts it down.

## Configuration

Config lives at `~/.localname/config.yaml`. Certificates in `~/.localname/certs/`, root CA in `~/.localname/ca/`, logs in `~/.localname/access.log`.

## Platform Support

- **macOS**: Full support
- **Linux**: Planned

## License

***REMOVED***
