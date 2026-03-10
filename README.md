<h1 align="center">⚡️ Slim</h1>

<p align="center">
  Simple command to get clean HTTPS local domains for your projects
</p>

<p align="center">
  <a href="https://slim.sh"><img src="https://img.shields.io/badge/website-slim.sh-0f172a?style=flat-square" alt="Website"></a>
  <img src="https://img.shields.io/badge/go-1.25%2B-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux-111827?style=flat-square" alt="Platform">
</p>

```
myapp.test        → localhost:3000
myapp.test/api    → localhost:8080
dashboard.test    → localhost:5173
app.loc           → localhost:4000
```

## Install

```bash
curl -sL https://slim.sh/install.sh | sh
```

or build from source

```bash
git clone https://github.com/kamranahmedse/slim.git
cd slim
make build
make install
```

Requires Go 1.25 or later.

## Quick Start

Create a custom HTTPs Domain for local development using `slim start`

```bash
slim start myapp --port 3000
# https://myapp.test → localhost:3000
```

To share your local project on a public URL

```bash
slim share --port 3000
# https://cheeky-panda.slim.show
```

## Local Usage

> Start or stop a local service using `slim start` or `slim stop`

```bash
slim start myapp --port 3000
slim start api --port 8080
slim stop myapp                  # stop one domain
slim stop                        # stop all domains
```

> If you don't specify the TLD, you get a `.test` domain. Specify a full domain to use any TLD:

```bash
slim start app.loc --port 3000   # https://app.loc → localhost:3000
slim start my.demo --port 4000   # https://my.demo → localhost:4000
```

> **Note:** Avoid `.local` — it's reserved for mDNS and can cause slow DNS resolution on macOS/Linux.

> Route different URL paths to different upstream ports on a single domain:

```bash
slim start myapp --port 3000 --route /api=8080 --route /ws=9000
```

> Define all services for a project in a `.slim.yaml` file at the project root:

```yaml
services:
  - domain: myapp
    port: 3000
    routes:
      - path: /api
        port: 8080
  - domain: dashboard
    port: 5173
  - domain: app.loc
    port: 4000
log_mode: minimal  # full | minimal | off
cors: true         # enable CORS headers on proxied responses
```

```bash
slim up                              # start all services
slim up --config /path/to/.slim.yaml # specify a config path
slim down                            # stop all project services
```

## Internet Sharing

> Expose a local server to the internet with a public `slim.show` URL. Requires `slim login` first.

```bash
slim share --port 3000                              # random subdomain
slim share --port 3000 --subdomain demo             # https://demo.slim.show
slim share --port 3000 --password secret            # password protected
slim share --port 3000 --ttl 30m                    # auto-expires after 30 minutes
slim share --port 3000 --domain myapp.example.com   # custom domain
```


## Logs and Diagnostics

```bash
slim list                # inspect running domains
slim list --json

slim logs                # view access logs
slim logs --follow myapp # tail logs for a domain
slim logs --flush        # clear log file

slim doctor              # run diagnostic checks
```

```
$ slim doctor
  ✓  CA certificate        valid, expires 2035-02-28
  ✓  CA trust              trusted by OS
  ✓  Port forwarding       active (80→10080, 443→10443)
  ✓  Hosts: myapp.test    present in /etc/hosts
  !  Daemon                not running
  ✓  Cert: myapp.test     valid, expires 2027-06-03
```

## Updating

Run `slim update` to update to latest version.

## Uninstall

> Remove everything: CA, certs, hosts entries, port-forward rules, config

```bash
slim uninstall
```

## License

[PolyForm Shield 1.0.0](./LICENSE) © [Kamran Ahmed](https://x.com/kamrify)
