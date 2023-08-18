package driverDB

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func OpenDb(dsn string) (*sql.DB, error) {
	db,err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	
	err = db.Ping()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return db, err
}