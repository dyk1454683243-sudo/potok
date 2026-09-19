> [!WARNING]
> Potok is in active development. Watch-sync (`potok sync`) is not implemented yet.

# Potok

**Potok** is a self-hosted, CLI-based tool for backing up and syncing [Obsidian](https://obsidian.md/) vaults with end-to-end encryption. Your notes stay yours — the server never sees your passwords or unencrypted data.

For more detail, check out the [Potok Docs](https://potok-docs.vercel.app/)

## Features

- **End-to-End Encryption** — Vaults are encrypted locally before leaving your device. The server only stores encrypted data.
- **Self-Hosted** — Run your own Potok server (`potokd`). No third-party cloud, no vendor lock-in.
- **Multiple Vaults** — Manage and sync multiple vaults independently.
- **Push / Pull** — Encrypt+upload and download+decrypt for registered vaults.
- **Cross-Platform** — Supports Windows and Linux. macOS is untested but might work.
- **Secure Key Storage** — Encryption passwords and API keys are stored in your OS keyring (Windows Credential Manager, macOS Keychain, Linux Secret Service).
- **Free & Open Source** — No file size limits, no file count limits, no paywalls.

## Commands

| Command | Description |
|---|---|
| `potok init` | Set server URL and API key |
| `potok vault-add` | Register a local folder as a vault |
| `potok vaults-list` | List vaults registered locally |
| `potok vault-remove [name]` | Remove a vault from local config |
| `potok remote-list` | List vaults available on the server |
| `potok remote-delete [name]` | Delete a vault from the server |
| `potok push [name]` | Encrypt and upload a vault |
| `potok pull [name]` | Download and decrypt into the registered local path |
| `potok doctor` | Run diagnostics on your setup |

`potok sync` (watch and auto-push) is **not** implemented yet.

## Getting Started

### Prerequisites

- A running Potok server (`potokd`)
- An API key from your server (shown once at registration)
- Go 1.25+ (if building from source; see `go.mod`)

### Install

```bash
go install github.com/mtiluk/potok/cmd/potok@latest
```

### Initialise

```bash
potok init
```

You'll be prompted for your server URL and API key. The URL is stored in the config file; the API key is stored in your OS keyring.

## Usage

### Register a vault

```bash
potok vault-add
```

Prompts for a vault name, local folder path, and encryption password. This only registers the vault locally — nothing is uploaded yet.

### List local vaults

```bash
potok vaults-list
```

### Push a vault to the server

```bash
potok push notes
```

Encrypts and uploads the vault. Creates the remote vault automatically on first push.

### Pull a vault from the server

```bash
potok pull notes
```

Downloads and decrypts into the vault's registered local path (from `vault-add`).

### Diagnostics

```bash
potok doctor
```

## Configuration

### Config file

| OS | Path |
|---|---|
| Linux (default) | `~/.potok/config.json` |
| Linux (XDG) | `$XDG_CONFIG_HOME/potok/config.json` |
| Windows | `%USERPROFILE%\.potok\config.json` |
| Override | `$POTOK_CONFIG_DIR/config.json` |

```json
{
  "server_url": "http://localhost:8080",
  "vaults": [
    {
      "name": "notes",
      "path": "/home/user/Documents/Obsidian/Notes",
      "remote_id": "",
      "last_synced_at": null
    }
  ]
}
```

### Sensitive data

Passwords and API keys are stored in your OS keyring under the `potok` service — not in config files.

| OS | Keyring backend |
|---|---|
| Linux | Secret Service (GNOME Keyring / KDE Wallet) |
| macOS | Keychain |
| Windows | Credential Manager |

## Security

- All encryption and decryption happens locally on your device.
- The server only stores encrypted blobs — it never sees your passwords or plaintext.
- Passwords and API keys are stored in your OS keyring, not in config files.
- Client crypto uses AES via `golang.org/x/crypto`.

## Roadmap

- [x] CLI skeleton and local vault management
- [x] OS keyring integration for passwords and API keys
- [x] Push — encrypt and upload vaults
- [x] Pull — download and decrypt vaults
- [ ] Automatic file watching and sync (`potok sync`)
- [ ] Incremental / file-level sync (today each push re-uploads)
- [ ] Conflict detection and handling
- [ ] Version history
- [ ] Web dashboard for server admin
- [ ] Cross-platform installers
