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
		verifiers              = PeerVerifiers{
			func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
				assert.Fail("no verifiers should have been called")
				return errors.New("no verifiers should have been called")
			},
		}
	)

	assert.Error(
		verifiers.VerifyPeerCertificate([][]byte{unparseableCertificate}, nil),
	)
}

func testPeerVerifiersSuccess(t *testing.T) {
	var (
		require  = require.New(t)
		random   = rand.New(rand.NewSource(1234))
		template = &x509.Certificate{
			SerialNumber: big.NewInt(35871293874),
		}
	)

	key, err := rsa.GenerateKey(random, 512)
	require.NoError(err)

	peerCert, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	require.NoError(err)

	for _, n := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", n), func(t *testing.T) {
			var (
				assert       = assert.New(t)
				executeCount int
				verifiers    PeerVerifiers
			)

			assert.Zero(verifiers.Len())
			for i := 0; i < n; i++ {
				verifiers.Append(func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
					executeCount++
					assert.Equal(template.SerialNumber, actual.SerialNumber)
					return nil
				})

				assert.Equal(i+1, verifiers.Len())
			}

			assert.NoError(
				verifiers.VerifyPeerCertificate([][]byte{peerCert}, nil),
			)

			assert.Equal(n, executeCount)
		})
	}
}

func testPeerVerifiersFailure(t *testing.T) {
	var (
		require  = require.New(t)
		random   = rand.New(rand.NewSource(8362))
		template = &x509.Certificate{
			SerialNumber: big.NewInt(9472387653),
		}
	)

	key, err := rsa.GenerateKey(random, 512)
	require.NoError(err)

	peerCert, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	require.NoError(err)

	for _, n := range []int{1, 2, 5} {
		t.Run(fmt.Sprintf("len=%d", n), func(t *testing.T) {
			var (
				assert       = assert.New(t)
				expectedErr  = PeerVerifyError{Reason: "expected"}
				executeCount int
				verifiers    PeerVerifiers
			)

			assert.Zero(verifiers.Len())
			for i := 0; i < n-1; i++ {
				verifiers.Append(func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
					executeCount++
					assert.Equal(template.SerialNumber, actual.SerialNumber)
					return nil
				})

				assert.Equal(i+1, verifiers.Len())
			}

			verifiers.Append(func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
				executeCount++
				assert.Equal(template.SerialNumber, actual.SerialNumber)
				return expectedErr
			})

			assert.Equal(n, verifiers.Len())

			assert.Equal(
				expectedErr,
				verifiers.VerifyPeerCertificate([][]byte{peerCert}, nil),
			)

			assert.Equal(n, executeCount)
		})
	}
}

func TestPeerVerifiers(t *testing.T) {
	t.Run("Empty", testPeerVerifiersEmpty)
	t.Run("UnparseableCertificate", testPeerVerifiersUnparseableCertificate)
	t.Run("Success", testPeerVerifiersSuccess)
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

func testNewTLSConfigNil(t *testing.T) {
	assert := assert.New(t)

	assert.Nil(NewTLSConfig(nil))
	assert.Nil(
		NewTLSConfig(
			nil,
			func(*x509.Certificate, [][]*x509.Certificate) error {
				return nil
			},
		),
	)
}

func testNewTLSConfigNoCertificate(t *testing.T) {
	var (
		assert = assert.New(t)
		config Config
	)

	tlsConfig, err := NewTLSConfig(&config)
	assert.NoError(err)
	assert.NotNil(tlsConfig)
}

func testNewTLSConfigMissingCertificate(t *testing.T) {
	var (
		assert = assert.New(t)
		config = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: "missing",
					KeyFile:         "missing",
				},
			},
		}
	)

	tlsConfig, err := NewTLSConfig(&config)
	assert.Error(err)
	assert.Nil(tlsConfig)
}

