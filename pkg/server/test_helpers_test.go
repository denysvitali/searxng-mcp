package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func loadJSONFixture(t *testing.T, fileName string) interface{} {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", fileName)
	payload, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded interface{}
	err = json.Unmarshal(payload, &decoded)
	require.NoError(t, err)

	return decoded
}
