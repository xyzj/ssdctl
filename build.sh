#!/bin/bash

ver=$(date +%y.%m.%d)

cd svr
CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$ver" -o ../extsvrd main.go
cd -
upx extsvrd

cd cli
CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$ver" -o ../extsvr main.go
cd -
upx extsvr

scp extsvrd wlstl:/home/shares/archiving/v5release/luwak_linux/programs/luwakctld
scp extsvr wlstl:/home/shares/archiving/v5release/luwak_linux/programs/luwakctl
