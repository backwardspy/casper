package dal

import (
	"casper/models"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// InitDB creates and returns a database connection.
func InitDB(dbPath string) *gorm.DB {
	db, err := gorm.Open(
		sqlite.Open(dbPath),
		&gorm.Config{
			// Logger: logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
			// 	SlowThreshold: 200 * time.Millisecond,
			// 	LogLevel:      logger.Info,
			// 	Colorful:      true,
			// }),
		},
	)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	log.Println("Connected to database.")

	db.AutoMigrate(&models.MeatballDay{}, &models.MeatballRole{}, &models.MeatballChannel{})
	log.Println("Migrated database.")

	return db
}

// UpsertMeatballDay inserts or updates the given meatball day.
func UpsertMeatballDay(meatballDay models.MeatballDay, db *gorm.DB) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "guild_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"month", "day"}),
	}).Create(&meatballDay).Error
}

// GetMeatballDay gets the meatball day for the given guild & user.
func GetMeatballDay(
	guildID string,
	userID string,
	db *gorm.DB,
) (*models.MeatballDay, error) {
	var meatballDay models.MeatballDay
	err := db.Where(
		&models.MeatballDay{
			GuildID: guildID,
			UserID:  userID,
		},
	).Take(&meatballDay).Error

	if err != nil {
		return nil, err
	}

	return &meatballDay, nil
}

// UpsertMeatballRole inserts or updates the given meatball role.
func UpsertMeatballRole(meatballRole models.MeatballRole, db *gorm.DB) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "guild_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role_id"}),
	}).Create(&meatballRole).Error
}

// UpsertMeatballChannel inserts or updates the given meatball channel.
func UpsertMeatballChannel(
	meatballChannel models.MeatballChannel,
	db *gorm.DB,
) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "guild_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"channel_id"}),
	}).Create(&meatballChannel).Error
}

// GetMeatballChannel returns the saved meatball channel for the given guild.
func GetMeatballChannel(
	guildID string,
	db *gorm.DB,
) (*models.MeatballChannel, error) {
	var meatballChannel models.MeatballChannel
	err := db.Where(
		&models.MeatballChannel{
			GuildID: guildID,
		},
	).Take(&meatballChannel).Error

	if err != nil {
		return nil, err
	}

	return &meatballChannel, nil
}

// GetNextMeatballDay gets the next occurring meatball day.
func GetNextMeatballDay(
	guildID string,
	db *gorm.DB,
) (*models.MeatballDay, error) {
	var meatballDays []models.MeatballDay
	err := db.Where(
		&models.MeatballDay{
			GuildID: guildID,
		},
	).Order(
		clause.OrderByColumn{
			Column: clause.Column{
				Name: "month",
			},
		},
	).Order(
		clause.OrderByColumn{
			Column: clause.Column{
				Name: "day",
			},
		},
	).Find(&meatballDays).Error

	if err != nil {
		return nil, err
	}

	if len(meatballDays) == 0 {
		return nil, nil
	}

	return findNextMeatballDay(meatballDays), nil
}

// Finds the next occurring meatball day.
func findNextMeatballDay(meatballDays []models.MeatballDay) *models.MeatballDay {
	now := time.Now()
	var next *models.MeatballDay

	for _, meatballDay := range meatballDays {
		if time.Month(meatballDay.Month) > now.Month() {
			next = &meatballDay
			break
		} else if time.Month(meatballDay.Month) == now.Month() && int(meatballDay.Day) >= now.Day() {
			next = &meatballDay
			break
		}
	}

	if next == nil {
		next = &meatballDays[0]
	}

	return next
}
