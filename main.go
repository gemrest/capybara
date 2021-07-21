package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
				if ctx.Pre%4 == 0 {
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
				if ctx.Pre%4 == 3 {
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
			u = ctx.URL.ResolveReference(u)

			if u.Scheme == "" || u.Scheme == "gemini" {
				if u.Host != ctx.Root.Host {
					u.Path = fmt.Sprintf("/x/%s%s", u.Host, u.Path)
				}
				u.Scheme = ""
				u.Host = ""
			}
			return template.URL(u.String())
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
{{- if .ExternalCSS }}
<link rel="stylesheet" type="text/css" href="{{.CSS | safeCSS}}">
{{- else }}
<style>
{{.CSS | safeCSS}}
</style>
{{- end }}
{{- end }}
<title>{{.Title}}</title>
<article{{if .Lang}} lang="{{.Lang}}"{{end}}>
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

var inputPage = template.Must(template.
	New("input").
	Funcs(template.FuncMap{
		"safeCSS": func(s string) template.CSS {
			return template.CSS(s)
		},
	}).
	Parse(`<!doctype html>
<html>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1" />
{{- if .CSS }}
{{- if .ExternalCSS }}
<link rel="stylesheet" type="text/css" href="{{.CSS | safeCSS}}">
{{- else }}
<style>
{{.CSS | safeCSS}}
</style>
{{- end }}
{{- end }}
<title>{{.Prompt}}</title>
<form method="POST">
	<label for="input">{{.Prompt}}</label>
	{{ if .Secret }}
	<input type="password" id="input" name="q" />
	{{ else }}
	<input type="text" id="input" name="q" />
	{{ end }}
</form>
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

li:not(:last-child) {
	margin-bottom: 0.5rem;
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

@media(prefers-color-scheme:dark) {
	html {
		background-color: #111;
		color: #eee;
	}

	blockquote {
		background-color: #000;
	}

	pre {
		background-color: #222;
	}

	a {
		color: #0087BD;
	}

	a:visited {
		color: #333399;
	}
}

label {
	display: block;
	font-weight: bold;
	margin-bottom: 0.5rem;
}

input {
	display: block;
	border: 1px solid #888;
	padding: .375rem;
	line-height: 1.25rem;
	transition: border-color .15s ease-in-out,box-shadow .15s ease-in-out;
	width: 100%;
}

input:focus {
	outline: 0;
	border-color: #80bdff;
	box-shadow: 0 0 0 0.2rem rgba(0,123,255,.25);
}
`

type GemtextContext struct {
	CSS         string
	ExternalCSS bool
	External    bool
	Lines       []gemini.Line
	Pre         int
	Resp        *gemini.Response
	Title       string
	Lang        string
	URL         *url.URL
	Root        *url.URL
}

type InputContext struct {
	CSS         string
	ExternalCSS bool
	Prompt      string
	Secret      bool
	URL         *url.URL
}

type GemtextHeading struct {
	Level int
	Text  string
}

func proxyGemini(req gemini.Request, external bool, root *url.URL,
	w http.ResponseWriter, r *http.Request, css string, externalCSS bool) {

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	client := gemini.Client{}
	resp, err := client.Do(ctx, &req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Gateway error: %v", err)
		return
	}
	defer resp.Body.Close()

	switch resp.Status {
	case 10, 11:
		w.Header().Add("Content-Type", "text/html")
		err = inputPage.Execute(w, &InputContext{
			CSS:         css,
			ExternalCSS: externalCSS,
			Prompt:      resp.Meta,
			Secret:      resp.Status == 11,
			URL:         req.URL,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("%v", err)))
		}
		return
	case 20:
		break // OK
	case 30, 31:
		to, err := url.Parse(resp.Meta)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, "Gateway error: bad redirect: %v", err)
		}
		next := req.URL.ResolveReference(to)
		if next.Scheme != "gemini" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "This page is redirecting you to %s", next)
			return
		}
		if external {
			next.Path = fmt.Sprintf("/x/%s/%s", next.Host, next.Path)
		}
		next.Host = r.URL.Host
		next.Scheme = r.URL.Scheme
		w.Header().Add("Location", next.String())
		w.WriteHeader(http.StatusFound)
		fmt.Fprintf(w, "Redirecting to %s", next)
		return
	case 40, 41, 42, 43, 44:
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "The remote server returned %d: %s", resp.Status, resp.Meta)
		return
	case 50, 51:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "The remote server returned %d: %s", resp.Status, resp.Meta)
		return
	case 52, 53, 59:
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "The remote server returned %d: %s", resp.Status, resp.Meta)
		return
	default:
		w.WriteHeader(http.StatusNotImplemented)
		fmt.Fprintf(w, "Proxy does not understand Gemini response status %d", resp.Status)
		return
	}

	m, params, err := mime.ParseMediaType(resp.Meta)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, "Gateway error: %d %s: %v", resp.Status, resp.Meta, err)
		return
	}

	if m != "text/gemini" {
		w.Header().Add("Content-Type", resp.Meta)
		io.Copy(w, resp.Body)
		return
	}

	if charset, ok := params["charset"]; ok {
		charset = strings.ToLower(charset)
		if charset != "utf-8" {
			w.WriteHeader(http.StatusNotImplemented)
			fmt.Fprintf(w, "Unsupported charset: %s", charset)
			return
		}
	}

	lang := params["lang"]

	w.Header().Add("Content-Type", "text/html")
	gemctx := &GemtextContext{
		CSS:         css,
		ExternalCSS: externalCSS,
		External:    external,
		Resp:        resp,
		Title:       req.URL.Host + " " + req.URL.Path,
		Lang:        lang,
		URL:         req.URL,
		Root:        root,
	}

	var title bool
	gemini.ParseLines(resp.Body, func(line gemini.Line) {
		gemctx.Lines = append(gemctx.Lines, line)
		if !title {
			if h, ok := line.(gemini.LineHeading1); ok {
				gemctx.Title = string(h)
				title = true
			}
		}
	})

	err = gemtextPage.Execute(w, gemctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "%v", err)
		return
	}
}

