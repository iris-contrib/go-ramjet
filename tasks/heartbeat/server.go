package heartbeat

import (
	"runtime"

	ramjet "github.com/Laisky/go-ramjet"
	"github.com/kataras/iris/v12"
)

func bindHTTP() {
	ramjet.Server.Get("/heartbeat", func(ctx iris.Context) {
		ctx.Writef("heartbeat with %v active goroutines", runtime.NumGoroutine())
	})
}
