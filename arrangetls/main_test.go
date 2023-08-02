// SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package arrangetls

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"os"
	"testing"
)

var (
	CertificateFile string
	KeyFile         string
)

func TestMain(m *testing.M) {
	certificate, err := CreateTestCertificate(&x509.Certificate{
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

	CertificateFile, KeyFile, err = CreateTestServerFiles(certificate)
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
