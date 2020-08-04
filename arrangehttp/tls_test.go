package arrangehttp

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

			for i := 0; i < n; i++ {
				verifiers = append(verifiers, func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
					executeCount++
					assert.Equal(template.SerialNumber, actual.SerialNumber)
					return nil
				})
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

			for i := 0; i < n-1; i++ {
				verifiers = append(verifiers, func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
					executeCount++
					assert.Equal(template.SerialNumber, actual.SerialNumber)
					return nil
				})
			}

			verifiers = append(verifiers, func(actual *x509.Certificate, _ [][]*x509.Certificate) error {
				executeCount++
				assert.Equal(template.SerialNumber, actual.SerialNumber)
				return expectedErr
			})

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
		assert    = assert.New(t)
		serverTLS TLS
	)

	tlsConfig, err := NewTLSConfig(&serverTLS)
	assert.NoError(err)
	assert.NotNil(tlsConfig)
}

func testNewTLSConfigMissingCertificate(t *testing.T) {
	var (
		assert    = assert.New(t)
		serverTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: "missing",
					KeyFile:         "missing",
				},
			},
		}
	)

	tlsConfig, err := NewTLSConfig(&serverTLS)
	assert.Error(err)
	assert.Nil(tlsConfig)
}

func testNewTLSConfigBasic(t *testing.T, certificateFile, keyFile string) {
	var (
		assert    = assert.New(t)
		require   = require.New(t)
		serverTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: certificateFile,
					KeyFile:         keyFile,
				},
			},
			MinVersion: 1,
			MaxVersion: 3,
		}
	)

	tlsConfig, err := NewTLSConfig(&serverTLS)
	require.NoError(err)
	require.NotNil(tlsConfig)

	assert.Equal(uint16(1), tlsConfig.MinVersion)
	assert.Equal(uint16(3), tlsConfig.MaxVersion)
	assert.Equal([]string{"http/1.1"}, tlsConfig.NextProtos)
	assert.Len(tlsConfig.Certificates, 1)
	assert.NotEmpty(tlsConfig.NameToCertificate) // verify that BuildNameToCertificate was run
	assert.Nil(tlsConfig.VerifyPeerCertificate)
	assert.Equal(tls.NoClientCert, tlsConfig.ClientAuth)
}

func testNewTLSConfigCustomNextProtos(t *testing.T, certificateFile, keyFile string) {
	var (
		assert    = assert.New(t)
		require   = require.New(t)
		serverTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: certificateFile,
					KeyFile:         keyFile,
				},
			},
			MinVersion: 1,
			MaxVersion: 3,
			NextProtos: []string{"http", "ftp"},
		}
	)

	tlsConfig, err := NewTLSConfig(&serverTLS)
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

func testNewTLSConfigVerifyPeerCertificate(t *testing.T, certificateFile, keyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		random   = rand.New(rand.NewSource(828675))
		template = &x509.Certificate{
			DNSNames:     []string{"foobar.example.com"},
			SerialNumber: big.NewInt(356493746),
		}

		serverTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: certificateFile,
					KeyFile:         keyFile,
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

	tlsConfig, err := NewTLSConfig(&serverTLS, extra)
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

func testNewTLSConfigClientCACertificateFile(t *testing.T, certificateFile, keyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: certificateFile,
					KeyFile:         keyFile,
				},
			},
			ClientCAs:  ExternalCertPool{certificateFile}, // this works as a bundle also
			MinVersion: 1,
			MaxVersion: 3,
			NextProtos: []string{"http", "ftp"},
		}
	)

	tlsConfig, err := NewTLSConfig(&serverTLS)
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
}

func testNewTLSConfigClientCACertificateFileMissing(t *testing.T, certificateFile, keyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		serverTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: certificateFile,
					KeyFile:         keyFile,
				},
			},
			ClientCAs: ExternalCertPool{"missing"},
		}
	)

	tlsConfig, err := NewTLSConfig(&serverTLS)
	assert.Error(err)
	require.Nil(tlsConfig)
}

func testNewTLSConfigClientCACertificateFileUnparseable(t *testing.T, certificateFile, keyFile string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	clientCACertificateFile, err := ioutil.TempFile("", "server*unparseable.cert")
	require.NoError(err)

	_, err = clientCACertificateFile.Write([]byte("unparseable"))
	require.NoError(err)

	defer os.Remove(clientCACertificateFile.Name())
	clientCACertificateFile.Close()

	var (
		serverTLS = TLS{
			Certificates: ExternalCertificates{
				{
					CertificateFile: certificateFile,
					KeyFile:         keyFile,
				},
			},
			ClientCAs: ExternalCertPool{clientCACertificateFile.Name()},
		}
	)

	tlsConfig, err := NewTLSConfig(&serverTLS)
	assert.Error(err)
	require.Nil(tlsConfig)
}

func TestNewTLSConfig(t *testing.T) {
	t.Run("Nil", testNewTLSConfigNil)
	t.Run("NoCertificate", testNewTLSConfigNoCertificate)
	t.Run("MissingCertificate", testNewTLSConfigMissingCertificate)

	certificateFile, keyFile := createServerFiles(t)
	defer os.Remove(certificateFile)
	defer os.Remove(keyFile)

	t.Run("Basic", func(t *testing.T) {
		testNewTLSConfigBasic(t, certificateFile, keyFile)
	})

	t.Run("CustomNextProtos", func(t *testing.T) {
		testNewTLSConfigCustomNextProtos(t, certificateFile, keyFile)
	})

	t.Run("VerifyPeerCertificate", func(t *testing.T) {
		testNewTLSConfigVerifyPeerCertificate(t, certificateFile, keyFile)
	})

	t.Run("ClientCACertificateFile", func(t *testing.T) {
		testNewTLSConfigClientCACertificateFile(t, certificateFile, keyFile)
	})

	t.Run("ClientCACertificateFileMissing", func(t *testing.T) {
		testNewTLSConfigClientCACertificateFileMissing(t, certificateFile, keyFile)
	})

	t.Run("ClientCACertificateFileUnparseable", func(t *testing.T) {
		testNewTLSConfigClientCACertificateFileUnparseable(t, certificateFile, keyFile)
	})
}
