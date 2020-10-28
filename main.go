package main

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"html/template"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"

	"git.sr.ht/~adnano/go-gemini"
	"git.sr.ht/~sircmpwn/getopt"
)

var gemtextPage = template.Must(template.
	New("gemtext").
	Funcs(template.FuncMap{
		"heading": func(line gemini.Line) *GemtextHeading {
			switch l := line.(type) {
			case gemini.LineHeading1:
				return &GemtextHeading{1, string(l)}
			case gemini.LineHeading2:
				return &GemtextHeading{2, string(l)}
			case gemini.LineHeading3:
				return &GemtextHeading{3, string(l)}
			default:
				return nil
			}
		},
		"link": func(line gemini.Line) *gemini.LineLink {
			switch l := line.(type) {
			case gemini.LineLink:
				return &l
			default:
				return nil
			}
		},
		"li": func(line gemini.Line) *gemini.LineListItem {
			switch l := line.(type) {
			case gemini.LineListItem:
				return &l
			default:
				return nil
			}
		},
		"pre_toggle_on": func(ctx *GemtextContext, line gemini.Line) *gemini.LinePreformattingToggle {
			switch l := line.(type) {
			case gemini.LinePreformattingToggle:
				if ctx.Pre % 4 == 0 {
					ctx.Pre += 1
					return &l
				}
				ctx.Pre += 1
				return nil
			default:
				return nil
			}
		},
		"pre_toggle_off": func(ctx *GemtextContext, line gemini.Line) *gemini.LinePreformattingToggle {
			switch l := line.(type) {
			case gemini.LinePreformattingToggle:
				if ctx.Pre % 4 == 3 {
					ctx.Pre += 1
					return &l
				}
				ctx.Pre += 1
				return nil
			default:
				return nil
			}
		},
		"pre": func(line gemini.Line) *gemini.LinePreformattedText {
			switch l := line.(type) {
			case gemini.LinePreformattedText:
				return &l
			default:
				return nil
			}
		},
		"quote": func(line gemini.Line) *gemini.LineQuote {
			switch l := line.(type) {
			case gemini.LineQuote:
				return &l
			default:
				return nil
			}
		},
		"text": func(line gemini.Line) *gemini.LineText {
			switch l := line.(type) {
			case gemini.LineText:
				return &l
			default:
				return nil
			}
		},
		"url": func(ctx *GemtextContext, s string) template.URL {
			u, err := url.Parse(s)
			if err != nil {
				return template.URL("error")
			}
			if u.Scheme == "" {
				return template.URL(s)
			}
			if u.Scheme == "gemini" {
				if u.Host != ctx.URL.Host {
					u.Path = fmt.Sprintf("/x/%s%s", u.Host, u.Path)
					u.Host = ""
				}
				u.Scheme = ""
				u.Host = ""
				return template.URL(u.String())
			}
			return template.URL(s)
		},
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
		"safeURL": func(s string) template.URL {
			return template.URL(s)
		},
	}).
	Parse(`<!doctype html>
<html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1" />
{{- if .CSS }}
<style>
{{.CSS | safeCSS}}
</style>
{{- end }}
<title>{{.Title}}</title>
<article>
	{{ $ctx := . -}}
	{{- $isList := false -}}
	{{- range .Lines -}}
	{{- if and $isList (not (. | li)) }}
	</ul>
	{{- $isList = false -}}
	{{- end -}}

	{{- with . | heading }}
	{{- $isList = false -}}
	<h{{.Level}}>{{.Text}}</h{{.Level}}>
	{{- end -}}

	{{- with . | link }}
	{{- $isList = false -}}
	<p>
	<a
		href="{{.URL | url $ctx}}"
	>{{if .Name}}{{.Name}}{{else}}{{.URL}}{{end}}</a>
	{{- end -}}

	{{- with . | quote }}
	{{- $isList = false -}}
	<blockquote>
		{{slice .String 1}}
	</blockquote>
	{{- end -}}

	{{- with . | pre_toggle_on $ctx }}
	<div aria-label="{{slice .String 3}}">
		<pre aria-hidden="true" alt="{{slice .String 3}}">
	{{- $isList = false -}}
	{{- end -}}
	{{- with . | pre -}}
	{{- $isList = false -}}
	{{.}}
{{ end -}}
	{{- with . | pre_toggle_off $ctx -}}
	{{- $isList = false -}}
		</pre>
	</div>
	{{- end -}}

	{{- with . | text }}
	{{- $isList = false }}
	<p>{{.}}
	{{- end -}}

	{{- with . | li }}
	{{- if not $isList }}
	<ul>
	{{- end -}}

	{{- $isList = true }}
		<li>{{slice .String 1}}</li>
	{{- end -}}

	{{- end }}
	{{- if $isList }}
	</ul>
	{{- end }}
</article>
<details>
	<summary>
		Proxied content from <a href="{{.URL.String | safeURL}}">{{.URL.String}}</a>
		{{if .External}}
		(external content)
		{{end}}
	</summary>
	<p>Gemini request details:
	<dl>
		<dt>Original URL</dt>
		<dd><a href="{{.URL.String | safeURL}}">{{.URL.String}}</a></dd>
		<dt>Status code</dt>
		<dd>{{.Resp.Status}}</dd>
		<dt>Meta</dt>
		<dd>{{.Resp.Meta}}</dd>
		<dt>Proxied by</dt>
		<dd><a href="https://sr.ht/~sircmpwn/kineto">kineto</a></dd>
	</dl>
	<p>Be advised that no attempt was made to verify the remote SSL certificate.
</details>
`))

