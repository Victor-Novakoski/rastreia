package route

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Victor-Novakoski/rastreia/internal/store"
)

func TestParseCode(t *testing.T) {
	for in, want := range map[string]string{
		"RS7K2M9QXA4P":    "RS7K2M9QXA4P",
		"  rs7k2m9qxa4p ": "RS7K2M9QXA4P",
		"https://rastreia.app/rastreio/RS7K2M9QXA4P":  "RS7K2M9QXA4P",
		"http://localhost:5173/rastreio/rs7k2m9qxa4p": "RS7K2M9QXA4P",
	} {
		assert.Equal(t, want, parseCode(in), in)
	}
}

func TestGroupStops(t *testing.T) {
	lat, lng := -23.55, -46.63
	row := func(id int64, cep, number, complement string) store.ListRouteItemsRow {
		d := store.Delivery{ID: id, PostalCode: cep, Street: "Rua A", Number: number, Complement: complement,
			District: "Centro", City: "São Paulo", Address: "Rua A, " + number}
		if id == 3 {
			d.Latitude, d.Longitude = &lat, &lng
		}
		return store.ListRouteItemsRow{Delivery: d}
	}
	legacy := store.ListRouteItemsRow{Delivery: store.Delivery{ID: 5, Address: "Rua  B, 1"}}
	legacyAgain := store.ListRouteItemsRow{Delivery: store.Delivery{ID: 6, Address: "rua b, 1"}}

	stops := groupStops([]store.ListRouteItemsRow{
		row(1, "01001000", "10", "Apto 1"), row(2, "01001000", "20", ""), legacy,
		row(3, "01001000", "10", "Apto 2"), legacyAgain,
	})

	if assert.Len(t, stops, 3) {
		assert.Equal(t, []int64{1, 3}, stops[0].items, "apartments of a building are one stop")
		assert.Equal(t, "Rua A, 10 - Centro, São Paulo", stops[0].Address, "without the complement")
		assert.Equal(t, &lat, stops[0].Latitude, "any package with a position places the stop")
		assert.Equal(t, []int64{2}, stops[1].items)
		assert.Equal(t, []int64{5, 6}, stops[2].items, "older addresses match by text")
		assert.Equal(t, []int{1, 2, 3}, []int{stops[0].Number, stops[1].Number, stops[2].Number})
	}
}
