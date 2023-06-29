package arrangetest

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"

	"github.com/stretchr/testify/suite"
	"github.com/xmidt-org/arrange/arrangetls"
)

// TLSSuite is a simple stretchr/testify suite that manages the lifecycle
// of a testing certificate.  Useful primarily for testing TLS code.
type TLSSuite struct {
	suite.Suite

	certificate     *tls.Certificate
	certificateFile string
	keyFile         string
}

func (suite *TLSSuite) SetupSuite() {
	var err error
	suite.certificate, err = arrangetls.CreateTestCertificate(&x509.Certificate{
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

	suite.Require().NoError(
		err,
		"Unable to generate test certificate",
	)

	suite.certificateFile, suite.keyFile, err = arrangetls.CreateTestServerFiles(suite.certificate)
	suite.Require().NoError(
		err,
		"Unable to create temporary server files",
	)
}

func (suite *TLSSuite) TearDownSuite() {
	if err := os.Remove(suite.certificateFile); err != nil {
		suite.T().Logf(
			"Unable to remove certificate file %s: %s", suite.certificateFile, err,
		)
	}

	if err := os.Remove(suite.keyFile); err != nil {
		suite.T().Logf(
			"Unable to remove key file %s: %s", suite.keyFile, err,
		)
	}
}
