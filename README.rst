Capybara
========

This is an `HTTP <https://en.wikipedia.org/wiki/Hypertext_Transfer_Protocol>`__ to `Gemini <https://gemini.circumlunar.space/>`__ proxy designed to provide service
for a single domain, i.e. to make your Gemini site available over HTTP. It
can proxy to any domain in order to facilitate linking to the rest of
Geminispace, but it defaults to a specific domain.

Usage
-----

.. code-block:: shell

  $ go build
  $ ./capybara [-b 127.0.0.1:8080] [-s style.css] [-e style.css] gemini://fuwn.space

-b
~~

The -b argument is optional and allows you to bind to an arbitrary address; by
default Capybara will bind to :code:`:8080`. You should set up some external reverse
proxy like nginx to forward traffic to this port and add TLS.

-s
~~

The -s argument is optional and allows you to specify a custom CSS filename.
The given file will be loaded from the local disk and placed in a :code:`<style>`
block. By default Capybara will serve its built-in style.

-e
~~

The -e argument is optional and allows you to specify a custom CSS URL. If
provided, the style.css given will be treated as a link to be put in the href
of a :code:`<link rel="stylesheet"...>` instead of being placed inline with the body
in a :code:`<style>` block like with the -s flag. The given stylesheet can be a
relative link, for instance :code:`-e /main.css` will serve :code:`main.css` from the root
of the proxied Gemini capsule.

License
~~~~~~~

`GNU General Public License v3.0 <./LICENSE>`__