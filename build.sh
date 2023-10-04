#!/bin/bash

ver=$(date +%y.%m.%d)

CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$ver" -o extsvrd svr/main.go
upx extsvrd

CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$ver" -o extsvr cli/main.go
upx extsvr

scp extsvrd wlstl:/home/shares/archiving/v5release/luwak_linux/programs/luwakctld
scp extsvr wlstl:/home/shares/archiving/v5release/luwak_linux/programs/luwakctl
