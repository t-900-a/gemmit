#!/bin/bash
pidof  fetchmonero >/dev/null
if [[ $? -ne 0 ]] ; then
        echo "Fetching Monero Transactions:     $(date)" >> /var/log/fetchentries.log
        /usr/local/bin/fetchmonero "postgres://<USERNAME>:<PASSWORD>@127.0.0.1/<DB_NAME>?sslmode=disable" "https://api.mymonero.com:8443" &> /var/log/fetchmonero.log &
fi
