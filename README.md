# UGOSERVER

## what the hell is this ???
This is a little side/hobby project I've taken on after not being satisfied with the current state of pbsds' hatena-server.

The code may be messy, as I am a novice programmer in Go and this is my first real project in it.


## why ???
Why not?


## when ???
As I'm rather busy with life at the moment and succumb to burnout like anyone else does, commits may be few and far between, but I try to work on this whenever I feel I can. No guarantees/ETAs for completeness or features are made.


## how ???
Using [nds-constrain't](https://github.com/KaeruTeam/nds-constraint), the DS can be tricked into thinking that SSL certificates signed using the Wii's client certificate are legitimate and valid.

An nginx server is used to redirect the HTTPS requests made by flipnote studio to the actual Go server in regular unencrypted HTTP, because, let's be fair, there's not much PII to give around from flipnote studio.
Then a simple dnsmasq server is run to redirect queries from Nintendo's domains to my own IP.

All of this is wrapped in docker and [webproc](https://github.com/jpillora/webproc), which provides a convenient web interface for viewing logs and the current loaded config file.

Some paths will have to be changed because they are hardcoded, like `/srv/` to whatever you want.

The database portion of this uses postgresql, and while I'm inexperienced in SQL, the features that it offers seem good to me vs the minor learning curve.


## how ??? 2: electric bogaloo
The general consensus for how a flipnote server should operate seems to be rather conformative to how Nintendo and Hatena originally did it. I don't agree with that, so if you're a seasoned hatena developer, some things may be weird or different. That's probably intended, but if it bugs you too much, feel free to open an issue.


## Contributing
Issues and pull requests are welcome.

I try to comment code as much as possible, maybe even when not needed, but I believe extra documentation for things that may seem obvious is better than no documentation.

If you have original flipnote hatena captures, that would be SUPER helpful. Please send them over, you can find my contacts on [my web page](https://furokku.github.io/sub/about.html).


## Credits & Thanks
Flipnote Studio - Nintendo
Flipnote Hatena - Hatena
[hatena-server](https://github.com/pbsds/hatena-server) - pbsds
[nds-constrain't](https://github.com/KaeruTeam/nds-constraint) - Project KaeruTeam
Extensive flipnote format documentation [here](https://github.com/Flipnote-Collective/flipnote-studio-docs/wiki) and [here](https://github.com/pbsds/hatena-server/wiki)
and others...
