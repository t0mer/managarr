// internal/storage/crypto_test.go
package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/galactica/internal/storage"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	secret := "test-secret-key"
	plaintext := []byte("sensitive api key value")

	ciphertext, err := storage.Encrypt(plaintext, secret)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	result, err := storage.Decrypt(ciphertext, secret)
	require.NoError(t, err)
	assert.Equal(t, plaintext, result)
}

func TestEncryptProducesUniqueNonce(t *testing.T) {
	secret := "test-secret"
	plaintext := []byte("same plaintext")

	a, err := storage.Encrypt(plaintext, secret)
	require.NoError(t, err)
	b, err := storage.Encrypt(plaintext, secret)
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
}

func TestDecryptFailsWithWrongKey(t *testing.T) {
	ciphertext, err := storage.Encrypt([]byte("secret"), "correct-key")
	require.NoError(t, err)

	_, err = storage.Decrypt(ciphertext, "wrong-key")
	assert.Error(t, err)
}
