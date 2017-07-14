package fpprof

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const (
	prefix    = "/debug/pprof/"
	cmdline   = "cmdline"
	profile   = "profile"
	symbol    = "symbol"
	traceSuff = "trace"
	////
	heap = "heap"
	////
	contentType  = "Content-Type"
	textPlain    = "text/plain; charset=utf-8"
	octeatStream = "application/octet-stream"
	textHtml     = "text/html; charset=utf-8"
	////
	errNoStartProfile   = "Could not enable CPU profiling: "
	errUnknownProfile   = "Unknown profile: "
	errNotEnableTracing = "Could not enable tracing: "
	////
	debugPostQuery  = "debug"
	gcPostQuery     = "gc"
	secondsGetQuery = "seconds"
	////
	num_symbols = "num_symbols: 1\n"
	////
	post = "POST"
)

var indexTmpl = template.Must(template.New("index").Parse(`<html>
<head>

<title>/debug/pprof/</title>

</head>

<body>

/debug/pprof/<br>

<br>

profiles:<br>

<table>

{{range .}}

<tr><td align=right>{{.Count}}<td><a href="{{.Name}}?debug=1">{{.Name}}</a>

{{end}}

</table>

<br>

<a href="goroutine?debug=2">full goroutine stack dump</a><br>

</body>

</html>

`))

func cmdlineHandler(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set(contentType, textPlain)
	ctx.Response.SetBodyString(strings.Join(os.Args, "\x00"))
}

func profileHandler(ctx *fasthttp.RequestCtx) {
	seconds := ctx.Request.URI().QueryArgs().GetUintOrZero(secondsGetQuery)
	if 0 == seconds {
		seconds = 30
	}
	ctx.Response.Header.Set(contentType, octeatStream)
	err := pprof.StartCPUProfile(ctx.Response.BodyWriter())
	if nil != err {
		ctx.Response.Header.Set(contentType, textPlain)
		ctx.Response.SetStatusCode(500)
		ctx.Response.SetBodyString(errNoStartProfile + err.Error() + "\n")
		return
	}
	time.Sleep(time.Duration(seconds) * time.Second)
	pprof.StopCPUProfile()
}

func symbolHandler(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set(contentType, textPlain)
	buf := bytes.NewBuffer(nil)
	buf.WriteString(num_symbols)
	var b *bufio.Reader
	if post == string(ctx.Method()) {
		b = bufio.NewReader(bytes.NewReader(ctx.Request.Body()))
	} else {
		b = bufio.NewReader(bytes.NewReader(ctx.Request.URI().QueryArgs().QueryString()))
	}

	for {
		word, err := b.ReadSlice('+')
		if nil == err {
			word = word[0 : len(word)-1]
		}
		pc, _ := strconv.ParseUint(string(word), 0, 64)
		if 0 != pc {
			f := runtime.FuncForPC(uintptr(pc))
			if nil != f {
				fmt.Fprintf(buf, "%#x %s\n", pc, f.Name())
			}
		}
		if nil != err {
			if io.EOF != err {
				fmt.Fprintf(buf, "reqding request: %v\n", err)
			}
			break
		}
	}
	buf.WriteTo(ctx.Response.BodyWriter())
}

func traceHandler(ctx *fasthttp.RequestCtx) {
	seconds := ctx.Request.URI().QueryArgs().GetUintOrZero(secondsGetQuery)
	if 0 == seconds {
		seconds = 30
	}
	ctx.Response.Header.Set(contentType, octeatStream)
	err := trace.Start(ctx.Response.BodyWriter())
	if nil != err {
		ctx.Response.Header.Set(contentType, textPlain)
		ctx.Response.SetStatusCode(500)
		ctx.Response.SetBodyString(errNotEnableTracing + err.Error() + "\n")
		return
	}
	time.Sleep(time.Duration(seconds) * time.Second)
	trace.Stop()

}

func Pprof(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Request.URI().Path())
	if strings.HasPrefix(path, prefix) {
		name := strings.TrimPrefix(path, prefix)
		if 0 < len(name) {
			switch name {
			case cmdline:
				cmdlineHandler(ctx)
				return
			case profile:
				profileHandler(ctx)
				return
			case symbol:
				symbolHandler(ctx)
			case traceSuff:
				traceHandler(ctx)
			default:
				ctx.Response.Header.Set(contentType, textPlain)
				debug := ctx.Request.URI().QueryArgs().GetUintOrZero(debugPostQuery)
				p := pprof.Lookup(name)
				if nil == p {
					ctx.Response.SetStatusCode(404)
					ctx.Response.SetBodyString(errUnknownProfile + name + "\n")
					return
				}
				gc := ctx.Request.URI().QueryArgs().GetUintOrZero(gcPostQuery)
				if 0 < gc && heap == name {
					runtime.GC()
				}
				p.WriteTo(ctx.Response.BodyWriter(), debug)
				return
			}
		} else {
			profiles := pprof.Profiles()
			ctx.Response.Header.Set(contentType, textHtml)
			indexTmpl.Execute(ctx.Response.BodyWriter(), profiles)
		}
	}
	//ctx.Response.SetStatusCode(404)
	return

}
