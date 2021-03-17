package arrangetls

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeerVerifierError(t *testing.T) {
	var (
		assert = assert.New(t)
		err    = PeerVerifyError{
			Reason: "expected error text",
		}
	)

	assert.Equal("expected error text", err.Error())
}

func testPeerVerifiersEmpty(t *testing.T) {
	var (
		assert = assert.New(t)
	)

	assert.NoError(PeerVerifiers{}.VerifyPeerCertificate(nil, nil))
}

func testPeerVerifiersUnparseableCertificate(t *testing.T) {
	var (
		assert                 = assert.New(t)
		unparseableCertificate = []byte("this is an unparseable certificate")
		verifiers              = NewPeerVerifiers(
			func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
				assert.Fail("no verifiers should have been called")
				return errors.New("no verifiers should have been called")
			},
		)
	)

	assert.Error(
		verifiers.VerifyPeerCertificate([][]byte{unparseableCertificate}, nil),
	)
}

func testPeerVerifiersSuccess(t *testing.T) {
	var (
		require  = require.New(t)
		random   = rand.New(rand.NewSource(1234)) //nolint:gosec // this is just a test
		template = &x509.Certificate{
			SerialNumber: big.NewInt(35871293874),
		}
	)

	key, err := rsa.GenerateKey(random, 2048)
	require.NoError(err)

	peerCert, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	require.NoError(err)

	for _, n := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", n), func(t *testing.T) {
			var (
				assert       = assert.New(t)
				executeCount int
				pvs          PeerVerifiers
			)

			for i := 0; i < n; i++ {
				pvs = pvs.Append(func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
					executeCount++
					assert.Equal(template.SerialNumber, actual.SerialNumber)
					return nil
				})
			}

			assert.NoError(
				pvs.VerifyPeerCertificate([][]byte{peerCert}, nil),
			)

			assert.Equal(n, executeCount)
		})
	}
}

func testPeerVerifiersExtend(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		random   = rand.New(rand.NewSource(1234)) //nolint:gosec // this is just a test
		template = &x509.Certificate{
			SerialNumber: big.NewInt(94782236446),
		}
	)

	key, err := rsa.GenerateKey(random, 2048)
	require.NoError(err)

	peerCert, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	require.NoError(err)

	called0 := false
	called1 := false
	pvs := NewPeerVerifiers(
		func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
			assert.Equal(template.SerialNumber, actual.SerialNumber)
			called0 = true
			return nil
		},
	).Extend(NewPeerVerifiers(
		func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
			assert.Equal(template.SerialNumber, actual.SerialNumber)
			called1 = true
			return nil
		},
	))

	assert.NoError(
		pvs.VerifyPeerCertificate([][]byte{peerCert}, nil),
	)

	assert.True(called0)
	assert.True(called1)
}

func testPeerVerifiersFailure(t *testing.T) {
	var (
		require  = require.New(t)
		random   = rand.New(rand.NewSource(8362)) //nolint:gosec // this is just a test
		template = &x509.Certificate{
			SerialNumber: big.NewInt(9472387653),
		}
	)

	key, err := rsa.GenerateKey(random, 2048)
	require.NoError(err)

	peerCert, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	require.NoError(err)

	for _, n := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", n), func(t *testing.T) {
			var (
				assert       = assert.New(t)
				expectedErr  = PeerVerifyError{Reason: "expected"}
				executeCount int
				pvs          PeerVerifiers
			)

			for i := 0; i < n-1; i++ {
				pvs = pvs.Append(func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
					executeCount++
					assert.Equal(template.SerialNumber, actual.SerialNumber)
					return nil
				})
			}

			pvs = pvs.Append(func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
				executeCount++
				assert.Equal(template.SerialNumber, actual.SerialNumber)
				return expectedErr
			})

			assert.Equal(
				expectedErr,
				pvs.VerifyPeerCertificate([][]byte{peerCert}, nil),
			)

			assert.Equal(n, executeCount)
		})
	}
}

func TestPeerVerifiers(t *testing.T) {
	t.Run("Empty", testPeerVerifiersEmpty)
	t.Run("UnparseableCertificate", testPeerVerifiersUnparseableCertificate)
	t.Run("Success", testPeerVerifiersSuccess)
	t.Run("Extend", testPeerVerifiersExtend)
	t.Run("Failure", testPeerVerifiersFailure)
}

func testPeerVerifyConfigEmpty(t *testing.T) {
	var (
		assert = assert.New(t)
		pvc    PeerVerifyConfig
	)

	assert.Nil(pvc.Verifier())
}

