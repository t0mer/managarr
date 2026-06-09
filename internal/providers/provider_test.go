// internal/providers/provider_test.go
package providers_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/galactica/internal/providers"
)

type fakeProvider struct{}

func (f *fakeProvider) Kind() providers.Kind { return providers.Kind("fake") }
func (f *fakeProvider) TestConnection(_ context.Context, _ providers.Instance) error { return nil }
func (f *fakeProvider) FetchLogs(_ context.Context, _ providers.Instance, _ time.Time) ([]providers.LogEntry, error) {
	return []providers.LogEntry{{Message: "test"}}, nil
}
func (f *fakeProvider) StreamLogs(_ context.Context, _ providers.Instance) (<-chan providers.LogEntry, error) {
	ch := make(chan providers.LogEntry)
	close(ch)
	return ch, nil
}
func (f *fakeProvider) Collect(_ context.Context, _ providers.Instance) ([]providers.Sample, error) {
	return []providers.Sample{{Metric: "test", Value: 1}}, nil
}

func TestRegisterAndGet(t *testing.T) {
	fp := &fakeProvider{}
	providers.Register(fp)

	got, err := providers.Get(providers.Kind("fake"))
	require.NoError(t, err)
	assert.Equal(t, providers.Kind("fake"), got.Kind())
}

func TestGetUnknownKindErrors(t *testing.T) {
	_, err := providers.Get(providers.Kind("unknown-xyz"))
	assert.Error(t, err)
}

func TestKindConstants(t *testing.T) {
	kinds := []providers.Kind{
		providers.KindSonarr, providers.KindRadarr, providers.KindLidarr,
		providers.KindJackett, providers.KindDeluge,
		providers.KindPlex, providers.KindEmby, providers.KindJellyfin,
	}
	assert.Len(t, kinds, 8)
	for _, k := range kinds {
		assert.NotEmpty(t, string(k))
	}
}
