package internal

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/kongken/monkey/service/datasrv/internal/conf"
)

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:  conf.Conf,
		Service: "api",
		// Router:  http.Router,
		InitFunc: []func() error{},
	})
	return app
}
