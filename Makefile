BINARY_NAME=ssdctld
BINARY_NAME_CLIENT=ssdctl
MAIN_VER=2.2.0

DIST_LINUX=_dist/${BINARY_NAME}
DIST_ARM64=_dist/${BINARY_NAME}-arm64
DIST_LINUX_CLIENT=_dist/${BINARY_NAME_CLIENT}
DIST_ARM64_CLIENT=_dist/${BINARY_NAME_CLIENT}-arm64

DATE_VER=`date '+%y%m%d.%H%M%S'`
GO_VER=`go version | cut -d \  -f 3`
BUILD_DATE=`date`
BUILD_OS=`uname -srv`
LDFLAGS="-s -w -X 'main.version=${MAIN_VER}.${DATE_VER}'"

# GOARCH for linux enable:
#	"amd64", "arm64", "mips64", "mips64le", "ppc64", "ppc64le", "riscv64", "s390x", "wasm"
#	"loong64" may need c source code
# Detail: https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63

# 编译所有版本，并发布到服务器
release: linux arm64
	@echo "copy files to server..."
	@scp -p ${DIST_LINUX} wlstl:/home/shares/archiving/v5release/luwak_linux/programs/luwakctld
	@scp -p ${DIST_LINUX_CLIENT} wlstl:/home/shares/archiving/v5release/luwak_linux/programs/luwakctl
	@scp -p ${DIST_ARM64} wlstl:/home/shares/archiving/v5release/luwak_arm64/bin/luwakctld
	@scp -p ${DIST_ARM64_CLIENT} wlstl:/home/shares/archiving/v5release/luwak_arm64/bin/luwakctl
	@echo "\nall done."

# 编译linux 64位版本
linux: modtidy
	@echo "building linux amd64 version..."
	@GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ${DIST_LINUX} -ldflags=${LDFLAGS} main.go
	@upx ${DIST_LINUX}
	@GOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -o ${DIST_LINUX_CLIENT} -ldflags=${LDFLAGS} cmd/main.go
	@upx ${DIST_LINUX_CLIENT}
	@echo "done.\n"

# 编译arm64/aarch64架构版本
arm64: modtidy
	@echo "building linux arm64/aarch64 version..."
	@GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o ${DIST_ARM64} -ldflags=${LDFLAGS} main.go
	@GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -o ${DIST_ARM64_CLIENT} -ldflags=${LDFLAGS} cmd/main.go
	@echo "done.\n"

# 更新go.mod文件
modtidy:
	@go mod tidy

# 更新所有依赖包
modupdate:
	@go get -u -v all
	@echo "done."

# 清理编译文件
clean:
	@rm -fv _dist/${BINARY_NAME}*
	@rm -fv _dist/${BINARY_NAME_CLIENT}*

