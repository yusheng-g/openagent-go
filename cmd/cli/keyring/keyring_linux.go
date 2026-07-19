//go:build linux

package keyring

import (
	"errors"
	"fmt"

	dbus "github.com/godbus/dbus/v5"
	gkr "github.com/zalando/go-keyring"
	"golang.org/x/sys/unix"
)

const (
	dbusServiceName        = "org.freedesktop.DBus"
	dbusObjectPath         = "/org/freedesktop/DBus"
	dbusNameHasOwnerMethod = "org.freedesktop.DBus.NameHasOwner"
	secretServiceName      = "org.freedesktop.secrets"

	// Linux kernel keyring parameters.
	keyringKeyType   = "user"
	keyringKeyPrefix = "openagent:"
)

// openBackend selects a Linux keyring backend. It prefers Secret Service
// (D-Bus), falling back to the kernel user keyring (KeyCtl) when no
// session bus / Secret Service is available. Returns ErrKeyringUnavailable
// when neither backend can be initialized.
func openBackend() (backend, error) {
	if isSecretServiceAvailable() {
		return secretServiceBackend{}, nil
	}
	if err := ensureKeyringLinked(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyringUnavailable, err)
	}
	return keyctlBackend{}, nil
}

// isSecretServiceAvailable checks whether the Secret Service D-Bus
// service is registered on the session bus. Unlike zalando/go-keyring's
// implicit probe (which triggers `dbus-launch` autolaunch via godbus),
// this explicitly calls dbus.SessionBus() — on recent godbus versions
// SessionBus returns an error in headless environments instead of
// invoking dbus-launch.
func isSecretServiceAvailable() bool {
	conn, err := dbus.SessionBus()
	if err != nil {
		return false
	}
	defer conn.Close()
	obj := conn.Object(dbusServiceName, dbusObjectPath)
	var hasOwner bool
	if err := obj.Call(dbusNameHasOwnerMethod, 0, secretServiceName).Store(&hasOwner); err != nil {
		return false
	}
	return hasOwner
}

// ensureKeyringLinked links the user keyring to the session keyring so
// that keys added by this process are visible to future sibling processes
// in the same session. Mirrors `ensureKeyringLinked()` in
// ~/projects/hdspace-models/credential/credential_linux.go. Returns an
// error when the kernel keyring is not usable (e.g. containers lacking
// the keyutils capability).
func ensureKeyringLinked() error {
	if _, err := unix.KeyctlInt(unix.KEYCTL_LINK,
		unix.KEY_SPEC_USER_KEYRING,
		unix.KEY_SPEC_SESSION_KEYRING, 0, 0); err != nil {
		return fmt.Errorf("keyctl link user keyring: %w", err)
	}
	// Probe the user keyring id (creates if missing). If this fails the
	// kernel doesn't support keyrings for this process and we should NOT
	// silently fall back to MemStore.
	if _, err := unix.KeyctlGetKeyringID(unix.KEY_SPEC_USER_KEYRING, true); err != nil {
		return fmt.Errorf("keyctl get user keyring id: %w", err)
	}
	return nil
}

// keyctlBackend stores base64-encoded secrets in the Linux kernel
// keyring under type "user" with descriptions "openagent:<service>:<key>".
type keyctlBackend struct{}

func keyctlDesc(service, key string) string {
	return keyringKeyPrefix + service + ":" + key
}

func (keyctlBackend) Get(service, key string) (string, error) {
	id, err := unix.KeyctlSearch(unix.KEY_SPEC_USER_KEYRING,
		keyringKeyType, keyctlDesc(service, key), 0)
	if err != nil {
		if errors.Is(err, unix.ENOKEY) || errors.Is(err, unix.ENOENT) {
			return "", gkr.ErrNotFound
		}
		return "", fmt.Errorf("keyctl search: %w", err)
	}
	// First call with nil buffer returns the required size.
	n, err := unix.KeyctlBuffer(unix.KEYCTL_READ, id, nil, 0)
	if err != nil {
		return "", fmt.Errorf("keyctl read (probe): %w", err)
	}
	buf := make([]byte, n)
	if _, err := unix.KeyctlBuffer(unix.KEYCTL_READ, id, buf, 0); err != nil {
		return "", fmt.Errorf("keyctl read: %w", err)
	}
	return b64Decode(string(buf))
}

func (keyctlBackend) Set(service, key, value string) error {
	desc := keyctlDesc(service, key)
	// Revoke an existing key with the same description to avoid duplicates.
	// AddKey would update an in-place key, but revoking first is simpler
	// and consistent across kernel versions.
	if id, err := unix.KeyctlSearch(unix.KEY_SPEC_USER_KEYRING,
		keyringKeyType, desc, 0); err == nil {
		_, _ = unix.KeyctlInt(unix.KEYCTL_REVOKE, id, 0, 0, 0)
	}
	if _, err := unix.AddKey(keyringKeyType, desc,
		[]byte(b64Encode(value)), unix.KEY_SPEC_USER_KEYRING); err != nil {
		return fmt.Errorf("keyctl add: %w", err)
	}
	return nil
}

func (keyctlBackend) Delete(service, key string) error {
	id, err := unix.KeyctlSearch(unix.KEY_SPEC_USER_KEYRING,
		keyringKeyType, keyctlDesc(service, key), 0)
	if err != nil {
		if errors.Is(err, unix.ENOKEY) || errors.Is(err, unix.ENOENT) {
			return gkr.ErrNotFound
		}
		return fmt.Errorf("keyctl search: %w", err)
	}
	if _, err := unix.KeyctlInt(unix.KEYCTL_UNLINK, id,
		unix.KEY_SPEC_USER_KEYRING, 0, 0); err != nil {
		return fmt.Errorf("keyctl unlink: %w", err)
	}
	return nil
}