# UGOSERVER

## what the hell is this ???
This is a little side/hobby project I've taken on after not being satisfied with the current state of pbsds' hatena-server.

The code may be messy, as I am a novice programmer in Go and this is my first real project in it.


## why ???
Why not?


## when ???
As I'm rather busy with life at the moment and succumb to burnout like anyone else does, commits may be few and far between, but I try to work on this whenever I feel I can. No guarantees/ETAs for completeness or features are made.


## how ???
Using [nds-constrain't](https://github.com/KaeruTeam/nds-constraint), the DS can be tricked into thinking that certificates signed using the Wii's client CA are legitimate and valid.

An nginx server paired with an older version of openssl is used to accept the authentication requests made in HTTPS using SSLv3, then redirected to the main server, because nobody wants to have SSLv3 enabled on their server.
Then a simple dnsmasq server is run to redirect queries from Nintendo's domains to my own IP.

All of this is wrapped in docker (for both security and convenience) and [webproc](https://github.com/jpillora/webproc), which provides a nice web interface for viewing logs and the current loaded config file.

Some paths will have to be changed because they are hardcoded, like `/srv/` to whatever you want.

The database portion of this uses postgresql, and while I'm inexperienced in SQL, the features that it offers seem good to me vs the minor learning curve.


## Setup
Create a certificate for your server using the commands in the nds-constraint github repo linked below, and put them in `crt/common.crt` and `crt/common.key`. You should add a SAN (subject alternative name) for `ugomemo.hatena.ne.jp`, unless you plan on not using the japanese region flipnote studio.
Change the ip in dnsmasq.conf to wherever you want your request redirected
Run `docker-compose up` to start the containers
Cd into `ugoserver/` and start the go server

Flipnote studio should now be able to connect to your replacement hatena server.


## Notes
Some things I observed while developing this are available in notes.md, and contains status quo on some features.

The txt files in ugoserver/hatena/static/ds/v2-xx/ should be UTF-16LE, otherwise the DS will not show any text. In vim this can be achieved by setting fileencoding to utf-16le and saving the file.

`ugoserver/hatena/static/ds/v2-xx` should be symlinked to regions you want to enable. v2-us, v2-eu and v2-jp can be inferred from the name, v2 is the initial rev2 release of flipnote studio in japan.

I probably should've used git to log changes before I made this public, but that's whatever


## Contributing
Issues and pull requests are welcome.

I try to comment code as much as possible, maybe even when not needed, but I believe extra documentation for things that may seem obvious is better than no documentation.

If you have original flipnote hatena captures, that would be SUPER helpful. Please send them over, you can find my contacts on [my web page](https://furokku.github.io/about.html).


## Credits & Thanks
[hatena-server](https://github.com/pbsds/hatena-server) - pbsds
[nds-constrain't](https://github.com/KaeruTeam/nds-constraint) - Project KaeruTeam
Extensive flipnote format documentation [here](https://github.com/Flipnote-Collective/flipnote-studio-docs/wiki) and [here](https://github.com/pbsds/hatena-server/wiki)
and others...
