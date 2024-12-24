# ugoserver
a little project implementing a hatena server in go, because why not

## why
there aren't any good servers if you'd like to run one, hatena-server is outdated as hell, gotena is unlicensed, etc.
<br>i work on this when i feel like it and when i have free time. don't expect etas/bother me to do things. 
<br>i hate doing any sort of frontend work so most commits are going to be underlying functionality before any real features get added
<br>issues and pull requests are welcome

## how
using [nds-constrain't](https://github.com/KaeruTeam/nds-constraint), the DS can be tricked into thinking that certificates signed using the Wii's client CA are valid
<br>due to the weak cipher suites supported by the ds, you must compile your distro's release of openssl with the enable-ssl3, enable-ssl3-method, enable-weak-ssl-ciphers options and ensure no-legacy isn't enabled, otherwise you will not be able to activate the old cipher suites (RC4-SHA, RC4-MD5)

## setup
* create a certificate for your server using the commands in the nds-constraint github repo, and put them in `crt/common.crt` and `crt/common.key`. You should add a SAN (subject alternative name) for `ugomemo.hatena.ne.jp`, unless you plan on not using the japanese region flipnote studio
* choose a db. There is support for either of postgresql or sqlite (using [go-sqlite3](https://github.com/mattn/go-sqlite3), requires CGO and can be slow to compile), specify build tags `postgres` or `sqlite` during compilation to choose one or the other (or both). This is necessary! otherwise you won't have any database support built in
* compile, make sure you have a build tag for whichever db set, if you choose sqlite ensure CGO_ENABLED=1 and gcc is in your PATH
* change configurations, sample configs are available as `default.json` and `nginx.example.conf`, `dnsmasq.example.conf`. set ips / directories / urls where necessary, preferably you should copy the default configs to some other file as they may get overwritten with future commits
* start the server. Without passing any command line arguments to it, it will attempt to read `default.json` in the current working directory. You can use a different config by passing the path to it as the first command line argument. The rest are ignored
* set the primary DNS on your console correctly and (!) set the secondary DNS to what you use. This is important, as the provided dnsmasq configuration will only redirect the flipnote hatena urls
<br>Flipnote studio should now be able to connect to your replacement hatena server


## Credits & Thanks
Original [hatena-server](https://github.com/pbsds/hatena-server) - pbsds
<br>flipnote hatena dumps (thank you a lot!) - pbsds
<br>[nds-constrain't](https://github.com/KaeruTeam/nds-constraint) - Project Kaeru
<br>Extensive flipnote format documentation [here](https://github.com/Flipnote-Collective/flipnote-studio-docs/wiki) and [here](https://github.com/pbsds/hatena-server/wiki)
<br>and likely others...
