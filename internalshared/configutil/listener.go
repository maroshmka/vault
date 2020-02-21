package configutil

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/go-sockaddr"
	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/vault/sdk/helper/parseutil"
	"github.com/hashicorp/vault/sdk/helper/tlsutil"
)

type ListenerTelemetry struct {
	UnauthenticatedMetricsAccess bool `hcl:"unauthenticated_metrics_access"`
}

// Listener is the listener configuration for the server.
type Listener struct {
	rawConfig map[string]interface{}

	Type    string
	Purpose string

	Address               string        `hcl:"address"`
	ClusterAddress        string        `hcl:"cluster_address"`
	MaxRequestSize        int64         `hcl:"max_request_size"`
	MaxRequestDuration    time.Duration `hcl:"-"`
	MaxRequestDurationRaw interface{}   `hcl:"max_request_duration"`
	RequireRequestHeader  bool          `hcl:"require_request_header"`

	TLSDisable                    bool     `hcl:"tls_disable"`
	TLSCertFile                   string   `hcl:"tls_cert_file"`
	TLSKeyFile                    string   `hcl:"tls_key_file"`
	TLSMinVersion                 string   `hcl:"tls_min_version`
	TLSCipherSuites               []uint16 `hcl:"-"`
	TLSCipherSuitesRaw            string   `hcl:"tls_cipher_suites"`
	TLSPreferServerCipherSuites   bool     `hcl:"tls_prefer_server_cipher_suites"`
	TLSRequireAndVerifyClientCert bool     `hcl:"tls_require_and_verify_client_cert"`
	TLSClientCAFile               string   `hcl:"tls_client_ca_file"`
	TLSDisableClientCerts         bool     `hcl:"tls_disable_client_certs"`

	HTTPReadTimeout          time.Duration `hcl:"-"`
	HTTPReadTimeoutRaw       interface{}   `hcl:"http_read_timeout"`
	HTTPReadHeaderTimeout    time.Duration `hcl:"-"`
	HTTPReadHeaderTimeoutRaw interface{}   `hcl:"http_read_header_timeout"`
	HTTPWriteTimeout         time.Duration `hcl:"-"`
	HTTPWriteTimeoutRaw      interface{}   `hcl:"http_write_timeout"`
	HTTPIdleTimeout          time.Duration `hcl:"-"`
	HTTPIdleTimeoutRaw       interface{}   `hcl:"http_idle_timeout"`

	ProxyProtocolBehavior           string                        `hcl:"proxy_protocol_behavior"`
	ProxyProtocolAuthorizedAddrs    []*sockaddr.SockAddrMarshaler `hcl:"-"`
	ProxyProtocolAuthorizedAddrsRaw interface{}                   `hcl:"proxy_protocol_authorized_addrs"`

	XForwardedForAuthorizedAddrs     []*sockaddr.SockAddrMarshaler `hcl:"-"`
	XForwardedForAuthorizedAddrsRaw  interface{}                   `hcl:"x_forwarded_for_authorized_addrs"`
	XForwardedForHopSkips            int                           `hcl:"x_forwarded_for_hop_skips"`
	XForwardedForRejectNotPresent    bool                          `hcl:"x_forwarded_for_reject_not_present"`
	XForwardedForRejectNotAuthorized bool                          `hcl:"x_forwarded_for_reject_not_authorized"`

	SocketMode  string `hcl:"socket_mode"`
	SocketUser  string `hcl:"socket_user"`
	SocketGroup string `hcl:"socket_group"`

	Telemetry ListenerTelemetry `hcl:"telemetry"`
}

func (l *Listener) GoString() string {
	return fmt.Sprintf("*%#v", *l)
}

