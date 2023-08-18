package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	driverDB "github.com/ruhancs/go-stripe/internal/driver"
	"github.com/ruhancs/go-stripe/internal/models"
)

const version = "1.0.0"

type config struct {
	port int
	env string
	db struct {
		dataSourceName string
	}
	stripe struct {
		secret string
		key string
	}
}

type application struct {
	config config
	infolog *log.Logger
	errorLog *log.Logger
	version string
	DB models.DbModel
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

	app.infolog.Println( fmt.Printf("Starting Back end server on port %d", app.config.port))

	return srv.ListenAndServe()
}

//main do backend
func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env")
	}
	var cfg config

	dbPassword := os.Getenv("DB_PASSWORD")
	dbUser := os.Getenv("DB_USER")

	//flag para usar na linha de comando
	flag.IntVar(&cfg.port, "port", 4001, "Server Port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application enviroment {development | production}")
	flag.StringVar(&cfg.db.dataSourceName, "dataSourceName", fmt.Sprintf(`%s:%s@tcp(localhost:3306)/widgets?parseTime=true&tls=false`,dbUser,dbPassword), "DSN")

	//definir as variaveis na linah de comando
	flag.Parse()

	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")

	//logs da app
	infolog := log.New(os.Stdout, " INFO\t", log.Ldate| log.Ltime)
	errorLog := log.New(os.Stdout, " ERROR\t", log.Ldate| log.Ltime| log.Lshortfile)

	conn,err := driverDB.OpenDb(cfg.db.dataSourceName)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer conn.Close()

	app := &application{
		config: cfg,
		infolog: infolog,
		errorLog: errorLog,
		version: version,
		DB: models.DbModel{DB: conn},
	}

	err = app.server()
	if err != nil {
		log.Fatal(err)
	}
}