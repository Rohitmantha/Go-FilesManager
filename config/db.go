package config

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"github.com/joho/godotenv"
)

var DB *sql.DB

func ConnectDB() {
	if err := godotenv.Load(); err != nil {
        log.Fatalf("Error loading .env file")
    }

    dbusername := os.Getenv("DB_USERNAME")
	dbpassword := os.Getenv("DB_PWD")
	dbhost := os.Getenv("DB_HOST") // Add DB_HOST to specify the host address
	dbport := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbusername, dbpassword, dbhost, dbport, dbname)
	var err error
	DB, err = sql.Open("mysql", dsn)
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
