package internal

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
)

func NewApp() *app.App {
	app := core.New(&app.Config{
		// Config:  conf.Conf,
		Service: "api",
		// Router:  http.Router,
		InitFunc: []func() error{},
	})
	return app
}
