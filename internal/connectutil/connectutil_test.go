package connectutil

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestH2CServerProtocols(t *testing.T) {
	t.Run("returns configured protocols", func(t *testing.T) {
		p := H2CServerProtocols()

		assert.NotNil(t, p)
		// Verify HTTP/1 is enabled
		assert.True(t, p.HTTP1())
		// Verify unencrypted HTTP/2 is enabled
		assert.True(t, p.UnencryptedHTTP2())
	})

	t.Run("returns fresh instance each call", func(t *testing.T) {
		p1 := H2CServerProtocols()
		p2 := H2CServerProtocols()

		// Should be different instances
		assert.NotSame(t, p1, p2)
	})
}

func TestH2CClient(t *testing.T) {
	t.Run("client is initialized", func(t *testing.T) {
		assert.NotNil(t, H2CClient)
		assert.NotNil(t, H2CClient.Transport)
	})

	t.Run("transport has HTTP/2 protocol configured", func(t *testing.T) {
		transport, ok := H2CClient.Transport.(*http.Transport)
		require.True(t, ok, "expected *http.Transport")

		// The Transport should have protocols configured
		assert.NotNil(t, transport.Protocols)
		assert.True(t, transport.Protocols.UnencryptedHTTP2())
	})
}
