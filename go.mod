module extsvr

go 1.21

toolchain go1.22.0

replace github.com/xyzj/gopsu => /config/go/src/github.com/xyzj/gopsu

require (
	github.com/xyzj/gopsu v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.2 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.3.0 // indirect
	github.com/ethereum/go-ethereum v1.13.14 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/holiman/uint256 v1.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/tidwall/gjson v1.17.1 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/tjfoc/gmsm v1.4.2-0.20220114090716-36b992c51540 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)
