# flipnote studio revisions

## initial release, only jp region
internally labeled UGOMEMO-V1

quirks:
<br>    - does not imply region or version in hatena urls
<br>    - fetches resources under `/ds` example `/ds/auth` `/ds/eula.txt`
<br>    - sends two GET requests to auth url instead of GET, then POST


## second release, only jp region
internally labeled UGOMEMO-V2

this release has the subtitle "Version 2" under the main title on the title screen
<br>different launcher icon

quirks:
<br>    - fetches resources under `/ds/v2` example `/ds/v2/auth` `/ds/v2/eula.txt`
<br>    - correctly sends one GET, then one POST request to auth url


## third release, worldwide
internally also labeled UGOMEMO-V2, but has some
<br>changes for other regions

does not have the version 2 subtitle on us/eu regions, don't know about jp
<br>because it fails to run on a hiyacfw 1.4.5U dsi

quirks:
<br>    - fetches resources under `/ds/v2-xx` where xx is region code us/eu/jp 
<br>      example `/ds/v2-us/auth` `/ds/v2-eu/en/eula.txt`
<br>    - non-jp releases have the flipnote hatena button on the main menu (? needs testing)
<br>    - some resources are fetched under `/ds/v2-xx/yy` where yy is language code en/es/de/fr/it/jp


## ugoku memou chou viewer (probably spelled that wrong)

version of flipnote studio for the nintendo ds
<br>looks to have hatena support
<br>need to test because don't know much about

quirks:
<br>    - fetches resources under `/ds/tv-jp` example `/ds/tv-jp/auth` `/ds/tv-jp/jp/eula.txt`
<br>    - seems to have same hatena features as 3rd release


## closing notes
I will probably not support the initial release because it would be too much work to properly
<br>handle the way it does two GET requests instead of GET then POST

second and third release support should be more or less easy other than being a little messy
<br>since the former does not supply a language for eula and confirm/

the DS viewer may be possible to do but as far as I know the .ugo parser in it is a little
<br>wonky


-----------------------------------


# internal flipnote hatena urls

## ugomemo://command
~~probably expects PPM as response~~
<br>nope. expects response with X-DSi-Dialog-Type http header. if this header is missing the 
<br>dsi will softlock

value set to 0 makes flipnote studio navigate to the url in the response body  (utf-8)
<br>value set to 1 makes flipnote studio display an error, text from the response body (utf-16le)


## ugomemo://createmail/{}
url should be set to ugomemo://createmail<br>
then it asks for mail/addresses.ugo which is just an ugomenu with a bunch of buttons with the url set to ugomemo://createmail/{id of recipient}<br>
it then sends POST request to mail.send with X-DSi-Mail-To header with the request body containing a mini flipnote
