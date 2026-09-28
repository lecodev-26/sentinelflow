package secretsv4

import (
	"context"
	"fmt"
	"os"
)

type Store interface {
	Get(context.Context, string) (string, error)
}
type EnvStore struct{ Prefix string }

func (s EnvStore) Get(_ context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("secret name required")
	}
	key := name
	if s.Prefix != "" {
		key = s.Prefix + name
	}
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("secret %q unavailable", name)
	}
	return v, nil
}
