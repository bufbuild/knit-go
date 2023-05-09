package knitgateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sort"
	"strings"
)

// These constants and vars are the defaults for crypto/tls package.
// But we set them explicitly just in case the defaults ever change.
// We don't want defaults changing without realizing it.

const (
	defaultServerMinTLSVersion = tls.VersionTLS10
	defaultClientMinTLSVersion = tls.VersionTLS12
)

var (
	defaultCipherSuites = []uint16{
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
	}

	allCipherSuites = map[string]uint16{
		// enabled by default:
		"TLS_RSA_WITH_AES_128_CBC_SHA":                  tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":                  tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":          tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		// disabled by default:
		"TLS_RSA_WITH_RC4_128_SHA":                tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	}
)

// ContextWithClientCert stores the given client certificate in the given context. This is set
// when handling an incoming request, allowing the client's identity to be sent to backends
// in request headers.
func ContextWithClientCert(ctx context.Context, clientCert *x509.Certificate) context.Context {
	return context.WithValue(ctx, clientCertContextKey{}, clientCert)
}

func clientCertFromContext(ctx context.Context) *x509.Certificate {
	cert, _ := ctx.Value(clientCertContextKey{}).(*x509.Certificate)
	return cert
}

type clientCertContextKey struct{}

func configureServerTLS(conf *externalServerTLSConfig) (*tls.Config, error) {
	if conf == nil {
		// not using TLS if no 'listen.tls' stanza provided
		return nil, nil
	}
	var tlsConf tls.Config
	if conf.Cert == "" || conf.Key == "" {
		return nil, fmt.Errorf("both 'cert' and 'key' properties must be specified")
	}
	certs, err := loadCertificateAndKey(conf.Cert, conf.Key)
	if err != nil {
		return nil, err
	}
	tlsConf.Certificates = certs

	if conf.MinVersion == "" {
		tlsConf.MinVersion = defaultServerMinTLSVersion
	} else {
		var err error
		tlsConf.MinVersion, err = parseTLSVersion(conf.MinVersion)
		if err != nil {
			return nil, err
		}
	}
	if conf.Ciphers == nil {
		// This is the default for crypto/tls. But we set it explicitly just
		// in case the default ever changes. We don't want our defaults
		// changing without realizing it.
		tlsConf.CipherSuites = defaultCipherSuites
	} else {
		var err error
		tlsConf.CipherSuites, err = parseCipherSuites(conf.Ciphers)
		if err != nil {
			return nil, err
		}
	}
	if conf.ClientCerts != nil {
		if conf.ClientCerts.Require {
			tlsConf.ClientAuth = tls.RequireAndVerifyClientCert
		} else {
			tlsConf.ClientAuth = tls.VerifyClientCertIfGiven
		}
		if conf.ClientCerts.CACert == "" {
			return nil, fmt.Errorf("'client_certs' config is enabled but 'ca_cert' property is missing")
		}
		var err error
		tlsConf.ClientCAs, err = loadCertificatePool(conf.ClientCerts.CACert)
		if err != nil {
			return nil, fmt.Errorf("'client_certs.cacert' config: %w", err)
		}
	}

	return &tlsConf, nil
}

func configureClientTLS(conf *externalClientTLSConfig) (*tls.Config, error) {
	var tlsConf tls.Config
	if conf == nil {
		tlsConf.MinVersion = defaultClientMinTLSVersion
		tlsConf.CipherSuites = defaultCipherSuites
		return &tlsConf, nil
	}

	if (conf.Cert == nil) != (conf.Key == nil) {
		return nil, fmt.Errorf("if either 'cert' and 'key' is specified then both must be specified")
	}
	if conf.Cert != nil {
		certs, err := loadCertificateAndKey(*conf.Cert, *conf.Key)
		if err != nil {
			return nil, err
		}
		tlsConf.Certificates = certs
	}

	if conf.MinVersion == nil {
		tlsConf.MinVersion = defaultClientMinTLSVersion
	} else {
		var err error
		tlsConf.MinVersion, err = parseTLSVersion(*conf.MinVersion)
		if err != nil {
			return nil, err
		}
	}
	if conf.Ciphers == nil {
		// This is the default for crypto/tls. But we set it explicitly just
		// in case the default ever changes. We don't want our defaults
		// changing without realizing it.
		tlsConf.CipherSuites = defaultCipherSuites
	} else {
		var err error
		tlsConf.CipherSuites, err = parseCipherSuites(conf.Ciphers)
		if err != nil {
			return nil, err
		}
	}

	if conf.SkipVerify != nil && *conf.SkipVerify {
		tlsConf.InsecureSkipVerify = true
	} else if conf.CACert != nil {
		var err error
		tlsConf.RootCAs, err = loadCertificatePool(*conf.CACert)
		if err != nil {
			return nil, fmt.Errorf("'cacert' config: %w", err)
		}
	}

	return &tlsConf, nil
}

