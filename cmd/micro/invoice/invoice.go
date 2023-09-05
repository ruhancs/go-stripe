package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const version = "1.0.0"

type config struct {
	port int
	smtp struct {
		host string
		port int
		username string
		password string
	}
	frontend string // url de reset de senha
}

type application struct {
	config config
	infolog *log.Logger
	errorLog *log.Logger
	version string
}

func (app *application) server() error {
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", app.config.port),
		Handler: app.routes(),//routes configurado em route.go
		IdleTimeout: 30 * time.Second,
		ReadTimeout: 10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	app.infolog.Println( fmt.Printf("Starting invoice service on port %d", app.config.port))

	return srv.ListenAndServe()
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env")
	}
	var cfg config
	
	//flag para usar na linha de comando
	flag.IntVar(&cfg.port, "port", 5000, "Server Port to listen on")
	flag.StringVar(&cfg.frontend, "frontend", "http://localhost:4000", "frontend url")
	flag.StringVar(&cfg.smtp.host, "smthost", "smtp.mailtrap.io", "smtp host")
	flag.IntVar(&cfg.smtp.port, "smtport", 587, "smtp port")
	//flag.StringVar(&cfg.smtp.username, "smtusername", "username", "smtp port")
	//flag.StringVar(&cfg.smtp.password, "password", "password", "smtp port")

	//definir as variaveis na linah de comando
	flag.Parse()

	cfg.smtp.username = os.Getenv("SMTP_USERNAME")
	cfg.smtp.password = os.Getenv("SMTP_PASSWORD")

	//logs da app
	infolog := log.New(os.Stdout, " INFO\t", log.Ldate| log.Ltime)
	errorLog := log.New(os.Stdout, " ERROR\t", log.Ldate| log.Ltime| log.Lshortfile)

	app := &application{
		config: cfg,
		infolog: infolog,
		errorLog: errorLog,
		version: version,
	}

	app.CreateDirIfNotExist("./invoices")

	err = app.server()
	if err != nil {
		log.Fatal(err)
	}
}