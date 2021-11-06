package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
)

const JWTWellKnown = "https://token.actions.githubusercontent.com/.well-known/openid-configuration"

type WellKnownData struct {
	SignatureTypes []string `json:"id_token_signing_alg_values_supported"`
	JWKSURI        string   `json:"jwks_uri"`
}

var (
	wellKnownCache      WellKnownData
	wellKnownLastUpdate time.Time
)

var jwksAutoRefresh *jwk.AutoRefresh

func GetWellKnownData() (WellKnownData, error) {
	if wellKnownCache.JWKSURI != "" && time.Now().UTC().Sub(wellKnownLastUpdate) < 24*time.Hour {
		return wellKnownCache, nil
	}

	httpClient := http.Client{Timeout: time.Second * 2}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, JWTWellKnown, nil)
	if err != nil {
		return WellKnownData{}, err
	}
	req.Header.Set("User-Agent", "actions-cache-server/1.0")
	res, err := httpClient.Do(req)
	if err != nil {
		return WellKnownData{}, err
	}

	if res.Body == nil {
		return WellKnownData{}, errors.New("http response body nil")
	}
	defer res.Body.Close()

	if err = json.NewDecoder(res.Body).Decode(&wellKnownCache); err != nil {
		return WellKnownData{}, err
	}

	if jwksAutoRefresh == nil {
		jwksAutoRefresh = jwk.NewAutoRefresh(context.Background())
	}
	jwksAutoRefresh.Configure(wellKnownCache.JWKSURI)

	wellKnownLastUpdate = time.Now().UTC()
	return wellKnownCache, nil
}

func LookupKey(keyID string) (interface{}, error) {
	wellKnownData, err := GetWellKnownData()
	if err != nil {
		return nil, err
	}

	set, err := jwksAutoRefresh.Fetch(context.Background(), wellKnownData.JWKSURI)
	if err != nil {
		return nil, err
	}

	// So set.LookupKeyID looks for Kid but we have x5t
	for i := 0; i < set.Len(); i++ {
		currentKey, ok := set.Get(i)
		if !ok {
			return nil, errors.New("could not get key, index out of range")
		}

		if currentKey.X509CertThumbprint() != keyID {
			continue
		}
		var keyData interface{}
		err = currentKey.Raw(&keyData)

		return keyData, err
	}
	return nil, errors.New("signing key not found")
}
