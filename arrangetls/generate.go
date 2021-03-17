package arrangetls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
)

// CreateTestCertificate creates a self-signed x509 ceritificate for use in testing
// TLS code.  A 1024-bit RSA key pair is used, and otherwise all defaults are taken.
func CreateTestCertificate(template *x509.Certificate) (*tls.Certificate, error) {
	var (
		key      *rsa.PrivateKey
		derBytes []byte
		err      error
	)

	key, err = rsa.GenerateKey(rand.Reader, 4096)
	if err == nil {
		derBytes, err = x509.CreateCertificate(
			rand.Reader,
			template,
			template,
			&key.PublicKey,
			key,
		)
	}

	return &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  key,
	}, err
}

// CreateTestServerFiles creates the certificate file and key file expected by
// net/http.Server, which is the basic model followed by mode golang TLS code.
//
// The supplied certificate must have at least (1) []byte in its Certificate chain.
// If not, this function will panic.  If it has more than (1) entry in its chain,
// only the first entry is written to the certificate file.
func CreateTestServerFiles(certificate *tls.Certificate) (certificateFileName, keyFileName string, err error) {
	var (
		certificateFile *os.File
		keyFile         *os.File
		keyBytes        []byte
	)

	certificateFile, err = ioutil.TempFile("", "test-cert-*.pem")
	if err == nil {
		defer certificateFile.Close()
		keyFile, err = ioutil.TempFile("", "test-key-*.pem")
	}

	if err == nil {
		defer keyFile.Close()
		err = pem.Encode(certificateFile, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certificate.Certificate[0],
		})
	}

	if err == nil {
		keyBytes, err = x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	}

	if err == nil {
		err = pem.Encode(keyFile, &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyBytes,
		})
	}

	if err == nil {
		certificateFileName = certificateFile.Name()
		keyFileName = keyFile.Name()
	}

	return
}
