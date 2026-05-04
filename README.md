# WeazlChat

WeazlChat is a private local-first AI chat TUI for vLLM and Ollama servers.

## Defaults

On first launch, WeazlChat writes `~/.config/weazlchat/config.json` with local defaults:

- `local-vllm`: `http://localhost:8000`
- model: `local-model`
- `local-ollama`: `http://localhost:11434`

The endpoint and model are read from config at runtime, not hardcoded into the chat client.

## Run

```sh
go run ./cmd/weazlchat
```

## Install

```sh
./scripts/install.sh
```

The installer builds `weazlchat`, places it in `~/.weazlchat/bin`, and adds that directory to your shell `PATH` when it is not already present.
It also asks for your provider type and URL, queries the provider for available models, writes `~/.config/weazlchat/config.json`, and starts the TUI.

Provider URLs should be base URLs:

- vLLM: `https://host:port` or `https://host`, without `/v1`
- Ollama: `http://host:11434`, without `/api`

The installer normalizes accidental `/v1` and `/api` suffixes before querying and saving.

The first run asks you to create a local history password. Session history and workspace saves are stored in SQLite with a bcrypt-protected vault and AES-GCM encrypted payloads.

## Keys

- `enter`: send message or select session
- `ctrl+n`: new session
- `ctrl+r`: resume from session history
- `ctrl+d`: delete selected session from session history
- `ctrl+s`: save current workspace view
- `ctrl+w`: list workspace saves
- `esc`: back to chat
- `ctrl+c`: quit
