package piv

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"math/big"
	"testing"
)

// These compile-time checks retain the v1 management API shape.
var (
	_ func(*YubiKey, [24]byte, [24]byte) error                      = (*YubiKey).SetManagementKey
	_ func(*YubiKey, [24]byte, Slot, Key) (crypto.PublicKey, error) = (*YubiKey).GenerateKey
	_ func(*YubiKey, [24]byte, Slot, crypto.PrivateKey, Key) error  = (*YubiKey).SetPrivateKeyInsecure
)

// TestRSAV1UnsupportedAlgorithms rejects newer key sizes before PIN or transport.
func TestRSAV1UnsupportedAlgorithms(t *testing.T) {
	for _, bits := range []int{3072, 4096} {
		t.Run(fmt.Sprintf("RSA%d", bits), func(t *testing.T) {
			pub := &rsa.PublicKey{N: new(big.Int).Lsh(big.NewInt(1), uint(bits-1)), E: 65537}
			prompts, rawCalls := 0, 0
			key, err := (&YubiKey{}).PrivateKey(SlotKeyManagement, pub, KeyAuth{
				PINPolicy: PINPolicyAlways,
				PINPrompt: func() (string, error) { prompts++; return "", errors.New("unexpected PIN") },
			})
			if err != nil {
				t.Fatal(err)
			}
			plain, err := key.(crypto.Decrypter).Decrypt(rand.Reader, make([]byte, pub.Size()), &rsa.OAEPOptions{Hash: crypto.SHA256, MGFHash: crypto.SHA256})
			if err == nil || plain != nil || prompts != 0 {
				t.Fatalf("unsupported key authenticated: %x/%v, prompts%d", plain, err, prompts)
			}
			tx := rsaAPDUFunc(func(apdu) ([]byte, error) { rawCalls++; return nil, nil })
			plain, err = ykDecryptRSA(tx, SlotKeyManagement, pub, make([]byte, pub.Size()))
			if err == nil || plain != nil || rawCalls != 0 {
				t.Fatalf("unsupported key transmitted: %x/%v, calls%d", plain, err, rawCalls)
			}
		})
	}
}
