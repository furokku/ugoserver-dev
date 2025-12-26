# ugoserver documentation

## introduction
ugoserver is a reimplementation of the hatena service, flipnote hatena

features, in its current state:
* nas/flipnote authentication
* user accounts (todo: email)
* uploading flipnotes
* channels
* uploading comments (todo: text)
* command search
* barebones user interface
(more internal stuff)
* user sessions
* make dynamic ugomenus on the fly
* cache static resources like images, text, web templates, etc
* rsa verify ppm signatures
* CLI for console administration (wip)

nx library:
* ppm, npf, ntft, nbf decoder
  - support jpeg and png by default, add extra encoders to ugotool if you would like something else
* npf, ntft, nbf encoder
  - encodes from go stdlib image.Image

ugotool:
* establish a console connection to ugoserver thru unix socket
* image viewer
* convert between flipnote image formats

end goals:
* tcp for console
* creating channels
* channel search
* pretty flipnote-side user interface
* mail system
* following creators and friends
* preferred sort mode based on above
* email
* maybe: web interface for viewing flipnotes


## basic usage
when you run ugoserver, it will look for a configuration in ./config.json<br>
you may specify your own config file by passing it in the first argument<br>
for now other args are ignored

first, all static assets will be loaded

then, a connection to the postgresql database will be established as configured. you must have the pgcrypto extension<br>
a sanity test will not be performed to check whether all of the correct objects are defined, this is your responsibility

finally, the http and socket listeners are started for the main services

it is intended for this to run behind a reverse proxy, such as nginx. that is why client ips are obtained
through the use of X-Real-IP header, and why the http server runs at port 9000

the flipnote studio expects https with ssl3 (ciphers RC4-MD5 or RC4-SHA) on some endpoints,
this can be sorted out with help from the nds-constrain't repository

## configuration
todo<br>
will finish all of this at some point