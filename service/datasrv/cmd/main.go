package main

import (
	"github.com/kongken/datasrv/service/datasrv/internal"
)

func main() {
	app := internal.NewApp()
	app.Run()
}
