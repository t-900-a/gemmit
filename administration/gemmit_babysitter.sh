#!/bin/bash
pidof  gemmit >/dev/null
if [[ $? -ne 0 ]] ; then
        echo "Restarting gemmit:     $(date)" >> /var/log/gemmit_babysitter.log
        tmux new-session -d -s tmux-gemmit "/usr/local/bin/gemmit \"<YOURDOMAIN>\" \"postgres://<USERNAME>:<PASSWORD>@127.0.0.1/<DB_NAME>?sslmode=disable\" &> /var/log/gemmit.log"
fi
