package arrangehttp

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"strings"
)

var (
	ErrTLSCertificateRequired         = errors.New("Both a certificateFile and keyFile are required")
	ErrUnableToAddClientCACertificate = errors.New("Unable to add client CA certificate")
)

// PeerVerifyError represents a verification error for a particular certificate
type PeerVerifyError struct {
	Certificate *x509.Certificate
	Reason      string
}

func (pve PeerVerifyError) Error() string {
	return pve.Reason
}

// PeerVerifier is a verification strategy for a peer certificate.
type PeerVerifier func(*x509.Certificate, [][]*x509.Certificate) error

// PeerVerifiers is a sequence of PeerVerifier objects.  This type handles
// parsing a certificate once, then invoking each PeerVerifier.
type PeerVerifiers []PeerVerifier

// Len returns the count of verifiers in this slice
func (pvs PeerVerifiers) Len() int {
	return len(pvs)
}

// Append tacks on more PeerVerifier strategies to this slice
func (pvs *PeerVerifiers) Append(more ...PeerVerifier) {
	*pvs = append(*pvs, more...)
}

// VerifyPeerCertificate may be used as the closure for crypto/tls.Config.VerifyPeerCertificate.
// Parsing is done once, then each PeerVerifier is invoked in sequence.  Any error short-circuits
// subsequent checks.
func (pvs PeerVerifiers) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(pvs) == 0 {
		return nil
	}

	for _, rawCert := range rawCerts {
		peerCert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return err
		}

		for _, pv := range pvs {
			if err := pv(peerCert, verifiedChains); err != nil {
				return err
			}
		}
	}

	return nil
}

// PeerVerifyConfig allows common checks against a client-side certificate to be configured externally.
// Any constraint that matches will result in a valid peer cert.
type PeerVerifyConfig struct {
	// DNSSuffixes enumerates any DNS suffixes that are checked.  A DNSName field of at least (1) peer cert
	// must have one of these suffixes.  If this field is not supplied, no DNS suffix checking is performed.
	// Matching is case insensitive.
	//
	// If any DNS suffix matches, that is sufficient for the peer cert to be valid.
	// No further checking is done in that case.
	DNSSuffixes []string

	// CommonNames lists the subject common names that at least (1) peer cert must have.  If not supplied,
	// no checking is done on the common name.  Matching common names is case sensitive.
	//
	// If any common name matches, that is sufficient for the peer cert to be valid.  No further
	// checking is done in that case.
	CommonNames []string
}

// Verifier produces a PeerVerifier strategy from these options.
// If nothing is configured, this method returns nil.
func (pvc PeerVerifyConfig) Verifier() PeerVerifier {
	if len(pvc.DNSSuffixes) > 0 || len(pvc.CommonNames) > 0 {
		// make a safe clone to host our closure
		var clone PeerVerifyConfig
		if len(pvc.DNSSuffixes) > 0 {
			clone.DNSSuffixes = make([]string, len(pvc.DNSSuffixes))
			for i, suffix := range pvc.DNSSuffixes {
				clone.DNSSuffixes[i] = strings.ToLower(suffix)
			}
		}

		if len(pvc.CommonNames) > 0 {
			clone.CommonNames = append(clone.CommonNames, pvc.CommonNames...)
		}

		return clone.verify
	}

	return nil
}

// verify is the PeerVerifier strategy that uses this configuration.
// This is typically invoked against a clone of the unmarshaled struct.
func (pvc PeerVerifyConfig) verify(peerCert *x509.Certificate, _ [][]*x509.Certificate) error {
	for _, suffix := range pvc.DNSSuffixes {
		for _, dnsName := range peerCert.DNSNames {
			if strings.HasSuffix(strings.ToLower(dnsName), suffix) {
				return nil
			}
		}

		// Allow the common name to be suffixed by a DNS suffix
		if strings.HasSuffix(strings.ToLower(peerCert.Subject.CommonName), suffix) {
			return nil
		}
	}

	for _, commonName := range pvc.CommonNames {
		if commonName == peerCert.Subject.CommonName {
			return nil
		}
	}

	return PeerVerifyError{
		Certificate: peerCert,
		Reason:      "No DNS name or common name matched",
	}
}

// ExternalCertificate represents a certificate with its key file on the filesystem.
// A server or client may have one or more associated external certificates.
type ExternalCertificate struct {
	CertificateFile string
	KeyFile         string
}

func (ec ExternalCertificate) Load() (tls.Certificate, error) {
	if len(ec.CertificateFile) > 0 && len(ec.KeyFile) > 0 {
		return tls.LoadX509KeyPair(ec.CertificateFile, ec.KeyFile)
	}

	return tls.Certificate{}, ErrTLSCertificateRequired
}

// ExternalCertificates is a sequence of externally available certificates
type ExternalCertificates []ExternalCertificate

