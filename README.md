# kineto

This is an [HTTP][http] to [Gemini][gemini] proxy designed to provide service
for a single domain, i.e. to make your Gemini site available over HTTP. It
can proxy to any domain in order to facilitate linking to the rest of
Geminispace, but it defaults to a specific domain.

[http]: https://en.wikipedia.org/wiki/Hypertext_Transfer_Protocol
[gemini]: https://gemini.circumlunar.space/

## Usage

```
$ go build
$ ./kineto [-b 127.0.0.1:8080] gemini://example.org
```

The -b argument is optional and allows you to bind to an arbitrary address; by
default kineto will bind to `:8080`. You should set up some external reverse
proxy like nginx to forward traffic to this port and add TLS.

## "kineto"?

It's named after the Contraves-Goerz Kineto Tracking Mount, which is used by
NASA to watch rockets as they ascend to orbit.

![](https://l.sr.ht/_frS.jpeg)
