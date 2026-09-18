> [!WARNING]
> Potok is in active development. Some advertised features are not implemented yet — see Commands and Roadmap below.

# Potok

**Potok** is a self-hosted, CLI-based tool for backing up and syncing [Obsidian](https://obsidian.md/) vaults with end-to-end encryption. Your notes stay yours — the server never sees your passwords or unencrypted data.

For more detail, check out the [Potok Docs](https://potok-docs.vercel.app/)

## Features

- **End-to-End Encryption** — Vault files are encrypted locally before upload. The server only stores encrypted blobs.
- **Self-Hosted** — Run your own Potok server. No third-party cloud, no vendor lock-in.
- **Multiple Vaults** — Register and manage several vaults independently.
- **Manual Push / Pull** — Encrypt and upload with `push`, download and decrypt with `pull`.
- **Cross-Platform** — Supports Windows and Linux. macOS is untested but might work.
- **Secure Key Storage** — Encryption passphrases and API keys are stored in your OS keyring (Windows Credential Manager, macOS Keychain, Linux Secret Service).
- **Free & Open Source** — No file size limits, no file count limits, no paywalls.

## Commands

| Command | Status | Description |
|---|---|---|
| `potok init` | available | Set server URL and API key |
| `potok vault-add` | available | Register a local folder as a vault |
| `potok vaults-list` | available | List vaults registered locally |
| `potok vault-remove [name]` | available | Remove a vault from local config |
| `potok remote-list` | available | List vaults available on the server |
| `potok remote-delete [name]` | available | Delete a vault from the server |
| `potok push [name]` | available | Encrypt and upload a vault (per-file blobs) |
| `potok pull [name]` | available | Download and decrypt into the registered path |
| `potok doctor` | available | Run diagnostics on your setup |
| `potok sync [name]` | **not implemented** | Planned watch / auto-sync |

## Getting Started

### Prerequisites

- A running Potok server (`potokd` in this repository)
- An API key from your server admin
- Go 1.25+ (if building from source)

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

Prompts for a vault name, local folder path, and encryption passphrase. This only registers the vault locally — nothing is uploaded yet.

### List local vaults

```bash
potok vaults-list
```

Shows all vaults registered on this device with their path and last sync time.

### Push a vault to the server

```bash
potok push notes
```

Encrypts each file and uploads blobs plus an encrypted manifest. Creates the remote vault automatically on first push.

### Pull a vault from the server

```bash
potok pull notes
```

Downloads and decrypts files into the vault's registered local path.

## Configuration

### Config file

| OS | Path |
|---|---|
| Linux / macOS | `~/.potok/config.json` (or `$XDG_CONFIG_HOME/potok/config.json`) |
| Windows | `%USERPROFILE%\.potok\config.json` |

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

Passphrases and API keys are stored in your OS keyring under the `potok` service — never in config files.

| OS | Keyring backend |
|---|---|
| Linux | Secret Service (GNOME Keyring / KDE Wallet) |
| macOS | Keychain |
| Windows | Credential Manager |

| Keyring entry | Value |
|---|---|
| `potok / api-key` | Your server API key |
| `potok / vault:{name}` | Encryption passphrase for that vault |

## Security

- All encryption and decryption happens locally on your device.
- The server only stores encrypted blobs — it never sees your passphrases or plaintext.
- Passphrases and API keys are stored in your OS keyring, not in config files.
- Encryption uses AES via `golang.org/x/crypto`.

## Roadmap

- [x] CLI skeleton and local vault management
- [x] OS keyring integration for passphrases and API keys
- [x] Push — encrypt and upload vaults (per-file blobs + manifest)
- [x] Pull — download and decrypt vaults
- [ ] Automatic file watching and sync (`potok sync`)
- [ ] Conflict detection and handling
- [ ] Version history
- [ ] Web dashboard for server admin
- [ ] Cross-platform installers