func ParseListeners(result *SharedConfig, list *ast.ObjectList) error {
	var err error
	result.Listeners = make([]*Listener, 0, len(list.Items))
	for _, item := range list.Items {
		key := "listener"
		if len(item.Keys) > 0 {
			key = item.Keys[0].Token.Value().(string)
		}

		var l Listener
		if err := hcl.DecodeObject(&l, item.Val); err != nil {
			return multierror.Prefix(err, fmt.Sprintf("listeners.%s:", key))
		}

		// Hacky way, for now, to get the values we want for sanitizing
		var m map[string]interface{}
		if err := hcl.DecodeObject(&m, item.Val); err != nil {
			return multierror.Prefix(err, fmt.Sprintf("listeners.%s:", key))
		}
		l.rawConfig = m

		l.Type = strings.ToLower(key)
		switch l.Type {
		case "tcp", "unix":
		default:
			return multierror.Prefix(fmt.Errorf("unsupported listener type %q", l.Type), fmt.Sprintf("listeners.%s:", key))
		}

		// Request Parameters
		{
			if l.MaxRequestSize < 0 {
				return multierror.Prefix(errors.New("max_request_size cannot be negative"), fmt.Sprintf("listeners.%s", key))
			}

			if l.MaxRequestDurationRaw != nil {
				if l.MaxRequestDuration, err = parseutil.ParseDurationSecond(l.MaxRequestDurationRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("error parsing max_request_duration: %w", err), fmt.Sprintf("listeners.%s", key))
				}
				if l.MaxRequestDuration < 0 {
					return multierror.Prefix(errors.New("max_request_duration cannot be negative"), fmt.Sprintf("listeners.%s", key))
				}
			}
		}

		// TLS Parameters
		{
			if l.TLSCipherSuitesRaw != "" {
				if l.TLSCipherSuites, err = tlsutil.ParseCiphers(l.TLSCipherSuitesRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("invalid value for tls_cipher_suites: %w", err), fmt.Sprintf("listeners.%s", key))
				}
			}
		}

		// HTTP timeouts
		{
			if l.HTTPReadTimeoutRaw != nil {
				if l.HTTPReadTimeout, err = parseutil.ParseDurationSecond(l.HTTPReadTimeoutRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("error parsing http_read_timeout: %w", err), fmt.Sprintf("listeners.%s", key))
				}
			}

			if l.HTTPReadHeaderTimeoutRaw != nil {
				if l.HTTPReadHeaderTimeout, err = parseutil.ParseDurationSecond(l.HTTPReadHeaderTimeoutRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("error parsing http_read_header_timeout: %w", err), fmt.Sprintf("listeners.%s", key))
				}
			}

			if l.HTTPWriteTimeoutRaw != nil {
				if l.HTTPWriteTimeout, err = parseutil.ParseDurationSecond(l.HTTPWriteTimeoutRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("error parsing http_write_timeout: %w", err), fmt.Sprintf("listeners.%s", key))
				}
			}

			if l.HTTPIdleTimeoutRaw != nil {
				if l.HTTPIdleTimeout, err = parseutil.ParseDurationSecond(l.HTTPIdleTimeoutRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("error parsing http_idle_timeout: %w", err), fmt.Sprintf("listeners.%s", key))
				}
			}
		}

		// Proxy Protocol config
		{
			if l.ProxyProtocolAuthorizedAddrsRaw != nil {
				if l.ProxyProtocolAuthorizedAddrs, err = parseutil.ParseAddrs(l.ProxyProtocolAuthorizedAddrsRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("error parsing proxy_protocol_authorized_addrs: %w", err), fmt.Sprintf("listeners.%s", key))
				}

				switch l.ProxyProtocolBehavior {
				case "allow_authorized", "deny_authorized":
					if len(l.ProxyProtocolAuthorizedAddrs) == 0 {
						return multierror.Prefix(errors.New("proxy_protocol_behavior set to allow or deny only authorized addresses but no proxy_protocol_authorized_addrs value"), fmt.Sprintf("listeners.%s", key))
					}
				}
			}
		}

		// X-Forwarded-For config
		{
			if l.XForwardedForAuthorizedAddrsRaw != nil {
				if l.XForwardedForAuthorizedAddrs, err = parseutil.ParseAddrs(l.XForwardedForAuthorizedAddrsRaw); err != nil {
					return multierror.Prefix(fmt.Errorf("error parsing x_forwarded_for_authorized_addrs: %w", err), fmt.Sprintf("listeners.%s", key))
				}
			}

			if l.XForwardedForHopSkips < 0 {
				return multierror.Prefix(fmt.Errorf("x_forwarded_for_hop_skips cannot be negative but set to %d", l.XForwardedForHopSkips), fmt.Sprintf("listeners.%s", key))
			}
		}

		result.Listeners = append(result.Listeners, &l)
	}

	return nil
}
