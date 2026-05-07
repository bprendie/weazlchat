package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	gossh "golang.org/x/crypto/ssh"

	"github.com/bprendie/weazlchat/internal/app"
	"github.com/bprendie/weazlchat/internal/config"
	"github.com/bprendie/weazlchat/internal/tui"
)

func main() {
	lipgloss.SetHasDarkBackground(true)

	cfg, cfgPath, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	if err := ensureAuthorizedKeys(cfg.SSH.AuthorizedKeysPath); err != nil {
		fmt.Fprintf(os.Stderr, "authorized keys: %v\n", err)
		os.Exit(1)
	}

	server, err := wish.NewServer(
		wish.WithAddress(cfg.SSH.Listen),
		wish.WithHostKeyPath(cfg.SSH.HostKeyPath),
		wish.WithAuthorizedKeys(cfg.SSH.AuthorizedKeysPath),
		wish.WithMiddleware(
			bubbletea.Middleware(sessionHandler(cfg, cfgPath)),
			logging.Middleware(),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ssh server: %v\n", err)
		os.Exit(1)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-done
		_ = server.Close()
	}()

	log.Info("starting weazlchat ssh", "listen", cfg.SSH.Listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "ssh server: %v\n", err)
		os.Exit(1)
	}
}

func sessionHandler(cfg config.Config, cfgPath string) bubbletea.Handler {
	return func(sess ssh.Session) (tea.Model, []tea.ProgramOption) {
		key := sess.PublicKey()
		if key == nil {
			wish.Fatalln(sess, "public key authentication is required")
			return nil, nil
		}
		dbPath := userDatabasePath(cfg.SSH.UserDataDir, key)
		rt, err := app.NewRuntime(cfg, cfgPath, dbPath)
		if err != nil {
			wish.Fatalln(sess, "failed to open user database: "+err.Error())
			return nil, nil
		}
		go func() {
			<-sess.Context().Done()
			_ = rt.Store.Close()
		}()
		opts := []tea.ProgramOption{tea.WithAltScreen(), tea.WithMouseCellMotion()}
		return tui.New(rt.Config, rt.ConfigPath, rt.Store, rt.ToolRegistry, tui.WithInitialKeyIgnore(2*time.Second)), opts
	}
}

func userDatabasePath(root string, key ssh.PublicKey) string {
	fingerprint := gossh.FingerprintSHA256(key)
	userID := sanitizeFingerprint(fingerprint)
	return filepath.Join(root, userID, "weazlchat.sqlite3")
}

var unsafePathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeFingerprint(fingerprint string) string {
	fingerprint = strings.TrimSpace(fingerprint)
	fingerprint = strings.TrimPrefix(fingerprint, "SHA256:")
	fingerprint = unsafePathChars.ReplaceAllString(fingerprint, "_")
	fingerprint = strings.Trim(fingerprint, "._-")
	if fingerprint == "" {
		return "unknown"
	}
	return fingerprint
}

func ensureAuthorizedKeys(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_ = f.Close()
	return fmt.Errorf("%s was created but is empty; add allowed SSH public keys before starting weazlchat-ssh", path)
}
