module extsvr

go 1.22

replace github.com/xyzj/gopsu => /config/go/src/github.com/xyzj/gopsu

require (
	github.com/xyzj/gopsu v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/pkg/errors v0.9.1 // indirect
