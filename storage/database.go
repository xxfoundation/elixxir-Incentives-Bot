////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"time"
)

// database interface holds function definitions for storage
type database interface {
	CheckUser(id string) (string, error)
	UseCode(id, code string) error
	CheckRegStatus(id *id.ID) (bool, error)
}

// DatabaseImpl struct implements the database interface with an underlying DB
type DatabaseImpl struct {
	db    *gorm.DB // Stored database connection
	udbDB *gorm.DB
}

type Code struct {
	Code  string `gorm:"primary_key;"`
	Uses  int    `gorm:"not null"`
	Total int    `gorm:"not null"`
	Users []User `gorm:"foreignKey:code;references:code"`
}

type User struct {
	ID   string `gorm:"primary_key"`
	Code string `gorm:"not null"`
}

// MapImpl struct implements the database interface with an underlying Map
type MapImpl struct {
	coupons map[string]*Code
	users   map[string]*Code
	sync.RWMutex
}

// newDatabase initializes the database interface
// Returns a database interface and error
func newDatabase(params Params, udbParams Params) (database, error) {
	var err, udbErr error
	var db, udbDb *gorm.DB
	// Connect to the database if the correct information is provided
	if params.Address != "" && params.Port != "" && udbParams.Address != "" && udbParams.Port != "" {
		// Create the database connection
		connectString := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			params.Address, params.Port, params.Username, params.DBName)
		udConnectString := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			udbParams.Address, udbParams.Port, udbParams.Username, udbParams.DBName)
		// Handle empty database password
		if len(params.Password) > 0 {
			connectString += fmt.Sprintf(" password=%s", params.Password)
		}
		if len(udbParams.Password) > 0 {
			connectString += fmt.Sprintf(" password=%s", udbParams.Password)
		}
		db, err = gorm.Open(postgres.Open(connectString), &gorm.Config{
			Logger: logger.New(jww.TRACE, logger.Config{LogLevel: logger.Info}),
		})
		udbDb, err = gorm.Open(postgres.Open(udConnectString), &gorm.Config{
			Logger: logger.New(jww.TRACE, logger.Config{LogLevel: logger.Info}),
		})
	}

	// Return the map-backend interface
	// in the event there is a database error or information is not provided
	if (params.Address == "" && params.Port == "" && udbParams.Address == "" && udbParams.Port == "") || err != nil || udbErr != nil {

		if err != nil {
			jww.WARN.Printf("Unable to initialize database backend: %+v", err)
		} else if udbErr != nil {
			jww.WARN.Printf("Unable to initialize UDB database backend: %+v", udbErr)
		} else {
			jww.WARN.Printf("Database backend connection information not provided")
		}

		defer jww.INFO.Println("Map backend initialized successfully!")

		mapImpl := &MapImpl{}

		return database(mapImpl), nil
	}

	// Get and configure the internal database ConnPool
	sqlDb, err := db.DB()
	if err != nil {
		return database(&DatabaseImpl{}), errors.Errorf("Unable to configure database connection pool: %+v", err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDb.SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the Database.
	sqlDb.SetMaxOpenConns(50)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be idle.
	sqlDb.SetConnMaxIdleTime(10 * time.Minute)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDb.SetConnMaxLifetime(12 * time.Hour)

	// Initialize the database schema
	// WARNING: Order is important. Do not change without database testing
	models := []interface{}{Code{}, User{}}
	for _, model := range models {
		err = db.AutoMigrate(model)
		if err != nil {
			return database(&DatabaseImpl{}), err
		}
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return &DatabaseImpl{db: db, udbDB: udbDb}, nil
}
