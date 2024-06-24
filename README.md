# UGOSERVER

## what the hell is this ???
This is a little side/hobby project I've taken on after not being satisfied with the current state of pbsds' hatena-server. Besides, the dependency on Sudomemo's DNS and authentication server and its implementation in hatena-server makes my head hurt.

The code may be messy, as I am a novice programmer in Go and this is my first real project in it.


## why ???
Why not?


## when ???
As I'm rather busy with life at the moment and succumb to burnout like anyone else does, commits may be few and far between, but I try to work on this whenever I feel I can. No guarantees/ETAs for completeness or features are made.
<br>
As of right now, viewing and uploading flipnotes works. Currently I'd like to add support for mail, sorting by hot/top flipnotes, comments/user account system and a pleasing website for it. Don't expect anything, however. I work on this when I find that I have the time and motivation, which is rarely nowadays.

## how ???
Using [nds-constrain't](https://github.com/KaeruTeam/nds-constraint), the DS can be tricked into thinking that certificates signed using the Wii's client CA are legitimate and valid.

The way the server works has changed a little bit<br>
It no longer relies on docker, so the amd64 limitation has mostly been lifted<br>
Compile your distro's release of openssl with the enable-ssl3, enable-ssl3-method and enable-weak-ssl-ciphers flags. Make sure you don't have no-legacy, otherwise you won't be able to use the legacy provider in openssl in order to enable the very limited ssl ciphers that the DS supports (RC4-SHA and RC4-MD5).<br>

## Setup
* Create a certificate for your server using the commands in the nds-constraint github repo linked below, and put them in `crt/common.crt` and `crt/common.key`. You should add a SAN (subject alternative name) for `ugomemo.hatena.ne.jp`, unless you plan on not using the japanese region flipnote studio
* Install postgresql. The database format might change at any point, so look at db.go for a reference of what you need in the table
* Change the ip in dnsmasq.conf and the nginx configuration (an example is provided, you should create a site in your installed version though) to wherever you want your request redirected
* Start the server. Without passing any command line arguments to it, it will attempt to read `default.json` in the current working directory. You can use a different config by passing the path to it as the first command line argument. The rest are ignored.
* Set the primary DNS on your console correctly and (!) set the secondary DNS to what you use. This is important, as the provided dnsmasq configuration will only redirect the servers that are hardcoded into the app.
<br>Flipnote studio should now be able to connect to your replacement hatena server.


## Notes
Some things I observed while developing this are available in notes.md, and contains status quo on some features.
Commits might be giant and happen in the middle of nowhere containing a lot of changes from where i got sudden bursts of motivation

## Contributing
Issues and pull requests are welcome.

I try to comment code as much as possible, maybe even when not needed, but I believe extra documentation for things that may seem obvious is better than no documentation.

If you have original flipnote hatena captures, that would be SUPER helpful. Please send them over, you can find my contacts on [my web page](https://floc.root.sx/about.html).


## Credits & Thanks
Original [hatena-server](https://github.com/pbsds/hatena-server) - pbsds
<br>flipnote hatena dumps (thank you a lot!) - pbsds
<br>[nds-constrain't](https://github.com/KaeruTeam/nds-constraint) - Project Kaeru
<br>Extensive flipnote format documentation [here](https://github.com/Flipnote-Collective/flipnote-studio-docs/wiki) and [here](https://github.com/pbsds/hatena-server/wiki)
<br>and likely others...
