package arrangehttp

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/xmidt-org/arrange/arrangetls"
)

var (
	CertificateFile string
	KeyFile         string
)

func TestMain(m *testing.M) {
	certificate, err := arrangetls.CreateTestCertificate(&x509.Certificate{
		SerialNumber: big.NewInt(837492837),
		Issuer: pkix.Name{
			CommonName: "test",
		},
		Subject: pkix.Name{
			CommonName: "test",
		},
		DNSNames: []string{
			"test.net",
		},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to generate test certificate: %s", err)
		os.Exit(1)
	}

	CertificateFile, KeyFile, err = arrangetls.CreateTestServerFiles(certificate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create temporary server files: %s", err)
		os.Exit(1)
	}

	os.Exit(func() int {
		defer os.Remove(CertificateFile)
		defer os.Remove(KeyFile)
		return m.Run()
	}())
}
