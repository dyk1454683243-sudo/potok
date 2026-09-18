> [!WARNING]
> Potok is in active development. Some commands are not yet implemented.

# Potok

**Potok** is a self-hosted, CLI-based tool for backing up and syncing [Obsidian](https://obsidian.md/) vaults with end-to-end encryption. Your notes stay yours — the server never sees your passwords or unencrypted data.

For more detail, check out the [Potok Docs](https://potok-docs.vercel.app/)

## Features

- **End-to-End Encryption** — Vaults are encrypted locally before leaving your device. The server only stores encrypted data.
- **Self-Hosted** — Run your own Potok server. No third-party cloud, no vendor lock-in.
- **Multiple Vaults** — Manage and sync multiple vaults independently.
- **Automatic Sync** — Watches your vault folder for changes and pushes them automatically.
- **Cross-Platform** — Supports Windows and Linux. macOS is untested but might work?
- **Secure Key Storage** — The CLI keeps encryption passwords and API keys in your OS keyring (Windows Credential Manager, macOS Keychain, Linux Secret Service). The server stores only a SHA-256 hash of each API key, never the live secret.
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
| `potok pull [name]` | Download and decrypt a vault |
| `potok sync [name]` | Watch and auto-sync a vault |
| `potok doctor` | Run diagnostics on your setup |

## Getting Started

### Prerequisites

- A running Potok server ([server setup guide](update))
- An API key from your server admin
- Go 1.21+ (if building from source)

### Install

```bash
go install github.com/michaeltukdev/Potok/cmd/client@latest
```

### Initialise

```bash
potok init
```

You'll be prompted for your server URL and API key. These are stored locally in `~/.potok/config.json` and your OS keyring respectively.

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

Shows all vaults registered on this device with their path and last sync time.

### Push a vault to the server

```bash
potok push notes
```

Encrypts and uploads the vault to your server. Creates the remote vault automatically on first push.

### Pull a vault from the server

```bash
potok pull notes --dest ~/Documents/Obsidian/Notes
```

Downloads and decrypts a vault into the specified directory.

### Sync a vault

```bash
potok sync notes
```

Long-running process that watches for local changes and pushes them automatically.

## Configuration

### Config file

| OS | Path |
|---|---|
| Linux | `~/.potok/config.json` |
| Windows | `%USERPROFILE%\.potok\config.json` |

```json
{
  "api_url": "http://localhost:8080",
  "vaults": [
    {
      "name": "notes",
      "path": "/home/user/Documents/Obsidian/Notes",
      "last_synced": ""
    }
  ]
}
```

### Sensitive data

The CLI stores passwords and API keys in your OS keyring under the `potok` service — never in config files. If an older `config.json` still has an `api_key` field, the next `potok` command moves it into the keyring and rewrites the file.

| OS | Keyring backend |
|---|---|
| Linux | Secret Service (GNOME Keyring / KDE Wallet) |
| macOS | Keychain |
| Windows | Credential Manager |

| Keyring entry | Value |
|---|---|
| `potok / api-key` | Your server API key |
| `potok / vault:{name}` | Encryption password for that vault |

## Security

- All encryption and decryption happens locally on your device.
- The server only stores encrypted blobs — it never sees your passwords or note plaintext.
- Client secrets (the live API key and vault passphrases) live in the OS keyring, not in `config.json`.
- Server-side API keys are stored as `sha256:` digests. Keys are 256-bit values from `crypto/rand`, so a fast cryptographic hash is enough to protect them at rest (the same approach used for GitHub personal access tokens). User passwords stay on bcrypt. The live key is shown once at registration and cannot be recovered from the database.
- Encryption uses AES-GCM via `golang.org/x/crypto`; passphrases are stretched with Argon2id.

## Roadmap

- [x] CLI skeleton and local vault management
- [x] OS keyring integration for passwords and API keys
- [x] Push — encrypt and upload vaults
- [x] Pull — download and decrypt vaults
- [x] Automatic file watching and sync
- [x] File-level sync (currently uploads entire vault)
- [ ] Conflict detection and handling
- [ ] Version history
- [ ] Web dashboard for server admin
- [ ] Cross-platform installers
