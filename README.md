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

Requires Go 1.25 or later to build from source.

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
localname start myapp -p 3000 --log-mode minimal  # access logs: full|minimal|off
localname list                       # see what's running + health
localname logs                       # tail request logs
localname logs -f myapp              # follow logs for one domain
localname logs --flush               # clear access logs
localname stop myapp                 # stop one domain
localname stop                       # stop everything
localname version                    # print version
```

### Uninstall

```bash
localname uninstall   # removes everything: CA, certs, hosts entries, port-forward rules, config
```

## How It Works

- **HTTPS**: A root CA is generated on first use and trusted in the system trust store (macOS Keychain or Linux CA store). Per-domain leaf certificates are created on demand and served via SNI.
- **Reverse proxy**: Go's `httputil.ReverseProxy` handles HTTP, HTTPS, and WebSocket upgrades natively — HMR for Next.js, Vite, etc. works out of the box.
- **Local resolution**: `/etc/hosts` entries are managed automatically.
- **LAN access**: mDNS (Bonjour) advertises domains so other devices on the network can reach them.
- **Port forwarding**: macOS `pfctl` or Linux `iptables` redirects ports 80/443 to unprivileged 10080/10443 so the proxy doesn't need root.
- **Daemon**: The proxy runs in the background. `start` launches it automatically, `stop` shuts it down.

## Configuration

Config lives at `~/.localname/config.yaml`. Certificates in `~/.localname/certs/`, root CA in `~/.localname/ca/`, logs in `~/.localname/access.log`.

Set access logging mode globally (persisted in config) with:

```bash
localname start myapp --port 3000 --log-mode full     # default
localname start myapp --port 3000 --log-mode minimal
localname start myapp --port 3000 --log-mode off
```

## Requirements

First-time setup requires `sudo` for:
- Trusting the root CA in the system trust store (macOS and Linux)
- Setting up port forwarding rules (macOS: `pfctl`, Linux: `iptables`)
- Managing `/etc/hosts` entries

On Linux, CA trust uses one of: `update-ca-certificates` (Debian/Ubuntu) or `update-ca-trust` (RHEL/Fedora/Arch-family setups).

After trusting the CA, you may need to restart your browser for it to recognize the new root certificate.

## Platform Support

- **macOS**: Full support (port forwarding via `pfctl`, CA trust via Keychain)
- **Linux**: Full support for hosts management, CA trust (via `update-ca-certificates` or `update-ca-trust`), and port forwarding (via `iptables`). Note that `systemd-resolved` and `avahi-daemon` often claim the `.local` TLD, which may require additional configuration.

## Notes on `.local` TLD

The `.local` TLD is reserved for mDNS (RFC 6762). localname uses this intentionally for LAN discovery, but be aware:
- Some corporate networks and VPNs intercept `.local` resolution, which may conflict.
- Existing Bonjour/Avahi services on the network may advertise the same hostnames.

## License

***REMOVED***
