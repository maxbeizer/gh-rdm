# gh-rdm — Remote Development Manager

A [`gh`](https://cli.github.com/) CLI extension that forwards clipboard (copy/paste), `open` commands, and **screenshot images** from remote SSH sessions back to your local machine.

Stolen from [BlakeWilliams/remote-development-manager](https://github.com/BlakeWilliams/remote-development-manager) and repackaged as a `gh` extension.

## How it works

1. A **server** runs on your local machine, listening on a unix socket.
2. The socket is forwarded to remote machines via the SSH `-R` flag.
3. **Client** commands on the remote side send copy/paste/open requests back through the tunnel.

## Installation

```bash
gh extension install maxbeizer/gh-rdm
```

## Quick start

Run the interactive setup wizard to get everything configured:

```bash
gh rdm setup
```

This walks you through starting the server, configuring SSH forwarding in `~/.ssh/config`, and setting up integrations (neovim, gh CLI browser, shell aliases).

### Or do it manually

Start the server and SSH into a remote host with clipboard forwarding in one shot:

```bash
gh rdm server & ssh -R 127.0.0.1:7391:$(gh rdm socket) user@remote-host
```

For GitHub Codespaces:

```bash
gh rdm server & gh cs ssh -- -R 127.0.0.1:7391:$(gh rdm socket)
```

## Usage

### Server (local machine)

```bash
# Start the server
gh rdm server

# Get socket path (useful for SSH config)
gh rdm socket

# Stop the server
gh rdm stop
```

### SSH with forwarding

Forward the local socket to the remote host so client commands can reach it:

```bash
ssh -R 127.0.0.1:7391:$(gh rdm socket) user@remote-host
```

### Client (remote machine)

```bash
# Copy to local clipboard
echo "hello" | gh rdm copy

# Paste from local clipboard
gh rdm paste

# Open URL in local browser
gh rdm open https://github.com

# Fetch latest screenshot from local Desktop
gh rdm screenshot

# Fetch clipboard image (use ⌘⇧⌃4 to screenshot to clipboard)
gh rdm clipboard-image

# Save screenshot to a custom directory
gh rdm screenshot -o ~/images
```

## Integrations

### Screenshots & Copilot CLI over SSH

When SSH'd into a Codespace, macOS screenshots reference local paths that the remote can't access.
Use `gh rdm screenshot` to pull images through the SSH tunnel:

```bash
# 1. Take a screenshot on your Mac (⌘⇧4)
# 2. In the Codespace terminal:
gh rdm screenshot
# Output:
#   📸 Saved: /tmp/screenshot-20260306-120000.png (245820 bytes)
#   @/tmp/screenshot-20260306-120000.png

# 3. Paste the @ reference into Copilot CLI
```

Or capture directly to clipboard and pull:

```bash
# 1. Screenshot to clipboard (⌘⇧⌃4)
# 2. In the Codespace:
gh rdm clipboard-image
```

### Tmux

Add to your shell profile so `pbcopy` works inside tmux over SSH:

```bash
alias pbcopy="gh rdm copy"
```

### Neovim

Configure the clipboard provider in your Neovim config:

```lua
vim.g.clipboard = {
  name = "gh-rdm",
  copy = {
    ["+"] = "gh rdm copy",
    ["*"] = "gh rdm copy",
  },
  paste = {
    ["+"] = "gh rdm paste",
    ["*"] = "gh rdm paste",
  },
  cache_enabled = true,
}
```

### GitHub CLI

Use `gh rdm open` as the browser for the GitHub CLI:

```bash
gh config set browser "gh rdm open"
```

### Zsh

Add to `~/.zshenv` so `open` works transparently on the remote:

```bash
alias open="gh rdm open"
```

## Development

```bash
make help          # see all targets
make build         # build binary
make test          # run tests
make ci            # build + vet + test-race
make install-local # install extension from checkout
```
