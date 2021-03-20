# Administrative notes

## Build

### Requirements

Go 1.16

If 1.16 is not available in your repo (as is with ubuntu), you may manually install it

https://www.kalilinux.in/2020/06/how-to-install-golang-in-kali-linux-new.html

```
git clone https://github.com/t-900-a/gemmit.git
cd gemmit
go build .
chmod +x gemmit
sudo cp gemmit /usr/local/bin
cd cmd/fetchentries
go build .
chmod +x fetchentries
sudo cp fetchentries /usr/local/bin
cd ../fetchmonero
go build .
chmod +x fetchmonero
sudo cp fetchmonero /usr/local/bin
```
## Certs
You may need to chmod and/or adjust chown certs, so that the server can read the certs.
```
mkdir /var/lib/gemini/
mkdir /var/lib/gemini/certs
cd /var/lib/gemini/certs
openssl req -x509 -newkey rsa:4096 -keyout key.rsa -out cert.pem -days 99999 -nodes
```

## Database
[Postgres quickstart](https://www.digitalocean.com/community/tutorials/how-to-install-postgresql-on-ubuntu-20-04-quickstart)
## Crons

Shell scripts are intended to be ran via a cron job

Example crontab

```
*/5 * * * * /home/gemmit/gemmit_babysitter.sh
*/13 * * * * /home/gemmit/fetchmonero.sh
0 5,17 * * * /home/gemmit/fetchentries.sh
```

gemmit_babysitter.sh : ensures gemmit server is running, if not it will start it

fetchmonero.sh : Refreshes the Monero transactions for all feeds

fetchentries.sh : Refreshes the entries for each feed