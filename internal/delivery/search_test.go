package delivery

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
)

func TestSearchPattern(t *testing.T) {
	cases := map[string]*string{
		"":                        nil,
		"   ":                     nil,
		"João":                    ptr("joao"),
		"  JOSÉ   da  Conceição ": ptr("jose da conceicao"),
		"rs7k2m":                  ptr("rs7k2m"),
		"50%_off":                 ptr(`50\%\_off`),
		`a\b`:                     ptr(`a\\b`),
		"ana\x00maria":            ptr("ana maria"),
		"\x00":                    nil,
	}
	for in, want := range cases {
		assert.Equal(t, want, searchPattern(in), "%q", in)
	}
}

func TestList_Search(t *testing.T) {
	fs := newFakeStore()
	svc := NewService(fs)

	_, err := svc.List(context.Background(), 1, ListInput{Search: " Maria "})
	require.NoError(t, err)
	assert.Equal(t, ptr("maria"), fs.lastList.Search)

	_, err = svc.List(context.Background(), 1, ListInput{Search: strings.Repeat("a", 101)})
	var verr *apperr.ValidationError
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Fields, "q")
}
