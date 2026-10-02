package notify

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"net/url"
	"strings"
	texttemplate "text/template"
	"time"
)

// Email is a rendered message, ready to send.
type Email struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// statusCopy is what the recipient reads for each status. Labels match the
// front (web/src/lib/status.ts).
var statusCopy = map[string]struct{ subject, line string }{
	"pending":    {"Recebemos sua entrega %s", "Sua entrega foi registrada e está aguardando coleta."},
	"picked_up":  {"Sua entrega %s foi coletada", "O motorista coletou sua entrega."},
	"in_transit": {"Sua entrega %s está em rota", "Sua entrega saiu para entrega e chega em breve."},
	"delivered":  {"Sua entrega %s foi entregue", "Sua entrega foi concluída."},
	"failed":     {"Não conseguimos entregar %s", "Não conseguimos fazer a entrega desta vez. Vamos tentar de novo."},
}

// The recipient's name is typed by an operator, so the HTML version goes
// through html/template, which escapes it.
var (
	textTmpl = texttemplate.Must(texttemplate.New("text").Parse(`Olá, {{.Name}}!

{{.Line}}

Código de rastreio: {{.Code}}
Acompanhe em: {{.Link}}

Atualizado em {{.When}}.
`))
	htmlTmpl = htmltemplate.Must(htmltemplate.New("html").Parse(`<!doctype html>
<html lang="pt-BR"><body style="font-family:system-ui,sans-serif;color:#1c1917;max-width:520px;margin:0 auto;padding:24px">
<p>Olá, {{.Name}}!</p>
<p>{{.Line}}</p>
<p>Código de rastreio: <strong>{{.Code}}</strong></p>
<p><a href="{{.Link}}" style="display:inline-block;background:#1c1917;color:#fff;padding:10px 16px;border-radius:6px;text-decoration:none">Acompanhar entrega</a></p>
<p style="color:#78716c;font-size:13px">Atualizado em {{.When}}.</p>
</body></html>
`))
	brazil = time.FixedZone("BRT", -3*60*60)
)

// Render builds the e-mail for a status change. trackingURL is the front's
// public tracking page, such as https://rastreia.example.com/rastreio.
func Render(m StatusChanged, trackingURL string) (Email, error) {
	c, ok := statusCopy[m.Status]
	if !ok {
		return Email{}, fmt.Errorf("unknown status %q", m.Status)
	}
	data := struct{ Name, Line, Code, Link, When string }{
		Name: firstName(m.RecipientName),
		Line: c.line,
		Code: m.TrackingCode,
		Link: strings.TrimRight(trackingURL, "/") + "/" + url.PathEscape(m.TrackingCode),
		When: m.OccurredAt.In(brazil).Format("02/01/2006 às 15:04"),
	}
	var text, html bytes.Buffer
	if err := textTmpl.Execute(&text, data); err != nil {
		return Email{}, err
	}
	if err := htmlTmpl.Execute(&html, data); err != nil {
		return Email{}, err
	}
	return Email{
		To:      m.RecipientEmail,
		Subject: fmt.Sprintf(c.subject, m.TrackingCode),
		Text:    text.String(),
		HTML:    html.String(),
	}, nil
}

func firstName(name string) string {
	if f := strings.Fields(name); len(f) > 0 {
		return f[0]
	}
	return name
}
