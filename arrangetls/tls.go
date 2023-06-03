package arrangetls

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

	// strongCipherSuites are the tls.CipherSuite values that are safe for TLS versions less than 1.3
	strongCipherSuites = []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	}
)

// PeerVerifyError represents a verification error for a particular certificate
type PeerVerifyError struct {
	Certificate *x509.Certificate
	Reason      string
}

// Error satisfies the error interface.  It returns the Reason text.
func (pve *PeerVerifyError) Error() string {
	return pve.Reason
}

// PeerVerifier is a verification strategy for a peer certificate.
type PeerVerifier func(*x509.Certificate, [][]*x509.Certificate) error

// PeerVerifiers is an immutable sequence of PeerVerifiers.  The zero value
// is an empty sequence.
type PeerVerifiers struct {
	v []PeerVerifier
}

// NewPeerVerifiers returns a PeerVerifiers given a sequence of strategies
func NewPeerVerifiers(more ...PeerVerifier) PeerVerifiers {
	return PeerVerifiers{
		v: append([]PeerVerifier{}, more...),
	}
}

// Append adds more PeerVerifier strategies to this slice and
// returns the result.  If no PeerVerifier strategies are supplied,
// this method returns this PeerVerifiers as is.  Otherwise, the
// returned instance is a distinct sequence which is the concatenation
// of this instance with this method's arguments.
func (pvs PeerVerifiers) Append(more ...PeerVerifier) PeerVerifiers {
	if len(more) > 0 {
		return PeerVerifiers{
			v: append(
				append([]PeerVerifier{}, pvs.v...),
				more...,
			),
		}
	}

	return pvs
}

// Extend adds another sequence of PeerVerifiers to this one, and returns the result
func (pvs PeerVerifiers) Extend(more PeerVerifiers) PeerVerifiers {
	return pvs.Append(more.v...)
}

// VerifyPeerCertificate may be used as the closure for crypto/tls.Config.VerifyPeerCertificate.
// Parsing is done once, then each PeerVerifier is invoked in sequence.  Any error short-circuits
// subsequent checks.
func (pvs PeerVerifiers) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(pvs.v) == 0 {
		return nil
	}

	for _, rawCert := range rawCerts {
		peerCert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return err
		}

		for _, pv := range pvs.v {
			if err := pv(peerCert, verifiedChains); err != nil {
				return err
			}
		}
	}

	return nil
}

// SetTo conditinally configures tls.Config.VerifyPeerCertificate.  If the supplied tls.Config
// is not nil and this sequence is not empty, tls.Config.VerifyPeerCertificate is set to this
// sequence's VerifyPeerCertificate method.  Otherwise, this method does nothing.
//
// Note that PeerVerifiers is immutable.  Any tls.Config.VerifyPeerCertificate that is set
// will be unaffected by any future use of this PeerVerifiers sequence.
func (pvs PeerVerifiers) SetTo(tc *tls.Config) {
	if tc != nil && len(pvs.v) > 0 {
		tc.VerifyPeerCertificate = pvs.VerifyPeerCertificate
	}
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

	return &PeerVerifyError{
		Certificate: peerCert,
		Reason:      "No DNS name or common name matched",
	}
}

// AppendTo adds a peer verifier to the supplied sequence if and only if
// this config instance is not nil and if at least one of its fields
// is configured.
func (pvc *PeerVerifyConfig) AppendTo(pvs PeerVerifiers) PeerVerifiers {
	if pvc != nil {
		if v := pvc.Verifier(); v != nil {
			pvs = pvs.Append(v)
		}
	}

	return pvs
}

// ExternalCertificate represents a certificate with its key file on the filesystem.
// A server or client may have one or more associated external certificates.
type ExternalCertificate struct {
	CertificateFile string
	KeyFile         string
}

// Load reads in the certificate and key files from the file system
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

