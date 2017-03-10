#!/bin/bash
if [ "$PLUTO_VERB" == "up-client" ]
then
  systemctl start www-admin.socket www-admin.service
elif [ "$PLUTO_VERB" == "down-client" ]
then
systemctl stop www-admin.socket www-admin.service
fi
