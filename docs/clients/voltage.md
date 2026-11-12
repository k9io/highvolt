# Voltage — Filesystem Scanner

`voltage` is an endpoint agent that scans local filesystems for files matching configured MIME types and submits them to `highvolt-server` for PII analysis.

## How it works

1. On startup, Voltage fetches its configuration from JSONAir and authenticates to `highvolt-server`.
2. Based on the detected operating system (Linux/BSD/macOS/Windows), it reads the appropriate directory list and MIME type list from the configuration.
3. It walks each configured directory recursively, skipping symlinks, devices, pipes, and sockets.
4. For each regular file it checks the configured exclude patterns (path substring match).
5. Files matching a configured MIME type are processed:
   - If the file exceeds `core.max_size`, it is skipped.
   - MD5, SHA1, and SHA256 hashes are computed in a single streaming pass.
   - The SHA256 is checked against the local registry — already-seen files are skipped.
   - The file is submitted to `highvolt-server` (query first, then submit if not found).
   - On successful submission, the SHA256 is recorded in the local registry.
6. After all directories are walked, Voltage sleeps for `core.sleep_interval` seconds (default 86400, one day) and repeats. With `--once`, or when run as the Windows scheduled task, it exits after a single pass instead.

## Local registry

Voltage maintains a local GOB (Go binary encoding) file that stores the SHA256 hashes of every file it has successfully submitted. This prevents re-uploading files that have not changed.

The registry path is computed from the host's persistent storage location. Use `--nuke` to delete it and force a full re-scan.

## Service management

Voltage uses the `kardianos/service` library to integrate with the OS service manager (systemd, launchd, Windows SCM).

```bash
# Install the service
./voltage install

# Start the service
./voltage start

# Stop the service
./voltage stop

# Restart the service
./voltage restart

# Uninstall the service
./voltage uninstall
```

## Flags

| Flag | Description |
|------|-------------|
| `--once` | Run a single scan pass and exit (useful for cron or testing) |
| `--nuke` | Delete the local registry database and exit |

## Configuration (via JSONAir)

```json
{
  "core": {
    "max_size": 104857600,
    "sleep_interval": 86400
  },
  "operating_systems": {
    "unix": {
      "directories": ["/home", "/var/data"],
      "mime_types": ["application/pdf", "image/jpeg"],
      "exclude": ["/proc", "/sys", ".git"]
    },
    "macos": {
      "directories": ["/Users"],
      "mime_types": ["application/pdf", "image/jpeg"],
      "exclude": [".Trash", "Library/Caches"]
    },
    "windows": {
      "directories": ["C:\\Users"],
      "mime_types": ["application/pdf", "image/jpeg"],
      "exclude": ["AppData\\Local\\Temp"]
    }
  },
  "syslog": {
    "host": "local",
    "proto": "tcp"
  },
  "highvolt": {
    "url": "https://highvolt.internal:8443",
    "pat": "your-highvolt-pat"
  }
}
```

The client derives the `/api/v1/highvolt/query` and `/api/v1/highvolt/submit` endpoints from `highvolt.url`; they do not need to be configured separately.

Notes on the fields above:

- `core.max_size` (bytes) is required. Files larger than this are skipped.
- `core.sleep_interval` (seconds) is optional and defaults to 86400. It only applies when Voltage runs as a persistent service on macOS/Linux/Unix.
- `operating_systems.<os>.mime_types` is required for the OS Voltage is running on; Voltage exits if it is empty. (Older configs used `mimetypes`; that key is no longer read and must be renamed to `mime_types`.)
- `syslog.proto` is required unless `syslog.host` is `local`.

## MIME detection

Voltage uses **magic byte detection** (not file extension) to classify files. The `util.GetFileMagic` function reads the file header to determine its true MIME type. This prevents attackers from disguising files with misleading extensions.