// Len returns the count of externally available certificates in this slice
func (ecs ExternalCertificates) Len() int {
	return len(ecs)
}

// AppendTo loads and appends each certificate in this slice.  Any error short
// circuits and returns that error together with the slice with any successfully
// loaded certificates.
func (ecs ExternalCertificates) AppendTo(certs []tls.Certificate) ([]tls.Certificate, error) {
	for _, ec := range ecs {
		cert, err := ec.Load()
		if err != nil {
			return certs, err
		}

		certs = append(certs, cert)
	}

	return certs, nil
}

// ExternalCertPool is a sequence of file names containing certificates to
// be added to an x509.CertPool.  Each file name must be PEM-encoded.
type ExternalCertPool []string

// Len returns the number of external files in this pool
func (ecp ExternalCertPool) Len() int {
	return len(ecp)
}

// AppendTo adds each PEM-encoded file from this external pool to the given
// x509.CertPool.  The number of certs added is returned, and any error will
// short circuit subsequent loading.
func (ecp ExternalCertPool) AppendTo(pool *x509.CertPool) (int, error) {
	var loaded int
	for _, ec := range ecp {
		pemCert, err := ioutil.ReadFile(ec)
		if err != nil {
			return loaded, err
		}

		if pool.AppendCertsFromPEM(pemCert) {
			loaded++
		} else {
			return loaded, ErrUnableToAddClientCACertificate
		}
	}

	return loaded, nil
}

// TLS represents the unmarshaled tls options for either a client or a server
type TLS struct {
	// Certificates is the required set of certificates to present to a client.  There must
	// be at least one entry in this slice.
	Certificates ExternalCertificates

	// RootCAs is the optional certificate pool for root certificates.  By default, the golang
	// library uses the system certificate pool if this is unset.
	RootCAs ExternalCertPool

	// ClientCAs is the optional certificate pool for certificates expected from a client.  Configure
	// this as part of mTLS.
	ClientCAs ExternalCertPool

	// ServerName is used by a client to validate the server's hostname.  This field is optional
	// and has no default.
	ServerName string

	// InsecureSkipVerify indicates whether a client should validate a server's certificate(s)
	InsecureSkipVerify bool

	// NextProtos is the list of supported application protocols.  Defaults to "http/1.1" if unset.
	NextProtos []string

	// MinVersion is the minimum required TLS version
	MinVersion uint16

	// MaxVersion is the maximum required TLS version
	MaxVersion uint16

	// PeerVerify specifies the certificate validation done on client certificates.
	// If supplied, this verifier strategy is merged with an extra PeerVerifiers
	// supplied in application code.
	PeerVerify *PeerVerifyConfig
}

// NewTLSConfig constructs a *tls.Config from an unmarshaled TLS instance.
// If the supplied TLS is nil, this method returns nil with no error.
//
// The extra PeerVerifiers, if supplied, are used to build the tls.Config.VerifyPeerCertificate
// strategy.
func NewTLSConfig(t *TLS, extra ...PeerVerifier) (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}

	var nextProtos []string
	if len(t.NextProtos) > 0 {
		for _, np := range t.NextProtos {
			nextProtos = append(nextProtos, np)
		}
	} else {
		// assume http/1.1 by default
		nextProtos = append(nextProtos, "http/1.1")
	}

	tc := &tls.Config{
		MinVersion:         t.MinVersion,
		MaxVersion:         t.MaxVersion,
		NextProtos:         nextProtos,
		ServerName:         t.ServerName,
		InsecureSkipVerify: t.InsecureSkipVerify,
	}

	var peerVerifiers PeerVerifiers
	if t.PeerVerify != nil {
		// A PeerVerifyConfig can return a nil function if nothing is configured
		if v := t.PeerVerify.Verifier(); v != nil {
			peerVerifiers.Append(v)
		}
	}

	peerVerifiers.Append(extra...)
	if peerVerifiers.Len() > 0 {
		tc.VerifyPeerCertificate = peerVerifiers.VerifyPeerCertificate
	}

	if certs, err := t.Certificates.AppendTo(nil); err != nil {
		return nil, err
	} else {
		tc.Certificates = certs
	}

	if t.RootCAs.Len() > 0 {
		rootCAs := x509.NewCertPool()
		if count, err := t.RootCAs.AppendTo(rootCAs); err != nil {
			return nil, err
		} else if count > 0 {
			tc.RootCAs = rootCAs
		}
	}

	if t.ClientCAs.Len() > 0 {
		clientCAs := x509.NewCertPool()
		if count, err := t.ClientCAs.AppendTo(clientCAs); err != nil {
			return nil, err
		} else if count > 0 {
			tc.ClientCAs = clientCAs
			tc.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	tc.BuildNameToCertificate()
	return tc, nil
}