func testPeerVerifyConfigSuccess(t *testing.T) {
	testData := []struct {
		peerCert x509.Certificate
		config   PeerVerifyConfig
	}{
		{
			peerCert: x509.Certificate{
				DNSNames: []string{"test.foobar.com"},
			},
			config: PeerVerifyConfig{
				DNSSuffixes: []string{"foobar.com"},
			},
		},
		{
			peerCert: x509.Certificate{
				DNSNames: []string{"first.foobar.com", "second.something.net"},
			},
			config: PeerVerifyConfig{
				DNSSuffixes: []string{"another.thing.org", "something.net"},
			},
		},
		{
			peerCert: x509.Certificate{
				Subject: pkix.Name{
					CommonName: "PCTEST-another.thing.org",
				},
			},
			config: PeerVerifyConfig{
				DNSSuffixes: []string{"another.thing.org", "something.net"},
			},
		},
		{
			peerCert: x509.Certificate{
				Subject: pkix.Name{
					CommonName: "A Great Organization",
				},
			},
			config: PeerVerifyConfig{
				CommonNames: []string{"A Great Organization"},
			},
		},
		{
			peerCert: x509.Certificate{
				Subject: pkix.Name{
					CommonName: "A Great Organization",
				},
			},
			config: PeerVerifyConfig{
				CommonNames: []string{"First Organization Doesn't Match", "A Great Organization"},
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				verifier = record.config.Verifier()
			)

			require.NotNil(verifier)
			assert.NoError(verifier(&record.peerCert, nil))
		})
	}
}

func testPeerVerifyConfigFailure(t *testing.T) {
	testData := []struct {
		peerCert x509.Certificate
		config   PeerVerifyConfig
	}{
		{
			peerCert: x509.Certificate{},
			config: PeerVerifyConfig{
				DNSSuffixes: []string{"foobar.net"},
				CommonNames: []string{"For Great Justice"},
			},
		},
		{
			peerCert: x509.Certificate{
				DNSNames: []string{"another.company.com"},
			},
			config: PeerVerifyConfig{
				DNSSuffixes: []string{"foobar.net"},
				CommonNames: []string{"For Great Justice"},
			},
		},
		{
			peerCert: x509.Certificate{
				Subject: pkix.Name{
					CommonName: "Villains For Hire",
				},
			},
			config: PeerVerifyConfig{
				DNSSuffixes: []string{"foobar.net"},
				CommonNames: []string{"For Great Justice"},
			},
		},
		{
			peerCert: x509.Certificate{
				DNSNames: []string{"another.company.com"},
				Subject: pkix.Name{
					CommonName: "Villains For Hire",
				},
			},
			config: PeerVerifyConfig{
				DNSSuffixes: []string{"foobar.net"},
				CommonNames: []string{"For Great Justice"},
			},
		},
	}

	for i, record := range testData {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var (
				assert  = assert.New(t)
				require = require.New(t)

				verifier = record.config.Verifier()
			)

			require.NotNil(verifier)
			err := verifier(&record.peerCert, nil)
			assert.Error(err)
			require.IsType(PeerVerifyError{}, err)
			assert.Equal(&record.peerCert, err.(PeerVerifyError).Certificate)
		})
	}
}

func TestPeerVerifyConfig(t *testing.T) {
	t.Run("Empty", testPeerVerifyConfigEmpty)
	t.Run("Success", testPeerVerifyConfigSuccess)
	t.Run("Failure", testPeerVerifyConfigFailure)
}

func testExternalCertificateSuccess(t *testing.T) {
	var (
		assert = assert.New(t)

		ec ExternalCertificate
	)

	ec.CertificateFile, ec.KeyFile = CertificateFile, KeyFile
	loaded, err := ec.Load()
	assert.NoError(err)
	assert.NotEqual(tls.Certificate{}, loaded)
}

func testExternalCertificateFailure(t *testing.T) {
	var (
		assert = assert.New(t)

		ec ExternalCertificate
	)

	loaded, err := ec.Load()
	assert.Error(err)
	assert.Equal(tls.Certificate{}, loaded)
}

func TestExternalCertificate(t *testing.T) {
	t.Run("Success", testExternalCertificateSuccess)
	t.Run("Failure", testExternalCertificateFailure)
}

