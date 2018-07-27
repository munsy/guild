package main

import (
	//"crypto/tls"
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	//"runtime"
	"encoding/json"
	"strings"
	"syscall"
	"time"

	"github.com/munsy/guild/config"
	//"github.com/munsy/guild/pkg/router"
	"github.com/munsy/gobattlenet"
	"golang.org/x/crypto/ssh/terminal"
)

// handleRealmStatus handles realm statuses for AngularJS
func handleRealmStatus(w http.ResponseWriter, r *http.Request) {
	settings := &battlenet.Settings{
		Client: &http.Client{Timeout: (10 * time.Second)},
		Locale: battlenet.Locale.AmericanEnglish,
		Region: battlenet.Regions.US,
	}

	client, err := battlenet.WoWClient(settings, "your-key-here")

	if nil != err {
		fmt.Fprintln(w, "There was an error :(")
		fmt.Println(w, err.Error())
		return
	}

	switch r.Method {
	case "GET":
		response, err := client.RealmStatus()

		if nil != err {
			fmt.Fprintln(w, "There was an error :(")
			fmt.Println(w, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response.Data)
		break
	default:
		fmt.Fprintln(w, "Sorry, nothing here!")
	}
}

func redirect(w http.ResponseWriter, req *http.Request) {
	// remove/add non default ports from req.Host
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
	db := &config.MariaDBConfig{
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

	cfg := &config.Config{
		db,
		tls,
	}

	err = cfg.DB.Test()
	if nil != err {
		fmt.Println("Database test failed:")
		fmt.Println(err.Error())
	}

	fmt.Println("Starting server...")

	/*
		router := router.NewRouter()

		staticHandler := http.StripPrefix("/", http.FileServer(http.Dir("./static/")))
		cssHandler := http.StripPrefix("/css/", http.FileServer(http.Dir("./static/css/")))
		jsHandler := http.StripPrefix("/js/", http.FileServer(http.Dir("./static/js/")))
		imagesHandler := http.StripPrefix("/images/", http.FileServer(http.Dir("./static/images/")))
		newsImagesHandler := http.StripPrefix("/images/news/", http.FileServer(http.Dir("./static/images/news/")))

		router.PathPrefix("/").Handler(staticHandler)
		router.PathPrefix("/css/").Handler(cssHandler)
		router.PathPrefix("/js/").Handler(jsHandler)
		router.PathPrefix("/images/").Handler(imagesHandler)
		router.PathPrefix("/images/news/").Handler(newsImagesHandler)
	*/

	// Creates a new serve mux
	mux := http.NewServeMux()

	// Create room for static files serving
	mux.Handle("/bootstrap/", http.StripPrefix("/bootstrap", http.FileServer(http.Dir("./bootstrap"))))
	mux.Handle("/css/", http.StripPrefix("/css", http.FileServer(http.Dir("./css"))))
	mux.Handle("/html/", http.StripPrefix("/html", http.FileServer(http.Dir("./html"))))
	mux.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir("./images"))))
	mux.Handle("/js/", http.StripPrefix("/js", http.FileServer(http.Dir("./js"))))

	// Roster test
	mux.HandleFunc("/api/v1/realms/status", handleRealmStatus)

	// Any other request, we should render our SPA's only html file,
	// Allowing angular to do the routing on anything else other then the api
	// and the files it needs for itself to work.
	// Order here is critical. This html should contain the base tag like
	// <base href="/"> *href here should match the HandleFunc path below
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "html/index.html")
	})

	if nil == cfg.TLS {
		fmt.Println("TLS configuration not set. Falling back to HTTP...")
		http.ListenAndServe(":80", mux)
	} else {
		fmt.Println("Redirecting HTTPS traffic to " + cfg.TLS.Addr)
		// Redirect all HTTP requests to HTTPS.
		go http.ListenAndServe(":80", http.HandlerFunc(redirect))

		// Start the server through TLS/SSL.
		log.Fatal(http.ListenAndServeTLS(cfg.TLS.Addr, cfg.TLS.CertFile, cfg.TLS.KeyFile, mux))
	}
}
