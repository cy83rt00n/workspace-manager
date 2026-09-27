package system

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InvalidKeyName is the diagnostic returned when the key name cleans to empty.
const InvalidKeyName = "Invalid key name"

// GenerateKeypair generates an ED25519 SSH keypair at <dir>/<clean>.key and
// <dir>/<clean>.pub, returning their paths. clean is derived from name by
// stripping the directory, the final extension, then one trailing ".key"/".pub".
// An empty clean name errors. An existing private key errors. Any non-zero exit
// of ssh-keygen or mv is now propagated (ADR-001 D6).
func (s *System) GenerateKeypair(ctx context.Context, name, dir string) (privatePath, publicPath string, err error) {
	clean := filepath.Base(name)
	clean = strings.TrimSuffix(clean, filepath.Ext(clean))
	if strings.HasSuffix(clean, ".key") {
		clean = strings.TrimSuffix(clean, ".key")
	} else if strings.HasSuffix(clean, ".pub") {
		clean = strings.TrimSuffix(clean, ".pub")
	}
	if clean == "" {
		return "", "", errors.New(InvalidKeyName)
	}

	private := filepath.Join(dir, clean+".key")
	public := filepath.Join(dir, clean+".pub")

	if _, statErr := os.Stat(private); statErr == nil {
		return "", "", fmt.Errorf("%s already exists", private)
	}

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	if _, _, err := s.Run(ctx, "ssh-keygen", "-t", "ed25519", "-f", private, "-C", "wsm-"+clean, "-N", ""); err != nil {
		return "", "", err
	}
	if _, _, err := s.Run(ctx, "mv", private+".pub", public); err != nil {
		return "", "", err
	}
	return private, public, nil
}
