package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pblaravel/game-server-l2-go/internal/apitest"
)

func main() {
	javaLogin := flag.String("java-login", "127.0.0.1:2107", "Java login client TCP")
	javaGS := flag.String("java-gsreg", "127.0.0.1:9015", "Java login↔game TCP")
	javaGame := flag.String("java-game", "127.0.0.1:7778", "Java game client TCP")
	goLogin := flag.String("go-login", "127.0.0.1:3107", "Go login client TCP")
	goGS := flag.String("go-gsreg", "127.0.0.1:19015", "Go login↔game TCP")
	goGame := flag.String("go-game", "127.0.0.1:17778", "Go game client TCP")
	account := flag.String("account", "apitest", "account to create/login")
	skipJava := flag.Bool("skip-java", false, "only check Go against the Java source contract")
	skipGo := flag.Bool("skip-go", false, "only dump Java")
	flag.Parse()

	hash := []byte{1, 2, 3, 4}
	contract := apitest.ExpectedJavaContract()
	exit := 0

	var javaSnap, goSnap *apitest.Snapshot
	if !*skipJava {
		javaSnap = apitest.Capture("java", apitest.Target{Login: *javaLogin, GSReg: *javaGS, Game: *javaGame}, *account+"java", hash)
		defer javaSnap.Close()
		fmt.Println("=== Java snapshot ===")
		fmt.Println(string(javaSnap.JSON()))
		if diffs := javaSnap.MatchContract(contract); len(diffs) > 0 {
			fmt.Println("Java vs source contract:")
			for _, d := range diffs {
				fmt.Println("  ", d)
			}
			exit = 1
		}
		if len(javaSnap.Errors) > 0 {
			fmt.Println("Java capture errors:", javaSnap.Errors)
			exit = 1
		}
	}
	if !*skipGo {
		goSnap = apitest.Capture("go", apitest.Target{Login: *goLogin, GSReg: *goGS, Game: *goGame}, *account+"go", hash)
		defer goSnap.Close()
		fmt.Println("=== Go snapshot ===")
		fmt.Println(string(goSnap.JSON()))
		if diffs := goSnap.MatchContract(contract); len(diffs) > 0 {
			fmt.Println("Go vs source contract:")
			for _, d := range diffs {
				fmt.Println("  ", d)
			}
			exit = 1
		}
		if len(goSnap.Errors) > 0 {
			fmt.Println("Go capture errors:", goSnap.Errors)
			exit = 1
		}
	}
	if javaSnap != nil && goSnap != nil {
		diffs := apitest.Diff(javaSnap, goSnap)
		fmt.Println("=== Java vs Go ===")
		if len(diffs) == 0 {
			fmt.Println("opcodes and layouts match")
		} else {
			for _, d := range diffs {
				fmt.Println("  ", d)
			}
			exit = 1
		}
	}
	os.Exit(exit)
}
