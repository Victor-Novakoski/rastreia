package notify_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/notify"
)

func msg(status string) notify.StatusChanged {
	return notify.StatusChanged{
		EventID: 7, TrackingCode: "RSABCDEFGH23", Status: status,
		RecipientName: "Maria Souza", RecipientEmail: "maria@example.com",
		OccurredAt: time.Date(2026, 10, 2, 13, 5, 0, 0, time.UTC),
	}
}

func TestRender(t *testing.T) {
	e, err := notify.Render(msg("in_transit"), "https://rastreia.example.com/rastreio/")
	require.NoError(t, err)
	assert.Equal(t, "maria@example.com", e.To)
	assert.Equal(t, "Sua entrega RSABCDEFGH23 está em rota", e.Subject)
	assert.Contains(t, e.Text, "Olá, Maria!")
	assert.Contains(t, e.Text, "https://rastreia.example.com/rastreio/RSABCDEFGH23")
	assert.Contains(t, e.Text, "02/10/2026 às 10:05", "shown in Brasília time")
	assert.Contains(t, e.HTML, `href="https://rastreia.example.com/rastreio/RSABCDEFGH23"`)
	assert.NotContains(t, e.Text, "Souza", "only the first name")
}

func TestRender_EveryStatus(t *testing.T) {
	for _, s := range []string{"pending", "picked_up", "in_transit", "delivered", "failed"} {
		e, err := notify.Render(msg(s), "http://localhost:5173/rastreio")
		require.NoError(t, err, s)
		assert.Contains(t, e.Subject, "RSABCDEFGH23", s)
	}
}

func TestRender_EscapesNameInHTML(t *testing.T) {
	m := msg("delivered")
	m.RecipientName = `<script>alert(1)</script> Souza`
	e, err := notify.Render(m, "http://localhost:5173/rastreio")
	require.NoError(t, err)
	assert.NotContains(t, e.HTML, "<script>")
	assert.Contains(t, e.HTML, "&lt;script&gt;")
}

func TestRender_UnknownStatus(t *testing.T) {
	_, err := notify.Render(msg("lost"), "http://localhost:5173/rastreio")
	assert.Error(t, err)
}
