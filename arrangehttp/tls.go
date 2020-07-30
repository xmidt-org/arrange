package arrangehttp

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io/ioutil"
	"strings"
)

var (
	ErrTlsCertificateRequired         = errors.New("Both a certificateFile and keyFile are required")
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
	// If any common name matches, that is sufficient for the peer cert to be valid.  No further checking is done in that case.
	CommonNames []string
}

// Verifier produces a PeerVerifier strategy from these options.
// If nothing is configured, this method returns nil.
func (pvc PeerVerifyConfig) Verifier() PeerVerifier {
	if len(pvc.DNSSuffixes) == 0 && len(pvc.CommonNames) == 0 {
		return nil
	}

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

// ServerTls represents the set of configurable options for a serverside tls.Config associated with a server.
type ServerTls struct {
	CertificateFile         string
	KeyFile                 string
	ClientCACertificateFile string
	ServerName              string
	NextProtos              []string
	MinVersion              uint16
	MaxVersion              uint16
	PeerVerify              PeerVerifyConfig
}

// NewServerTlsConfig produces a *tls.Config from a set of configuration options.  If the supplied set of options
// is nil, this function returns nil with no error.
//
// If supplied, the PeerVerifier strategies will be executed as part of peer verification.  This allows application-layer
// logic to be injected.
func NewServerTlsConfig(t *ServerTls, extra ...PeerVerifier) (*tls.Config, error) {
	if t == nil {
		return nil, nil
	}

	if len(t.CertificateFile) == 0 || len(t.KeyFile) == 0 {
		return nil, ErrTlsCertificateRequired
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
		MinVersion: t.MinVersion,
		MaxVersion: t.MaxVersion,
		ServerName: t.ServerName,
		NextProtos: nextProtos,
	}

	var peerVerifiers PeerVerifiers
	if pv := t.PeerVerify.Verifier(); pv != nil {
		peerVerifiers = append(peerVerifiers, pv)
	}

	peerVerifiers = append(peerVerifiers, extra...)
	if len(peerVerifiers) > 0 {
		tc.VerifyPeerCertificate = peerVerifiers.VerifyPeerCertificate
	}

	if cert, err := tls.LoadX509KeyPair(t.CertificateFile, t.KeyFile); err != nil {
		return nil, err
	} else {
		tc.Certificates = []tls.Certificate{cert}
	}

	if len(t.ClientCACertificateFile) > 0 {
		caCert, err := ioutil.ReadFile(t.ClientCACertificateFile)
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, ErrUnableToAddClientCACertificate
		}

		tc.ClientCAs = caCertPool
		tc.ClientAuth = tls.RequireAndVerifyClientCert
	}

	tc.BuildNameToCertificate()
	return tc, nil
}
