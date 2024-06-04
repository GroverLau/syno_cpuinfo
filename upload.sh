#!/bin/bash

PASSWORD='Kiss@1234'
LOCAL_FILE='main'
REMOTE_USER='root'
REMOTE_HOST='10.10.10.2'
REMOTE_PATH='/root/'

go build ./
if [ $? == 0 ];then
	sshpass -p "$PASSWORD" scp "$LOCAL_FILE" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH"
fi
