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
It also asks for your provider type and URL, queries the provider for available models, optionally asks for tool API keys, writes `~/.config/weazlchat/config.json`, and starts the TUI.

Provider URLs should be base URLs:

- vLLM: `https://host:port` or `https://host`, without `/v1`
- Ollama: `http://host:11434`, without `/api`

The installer normalizes accidental `/v1` and `/api` suffixes before querying and saving.
Tool API keys are optional. During setup, blank keeps an existing saved key and `-` clears it.

The first run asks you to create a local history password. Session history and workspace saves are stored in SQLite with a bcrypt-protected vault and AES-GCM encrypted payloads.

## Keys

- `enter`: send message or select session
- `up` / `down`: recall previous prompts in the current session
- `pgup` / `pgdown`: scroll chat history
- `home` / `end`: jump to top or bottom of chat history
- `ctrl+n`: new session
- `ctrl+r`: resume from session history
- `ctrl+d`: delete selected session from session history
- `ctrl+s`: save current workspace view
- `ctrl+w`: list workspace saves
- `ctrl+t`: trim context into a summary checkpoint
- `esc`: back to chat
- `ctrl+c`: quit

## Context Trimming

The status line includes a Bubble Charm progress bar showing estimated context usage for the active provider. Configure the provider's `context_window` in `~/.config/weazlchat/config.json`; it defaults to `32768` tokens.

Press `ctrl+t` to ask the active model to summarize the current conversation into a compact checkpoint. The summary target scales with the configured context window, bounded between 500 and 2000 tokens. Future requests send that checkpoint summary plus only newer messages, instead of replaying the entire session from the beginning.

WeazlChat automatically trims context when the estimate reaches 97% of the configured context window. The current user prompt stays outside the checkpoint and is sent normally after the trim finishes.

## Paste And Copy

Large pasted blocks are stored as the full prompt payload but shown compactly in the input bar as `[PASTED n lines]`.

The TUI does not capture the mouse, so normal terminal highlight/copy works in the chat viewport. Use the keyboard scroll keys above when reviewing longer conversations.

## Tool Support

WeazlChat supports function calling (tools) that allow the AI model to interact with external services and perform calculations. Tools are executed automatically for safe operations (read-only) and can be configured in your config file.

### Enabling Tools

The installer can write this section for you. To edit it manually, update `~/.config/weazlchat/config.json`:

```json
{
  "tools": {
    "enabled": true,
    "auto_execute_safe": true,
    "alpha_vantage_api_key": "YOUR_ALPHA_VANTAGE_API_KEY_HERE",
    "brave_api_key": "YOUR_BRAVE_API_KEY_HERE",
    "workspace_roots": ["/home/user/Code", "/home/user/Notes"],
    "max_output_chars": 12000,
    "max_file_bytes": 1048576
  }
}
```

**Configuration Options:**

- `enabled`: Set to `true` to enable tool support (default: `false`)
- `auto_execute_safe`: Automatically execute safe (read-only) tools without confirmation (default: `true`)
- `alpha_vantage_api_key`: API key for stock price lookups (optional, get free key at https://www.alphavantage.co/support/#api-key)
- `brave_api_key`: API key for Brave web search lookups (optional)
- `workspace_roots`: Directories that file, shell, and SQLite tools may read from
- `max_output_chars`: Maximum characters returned by tools before truncation
- `max_file_bytes`: Maximum file size for file search/read tools

### Available Tools

#### Calculator
Performs basic mathematical operations. Always available when tools are enabled.

**Operations:** add, subtract, multiply, divide, power, sqrt, percentage

**Example prompts:**
- "What is 15% of 250?"
- "Calculate the square root of 144"
- "What's 2 to the power of 10?"

#### Current Time
Returns the current date and time for the local machine or a requested IANA timezone. Always available when tools are enabled.

**Example prompts:**
- "What time is it?"
- "What is the current date in UTC?"
- "What time is it in America/New_York?"

#### Weather
Fetches current weather and short forecasts with Open-Meteo. Always available when tools are enabled and does not require an API key.

**Example prompts:**
- "What's the weather in Philadelphia?"
- "Get a 3 day forecast for Boston, MA"
- "What's the weather in Berlin in celsius?"

#### Stock Price
Fetches current stock prices and market data. Requires Alpha Vantage API key.

**Example prompts:**
- "What's the current price of IBM stock?"
- "Get me the latest stock info for AAPL"
- "How is Microsoft (MSFT) doing today?"

#### Web Search
Searches the web with Brave Search and returns top result titles, URLs, snippets, and dates when available. Requires Brave API key.

**Example prompts:**
- "Search the web for the latest Go release notes"
- "Find current information about Bubble Tea tool calling examples"
- "Look up recent news about local AI models"

#### Fetch URL
Fetches an HTTP or HTTPS URL and returns readable text. Private and local network addresses are rejected.

**Example prompts:**
- "Fetch this documentation page and summarize it: https://example.com/docs"
- "Read the first article from those search results"

#### Local Files
Read-only file tools work only under configured `workspace_roots`.

**Tools:** `list_files`, `search_files`, `read_file`, `create_file`

**Example prompts:**
- "Search my notes for MCP"
- "Read README.md in this repo"
- "List Go files under the TUI package"
- "Write that answer to answer.md"

`create_file` only creates new text files under `workspace_roots`; it refuses to overwrite existing files.

#### Read-Only Command
Runs allowlisted read-only commands under configured `workspace_roots`. Commands are passed as command plus args, not shell strings.

**Allowlisted examples:** `pwd`, `ls`, `find`, `rg`, `cat`, `git status`, `git diff`, `git log`, `git show`, `go test`, `npm test`

#### SQLite Query
Runs read-only SQLite queries against database files under configured `workspace_roots`.

**Allowed SQL:** `SELECT`, `WITH`, `EXPLAIN`, and `PRAGMA table_info`

#### Local Memory
Stores explicit encrypted local memories in WeazlChat's database.

**Tools:** `remember`, `recall`, `list_memories`, `forget`

### How It Works

1. When you ask a question that requires a tool, the AI model will automatically call the appropriate tool
2. Tool payloads stay hidden from the chat transcript
3. The tool result is fed back to the model, which then provides a natural language response
4. All tool calls and results are encrypted and stored in your local database

### Model Requirements

**vLLM:** Your model must support function calling (e.g., models fine-tuned for tool use)

**Ollama:** Use models with native tool support like:
- `llama3.1` (8B, 70B, 405B)
- `mistral-nemo`
- `qwen2.5`

### Security

- **Safe tools** execute automatically and are designed to be read-only, create-only text file writes, or explicitly local memory operations
- File, shell, and SQLite tools are confined to configured `workspace_roots`
- `create_file` refuses overwrites
- Shell commands are allowlisted and do not execute through a shell
- URL fetch rejects private and local network addresses
- Tool output is truncated before it is sent back to the model
- Tool execution happens locally in the WeazlChat process
- API keys are stored in your local config file (not shared)
- Chat history, tool interactions, and memories are encrypted in your local database

### Example Config

See `config.example.json` for a complete configuration example with tools enabled.
