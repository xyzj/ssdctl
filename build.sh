#!/bin/bash

ver=$(date +%y.%m.%d)

CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$ver" -o luwakctld svr/main.go
upx luwakctld

CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$ver" -o luwakctl cli/main.go
upx luwakctl

scp luwakctld wlstl:/home/shares/archiving/v5release/luwak_linux/programs
scp luwakctl wlstl:/home/shares/archiving/v5release/luwak_linux/programs
