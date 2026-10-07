package delivery

import "strings"

// maxSearch caps the text of a search; codes, names and e-mails are shorter.
const maxSearch = 100

// accents turns the accented letters of Portuguese (and ñ) into plain ones.
// The ListDeliveries query drops the same letters from the recipient's name.
var accents = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// likeWildcards escapes what LIKE would read as a pattern, so a search for
// "50%" looks for those characters.
var likeWildcards = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// foldAccents lowers s and drops its accents: "João" and "JOAO" both become "joao".
func foldAccents(s string) string {
	return accents.Replace(strings.ToLower(s))
}

// searchPattern turns what the carrier typed into the search parameter of
// ListDeliveries, or nil when there is nothing to look for.
func searchPattern(q string) *string {
	q = strings.Join(strings.Fields(q), " ")
	if q == "" {
		return nil
	}
	p := likeWildcards.Replace(foldAccents(q))
	return &p
}
