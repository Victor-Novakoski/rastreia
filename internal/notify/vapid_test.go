package notify_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/notify"
	"github.com/Victor-Novakoski/rastreia/internal/store"
)

// The push services read the contact from the JWT's sub claim; Apple refuses
// a malformed one. webpush-go adds "mailto:" by itself.
func TestWebPush_VAPIDSubject(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	priv, pub, err := webpush.GenerateVAPIDKeys()
	require.NoError(t, err)
	k, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	send := notify.WebPush(notify.VAPID{PublicKey: pub, PrivateKey: priv, Subject: "mailto:contato@rastreia.dev"})
	_, err = send(context.Background(), []byte(`{}`), store.PushSubscription{
		Endpoint: srv.URL + "/x",
		P256dh:   base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes()),
		Auth:     base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
	})
	require.NoError(t, err)
	jwt := strings.TrimPrefix(strings.Split(auth, ",")[0], "vapid t=")
	payload, err := base64.RawURLEncoding.DecodeString(strings.Split(jwt, ".")[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(payload, &claims))
	assert.Equal(t, "mailto:contato@rastreia.dev", claims["sub"])
}
