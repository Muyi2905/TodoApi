package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func initDb() {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		panic("dsn envireonment variable is not set")
	}
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("error connecting to database", err)
	}
	fmt.Println("Database connection successful")
}

func main() {
	initDb()
	r := gin.Default()
	err := r.Run()
	if err != nil {
		fmt.Errorf("error starting server %v", err)
	}

}