// TODO: let user customize this
const defaultCSS = `html {
	font-family: sans-serif;
	color: #080808;
}

body {
	max-width: 920px;
	margin: 0 auto;
	padding: 1rem 2rem;
}

blockquote {
	background-color: #eee;
	border-left: 3px solid #444;
	margin: 1rem -1rem 1rem calc(-1rem - 3px);
	padding: 1rem;
}

ul {
	margin-left: 0;
	padding: 0;
}

li {
	padding: 0;
}

a {
	position: relative;
}

a:before {
	content: '⇒';
	color: #999;
	text-decoration: none;
	font-weight: bold;
	position: absolute;
	left: -1.25rem;
}

pre {
	background-color: #eee;
	margin: 0 -1rem;
	padding: 1rem;
	overflow-x: auto;
}

details:not([open]) summary,
details:not([open]) summary a {
	color: gray;
}

details summary a:before {
	display: none;
}

dl dt {
	font-weight: bold;
}

dl dt:not(:first-child) {
	margin-top: 0.5rem;
}
`

type GemtextContext struct {
	CSS      string
	External bool
	Lines    []gemini.Line
	Pre      int
	Resp     *gemini.Response
	Title    string
	URL      *url.URL
}

type GemtextHeading struct {
	Level int
	Text  string
}

func proxyGemini(req gemini.Request, external bool,
	w http.ResponseWriter, r *http.Request) {
	client := gemini.Client{
		TrustCertificate: func(_ string, _ *x509.Certificate, _ *gemini.KnownHosts) error {
			return nil
		},
	}
	u := &url.URL{}
	*u = *req.URL
	if !strings.Contains(req.URL.Host, ":") {
		req.URL.Host = req.URL.Host + ":1965"
	}
	if !strings.Contains(req.Host, ":") {
		req.Host = req.Host + ":1965"
	}

	resp, err := client.Send(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("Gateway error: %v", err)))
		return
	}

	switch resp.Status {
	case 20:
		break // OK
	case 30:
	case 31:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(fmt.Sprintf("This URL redirects to %s", resp.Meta)))
		return
	case 40:
	case 41:
	case 42:
	case 43:
	case 44:
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf("The remote server returned %d: %s", resp.Status, resp.Meta)))
		return
	case 50:
	case 51:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("The remote server returned %d: %s", resp.Status, resp.Meta)))
		return
	case 52:
	case 53:
	case 59:
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(fmt.Sprintf("The remote server returned %d: %s", resp.Status, resp.Meta)))
		return
	default:
		w.WriteHeader(http.StatusNotImplemented)
		w.Write([]byte(fmt.Sprintf("Proxy does not understand Gemini response status %d", resp.Status)))
		return
	}

	// XXX: We could use the params I guess
	m, _, err := mime.ParseMediaType(resp.Meta)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(fmt.Sprintf("Gateway error: %v", err)))
		return
	}

	if m != "text/gemini" {
		w.Header().Add("Content-Type", resp.Meta)
		w.Write(resp.Body)
		return
	}

	w.Header().Add("Content-Type", "text/html")
	text := gemini.Parse(bytes.NewReader(resp.Body))
	ctx := &GemtextContext{
		CSS:      defaultCSS,
		External: external,
		Lines:    []gemini.Line(text),
		Resp:     resp,
		Title:    req.URL.Host + " " + req.URL.Path,
		URL:      u,
	}
	for _, line := range text {
		if h, ok := line.(gemini.LineHeading1); ok {
			ctx.Title = string(h)
			break
		}
	}
	err = gemtextPage.Execute(w, ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%v", err)))
		return
	}
}

func main() {
	var (
		bind string = ":8080"
	)

	opts, optind, err := getopt.Getopts(os.Args, "b:c:")
	if err != nil {
		log.Fatal(err)
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'b':
			bind = opt.Value
		}
	}

	args := os.Args[optind:]
	if len(args) != 1 {
		log.Fatalf("Usage: %s <gemini root>", os.Args[0])
	}
	root, err := url.Parse(args[0])
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("404 Not found"))
			return
		}

		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("404 Not found"))
			return
		}

		req := gemini.Request{}
		req.URL = &url.URL{}
		req.URL.Scheme = root.Scheme
		req.URL.Host = root.Host
		req.URL.Path = r.URL.Path
		req.Host = root.Host
		proxyGemini(req, false, w, r)
	}))

	http.Handle("/x/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("404 Not found"))
			return
		}

		path := strings.SplitN(r.URL.Path, "/", 4)
		if len(path) != 4 {
			path = append(path, "")
		}
		req := gemini.Request{}
		req.URL, err = url.Parse(fmt.Sprintf("gemini://%s/%s", path[2], path[3]))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error: %v", err)))
			return
		}
		req.Host = path[2]
		log.Printf("%s (external) %s%s", r.Method, path[2], path[3])
		proxyGemini(req, true, w, r)
	}))

	log.Printf("HTTP server listening on %s", bind)
	log.Fatal(http.ListenAndServe(bind, nil))
}
