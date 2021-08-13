Capybara
========

.. raw:: html

  <a href="https://github.com/fuwn/capybara">
    <img
      src="https://raw.githubusercontent.com/fuwn/capybara/main/assets/Capybara_Logo_1000x1000.png"
      alt="Whirl"
      width="220">
  </a>
  
Synopsis
--------

Capybara is an `HTTP <https://en.wikipedia.org/wiki/Hypertext_Transfer_Protocol>`__
to `Gemini <https://gemini.circumlunar.space/>`__ proxy designed to ... proxy ...
Gemini capsules over to HTTP.

Capybara also happens to be a fork of `Kineto <https://sr.ht/~sircmpwn/kineto/>`__. If you
think you don't need any of the modern day enhancements that Capybara offers, check out
Kineto for a more bare, raw-er experience.

Usage
-----

.. code-block:: shell

  $ go build
  $ ./capybara [-b 127.0.0.1:8080] [-s style.css] [-e style.css] gemini://fuwn.space

:code:`-b` (:code:`BIND`)
~~~~~~~~~~~~~~~~~~~~~~~~~

The :code:`-b` argument is optional and allows you to bind to an arbitrary address;
by default Capybara will bind to :code:`:8080`. You should set up some external
reverse proxy like nginx to forward traffic to this port and add TLS.

:code:`-s` (:code:`CSS`)
~~~~~~~~~~~~~~~~~~~~~~~~

The :code:`-s` argument is optional and allows you to specify a custom CSS filename.
The given file will be loaded from the local disk and placed in a
:code:`<style>` block. By default Capybara will serve its built-in style.

:code:`-e` (:code:`CSS_EXTERNAL`)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

The :code:`-e` argument is optional and allows you to specify a custom CSS URL.
If provided, the style.css given will be treated as a link to be put in the href
of a :code:`<link rel="stylesheet"...>` instead of being placed inline with the
body in a :code:`<style>` block like with the :code:`-s` flag. The given stylesheet can
be a relative link, for instance :code:`-e /main.css` will serve
:code:`main.css` from the root of the proxied Gemini capsule.

Docker
------

.. code-block:: shell

  $ docker run -d -e ROOT='gemini://fuwn.space' [-e BIND='127.0.0.1:8080'] [-e CSS='style.css'] [-e CSS_EXTERNAL='style.css'] fuwn/capybara

Docker Compose
--------------

Edit the :code:`docker-compose.yml` file to your liking, then...

.. code-block:: shell

  $ docker-compose up -d

License
~~~~~~~

`GNU General Public License v3.0 <./LICENSE>`__
