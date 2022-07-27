package signatures

import (
	"crypto/sha256"
	"crypto/sha512"
	"golang.org/x/crypto/ssh"
	"hash"
)

type Signature struct {
	Signature *ssh.Signature
	PK        ssh.PublicKey
	HashAlg   string
	Namespace string
}

// https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.sshsig#L81
type MessageWrapper struct {
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Hash          string
}

// https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.sshsig#L34
type WrappedSig struct {
	MagicHeader   [6]byte
	Version       uint32
	PublicKey     string
	Namespace     string
	Reserved      string
	HashAlgorithm string
	Signature     string
}

const (
	pemType     = "SSH SIGNATURE"
	magicHeader = "SSHSIG"
)

var supportedHashAlgorithms = map[string]func() hash.Hash{
	"sha256": sha256.New,
	"sha512": sha512.New,
}
