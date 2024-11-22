module extsvr

go 1.22

replace github.com/xyzj/toolbox => /config/go/src/github.com/xyzj/toolbox

require (
	github.com/xyzj/toolbox v0.0.0-00010101000000-000000000000
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/pkg/errors v0.9.1 // indirect
