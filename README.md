# fpprof
Package pprof serves via fasthttp

This package like net/http/pprof but for [fasthttp package](https://github.com/valyala/fasthttp)

Example:
```go
package main

import (
        "fmt"
        "time"

        "github.com/t0pep0/fpprof"
        "github.com/valyala/fasthttp"
)

func Something() {
        i := uint64(0)
        for 0xFFFFFFFFFFFFFFF0 > i {
                i += 1
                time.Sleep(time.Duration(i))
        }
}

func main() {
        fmt.Println("start")
        fasthttp.ListenAndServe("localhost:6060", fpprof.Pprof)
        go Something()
}
```
