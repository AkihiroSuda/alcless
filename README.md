[[🍺**Homebrew**]](#homebrew)
[[✦**Gemini**]](#gemini)
[[֎**Codex**]](#codex)
[[✴️**Claude Code**]](#claude-code)

# Alcoholless: lightweight security sandbox for Homebrew, AI agents, etc.

Alcoholless is a lightweight security sandbox for macOS programs.

While Alcoholless was originally made for the sake of securing Homebrew, basically it can be used for almost any CLI programs on macOS.
Notably, Alcoholless is useful for allowing an AI agent to run shell commands with less risk of [breaking the host operating system](https://old.reddit.com/r/ClaudeAI/comments/1pgxckk/claude_cli_deleted_my_entire_home_directory_wiped/).

See also my blog article: <https://medium.com/nttlabs/alcoholless-lightweight-security-sandbox-for-macos-ccf0d1927301>

## Examples
### Homebrew
Alcoholless Homebrew (`alcless brew`) executes Homebrew in a separate environment
so as to reduce concerns around potential supply chain attacks.

```bash
cd ~/SOME_DIRECTORY
alcless brew install xz
alcless xz SOME_FILE
```

In the example above, `xz` works as a separate user with an access for the copy of the current directory.
The changed files are synced back to the current directory when the command exits, with a confirmation screen (see below).
Other directories are inaccessible, as long as the permissions are set correctly.

<details><summary>Confirmation screen</summary>

```console
$ alcless xz SOME_FILE
0:00AM INF ➡️Syncing the files src=/Users/USER/SOME_DIRECTORY/ dst=default:/Users/alcless_USER_default/Users/USER/SOME_DIRECTORY
0:00AM INF ⬅️Syncing the files back (dry run) src=default:/Users/alcless_USER_default/Users/USER/SOME_DIRECTORY/ dst=/Users/USER/SOME_DIRECTORY
*deleting SOME_FILE
.d..t.... ./
>f+++++++ SOME_FILE.xz
0:00AM INF ⬅️Syncing the files back src=default:/Users/alcless_USER_default/Users/user/tmp/ dst=/Users/USER/tmp
⚠️  The following commands will be executed:
rsync -rai --delete -e '/usr/local/bin/alclessctl shell --workdir=/ --plain' default:/Users/alcless_USER_default/Users/USER/SOME_DIRECTORY/ /Users/USER/SOME_DIRECTORY
❓ Press return to continue, or Ctrl-C to abort
[RETURN]
CONTINUE
*deleting SOME_FILE
.d..t.... ./
>f+++++++ SOME_FILE.xz
```

</details>

> [!IMPORTANT]
>
> Alcoholless uses an [unsupported installation mode](https://docs.brew.sh/Installation#untar-anywhere-unsupported) of Homebrew
> that uses a custom installation prefix.
>
> **Do NOT report any issue that happens with Alcoholless to the upstream Homebrew.**

### AI agents

Alcoholless is useful for sandboxing AI coding agents too.
The file changes made by the AI are committed to the host filesystem only after the user confirmation.

#### Gemini

```bash
cd ~/SOME_DIRECTORY
alcless brew install gemini-cli
alcless gemini
```

#### Codex

```bash
cd ~/SOME_DIRECTORY
alcless brew install codex
alcless codex
```

#### Claude Code

```bash
cd ~/SOME_DIRECTORY
alcless brew install claude-code
alcless claude
```

<details>
<summary>Troubleshooting</summary>
<p>

Claude Code may need extra steps for the initial setup
(issue [#52](https://github.com/AkihiroSuda/alcless/issues/52)):

- Switch the desktop user to `alcless_USER_default` via the Fast User Switching icon in the macOS menubar.
- Run the following commands in the `alcless_USER_default` desktop:
```bash
brew install claude-code
claude
```
- Authenticate with Anthropic in the initial screen of `claude`.
- Log out from the `alcless_USER_default` desktop.
- Run the following commands in the main desktop:
```bash
cd ~/SOME_DIRECTORY
alcless zsh -c "security unlock-keychain && claude"
```

</p>
</details>

> [!TIP]
>
> AI coding agents typically prints the authentication URL on the first run.
> Make sure to copy and paste the URL in a single line.
> (Hint: use TextEdit to eliminate extra line delimiters)

## Install

Requirements:
- macOS
- [Go](https://go.dev)

To install Alcoholless, run:

```bash
make
sudo make install
```

Makefile variables:
- `PREFIX`: installation prefix (default: `/usr/local`)

## Usage

To initialize the "default" sandbox (user account `alcless_${USER}_default`):
```
alclessctl create default
```

To run a command:
```
alclessctl shell default -- brew install xz
```
or
```
alcless brew install xz
```

To run a command, without rsyncing the current directory:
```
alclessctl shell --plain default bash
```
or
```
alcless --plain bash
```

To remove the sandbox:
```
alclessctl delete default
```

The command line is designed to be similar to [`limactl`](https://lima-vm.io/docs/usage/).

## How it works
Just plain old utilities under the hood: `sudo`, `su`, `pam_launchd`, and `rsync`.

A future version may also incorporate [FSKit](https://developer.apple.com/documentation/fskit/) to replace `rsync`.

### Security notice
Alcoholless creates `/etc/sudoers.d/alcless_exampleuser_default` for the user `exampleuser`, with the following content:
```
exampleuser ALL=(root) NOPASSWD: /usr/bin/su - alcless_exampleuser_default -c *
```

This `sudo` configuration allows `exampleuser` to run `/usr/bin/su - alcless_exampleuser_default -c *` as the `root` user,
without the password.

The `su` command being executed through `sudo` can run an arbitrary command as the sandbox user `alcless_exampleuser_default`.

See [FAQs](#faqs) for the reason why `su` is wrapped inside `sudo`.

- - -

## Advanced information

### FAQs
#### Why wrap `su` inside `sudo`?
Because `sudo` doesn't isolate "a specific Mach bootstrap subset, audit session and other characteristics not recognized by POSIX" (see `launchd(8)`),
while `su` isolates them.

e.g., `sudo -u alcless_exampleuser_default open -a TextEdit` opens the `TextEdit` application as the current user, not as `alcless_exampleuser_default`.

This issue could be solved by copying the `pam_launchd.so` configuration from `/etc/pam.d/su` to `/etc/pam.d/sudo`,
however, touching such system configuration files might be scary.

So, the current workaround is to just wrap `su` inside `sudo`.

#### Why not use VM?
Because VM has several disadvantages:
- Non-negligible performance overhead
- High disk consumption
- No direct access to the host hardware (GPU, etc.)
- Localhost address inaccessible from the host
- Does not work on GitHub Actions etc. due to lack of the support for nested virtualization

#### Why not support Linux and FreeBSD?
Because Linux and FreeBSD already have containers.

#### How does Alcoholless relate to Lima?
- Alcoholless (**Lightweight**): run commands as a separate macOS user (not a VM, nor a container)
- [Lima](https://lima-vm.io/) (**Strong security**): run commands in a VM
  ([Linux](https://lima-vm.io/docs/usage/guests/linux/), [macOS](https://lima-vm.io/docs/usage/guests/macos/), etc.)

The `alclessctl` CLI is designed to mimic the `limactl` CLI for an easier learning,
however, Alcoholless does not use Lima under the hood.
