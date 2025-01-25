# flipnote revisions
there are three revisions of flipnote
<br>1 and 2 are japan only
<br>3 is worldwide
<br>
<br>rev 1 isn't like the others, it sends two GETs to auth and doesn't use regular ugomenus
<br>index is hardcoded into the app, the buttons make requests for .lst files
<br>the checkmark toggles ?mode=safe on movies.lst, so it's probably not usable in rev2/rev3
<br>side by side buttons are also likely not feasible
<br>
<br>rev2 and 3 are basically the same, rev3 is just an update to support other regions
<br>rev2 sends requests under v2, rev3 under v2-us/eu/jp, chou viewer uses tv-jp so is probably also rev3

# internal flipnote hatena urls

## ugomemo://command
expects response with X-DSi-Dialog-Type http header. if this header is missing the 
<br>dsi will softlock

value set to 0 makes flipnote studio navigate to the url in the response body  (utf-8)
<br>value set to 1 makes flipnote studio display text from the response body (utf-16le)


## ugomemo://createmail/{}
url should be set to ugomemo://createmail
<br>then it asks for mail/addresses.ugo which is just an ugomenu with a bunch of buttons with the url set to ugomemo://createmail/{id of recipient}
<br>it then sends POST request to mail.send with X-DSi-Mail-To header with the request body containing a mini flipnote. TODO?