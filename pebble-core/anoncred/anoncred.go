package anoncred

/*
SecretCredential interface: Defines the methods for a secret credential,
including Bytes, Public, and SerialNo.
*/
type SecretCredential interface {
	Bytes() []byte
	Public() (PublicCredential, error)
	SerialNo() []byte
}

/*
PublicCredential interface: Defines the method for a public credential, which is Bytes.
*/
type PublicCredential interface {
	Bytes() []byte
}

/*
CredentialSet interface: Defines the methods for a credential set, including Len, Sign, and Verify.
*/
type CredentialSet interface {
	Len() int
	Sign(secret SecretCredential, msg []byte) ([]byte, error)
	Verify(serialNo, sig, msg []byte) error
}

/*
CredentialSystem interface: Defines the methods for a credential system,
such as generating secret credentials, reading secret and public credentials, and creating a credential set.
*/
type CredentialSystem interface {
	GenerateSecretCredential() (SecretCredential, error)
	ReadSecretCredential(p []byte) (SecretCredential, error)
	ReadPublicCredential(p []byte) (PublicCredential, error)
	MakeCredentialSet(credentials []PublicCredential) (CredentialSet, error)
}
