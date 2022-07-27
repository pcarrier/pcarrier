package signatures

import (
	"io"

	"golang.org/x/crypto/ssh"
)

// Verify verifies the signature of the given data and the armored signature using the given public key and the namespace.
// If the namespace is empty, the default namespace (file) is used.
func Verify(message io.Reader, signature *Signature) error {
	// Hash the message so we can verify it against the signature.
	h := supportedHashAlgorithms[signature.HashAlg]()
	if _, err := io.Copy(h, message); err != nil {
		return err
	}
	hm := h.Sum(nil)

	toVerify := MessageWrapper{
		Namespace:     signature.Namespace,
		HashAlgorithm: signature.HashAlg,
		Hash:          string(hm),
	}
	signedMessage := ssh.Marshal(toVerify)
	signedMessage = append([]byte(magicHeader), signedMessage...)
	return signature.PK.Verify(signedMessage, signature.Signature)
}
