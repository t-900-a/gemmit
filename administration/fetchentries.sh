#!/bin/bash
pidof  fetchentries >/dev/null
if [[ $? -ne 0 ]] ; then
        echo "Fetching entries:     $(date)" >> /var/log/fetchentries.log
        /usr/local/bin/fetchentries "postgres://<USERNAME>:<PASSWORD>@127.0.0.1/<DB_NAME>?sslmode=disable" &> /var/log/fetchentries.log &
fi
