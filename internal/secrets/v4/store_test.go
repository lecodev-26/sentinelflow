package secretsv4

import (
	"context"
	"os"
	"testing"
)

func TestEnvStore(t *testing.T) {
	os.Setenv("SF_SECRET_X", "value")
	defer os.Unsetenv("SF_SECRET_X")
	v, e := EnvStore{Prefix: "SF_SECRET_"}.Get(context.Background(), "X")
	if e != nil || v != "value" {
		t.Fatal(v, e)
	}
}
