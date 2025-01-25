# ugoserver

## overview
Ugoserver is a replacement server for Flipnote Studio's online functionality, Flipnote Hatena
<br>Issues and pull requests are welcome
<br>
<br>As part of development, I've written a library to convert to/from the image formats that Flipnote Studio uses. Namely npf, nbf, ntft and ppm. You can find this in the img folder

## quick setup
A few things need to be done prior to starting the server, namely
* ensure your system's SSL library supports RC4-MD5 or RC4-SHA ciphers. They're weak, but they're all the DSi supports. You can enable these in openssl at compile time with enable-ssl3, enable-ssl3-method, enable-weak-ssl-ciphers options, after that enable the legacy provider in the system configuration and set the SECLEVEL to 0 (it's insecure... but it's required to enable those ancient ciphersuites)
* generate bogus nintendo CA signed certs for flipnote.hatena.com, nas.nintendowifi.net (and ugomemo.hatena.ne.jp if you want to use japanese region flipnote studio with this). The commands to do this can be found in the nds-constraint github repository
* install PostgreSQL. A script to create all necessary tables/functions is available, use that, and insert any required data, like channels manually
* compile the server, there is a makefile provided to make this easy
* change the configuration as necessary. At a minimum, you need to set the options related to the database, so that it can connect, and the options for the directories with static content. an example is available
* configure a reverse proxy to redirect queries to this server @ port 9000. an example nginx config is available
* configure a dns to redirect flipnote's queries to this server. example dnsmasq and bind9 named config are available
* start the server. It will attempt to read the configuration file as the first argument passed (others ignored), and will default to config.json in the current working directory if not found.
<br>you should be able to set the dns and connect after doing all of this.
<br>Templates will be read from dir/static/template/\*.html, predefined menus from dir/static/menu/\*.json, text content from dir/static/txt (dir is set in the config file)
<br>
<br>2xxxx, 33xxx codes mean something is up, possibly with dns/nas
<br>304xxx means something is up with the html/menu bits of online mode
<br>304001 usually appears when the console received a response, but no data
<br>304605 can mean a variety of things and isn't easy to diagnose. HTTP status codes are also displayed as 304xxx

## current state
I have quite a few things planned for this server, so maybe there will be more commits pushed here
<br>a current todo list is available in the code
<br>at some point I wish to make a wiki for this, but that's after all of the core functionality is done

## support development $$
This is all a big hobby project. It's MIT licensed. If you want to run your paid server or whatever, feel free to do so. I do not care
<br>However, if you'd like to see more of this server, you can support me by donating via [paypal](https://www.paypal.com/donate/?hosted_button_id=YFLWW24WGMGS8)
<br> 65.40 ukrainian hryvnia buys me a monster energy drink, that's about $1.70, this is my lifeblood

## Credits & Thanks
Original [hatena-server](https://github.com/pbsds/hatena-server), some code was helpful in understanding how Flipnote works - pbsds
<br>flipnote hatena assets (thanks a bunch) - pbsds
<br>[nds-constrain't](https://github.com/KaeruTeam/nds-constraint) - Project Kaeru
<br>Very good format documentation [here](https://github.com/Flipnote-Collective/flipnote-studio-docs/wiki) and [here](https://github.com/pbsds/hatena-server/wiki)
<br>and likely others...