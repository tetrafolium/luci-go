# luci-go: LUCI services and tools in Go

[![GoDoc](https://godoc.org/github.com/tetrafolium/luci-go?status.svg)](https://godoc.org/github.com/tetrafolium/luci-go)


## Installing

LUCI Go code is meant to be worked on from an Chromium
[infra.git](https://chromium.googlesource.com/infra/infra.git) checkout, which
enforces packages versions and Go toolchain version. First get fetch via
[depot_tools.git](https://chromium.googlesource.com/chromium/tools/depot_tools.git)
then run:

    fetch infra
    cd infra/go
    eval `./env.py`
    cd src/github.com/tetrafolium/luci-go


## Contributing

Contributing uses the same flow as [Chromium
contributions](https://www.chromium.org/developers/contributing-code).
