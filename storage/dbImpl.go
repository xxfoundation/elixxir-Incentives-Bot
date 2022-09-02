////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"gorm.io/gorm"
)

func (db *DatabaseImpl) CheckUser(id string) (string, error) {
	u := &User{}
	err := db.db.Where("id = ?", id).Take(&u).Error
	if err != nil {
		return "", err
	}
	return u.Code, nil
}

func (db *DatabaseImpl) UseCode(id, code string) error {
	return db.db.Transaction(func(tx *gorm.DB) error {
		u := &User{
			ID:   id,
			Code: code,
		}
		err := tx.Create(&u).Error
		if err != nil {
			return errors.WithMessage(err, "Failed to add user")
		}

		c := &Code{}
		err = tx.Model(&c).Where("code = ?", code).
			Updates(map[string]interface{}{
				"uses":  gorm.Expr("uses + ?", 1),
				"total": gorm.Expr("total + ?", 10),
			}).Error
		if err != nil {
			return errors.WithMessage(err, "Failed to use code")
		}
		return nil
	})
}

func (db *DatabaseImpl) CheckRegStatus(id *id.ID) (bool, error) {
	var count int
	err := db.udbDB.Raw("select count(*) from users inner join facts on users.id = facts.user_id where users.id = ? and facts.type = ?", "\\"+id.HexEncode()[1:], fact.Phone).Scan(&count).Error
	if err != nil {
		return false, errors.WithMessage(err, "Failed to get registration status")
	}
	return count > 0, nil
}