func testExternalCertificatesSuccess(t *testing.T) {
	CertificateFile, KeyFile := CertificateFile, KeyFile
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert = assert.New(t)
				ecs    ExternalCertificates
			)

			assert.Zero(ecs.Len())
			for i := 0; i < length; i++ {
				ecs.Append(ExternalCertificate{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				})

				assert.Equal(i+1, ecs.Len())
			}

			loaded, err := ecs.AppendTo(nil)
			assert.NoError(err)
			assert.Len(loaded, length)

			loaded, err = ecs.AppendTo(loaded)
			assert.NoError(err)
			assert.Len(loaded, 2*length)
		})
	}
}

func testExternalCertificatesFailure(t *testing.T) {
	CertificateFile, KeyFile := CertificateFile, KeyFile
	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert = assert.New(t)
				ecs    ExternalCertificates
			)

			assert.Zero(ecs.Len())
			for i := 0; i < length-1; i++ {
				ecs.Append(ExternalCertificate{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				})

				assert.Equal(i+1, ecs.Len())
			}

			ecs.Append(ExternalCertificate{}) // this will always fail
			assert.Equal(length, ecs.Len())

			loaded, err := ecs.AppendTo(nil)
			assert.Error(err)
			assert.Equal(length-1, len(loaded)) // the successful loads should be appended

			loaded, err = ecs.AppendTo(loaded)
			assert.Error(err)
			assert.Equal(2*(length-1), len(loaded)) // the successful loads should be appended
		})
	}
}

func TestExternalCertificates(t *testing.T) {
	t.Run("Success", testExternalCertificatesSuccess)
	t.Run("Failure", testExternalCertificatesFailure)
}

func testExternalCertPoolSuccess(t *testing.T) {
	for _, length := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert = assert.New(t)
				ecp    ExternalCertPool
			)

			assert.Zero(ecp.Len())
			for i := 0; i < length; i++ {
				ecp.Append(CertificateFile)
				assert.Equal(i+1, ecp.Len())
			}

			certPool := x509.NewCertPool()
			count, err := ecp.AppendTo(certPool)
			assert.NoError(err)
			assert.Equal(length, count)
		})
	}
}

func testExternalCertPoolMissingFile(t *testing.T) {
	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert = assert.New(t)
				ecp    ExternalCertPool
			)

			assert.Zero(ecp.Len())
			for i := 0; i < length-1; i++ {
				ecp.Append(CertificateFile)
				assert.Equal(i+1, ecp.Len())
			}

			ecp.Append("missing")
			assert.Equal(length, ecp.Len())

			certPool := x509.NewCertPool()
			count, err := ecp.AppendTo(certPool)
			assert.Error(err)
			assert.Equal(length-1, count) // the successes should have been added
		})
	}
}

func testExternalCertPoolInvalidFile(t *testing.T) {
	require := require.New(t)
	invalidFile, err := ioutil.TempFile("", "invalid.*.cert")
	require.NoError(err)

	defer os.Remove(invalidFile.Name())
	_, err = invalidFile.WriteString("this is not valid PEM")
	invalidFile.Close()
	require.NoError(err)

	for _, length := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", length), func(t *testing.T) {
			var (
				assert = assert.New(t)
				ecp    ExternalCertPool
			)

			assert.Zero(ecp.Len())
			for i := 0; i < length-1; i++ {
				ecp.Append(CertificateFile)
				assert.Equal(i+1, ecp.Len())
			}

			ecp.Append(invalidFile.Name())
			assert.Equal(length, ecp.Len())

			certPool := x509.NewCertPool()
			count, err := ecp.AppendTo(certPool)
			assert.Error(err)
			assert.Equal(length-1, count) // the successes should have been added
		})
	}
}

func TestExternalCertPool(t *testing.T) {
	t.Run("Success", testExternalCertPoolSuccess)
	t.Run("MissingFile", testExternalCertPoolMissingFile)
	t.Run("InvalidFile", testExternalCertPoolInvalidFile)
}

func testConfigNil(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() {
		var c *Config
		tc, err := c.New()

		assert.Nil(tc)
		assert.NoError(err)
	})
}

func testConfigNoCertificate(t *testing.T) {
	var (
		assert = assert.New(t)
		c      Config
	)

	tc, err := c.New()
	assert.NoError(err)
	assert.NotNil(tc)
}

func testConfigMissingCertificate(t *testing.T) {
	var (
		assert = assert.New(t)
		c      = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: "missing",
					KeyFile:         "missing",
				},
			},
		}
	)

	tc, err := c.New()
	assert.Error(err)
	assert.Nil(tc)
}

func testConfigBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c       = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			MinVersion:         1,
			MaxVersion:         3,
			ServerName:         "foobar.com",
			InsecureSkipVerify: true,
		}
	)

	tc, err := c.New()
	require.NoError(err)
	require.NotNil(tc)

	assert.Equal(uint16(1), tc.MinVersion)
	assert.Equal(uint16(3), tc.MaxVersion)
	assert.Equal([]string{"http/1.1"}, tc.NextProtos)
	assert.Len(tc.Certificates, 1)
	assert.Equal("foobar.com", tc.ServerName)
	assert.True(tc.InsecureSkipVerify)
	assert.NotEmpty(tc.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Nil(tc.VerifyPeerCertificate)
	assert.Equal(tls.NoClientCert, tc.ClientAuth)
}

func testConfigCustomNextProtos(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		c       = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			MinVersion: 1,
			MaxVersion: 3,
			NextProtos: []string{"http", "ftp"},
		}
	)

	tc, err := c.New()
	require.NoError(err)
	require.NotNil(tc)

	assert.Equal(uint16(1), tc.MinVersion)
	assert.Equal(uint16(3), tc.MaxVersion)
	assert.Equal([]string{"http", "ftp"}, tc.NextProtos)
	assert.Len(tc.Certificates, 1)
	assert.NotEmpty(tc.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Nil(tc.VerifyPeerCertificate)
	assert.Equal(tls.NoClientCert, tc.ClientAuth)
}

func testConfigVerifyPeerCertificate(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		random   = rand.New(rand.NewSource(828675)) //nolint:gosec // this is just a test
		template = &x509.Certificate{
			DNSNames:     []string{"foobar.example.com"},
			SerialNumber: big.NewInt(356493746),
		}

		c = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			PeerVerify: &PeerVerifyConfig{
				DNSSuffixes: []string{"example.com"},
			},
		}

		extraErr = errors.New("expected error")
		extra    = func(peerCert *x509.Certificate, chain [][]*x509.Certificate) error {
			assert.Equal(template.SerialNumber, peerCert.SerialNumber)
			return extraErr
		}
	)

	key, err := rsa.GenerateKey(random, 2048)
	require.NoError(err)

	peerCert, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	require.NoError(err)

	tc, err := c.New(extra)
	require.NoError(err)
	require.NotNil(tc)

	assert.Zero(tc.MinVersion)
	assert.Zero(tc.MaxVersion)
	assert.Equal([]string{"http/1.1"}, tc.NextProtos)
	assert.Len(tc.Certificates, 1)
	assert.NotEmpty(tc.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Equal(tls.NoClientCert, tc.ClientAuth)

	require.NotNil(tc.VerifyPeerCertificate)
	assert.Equal(
		extraErr,
		tc.VerifyPeerCertificate([][]byte{peerCert}, nil),
	)
}

func testConfigCertPools(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		c = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			RootCAs:    ExternalCertPool{CertificateFile}, // this works as a bundle also
			ClientCAs:  ExternalCertPool{CertificateFile}, // this works as a bundle also
			MinVersion: 1,
			MaxVersion: 3,
			NextProtos: []string{"http", "ftp"},
		}
	)

	tc, err := c.New()
	require.NoError(err)
	require.NotNil(tc)

	assert.Equal(uint16(1), tc.MinVersion)
	assert.Equal(uint16(3), tc.MaxVersion)
	assert.Equal([]string{"http", "ftp"}, tc.NextProtos)
	assert.Len(tc.Certificates, 1)
	assert.NotEmpty(tc.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Nil(tc.VerifyPeerCertificate)
	assert.Equal(tls.RequireAndVerifyClientCert, tc.ClientAuth)
	assert.NotNil(tc.ClientCAs)
	assert.NotNil(tc.RootCAs)
}

func testConfigClientCAsError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		c = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			ClientCAs: ExternalCertPool{"missing"},
		}
	)

	tc, err := c.New()
	assert.Error(err)
	require.Nil(tc)
}

func testConfigRootCAsError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		c = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			RootCAs: ExternalCertPool{"missing"},
		}
	)

	tc, err := c.New()
	assert.Error(err)
	require.Nil(tc)
}

func TestConfig(t *testing.T) {
	t.Run("Nil", testConfigNil)
	t.Run("NoCertificate", testConfigNoCertificate)
	t.Run("MissingCertificate", testConfigMissingCertificate)
	t.Run("Basic", testConfigBasic)
	t.Run("CustomNextProtos", testConfigCustomNextProtos)
	t.Run("VerifyPeerCertificate", testConfigVerifyPeerCertificate)
	t.Run("CertPools", testConfigCertPools)
	t.Run("RootCAsError", testConfigRootCAsError)
	t.Run("ClientCAsError", testConfigClientCAsError)
}
