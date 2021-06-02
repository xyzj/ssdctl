#!/usr/bin/env python
# -*- coding:utf-8 -*-

import time
import os
import sys
import platform


def build(outname, goos, goarch, enableups, mainver):
    r = os.popen('go version')
    gover = r.read().strip().replace("go version ", "")
    pf = "{0}({1})".format(platform.platform(), platform.node())
    r.close()
    if goos == "windows":
        if goarch == "386":
            outpath = "../dist_x86"
        else:
            outpath = "../dist_win"
    else:
        outpath = "../../luwak/dist_linux"
    outname = outpath + "/" + outname
    buildcmd = 'GOOS={5} GOARCH={6} go build -tags=jsoniter -ldflags="-s -w -X main.version={1} -X \'main.buildDate={2}\' -X \'main.goVersion={3}\' -X \'main.platform={4}\'" -o {0} main.go'.format(
        outname, mainver, time.ctime(time.time()), gover, pf, goos, goarch)
    # print(buildcmd)
    os.system(buildcmd)
    if enableups:
        os.system("upx {0}".format(outname))


def build_service(outname, goos, goarch, enableups):

    r = os.popen('go version')
    gover = r.read().strip().replace("go version ", "")
    pf = "{0}({1})".format(platform.platform(), platform.node())
    r.close()
    if goos == "windows":
        if goarch == "386":
            outpath = "../dist_x86"
        else:
            outpath = "../dist_win"
    else:
        outpath = "../../luwak/dist_linux"
    outname = outpath + "/" + outname
    buildcmd = 'GOOS={5} GOARCH={6} go build -tags=jsoniter -ldflags="-s -w -H windowsgui -X main.version={1} -X \'main.buildDate={2}\' -X \'main.goVersion={3}\' -X \'main.platform={4}\'" -o {0} main.go'.format(
        outname, mainver, time.ctime(time.time()), gover, pf, goos, goarch)
    # print(buildcmd)
    os.system(buildcmd)
    if enableups:
        os.system("upx dist_win/{0}".format(outname))


if __name__ == "__main__":
    x = time.localtime()
    y = 0
    try:
        y = sys.argv[1]
    except:
        pass
    if y == 0:
        y = x[3] * 60 * 60 + x[4] * 60 + x[5]
    result = os.popen('git describe --tags')
    ver = result.read().strip().replace("v","")
    if ver == "":
        ver = "1.0.0"
    mainver = ver + ".{0}.{1}".format(time.strftime("%y%m%d", x), y)
    pf = "linux"
    try:
        pf = sys.argv[2]
    except:
        pass
    if "linux" in pf:
        print("\n=== start build linux x64 ...")
        build("luwakctl", "linux", "amd64", True, mainver)

    os.system(
        "scp -r ../../luwak/dist_linux/luwakctl wlstl:/home/shares/archiving/v5release/luwak_linux"
    )
