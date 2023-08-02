/**
 * Copyright 2023 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package arrangetls

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"os"

	"github.com/stretchr/testify/suite"
)

// Suite is a simple stretchr/testify suite that manages the lifecycle
// of a testing certificate.  Useful primarily for testing TLS code.
type Suite struct {
	suite.Suite

	certificate     *tls.Certificate
	certificateFile string
	keyFile         string
}

// Config returns a configuration object using this suite's certificate.
func (suite *Suite) Config() *Config {
	return &Config{
		Certificates: ExternalCertificates{
			{
				CertificateFile: suite.certificateFile,
				KeyFile:         suite.keyFile,
			},
		},
	}
}

// TLSConfig creates a new *tls.Config using the certificate generated in setup.
func (suite *Suite) TLSConfig() *tls.Config {
	tlsConfig, err := suite.Config().New()
	suite.Require().NoError(err)
	suite.Require().NotNil(tlsConfig)
	return tlsConfig
}

// SetupSuite creates a testing certificate and stores the certificate and its
// private key in temporary files.
func (suite *Suite) SetupSuite() {
	var err error
	suite.certificate, err = CreateTestCertificate(&x509.Certificate{
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

	suite.certificateFile, suite.keyFile, err = CreateTestServerFiles(suite.certificate)
	suite.Require().NoError(
		err,
		"Unable to create temporary server files",
	)
}

// TearDownSuite cleans up the temporary files created in setup.
func (suite *Suite) TearDownSuite() {
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
