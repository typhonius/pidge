<p align="center">
  <img src="logo.svg" width="96" alt="pidge" />
</p>

# pidge

A CLI and webhook server for [Android SMS Gateway](https://docs.sms-gate.app/). Send and receive SMS from the command line, and run a persistent server that captures incoming messages via webhooks.

## Prerequisites

- Go 1.22+
- [Android SMS Gateway](https://docs.sms-gate.app/) app installed on an Android phone
- The phone and your server must be on the same network (or reachable via Tailscale, VPN, etc.)

## Installation

```bash
go install github.com/typhonius/pidge@latest
```

## Configuration

Run `pidge setup` to create `~/.config/pidge/config.toml` interactively. The wizard tests the connection before saving.

```toml
[gateway]
url = "http://192.168.1.100:8080"
username = "admin"
password = "secret"

[server]
listen = ":3851"
db_path = "~/.config/pidge/pidge.db"
webhook_secret = ""
auto_register = false
webhook_url = ""
tls_cert = ""
tls_key = ""
```

**`[gateway]`** — Connection to the Android SMS Gateway app's local API (`url`, `username`, `password`).

**`[server]`** — Settings for `pidge serve`. See [Receiving SMS](#receiving-sms) below.

| Field            | Description                                              | Default                       |
|------------------|----------------------------------------------------------|-------------------------------|
| `listen`         | Address to bind the server                               | `:3851`                       |
| `db_path`        | SQLite database for received messages                    | `~/.config/pidge/pidge.db`    |
| `webhook_secret` | HMAC-SHA256 secret for verifying webhook POSTs           | _(empty — no verification)_   |
| `auto_register`  | Auto-register a webhook on the gateway at startup        | `false`                       |
| `webhook_url`    | URL the gateway should POST to (used with auto_register) | _(empty)_                     |
| `tls_cert`       | Path to TLS certificate file                             | _(empty — plain HTTP)_        |
| `tls_key`        | Path to TLS private key file                             | _(empty — plain HTTP)_        |

Config values can be overridden with environment variables: `PIDGE_URL`, `PIDGE_USER`, `PIDGE_PASS`, `PIDGE_LISTEN`, `PIDGE_DB_PATH`, `PIDGE_WEBHOOK_SECRET`.

## CLI Commands

All commands support `--json` for machine-readable output and `--config <path>` to use an alternate config file.

| Command                          | Description                              |
|----------------------------------|------------------------------------------|
| `pidge setup`                    | Interactive config wizard                |
| `pidge send <number> <message>`  | Send an SMS                              |
| `pidge inbox`                    | List received messages                   |
| `pidge ack <id>`                 | Mark a message as processed              |
| `pidge unack <id>`               | Mark a message as unprocessed            |
| `pidge status <message-id>`      | Check delivery status of a sent message  |
| `pidge health`                   | Check gateway health                     |
| `pidge logs`                     | View device logs (last 24h)              |
| `pidge settings`                 | View device settings                     |
| `pidge webhooks list`            | List registered webhooks                 |
| `pidge webhooks add <url> <event>` | Register a webhook                     |
| `pidge webhooks delete <id>`     | Delete a webhook                         |

`pidge inbox` reads from the local SQLite store populated by `pidge serve`. **You must have `pidge serve` running to receive and view incoming SMS** — the gateway has no inbox API, so incoming messages are only captured via webhooks.

## Receiving SMS

`pidge serve` starts a long-running server that receives webhooks from the gateway when SMS messages arrive and stores them in SQLite. It also exposes a REST API for reading messages and sending SMS.

### HTTPS required

The Android SMS Gateway app requires a **trusted HTTPS** endpoint for webhook delivery — it will silently fail to POST to plain HTTP or self-signed certs.

One way to get trusted certs is with [Tailscale HTTPS](https://tailscale.com/kb/1153/enabling-https):

```bash
sudo tailscale cert \
  --cert-file ~/.config/pidge/your-host.ts.net.crt \
  --key-file ~/.config/pidge/your-host.ts.net.key \
  your-host.ts.net
```

Then configure the server to use them:

```toml
[server]
listen = ":3851"
tls_cert = "~/.config/pidge/your-host.ts.net.crt"
tls_key = "~/.config/pidge/your-host.ts.net.key"
auto_register = true
webhook_url = "https://your-host.ts.net:3851/webhook"
```

### Firewall

The server port (default 3851) must be open for the gateway phone to reach it. If using Tailscale, make sure the port is accepted on the Tailscale interface:

```bash
# nftables example
sudo nft add rule inet filter input iifname "tailscale0" tcp dport 3851 accept
```

### REST API

| Method | Endpoint                         | Description              |
|--------|----------------------------------|--------------------------|
| GET    | `/api/messages`                  | List received messages   |
| GET    | `/api/messages/{id}`             | Get a single message     |
| POST   | `/api/messages/{id}/processed`   | Mark message as processed   |
| DELETE | `/api/messages/{id}/processed`   | Mark message as unprocessed |
| POST   | `/api/send`                      | Send an SMS                 |
| GET    | `/api/health`                    | Server + gateway health  |

`GET /api/messages` supports query parameters: `phone`, `since`, `before` (RFC3339), `processed` (bool), `limit`, `offset`.

`POST /api/send` takes JSON: `{"phoneNumber": "+1...", "message": "..."}`.

### Webhook endpoints

The gateway POSTs to `POST /` or `POST /webhook`. If `webhook_secret` is set, the server verifies `X-Signature` and `X-Timestamp` headers via HMAC-SHA256.

## Gateway phone setup

- **Disable RCS** — the gateway sends plain SMS; RCS can interfere
- **Grant all permissions** — SMS, phone, contacts, notification access
- **Disable battery optimization** for the gateway app
- **Tailscale VPN** — turn off "Block connections without VPN" in Android settings, otherwise the gateway can't deliver webhooks over the tailnet

## Known gotchas

- **Pi-hole / DNS filtering**: The phone may not resolve the webhook hostname. Add it to Pi-hole's local DNS or use the Tailscale IP directly.
- **Self-signed TLS certs**: Android rejects them silently. Use Tailscale HTTPS certs or a real CA.

## License

MIT
