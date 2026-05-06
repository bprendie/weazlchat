#!/usr/bin/env bash
set -euo pipefail

APP_NAME="weazlchat"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_ROOT="${WEAZLCHAT_HOME:-"$HOME/.weazlchat"}"
BIN_DIR="$INSTALL_ROOT/bin"
BIN_PATH="$BIN_DIR/$APP_NAME"
GO_CACHE="${GOCACHE:-"$REPO_ROOT/.gocache"}"
GO_MOD_CACHE="${GOMODCACHE:-"$REPO_ROOT/.gomodcache"}"

mkdir -p "$BIN_DIR" "$GO_CACHE" "$GO_MOD_CACHE"

echo "Building $APP_NAME..."
(
  cd "$REPO_ROOT"
  GOCACHE="$GO_CACHE" GOMODCACHE="$GO_MOD_CACHE" go build -buildvcs=false -o "$BIN_PATH" ./cmd/weazlchat
)

chmod 0755 "$BIN_PATH"

path_line='export PATH="$HOME/.weazlchat/bin:$PATH"'
marker_begin="# >>> weazlchat path >>>"
marker_end="# <<< weazlchat path <<<"

choose_profile() {
  local shell_name
  shell_name="$(basename "${SHELL:-}")"
  case "$shell_name" in
    zsh) echo "$HOME/.zshrc" ;;
    bash)
      if [[ -f "$HOME/.bashrc" ]]; then
        echo "$HOME/.bashrc"
      else
        echo "$HOME/.profile"
      fi
      ;;
    fish) echo "" ;;
    *) echo "$HOME/.profile" ;;
  esac
}

if [[ ":$PATH:" != *":$BIN_DIR:"* ]]; then
  profile="$(choose_profile)"
  if [[ -n "$profile" ]]; then
    touch "$profile"
    if ! grep -Fq "$marker_begin" "$profile"; then
      {
        echo ""
        echo "$marker_begin"
        echo "$path_line"
        echo "$marker_end"
      } >> "$profile"
      echo "Added $BIN_DIR to PATH in $profile"
    else
      echo "PATH block already exists in $profile"
    fi
  else
    echo "Fish shell detected. Add this to your fish config:"
    echo "set -gx PATH $BIN_DIR \$PATH"
  fi
fi

echo "Installed $APP_NAME to $BIN_PATH"
echo "If your shell cannot find it yet, restart the shell or run:"
echo "  $path_line"

echo ""
echo "Configuring provider and optional tools..."
(
  cd "$REPO_ROOT"
  GOCACHE="$GO_CACHE" GOMODCACHE="$GO_MOD_CACHE" go run -buildvcs=false ./cmd/weazlchat-setup
)

if [[ "${WEAZLCHAT_SKIP_LAUNCH:-}" == "1" ]]; then
  echo "Skipping first launch because WEAZLCHAT_SKIP_LAUNCH=1"
else
  echo ""
  echo "Launching $APP_NAME..."
  exec "$BIN_PATH"
fi
