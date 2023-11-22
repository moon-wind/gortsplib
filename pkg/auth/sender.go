package auth

import (
	"fmt"
	"strings"

	"github.com/moon-wind/gortsplib/pkg/base"
	"github.com/moon-wind/gortsplib/pkg/headers"
)

func findHeader(v base.HeaderValue, prefix string) string {
	for _, vi := range v {
		if strings.HasPrefix(vi, prefix) {
			return vi
		}
	}
	return ""
}

// Sender allows to send credentials.
type Sender struct {
	user   string
	pass   string
	method headers.AuthMethod
	realm  string
	nonce  string
}

// NewSender allocates a Sender.
// It requires a WWW-Authenticate header (provided by the server)
// and a set of credentials.
func NewSender(v base.HeaderValue, user string, pass string) (*Sender, error) {
	// prefer digest
	if v0 := findHeader(v, "Digest"); v0 != "" {
		var auth headers.Authenticate
		err := auth.Unmarshal(base.HeaderValue{v0})
		if err != nil {
			return nil, err
		}

		if auth.Realm == nil {
			return nil, fmt.Errorf("realm is missing")
		}

		if auth.Nonce == nil {
			return nil, fmt.Errorf("nonce is missing")
		}

		return &Sender{
			user:   user,
			pass:   pass,
			method: headers.AuthDigest,
			realm:  *auth.Realm,
			nonce:  *auth.Nonce,
		}, nil
	}

	if v0 := findHeader(v, "Basic"); v0 != "" {
		var auth headers.Authenticate
		err := auth.Unmarshal(base.HeaderValue{v0})
		if err != nil {
			return nil, err
		}

		if auth.Realm == nil {
			return nil, fmt.Errorf("realm is missing")
		}

		return &Sender{
			user:   user,
			pass:   pass,
			method: headers.AuthBasic,
			realm:  *auth.Realm,
		}, nil
	}

	return nil, fmt.Errorf("no authentication methods available")
}

// AddAuthorization adds the Authorization header to a Request.
func (se *Sender) AddAuthorization(req *base.Request) {
	urStr := req.URL.CloneWithoutCredentials().String()

	h := headers.Authorization{
		Method: se.method,
	}

	switch se.method {
	case headers.AuthBasic:
		h.BasicUser = se.user
		h.BasicPass = se.pass

	default: // headers.AuthDigest
		response := md5Hex(md5Hex(se.user+":"+se.realm+":"+se.pass) + ":" +
			se.nonce + ":" + md5Hex(string(req.Method)+":"+urStr))

		h.DigestValues = headers.Authenticate{
			Method:   headers.AuthDigest,
			Username: &se.user,
			Realm:    &se.realm,
			Nonce:    &se.nonce,
			URI:      &urStr,
			Response: &response,
		}
	}

	if req.Header == nil {
		req.Header = make(base.Header)
	}

	req.Header["Authorization"] = h.Marshal()
}