func performIfEnv(key string, do func()) {
	if len(os.Getenv(key)) != 0 {
		do()
	}
}

func main() {
	var (
		bind     string = ":8080"
		css      string = defaultCSS
		external bool   = false
	)

	opts, optind, err := getopt.Getopts(os.Args, "b:c:s:e:")
	if err != nil {
		log.Fatal(err)
	}

	performIfEnv("BIND", func() {
		bind = os.Getenv("BIND")
	})
	performIfEnv("CSS", func() {
		external = false
		cssContent, err := ioutil.ReadFile(os.Getenv("CSS"))
		if err == nil {
			css = string(cssContent)
		} else {
			log.Fatalf("Error opening custom CSS from '%s': %v", os.Getenv("CSS"), err)
		}
	})
	performIfEnv("CSS_EXTERNAL", func() {
		external = true
		css = os.Getenv("CSS_EXTERNAL")
	})

	for _, opt := range opts {
		switch opt.Option {
		case 'b':
			bind = opt.Value
		case 's':
			external = false
			cssContent, err := ioutil.ReadFile(opt.Value)
			if err == nil {
				css = string(cssContent)
			} else {
				log.Fatalf("Error opening custom CSS from '%s': %v", opt.Value, err)
			}
		case 'e':
			external = true
			css = opt.Value
		}
	}

	args := os.Args[optind:]
	var (
		envRoot string
		root    *url.URL
	)
	if len(args) != 1 {
		envRoot = os.Getenv("ROOT")
		if len(envRoot) == 0 {
			log.Fatalf("Usage: %s <gemini root>", os.Args[0])
		}
	} else {
		root, err = url.Parse(args[0])
	}
	if len(envRoot) != 0 {
		root, err = url.Parse(envRoot)
	}
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			if q, ok := r.Form["q"]; !ok {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
			} else {
				w.Header().Add("Location", "?"+q[0])
				w.WriteHeader(http.StatusFound)
				w.Write([]byte("Redirecting"))
			}
			return
		}

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
		req.URL.RawQuery = r.URL.RawQuery
		proxyGemini(req, false, root, w, r, css, external)
	}))

	http.Handle("/x/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			r.ParseForm()
			if q, ok := r.Form["q"]; !ok {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad request"))
			} else {
				w.Header().Add("Location", "?"+q[0])
				w.WriteHeader(http.StatusFound)
				w.Write([]byte("Redirecting"))
			}
			return
		}

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
		req.URL = &url.URL{}
		req.URL.Scheme = "gemini"
		req.URL.Host = path[2]
		req.URL.Path = "/" + path[3]
		req.URL.RawQuery = r.URL.RawQuery
		log.Printf("%s (external) %s%s", r.Method, r.URL.Host, r.URL.Path)
		proxyGemini(req, true, root, w, r, css, external)
	}))

	log.Printf("HTTP server listening on %s", bind)
	log.Fatal(http.ListenAndServe(bind, nil))
}
