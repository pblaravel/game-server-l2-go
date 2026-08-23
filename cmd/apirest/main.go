package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/pblaravel/game-server-l2-go/internal/apitest"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:18080", "HTTP listen address")
	login := flag.String("login", "127.0.0.1:2107", "login client TCP")
	gsreg := flag.String("gsreg", "127.0.0.1:9015", "login↔game TCP")
	game := flag.String("game", "127.0.0.1:7778", "game client TCP")
	flag.Parse()

	api := &apitest.REST{Target: apitest.Target{Login: *login, GSReg: *gsreg, Game: *game}}
	log.Printf("REST facade on %s → login=%s gsreg=%s game=%s", *listen, *login, *gsreg, *game)
	log.Fatal(http.ListenAndServe(*listen, api.Handler()))
}
