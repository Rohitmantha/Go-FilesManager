package config

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func ConnectDB() {
	var err error
	DB, err = sql.Open("mysql", "root:@tcp(localhost:3306)/authdb")
	if err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	// Ping the DB to test connection
	err = DB.Ping()
	if err != nil {
		log.Fatal("Cannot connect to DB:", err)
	}
	fmt.Println("Connected to the database successfully")
}