func testNewTLSConfigBasic(t *testing.T, CertificateFile, KeyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		config  = Config{
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

	tlsConfig, err := NewTLSConfig(&config)
	require.NoError(err)
	require.NotNil(tlsConfig)

	assert.Equal(uint16(1), tlsConfig.MinVersion)
	assert.Equal(uint16(3), tlsConfig.MaxVersion)
	assert.Equal([]string{"http/1.1"}, tlsConfig.NextProtos)
	assert.Len(tlsConfig.Certificates, 1)
	assert.Equal("foobar.com", tlsConfig.ServerName)
	assert.True(tlsConfig.InsecureSkipVerify)
	assert.NotEmpty(tlsConfig.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Nil(tlsConfig.VerifyPeerCertificate)
	assert.Equal(tls.NoClientCert, tlsConfig.ClientAuth)
}

func testNewTLSConfigCustomNextProtos(t *testing.T, CertificateFile, KeyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		config  = Config{
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

	tlsConfig, err := NewTLSConfig(&config)
	require.NoError(err)
	require.NotNil(tlsConfig)

	assert.Equal(uint16(1), tlsConfig.MinVersion)
	assert.Equal(uint16(3), tlsConfig.MaxVersion)
	assert.Equal([]string{"http", "ftp"}, tlsConfig.NextProtos)
	assert.Len(tlsConfig.Certificates, 1)
	assert.NotEmpty(tlsConfig.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Nil(tlsConfig.VerifyPeerCertificate)
	assert.Equal(tls.NoClientCert, tlsConfig.ClientAuth)
}

func testNewTLSConfigVerifyPeerCertificate(t *testing.T, CertificateFile, KeyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		random   = rand.New(rand.NewSource(828675))
		template = &x509.Certificate{
			DNSNames:     []string{"foobar.example.com"},
			SerialNumber: big.NewInt(356493746),
		}

		config = Config{
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

	key, err := rsa.GenerateKey(random, 512)
	require.NoError(err)

	peerCert, err := x509.CreateCertificate(random, template, template, &key.PublicKey, key)
	require.NoError(err)

	tlsConfig, err := NewTLSConfig(&config, extra)
	require.NoError(err)
	require.NotNil(tlsConfig)

	assert.Zero(tlsConfig.MinVersion)
	assert.Zero(tlsConfig.MaxVersion)
	assert.Equal([]string{"http/1.1"}, tlsConfig.NextProtos)
	assert.Len(tlsConfig.Certificates, 1)
	assert.NotEmpty(tlsConfig.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Equal(tls.NoClientCert, tlsConfig.ClientAuth)

	require.NotNil(tlsConfig.VerifyPeerCertificate)
	assert.Equal(
		extraErr,
		tlsConfig.VerifyPeerCertificate([][]byte{peerCert}, nil),
	)
}

func testNewTLSConfigCertPools(t *testing.T, CertificateFile, KeyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		config = Config{
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

	tlsConfig, err := NewTLSConfig(&config)
	require.NoError(err)
	require.NotNil(tlsConfig)

	assert.Equal(uint16(1), tlsConfig.MinVersion)
	assert.Equal(uint16(3), tlsConfig.MaxVersion)
	assert.Equal([]string{"http", "ftp"}, tlsConfig.NextProtos)
	assert.Len(tlsConfig.Certificates, 1)
	assert.NotEmpty(tlsConfig.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Nil(tlsConfig.VerifyPeerCertificate)
	assert.Equal(tls.RequireAndVerifyClientCert, tlsConfig.ClientAuth)
	assert.NotNil(tlsConfig.ClientCAs)
	assert.NotNil(tlsConfig.RootCAs)
}

func testNewTLSConfigClientCAsError(t *testing.T, CertificateFile, KeyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		config = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			ClientCAs: ExternalCertPool{"missing"},
		}
	)

	tlsConfig, err := NewTLSConfig(&config)
	assert.Error(err)
	require.Nil(tlsConfig)
}

func testNewTLSConfigRootCAsError(t *testing.T, CertificateFile, KeyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		config = Config{
			Certificates: ExternalCertificates{
				{
					CertificateFile: CertificateFile,
					KeyFile:         KeyFile,
				},
			},
			RootCAs: ExternalCertPool{"missing"},
		}
	)

	tlsConfig, err := NewTLSConfig(&config)
	assert.Error(err)
	require.Nil(tlsConfig)
}

func TestNewTLSConfig(t *testing.T) {
	t.Run("Nil", testNewTLSConfigNil)
	t.Run("NoCertificate", testNewTLSConfigNoCertificate)
	t.Run("MissingCertificate", testNewTLSConfigMissingCertificate)

	t.Run("Basic", func(t *testing.T) {
		testNewTLSConfigBasic(t, CertificateFile, KeyFile)
	})

	t.Run("CustomNextProtos", func(t *testing.T) {
		testNewTLSConfigCustomNextProtos(t, CertificateFile, KeyFile)
	})

	t.Run("VerifyPeerCertificate", func(t *testing.T) {
		testNewTLSConfigVerifyPeerCertificate(t, CertificateFile, KeyFile)
	})

	t.Run("CertPools", func(t *testing.T) {
		testNewTLSConfigCertPools(t, CertificateFile, KeyFile)
	})

	t.Run("RootCAsError", func(t *testing.T) {
		testNewTLSConfigRootCAsError(t, CertificateFile, KeyFile)
	})

	t.Run("ClientCAsError", func(t *testing.T) {
		testNewTLSConfigClientCAsError(t, CertificateFile, KeyFile)
	})
}
