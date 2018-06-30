package main

import (
	//"crypto/tls"
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	//"runtime"
	"strings"
	"syscall"

	"github.com/munsy/guild/conf"
	"github.com/munsy/guild/pkg/router"
	"golang.org/x/crypto/ssh/terminal"
)

func redirect(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	var target string

	target = "https://" + req.Host + req.URL.Path

	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}

	http.Redirect(w, req, target,
		http.StatusTemporaryRedirect)
}

func credentials() (string, string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("Bad password read")
		panic(err)
	}
	password := string(bytePassword)

	return strings.TrimSpace(username), strings.TrimSpace(password)
}

func main() {
	fmt.Println("Initializing...")
	var username, password string
	//fmt.Println("Setting up runtime...")
	//runtime.GOMAXPROCS(runtime.NumCPU()) // Use max amount of cores
	//fmt.Println("Set runtime to use maximum amount of cores.")
	if 3 == len(os.Args) {
		username = os.Args[1]
		password = os.Args[2]
	} else {
		username, password = credentials()
	}
	db := &conf.MariaDBConfig{
		username,
		"",
		password,
		"localhost",
		"3306",
		"guild",
		"",
	}

	fmt.Println("Configuring server settings...")

	tls, err := db.GetTLS()
	if nil != err {
		fmt.Println("TLS retrieval attempt failed:")
		fmt.Println(err.Error())
	}

	cfg := &conf.Config{
		db,
		tls,
	}

	err = cfg.DB.Test()
	if nil != err {
		fmt.Println("Database test failed:")
		fmt.Println(err.Error())
	}

	fmt.Println("Starting server...")

	router := router.NewRouter()

	cssHandler := http.StripPrefix("/css/", http.FileServer(http.Dir("./web/css/")))
	jsHandler := http.StripPrefix("/js/", http.FileServer(http.Dir("./web/js/")))
	imagesHandler := http.StripPrefix("/images/", http.FileServer(http.Dir("./web/images/")))
	newsImagesHandler := http.StripPrefix("/images/news/", http.FileServer(http.Dir("./web/images/news/")))

	router.PathPrefix("/css/").Handler(cssHandler)
	router.PathPrefix("/js/").Handler(jsHandler)
	router.PathPrefix("/images/").Handler(imagesHandler)
	router.PathPrefix("/images/news/").Handler(newsImagesHandler)

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	if nil == cfg.TLS {
		fmt.Println("TLS configuration not set. Falling back to HTTP...")
		http.ListenAndServe(":80", nil)
	} else {
		fmt.Println("Redirecting HTTPS traffic to " + cfg.TLS.Addr)
		// Redirect all HTTP requests to HTTPS.
		go http.ListenAndServe(":80", http.HandlerFunc(redirect))

		// Start the server through TLS/SSL.
		log.Fatal(http.ListenAndServeTLS(cfg.TLS.Addr, cfg.TLS.CertFile, cfg.TLS.KeyFile, nil))
	}
}