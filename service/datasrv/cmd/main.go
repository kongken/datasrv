package main

import (
	"github.com/kongken/monkey/service/datasrv/internal"
)

func main() {
	app := internal.NewApp()
	app.Run()
}
