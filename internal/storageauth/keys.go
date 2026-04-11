package storageauth

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"

	"github.com/moabukar/miniblue/internal/store"
)

// SharedKeyContextKey is the store key prefix for ARM-created accounts (data-plane auth).
const SharedKeyContextKeyPrefix = "blob:sharedkeyctx:"

// DeterministicAccountKey returns a stable 64-byte value as base64 (Azure storage key shape).
func DeterministicAccountKey(sub, rg, account, keyID string) string {
	h1 := sha256.Sum256([]byte("miniblue|storage|" + sub + "|" + rg + "|" + account + "|" + keyID))
	h2 := sha256.Sum256([]byte("miniblue|storage|alt|" + sub + "|" + rg + "|" + account + "|" + keyID))
	full := append(h1[:], h2[:]...)
	return base64.StdEncoding.EncodeToString(full)
}

// PersistSharedKeyContext records subscription + resource group for deriving the same keys as listKeys.
func PersistSharedKeyContext(s *store.Store, sub, rg, account string) {
	s.Set(SharedKeyContextKeyPrefix+account, sub+"\n"+rg)
}

// DeleteSharedKeyContext removes persisted key context (call when deleting the storage account).
func DeleteSharedKeyContext(s *store.Store, account string) {
	s.Delete(SharedKeyContextKeyPrefix + account)
}

// SharedKeyContext returns sub, rg if the account was created via ARM.
func SharedKeyContext(s *store.Store, account string) (sub, rg string, ok bool) {
	v, ok := s.Get(SharedKeyContextKeyPrefix + account)
	if !ok {
		return "", "", false
	}
	ctx, ok := v.(string)
	if !ok {
		return "", "", false
	}
	parts := strings.SplitN(ctx, "\n", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// AccountKeyBytes returns decoded key1 and key2 for HMAC verification.
func AccountKeyBytes(s *store.Store, account string) (key1, key2 []byte, ok bool) {
	sub, rg, ok := SharedKeyContext(s, account)
	if !ok {
		return nil, nil, false
	}
	k1, err := base64.StdEncoding.DecodeString(DeterministicAccountKey(sub, rg, account, "1"))
	if err != nil {
		return nil, nil, false
	}
	k2, err := base64.StdEncoding.DecodeString(DeterministicAccountKey(sub, rg, account, "2"))
	if err != nil {
		return nil, nil, false
	}
	return k1, k2, true
}
