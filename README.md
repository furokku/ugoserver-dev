# UGOSERVER

## what the hell is this ???
This is a little side/hobby project I've taken on after not being satisfied with the current state of pbsds' hatena-server. Besides, the dependency on Sudomemo's DNS and authentication server and its implementation in hatena-server makes my head hurt.

The code may be messy, as I am a novice programmer in Go and this is my first real project in it.


## why ???
Why not?


## when ???
As I'm rather busy with life at the moment and succumb to burnout like anyone else does, commits may be few and far between, but I try to work on this whenever I feel I can. No guarantees/ETAs for completeness or features are made.
<br>
As of right now, viewing and uploading flipnotes works. Currently I'd like to add support for mail, sorting by hot/top flipnotes, comments/user account system and a pleasing website for it, but that's TBD

## how ???
Using [nds-constrain't](https://github.com/KaeruTeam/nds-constraint), the DS can be tricked into thinking that certificates signed using the Wii's client CA are legitimate and valid.

An nginx server paired with an older version of openssl is used to accept the authentication requests made in HTTPS using SSLv3, then redirected to the main server, because nobody wants to have SSLv3 enabled on their server.
<br>Then a simple dnsmasq server is run to redirect queries from Nintendo's domains to my own IP.

All of this is wrapped in docker (for both security and convenience) and [webproc](https://github.com/jpillora/webproc), which provides a nice web interface for viewing logs and the current loaded config file.

Some paths will have to be changed because they are hardcoded, like `/srv/hatena` to whatever you want.
<br>I think it's a more or less sane default, so that's what I use

The database portion of this uses postgresql, and while I'm inexperienced in SQL, the features that it offers seem good to me vs the minor learning curve.


## Setup
READ: if you plan to run this on an ARM vps (like oracle's free tier), the nginx docker container will not work! This is because alpine:3.4, the last docker image that ships an openssl binary which supports SSLv3 without too much hassle, does not have ARM images! Make sure you have somewhere else to run it.
* Install postgresql and create a database called `ugo`. The default table is called `flipnotes` with values (id serial primary key; author\_id varchar(16) not null; filename varchar(24) not null, uploaded\_at timestamp default now() )
* Create a certificate for your server using the commands in the nds-constraint github repo linked below, and put them in `crt/common.crt` and `crt/common.key`. You should add a SAN (subject alternative name) for `ugomemo.hatena.ne.jp`, unless you plan on not using the japanese region flipnote studio
* Change the ip in dnsmasq.conf and proxy.conf to wherever you want your request redirected
* Run `docker-compose up` to start the containers
* Start the Go server, and don't forget to add the DBUSER and DBPASS environment variables in order to be able to connect to the database
* Set the primary DNS on your console correctly and (!) set the secondary DNS to what you use. This is important, as the provided dnsmasq configuration is very basic and only redirects the old flipnote servers
* It's recommended to configure http authentication for webproc if you're exposing this to the internet.

<br>Flipnote studio should now be able to connect to your replacement hatena server.


## Notes
Some things I observed while developing this are available in notes.md, and contains status quo on some features.

I probably should've used git to log changes before I made this public, but that's whatever


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
