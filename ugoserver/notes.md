# flipnote studio revisions

## initial release, only jp region
internally labeled UGOMEMO-V1

quirks:
    - does not imply region or version in hatena urls
    - fetches resources under `/ds` example `/ds/auth` `/ds/eula.txt`
    - sends two GET requests to auth url instead of GET, then POST


## second release, only jp region
internally labeled UGOMEMO-V2

this release has the subtitle "Version 2" under the main title on the title screen
different launcher icon

quirks:
    - fetches resources under `/ds/v2` example `/ds/v2/auth` `/ds/v2/eula.txt`
    - correctly sends one GET, then one POST request to auth url


## third release, worldwide
internally also labeled UGOMEMO-V2, but has some
changes for other regions

does not have the version 2 subtitle on us/eu regions, don't know about jp
because it fails to run on a hiyacfw 1.4.5U dsi

quirks:
    - fetches resources under `/ds/v2-xx` where xx is region code us/eu/jp 
      example `/ds/v2-us/auth` `/ds/v2-eu/en/eula.txt`
    - non-jp releases have the flipnote hatena button on the main menu (? needs testing)
    - some resources are fetched under `/ds/v2-xx/yy` where yy is language code en/es/de/fr/it/jp


## ugoku memou chou viewer (probably spelled that wrong)

version of flipnote studio for the nintendo ds
looks to have hatena support
need to test because don't know much about

quirks:
    - fetches resources under `/ds/tv-jp` example `/ds/tv-jp/auth` `/ds/tv-jp/jp/eula.txt`
    - seems to have same hatena features as 3rd release


## closing notes
I will probably not support the initial release because it would be too much work to properly
handle the way it does two GET requests instead of GET then POST

second and third release support should be more or less easy other than being a little messy
since the former does not supply a language for eula and confirm/

the DS viewer may be possible to do but as far as I know the .ugo parser in it is a little
wonky


-----------------------------------


# internal flipnote hatena urls

## ugomemo://command
~~probably expects PPM as response~~
nope. expects response with X-DSi-Dialog-Type http header. if this header is missing the 
dsi will softlock

value set to 0 makes flipnote studio navigate to the url in the response body  (utf-8)
value set to 1 makes flipnote studio display an error, text from the response body (utf-16le)


## ugomemo://createmail/{}
url should be set to ugomemo://createmail, then in `mail/addresses.ugo` set {} to the recipient
sends POST request to `mail.send` with X-DSi-Mail-To header
set to whatever is in {}. sent as a mini flipnote, so no sound/colors
