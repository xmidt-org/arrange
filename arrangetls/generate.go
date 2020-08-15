package arrangetls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
)

// CreateTestCertificate creates a self-signed x509 ceritificate for use in testing
// TLS code.  A 1024-bit RSA key pair is used, and otherwise all defaults are taken.
func CreateTestCertificate(template *x509.Certificate) (*tls.Certificate, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&key.PublicKey,
		key,
	)

	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  key,
	}, nil
}

// CreateTestServerFiles creates the certificate file and key file expected by
// net/http.Server, which is the basic model followed by mode golang TLS code.
func CreateTestServerFiles(certificate *tls.Certificate) (certificateFileName, keyFileName string, err error) {
	if len(certificate.Certificate) != 1 {
		err = errors.New("Only (1) DER-encoded certificate is supported")
		return
	}

	var certificateFile *os.File
	certificateFile, err = ioutil.TempFile("", "test-cert-*.pem")
	if err != nil {
		return
	}

	var keyFile *os.File
	keyFile, err = ioutil.TempFile("", "test-key-*.pem")
	if err != nil {
		certificateFile.Close()
		return
	}

	defer certificateFile.Close()
	defer keyFile.Close()

	err = pem.Encode(certificateFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate.Certificate[0],
	})

	if err != nil {
		return
	}

	var keyDERBytes []byte
	keyDERBytes, err = x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	if err != nil {
		return
	}

	err = pem.Encode(keyFile, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyDERBytes,
	})

	if err != nil {
		return
	}

	certificateFileName = certificateFile.Name()
	keyFileName = keyFile.Name()
	return
}