// Appends adds external certificates to this sequence
func (ecs *ExternalCertificates) Append(more ...ExternalCertificate) {
	*ecs = append(*ecs, more...)
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

// ExternalCertPool is a sequence of file names containing PEM-encoded certificates
// or certificate bundles to be added to an x509.CertPool
type ExternalCertPool []string

// Len returns the number of external files in this pool
func (ecp ExternalCertPool) Len() int {
	return len(ecp)
}

// Appends adds file names to this external cert pool
func (ecp *ExternalCertPool) Append(more ...string) {
	*ecp = append(*ecp, more...)
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

// Config represents the unmarshaled tls options for either a client or a server
type Config struct {
	// Certificates is the set of certificates to present to a client.  This field is
	// required for servers, and optional for clients.
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

	// MinVersion is the minimum required TLS version.  If unset, the internal crypto/tls default is used.
	MinVersion uint16

	// MaxVersion is the maximum required TLS version.  If unset, the internal crypto/tls default is used.
	MaxVersion uint16

	// PeerVerify specifies the certificate validation done on client certificates.
	// If supplied, this verifier strategy is merged with any extra PeerVerifiers
	// supplied in application code.
	PeerVerify *PeerVerifyConfig
}

// nextProtos returns the appropriate next protocols for the TLS handshake.  By default, http/1.1 is used.
func (c *Config) nextProtos() []string {
	nextProtos := append([]string{}, c.NextProtos...)
	if len(nextProtos) == 0 {
		// assume http/1.1 by default
		nextProtos = append(nextProtos, "http/1.1")
	}

	return nextProtos
}

// enforceVersions ensures certain constraints on the TLS version are met.
func (c *Config) enforceVersions(tc *tls.Config) {
	// If MinVersion was unset in configuration, explicitly establish it as 1.3.
	// This is different from the default crypto/tls behavior, as that package
	// defaults to 1.0 if MinVersion is unset.
	if tc.MinVersion == 0 {
		tc.MinVersion = tls.VersionTLS13
	}

	// If MaxVersion is set and less than MinVersion, set it explicitly to MinVersion.
	// We don't need to worry about the case where MaxVersion is unset, as crypto/tls
	// uses 1.3 in that case.
	if tc.MaxVersion != 0 && tc.MaxVersion < tc.MinVersion {
		tc.MaxVersion = tc.MinVersion
	}
}

// peerVerifiers configures the application-layer peer verifier code.
func (c *Config) peerVerifiers(tc *tls.Config, extra ...PeerVerifier) {
	var pvs PeerVerifiers
	pvs = c.PeerVerify.AppendTo(pvs)
	pvs = pvs.Append(extra...)
	pvs.SetTo(tc)
}

// certificates configures the TLS certificates defined in this configuration.
func (c *Config) certificates(tc *tls.Config) error {
	if certs, err := c.Certificates.AppendTo(nil); err != nil {
		return err
	} else {
		tc.Certificates = certs
	}

	if c.RootCAs.Len() > 0 {
		rootCAs := x509.NewCertPool()
		if count, err := c.RootCAs.AppendTo(rootCAs); err != nil {
			return err
		} else if count > 0 {
			tc.RootCAs = rootCAs
		}
	}

	if c.ClientCAs.Len() > 0 {
		clientCAs := x509.NewCertPool()
		if count, err := c.ClientCAs.AppendTo(clientCAs); err != nil {
			return err
		} else if count > 0 {
			tc.ClientCAs = clientCAs
			tc.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	// NOTE: This method is deprecated, but in order not to break
	// older code we call it here, for now.
	tc.BuildNameToCertificate() //nolint:staticcheck
	return nil
}

// New constructs a *tls.Config from this Config instance, usually unmarshaled
// from some external source.  If this instance is nil, it returns nil with no error.
//
// The extra PeerVerifiers, if supplied, are used to build the tls.Config.VerifyPeerCertificate
// strategy.
func (c *Config) New(extra ...PeerVerifier) (*tls.Config, error) {
	if c == nil {
		return nil, nil
	}

	tc := &tls.Config{
		MinVersion:         c.MinVersion,
		MaxVersion:         c.MaxVersion,
		NextProtos:         c.nextProtos(),
		ServerName:         c.ServerName,
		InsecureSkipVerify: c.InsecureSkipVerify, //nolint:gosec // the caller set this explicitly

		// always use the strong cipher suites for tls versions < 1.3
		CipherSuites: strongCipherSuites,
	}

	c.enforceVersions(tc)
	c.peerVerifiers(tc, extra...)
	if err := c.certificates(tc); err != nil {
		return nil, err
	}

	return tc, nil
}
