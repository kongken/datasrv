package internal

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
	"github.com/kongken/datasrv/service/datasrv/internal/conf"
)

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:  conf.Conf,
		Service: "datasrv",
		// Router:  http.Router,
		InitFunc: []func() error{},
	})
	return app
}
