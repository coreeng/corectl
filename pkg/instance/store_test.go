package instance

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreManagesBuiltInCustomCurrentAndGenerationBindings(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "platform.json")}

	selected, err := store.Resolve("")
	require.NoError(t, err)
	require.Equal(t, Instance{Name: BuiltInName, Origin: BuiltInOrigin}, selected)

	require.NoError(t, store.Add("local", "HTTP://LOCALHOST:8080/"))
	require.NoError(t, store.Use("local"))
	selected, err = store.Resolve("")
	require.NoError(t, err)
	require.Equal(t, "http://localhost:8080", selected.Origin)

	require.NoError(t, store.SetBinding(selected.Origin, "cluster-1", Binding{Generation: "2", ManagedContext: "managed"}))
	binding, ok, err := store.Binding(selected.Origin, "cluster-1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "2", binding.Generation)

	require.NoError(t, store.Remove("local"))
	selected, err = store.Resolve("")
	require.NoError(t, err)
	require.Equal(t, BuiltInName, selected.Name)
	require.Error(t, store.Remove(BuiltInName))
}

func TestNormalizeOriginRejectsUnsafeURLs(t *testing.T) {
	normalized, err := NormalizeOrigin("HTTPS://PORTAL.EXAMPLE.COM:443/")
	require.NoError(t, err)
	require.Equal(t, "https://portal.example.com", normalized)

	for _, raw := range []string{"http://example.com", "https://example.com/path", "https://user@example.com", "not-a-url"} {
		_, err := NormalizeOrigin(raw)
		require.Error(t, err, raw)
	}
}
