////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Handles the high level storage API.
// This layer merges the business logic layer and the database layer

package storage

import (
	"errors"
	"fmt"
	"gitlab.com/xx_network/primitives/id"
	"gorm.io/gorm"
)

// Params for creating a storage object
type Params struct {
	Username string
	Password string
	DBName   string
	Address  string
	Port     string
}

// Storage struct interfaces with the API for the storage layer
type Storage struct {
	// Stored Database interface
	database
}

// NewStorage creates a new Storage object wrapping a database interface
// Returns a Storage object, and error
func NewStorage(params Params, udbParams Params) (*Storage, error) {
	db, err := newDatabase(params, udbParams)
	storage := &Storage{db}
	return storage, err
}

// Register a user with the incentives bot.  Returns a response string
func (s *Storage) Register(uid *id.ID, code string) string {
	var strResponse string
	// Check if user has registered already
	usedCode, err := s.CheckUser(uid.String())
	if err != nil {
		// If err is recordNotFound continue
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Check registration status with UDB
			registered, err := s.CheckRegStatus(uid)
			if err != nil {
				// Failed to check UDB registration status
				strResponse = fmt.Sprintf("Could not use code %s (failed to check udb registration status): %+v", code, err)
			} else if !registered {
				// User has not registered a phone number with UDB
				strResponse = fmt.Sprintf("Could not use code %s (must have registered a phone number with UD)", code)
			} else {
				// Attempt to use the code sent
				err = s.UseCode(uid.String(), code)
				if err != nil {
					// Failed to use the code
					strResponse = fmt.Sprintf("Could not use code %s: %s", code, err.Error())
				} else {
					// Successfully registered with incentives
					strResponse = fmt.Sprintf("Thank you for using the xx messenger!  Your referral code %s has been registered.", code)
				}
			}
		} else {
			// Received unexpected error
			strResponse = fmt.Sprintf("Could not check user in database: %+v", err)
		}
	} else {
		// Registered already with incentives
		strResponse = fmt.Sprintf("User has already registered with incentives using code %s", usedCode)
	}
	return strResponse
}
