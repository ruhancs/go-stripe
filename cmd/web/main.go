package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
	driverDB "github.com/ruhancs/go-stripe/internal/driver"
	"github.com/ruhancs/go-stripe/internal/models"
)

const version = "1.0.0"
const cssVersion = "1"
var session *scs.SessionManager

type config struct {
	port int
	env string
	api string
	db struct {
		dataSourceName string
	}
	stripe struct {
		secret string
		key string
	}
	secretKey string
	frontend string
}

type application struct {
	config config
	infolog *log.Logger
	errorLog *log.Logger
	templateCahe map[string]*template.Template
	version string
	DB models.DbModel
	Session *scs.SessionManager
}

func (app *application) server() error {
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", app.config.port),
		Handler: app.routes(),
		IdleTimeout: 30 * time.Second,
		ReadTimeout: 10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	app.infolog.Println( fmt.Printf("Starting HTTP server on port %d", app.config.port))

	return srv.ListenAndServe()
}

func main() {
	
	gob.Register(TransactionData{})//colocar um map de string na sessao para inserir valores
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env")
	}
	var cfg config

	dbPassword := os.Getenv("DB_PASSWORD")
	dbUser := os.Getenv("DB_USER")

	//flag para usar na linha de comando
	flag.IntVar(&cfg.port, "port", 4000, "Server Port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application enviroment {development | production}")
	flag.StringVar(&cfg.db.dataSourceName, "dataSourceName", fmt.Sprintf(`%s:%s@tcp(localhost:3306)/widgets?parseTime=true&tls=false`,dbUser,dbPassword), "DSN")
	flag.StringVar(&cfg.api, "api", "http://localhost:4001", "URL to api")
	flag.StringVar(&cfg.frontend, "frontend", "http://localhost:4000", "frontend url")

	//definir as variaveis na linah de comando
	flag.Parse()

	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")

	cfg.secretKey = os.Getenv("SECRET_RESET_PASSWORD")

	//logs da app
	infolog := log.New(os.Stdout, " INFO\t", log.Ldate| log.Ltime)
	errorLog := log.New(os.Stdout, " ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	conn,err := driverDB.OpenDb(cfg.db.dataSourceName)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer conn.Close()

	//configurar a sessao do usuario
	session = scs.New()
	session.Lifetime = 24 * time.Hour //tempo de vida da sessao 24h
	session.Store = mysqlstore.New(conn)

	templateCache := make(map[string] *template.Template)

	app:= &application{
		config: cfg,
		infolog: infolog,
		errorLog: errorLog,
		templateCahe: templateCache,
		version: version,
		DB: models.DbModel{DB: conn},
		Session: session,
	}

	//executar em background conexao com websocket
	go app.ListenToWsChannel()

	err = app.server()
	if err != nil {
		app.errorLog.Println(err)
		log.Fatal(err)
	}
}