func mergeClientTLSConfig(conf, defaults *externalClientTLSConfig) *externalClientTLSConfig {
	if conf == nil {
		return defaults
	}
	if defaults == nil {
		return conf
	}

	merged := *conf
	if merged.MinVersion == nil {
		merged.MinVersion = defaults.MinVersion
	}
	if merged.Ciphers == nil {
		merged.Ciphers = defaults.Ciphers
	}
	if merged.CACert == nil {
		merged.CACert = defaults.CACert
	}
	if merged.SkipVerify == nil {
		merged.SkipVerify = defaults.SkipVerify
	}
	if merged.Cert == nil {
		merged.Cert = defaults.Cert
	}
	if merged.Key == nil {
		merged.Key = defaults.Key
	}
	// ServerName property intentionally skipped since defaults should not include it

	return &merged
}

func parseTLSVersion(v string) (uint16, error) {
	switch v {
	case tlsVersion10:
		return tls.VersionTLS10, nil
	case tlsVersion11:
		return tls.VersionTLS11, nil
	case tlsVersion12:
		return tls.VersionTLS12, nil
	case tlsVersion13:
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("%q is not a valid TLS version; instead use %q, %q, %q, or %q",
			v, tlsVersion10, tlsVersion11, tlsVersion12, tlsVersion13)
	}
}

func parseCipherSuites(conf *externalCiphersConfig) ([]uint16, error) {
	switch {
	case conf.Allow != nil && conf.Disallow != nil:
		return nil, fmt.Errorf("ciphers config must specify either allow or disallow, but not both")
	case conf.Allow != nil:
		ciphers := make([]uint16, len(conf.Allow))
		for i, cipher := range conf.Allow {
			cipherID, ok := allCipherSuites[cipher]
			if !ok {
				return nil, fmt.Errorf("allowed cipher %q is not a valid cipher suite name", cipher)
			}
			ciphers[i] = cipherID
		}
		return ciphers, nil
	case conf.Disallow != nil:
		// Start with the set of all cipher suites
		cipherSet := make(map[uint16]struct{}, len(allCipherSuites))
		for _, v := range allCipherSuites {
			cipherSet[v] = struct{}{}
		}
		// remove the ones that are disallowed
		for _, cipher := range conf.Disallow {
			cipherID, ok := allCipherSuites[cipher]
			if !ok {
				return nil, fmt.Errorf("disallowed cipher %q is not a valid cipher suite name", cipher)
			}
			delete(cipherSet, cipherID)
		}
		// return the remaining ones as a slice
		ciphers := make([]uint16, 0, len(cipherSet))
		for id := range cipherSet {
			ciphers = append(ciphers, id)
		}
		sort.Slice(ciphers, func(i, j int) bool {
			return ciphers[i] < ciphers[j]
		})
		return ciphers, nil
	default:
		return nil, fmt.Errorf("ciphers config must specify either allow or disallow; neither was present")
	}
}

func loadCertificateAndKey(cert, key string) ([]tls.Certificate, error) {
	if cert == "" {
		return nil, fmt.Errorf("cert filename is empty")
	}
	if key == "" {
		return nil, fmt.Errorf("key filename is empty")
	}
	certData, err := os.ReadFile(cert)
	if err != nil {
		return nil, errorHasFilename(err, cert)
	}
	keyData, err := os.ReadFile(key)
	if err != nil {
		return nil, errorHasFilename(err, key)
	}
	certPair, err := tls.X509KeyPair(certData, keyData)
	if err != nil {
		return nil, err
	}
	certPair.Leaf, err = x509.ParseCertificate(certPair.Certificate[0])
	if err != nil {
		return nil, err
	}
	return []tls.Certificate{certPair}, nil
}

func loadCertificatePool(cacert string) (*x509.CertPool, error) {
	if cacert == "" {
		return nil, fmt.Errorf("filename is empty")
	}
	data, err := os.ReadFile(cacert)
	if err != nil {
		return nil, errorHasFilename(err, cacert)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("no certificates could be successfully parsed from file %s", cacert)
	}
	return pool, nil
}

func errorHasFilename(err error, filename string) error {
	if strings.Contains(err.Error(), filename) {
		return err
	}
	return fmt.Errorf("%s: %w", filename, err)
}
