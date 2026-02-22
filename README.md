# <img src="logo.svg" width="32" alt="" valign="middle" /> pidge

Send and receive SMS from the command line. A CLI and webhook server for [Android SMS Gateway](https://docs.sms-gate.app/).

---

## Quick start

```bash
go install github.com/typhonius/pidge@latest
pidge setup        # interactive config wizard — tests the connection before saving
pidge send "+1234567890" "Hello from pidge"
pidge health
```

## Commands

| Command | Description |
|---------|-------------|
| `pidge setup` | Interactive config wizard |
| `pidge send <number> <message>` | Send an SMS |
| `pidge inbox` | List received messages |
| `pidge ack <id>` | Mark a message as processed |
| `pidge unack <id>` | Mark a message as unprocessed |
| `pidge stop` | Gracefully stop the server |
| `pidge status <message-id>` | Check delivery status of a sent message |
| `pidge health` | Check gateway health |
| `pidge logs` | View device logs (last 24h) |
| `pidge settings` | View device settings |
| `pidge webhooks list` | List registered webhooks |
| `pidge webhooks add <url> <event>` | Register a webhook |
| `pidge webhooks delete <id>` | Delete a webhook |

All commands support `--json` for machine-readable output and `--config <path>` for an alternate config file.

## Receiving SMS

`pidge inbox` reads from a local SQLite store populated by `pidge serve`. **You must have `pidge serve` running to receive and view incoming SMS** — the gateway has no inbox API, so incoming messages are only captured via webhooks.

`pidge serve` starts a long-running server that receives webhooks from the gateway when SMS messages arrive, stores them in SQLite, and exposes a REST API.

### HTTPS required

The gateway app requires a **trusted HTTPS** endpoint — it silently fails to POST to plain HTTP or self-signed certs. [Tailscale HTTPS](https://tailscale.com/kb/1153/enabling-https) is the easiest way:

```bash
sudo tailscale cert \
  --cert-file ~/.config/pidge/your-host.ts.net.crt \
  --key-file  ~/.config/pidge/your-host.ts.net.key \
  your-host.ts.net
```

Then point pidge at the certs:

```toml
[server]
tls_cert = "~/.config/pidge/your-host.ts.net.crt"
tls_key  = "~/.config/pidge/your-host.ts.net.key"
auto_register = true
webhook_url   = "https://your-host.ts.net:3851/"
```

### Firewall

The server port (default `3851`) must be reachable by the gateway phone. If using Tailscale, allow it on the Tailscale interface:

```bash
sudo nft add rule inet filter input iifname "tailscale0" tcp dport 3851 accept
```

### REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/messages` | List messages (`?phone`, `?since`, `?before`, `?processed`, `?limit`, `?offset`) |
| `GET` | `/api/messages/{id}` | Get a single message |
| `POST` | `/api/messages/{id}/processed` | Mark as processed |
| `DELETE` | `/api/messages/{id}/processed` | Mark as unprocessed |
| `POST` | `/api/send` | Send an SMS — `{"phoneNumber": "+1...", "message": "..."}` |
| `GET` | `/api/health` | Server + gateway health |

The gateway POSTs incoming SMS to `/` or `/webhook`. If `webhook_secret` is configured, the server verifies `X-Signature` and `X-Timestamp` via HMAC-SHA256.

## Configuration

`pidge setup` creates `~/.config/pidge/config.toml`:

```toml
[gateway]
url      = "http://192.168.1.100:8080"
username = "admin"
password = "secret"

[server]
listen         = ":3851"
db_path        = "~/.config/pidge/pidge.db"
webhook_secret = ""
auto_register  = false
webhook_url    = ""
tls_cert       = ""
tls_key        = ""
```

<details>
<summary>Server config reference</summary>

| Field | Description | Default |
|-------|-------------|---------|
| `listen` | Address to bind | `:3851` |
| `db_path` | SQLite database path | `~/.config/pidge/pidge.db` |
| `webhook_secret` | HMAC-SHA256 secret for verifying POSTs | _(none)_ |
| `auto_register` | Register webhook on gateway at startup | `false` |
| `webhook_url` | URL the gateway should POST to | _(none)_ |
| `tls_cert` | TLS certificate file | _(plain HTTP)_ |
| `tls_key` | TLS private key file | _(plain HTTP)_ |

</details>

Environment variable overrides: `PIDGE_URL`, `PIDGE_USER`, `PIDGE_PASS`, `PIDGE_LISTEN`, `PIDGE_DB_PATH`, `PIDGE_WEBHOOK_SECRET`.

## Phone setup

- **Disable RCS** — the gateway sends plain SMS; RCS can interfere
- **Grant all permissions** — SMS, phone, contacts, notification access
- **Disable battery optimization** for the gateway app
- **Tailscale** — turn off "Block connections without VPN" in Android settings, otherwise webhooks can't reach the tailnet

## Gotchas

- **Pi-hole / DNS filtering** — the phone may not resolve the webhook hostname. Add it to Pi-hole's local DNS or use the Tailscale IP directly.
- **Self-signed certs** — Android rejects them silently. Use Tailscale HTTPS or a real CA.
- **Duplicate webhooks** — the gateway retries with new event IDs. pidge deduplicates by message content + timestamp automatically.

## License

MIT
