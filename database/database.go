package database

import (
	"cafe/model"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDatabase() {
	var err error

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=63946229 dbname=cafe port=5432 sslmode=disable"
	}

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Bazanyň birikdirilmegi şowsuz boldy! Ýalňyşlyk: %v", err)
	}

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Bazanyň birikdirilmegi şowsuz boldy! Ýalňyşlyk: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Maglumat bazasyna birikmek mümkin däl: %v", err)
	}

	err = DB.AutoMigrate(
		&model.Cafe{},
		&model.CafePhone{},
		&model.User{},
		&model.FoodCategory{},
		&model.Food{},
	)
	if err != nil {
		log.Fatalf("Migrasiýa şowsuz boldy: %v", err)
	}

	log.Println("Bazanyň birikdirilmegi we migrasiýasy üstünlikli tamamlandy!")
}
