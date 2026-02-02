# Alcless (Bun/TypeScript Version)

Isolated macOS user sessions with pre-baked tools. Fast, focused, and one-shot.

## Features

1.  **Bulk Management**: Create or delete multiple sessions in a single command.
2.  **No Password**: Sessions are configured for instant access from the host user via `sudo su -`.
3.  **Pre-baked Tools**: Every session comes with:
    *   GitHub CLI (`gh`)
    *   `jq`
    *   Python 3.12 (via UV)
    *   UV
    *   Bun
4.  **Speed**: Parallel provisioning with a focused progress bar.

## Installation

Ensure you have [Bun](https://bun.sh) installed.

```bash
git clone https://github.com/AkihiroSuda/alcless.git
cd alcless
bun install
```

## Usage

### Create Sessions
Create one or more sessions.

```bash
./bin/alcless create session1 session2
# OR using the bun alias
bun containers session1 session2
```

### List Sessions
List all currently active sessions.

```bash
./bin/alcless list
```

### Run Commands / Shell
Execute a command in a session or open an interactive shell.

```bash
# Open an interactive shell
./bin/alcless exec session1

# Run a specific command
./bin/alcless exec session1 "uv --version"
```

### Delete Sessions
Remove sessions and their associated sudoers configuration.

```bash
./bin/alcless delete session1 session2
```

## Security & Permissions

### Suppressing the "Administration" Popup
On modern macOS, `sysadminctl` triggers a GUI prompt. To run `alcless` silently:
1.  Open **System Settings**.
2.  Go to **Privacy & Security** > **Full Disk Access**.
3.  Add and toggle your Terminal (e.g., **Ghostty**, **Terminal.app**, or **iTerm2**) to **ON**.
4.  Restart your terminal.

### Access Control
Sessions follow the naming convention `alcl_${USER}_${INSTANCE}`. They are configured with `NOPASSWD` sudoers fragments for the host user, allowing seamless transitions while maintaining system-level isolation.

## Development

### Run Tests
```bash
bun test
```

### Typecheck
```bash
bun run typecheck
```

## License
MIT
