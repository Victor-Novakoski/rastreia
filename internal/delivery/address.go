package delivery

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Victor-Novakoski/rastreia/internal/apperr"
)

// recipient holds every field of a delivery the carrier fills in: who
// receives it and where. Create validates all of it; Update merges the
// change into the current delivery and validates the result.
type recipient struct {
	Name, Email, Phone string
	PostalCode         string
	Street             string
	Number             string
	Complement         string
	District           string
	City               string
	State              string
	Reference          string
	Latitude           *float64
	Longitude          *float64
}

var states = []string{
	"AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG", "PA",
	"PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
}

const (
	maxName       = 120
	maxEmail      = 254
	maxStreet     = 200
	maxNumber     = 20
	maxComplement = 100
	maxDistrict   = 100
	maxCity       = 100
	maxReference  = 300
)

// normalize trims the text fields and keeps only the digits of the CEP and
// the phone, so "01001-000" and "(11) 98765-4321" are stored the same way
// whatever the form sent.
func (r *recipient) normalize() {
	for _, f := range []*string{
		&r.Name, &r.Street, &r.Number, &r.Complement, &r.District, &r.City, &r.Reference,
	} {
		*f = strings.Join(strings.Fields(*f), " ")
	}
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.State = strings.ToUpper(strings.TrimSpace(r.State))
	r.Phone = digits(r.Phone)
	r.PostalCode = digits(r.PostalCode)
}

func (r *recipient) validateContact(v apperr.Validator) {
	v.Check(r.Name != "", "recipient_name", "is required")
	v.Check(utf8.RuneCountInString(r.Name) <= maxName, "recipient_name", "must have at most 120 characters")
	v.Check(len(r.Email) <= maxEmail && validEmail(r.Email), "recipient_email", "must be a valid e-mail")
}

// validatePhone accepts a Brazilian number with the area code: 10 digits
// for a landline, 11 for a mobile.
func (r *recipient) validatePhone(v apperr.Validator) {
	v.Check(len(r.Phone) == 10 || len(r.Phone) == 11, "recipient_phone", "must have the area code and 10 or 11 digits")
}

func (r *recipient) validateAddress(v apperr.Validator) {
	v.Check(len(r.PostalCode) == 8, "postal_code", "must have 8 digits")
	required := func(value, field string, maxLen int) {
		v.Check(value != "", field, "is required")
		v.Check(utf8.RuneCountInString(value) <= maxLen, field, fmt.Sprintf("must have at most %d characters", maxLen))
	}
	required(r.Street, "street", maxStreet)
	required(r.Number, "number", maxNumber)
	required(r.District, "district", maxDistrict)
	required(r.City, "city", maxCity)
	v.Check(slices.Contains(states, r.State), "state", "must be a Brazilian state (UF)")
	v.Check(utf8.RuneCountInString(r.Complement) <= maxComplement, "complement", "must have at most 100 characters")
	v.Check(utf8.RuneCountInString(r.Reference) <= maxReference, "address_reference", "must have at most 300 characters")
	v.Check((r.Latitude == nil) == (r.Longitude == nil), "latitude", "must be sent together with longitude")
	if r.Latitude != nil {
		v.Check(validCoordinate(*r.Latitude, 90), "latitude", "must be between -90 and 90")
	}
	if r.Longitude != nil {
		v.Check(validCoordinate(*r.Longitude, 180), "longitude", "must be between -180 and 180")
	}
}

func validCoordinate(c, limit float64) bool {
	return !math.IsNaN(c) && c >= -limit && c <= limit
}

// fullAddress is the address in one line, as on a shipping label:
// "Rua A, 10, Apto 2 - Centro, São Paulo - SP, 01001-000".
func (r *recipient) fullAddress() string {
	line := r.Street + ", " + r.Number
	if r.Complement != "" {
		line += ", " + r.Complement
	}
	return fmt.Sprintf("%s - %s, %s - %s, %s-%s", line, r.District, r.City, r.State, r.PostalCode[:5], r.PostalCode[5:])
}

func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}
