This file describes the usage and setup of SSL certs for HTTPS.

# Initial setup

The easiest way is to use **letsencrypt**, so follow the tutorial for your platform.
In the end you should have some certificates in `/etc/letsencrypt/live/<domain>/...`.

## Usage in docker containers

The `docker-compose.yml` embeds the `/etc/letsencrypt` folder as volume so that the applications (e.g. the nginx server) can access the certificate.

# Configuration

## Server configuration

The server has ssl-specific entries in the config file.
Here an excerpt from the `prod.json` config:

```json
{
	"server-url": "https://stm.hauke-stieler.de",
	"ssl-cert-file": "/etc/letsencrypt/live/stm.hauke-stieler.de/fullchain.pem",
	"ssl-key-file": "/etc/letsencrypt/live/stm.hauke-stieler.de/privkey.pem",
	...
}
```

Using `https://` as protocol indicates that the given certificated should be used.
Just using `http://` will ignore these certificates.

In the end, specify your config with the `-c` flag like this:
```bash
./stm-server -c ./config/prod.json
```

## Client configuration

The client also uses docker to run but here the nginx server must be configured (not the actual angular application).

When building the container, the `nginx.conf` file from the client directory is used.
Here the most important entries for HTTPS:

```
server {
	listen 443 ssl;
	server_name stm.hauke-stieler.de;

	ssl_certificate /etc/letsencrypt/live/stm.hauke-stieler.de/cert.pem;
	ssl_certificate_key /etc/letsencrypt/live/stm.hauke-stieler.de/privkey.pem;

	# ...
}
```

When building the docker container for the client, the `nginx.conf` file will be copied into the container.
Therefore, just starting the container will use this config file and changing this file requires to rebuild the container.

# Automatic renewal

I use the systemd timer functionality to trigger a renewal of the certificate.
This tutorial is pretty simple and straight forward, however, I changes some things: https://stevenwestmoreland.com/2017/11/renewing-certbot-certificates-using-a-systemd-timer.html

## Systemd timer

Specifies how often the certbot should try to renew the certificate.

The file `/lib/systemd/system/certbot.timer`:

```
[Unit]
Description=Certbot renewal

[Timer]
OnBootSec=5m
OnUnitActiveSec=1d

[Install]
WantedBy=multi-user.target
```

## Systemd service

Specified how the certbot should renew the certificate.
Here the post-hook also restarts all the docker container.

The file `/lib/systemd/system/certbot.service`:

```
[Unit]
Description=Certbot

[Service]
Type=oneshot
PrivateTmp=true
ExecStart=/usr/bin/certbot renew --post-hook "bash -c \"cd /root/simple-task-manager && docker-compose restart\""

[Install]
WantedBy=multi-user.target
```

## Setup Systemd

1. Create the two files mentioned above (or edit them, they probably already exist)
2. Reload via `systemd daemon-reload`
3. Restart the timer with `systemctl restart certbot.timer`

Now, probably nothing happens unless you used a very low `OnUnitActiveSec` and `OnBootSec` value.
To check everything (maybe there are starting errors), check the logs with `journalctl -f -u certbot.*`.