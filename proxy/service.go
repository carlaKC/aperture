package proxy

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/lightninglabs/kirin/auth"
	"github.com/lightninglabs/kirin/freebie"
)

var (
	filePrefix       = "!file"
	filePrefixHex    = filePrefix + "+hex"
	filePrefixBase64 = filePrefix + "+base64"
)

// Service generically specifies configuration data for backend services to the
// Kirin proxy.
type Service struct {
	// TLSCertPath is the optional path to the service's TLS certificate.
	TLSCertPath string `long:"tlscertpath" description:"Path to the service's TLS certificate"`

	// Address is the service's IP address and port.
	Address string `long:"address" description:"service instance rpc address"`

	// Protocol is the protocol that should be used to connect to the
	// service. Currently supported is http and https.
	Protocol string `long:"protocol" description:"service instance protocol"`

	// Auth is the authentication level required for this service to be
	// accessed. Valid values are "on" for full authentication, "freebie X"
	// for X free requests per IP address before authentication is required
	// or "off" for no authentication.
	Auth auth.Level `long:"auth" description:"required authentication"`

	// HostRegexp is a regular expression that is tested against the 'Host'
	// HTTP header field to find out if this service should be used.
	HostRegexp string `long:"hostregexp" description:"Regular expression to match the host against"`

	// PathRegexp is a regular expression that is tested against the path
	// of the URL of a request to find out if this service should be used.
	PathRegexp string `long:"pathregexp" description:"Regular expression to match the path of the URL against"`

	// Headers is a map of strings that defines header name and values that
	// should always be passed to the backend service, overwriting any
	// headers with the same name that might have been set by the client
	// request.
	// If the value of a header field starts with the prefix "!file+hex:",
	// the rest of the value is treated as a path to a file and the content
	// of that file is sent to the backend with each call (hex encoded).
	// If the value starts with the prefix "!file+base64:", the content of
	// the file is sent encoded as base64.
	Headers map[string]string `long:"headers" description:"Header fields to always pass to the service"`

	freebieDb freebie.DB
}

// prepareServices prepares the backend service configurations to be used by the
// proxy.
func prepareServices(services []*Service) error {
	for _, service := range services {
		// Each freebie enabled service gets its own store.
		if service.Auth.IsFreebie() {
			service.freebieDb = freebie.NewMemIpMaskStore(
				service.Auth.FreebieCount(),
			)
		}

		// Replace placeholders/directives in the header fields with the
		// actual desired values.
		for key, value := range service.Headers {
			if !strings.HasPrefix(value, filePrefix) {
				continue
			}

			parts := strings.Split(value, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid header config, " +
					"must be '!file+hex:path'")
			}
			prefix, fileName := parts[0], parts[1]
			bytes, err := ioutil.ReadFile(fileName)
			if err != nil {
				return err
			}

			// There are two supported formats to encode the file
			// content in: hex and base64.
			switch {
			case prefix == filePrefixHex:
				newValue := hex.EncodeToString(bytes)
				service.Headers[key] = newValue

			case prefix == filePrefixBase64:
				newValue := base64.StdEncoding.EncodeToString(
					bytes,
				)
				service.Headers[key] = newValue

			default:
				return fmt.Errorf("unsupported file prefix "+
					"format %s", value)
			}
		}
	}
	return nil
}
