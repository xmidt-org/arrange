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
