////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

//func TestStorage(t *testing.T) {
//	db, err := NewStorage(Params{
//		Username: "jonahhusson",
//		Password: "",
//		DBName:   "cmix_server",
//		Address:  "0.0.0.0",
//		Port:     "5432",
//	}, Params{
//		Username: "jonahhusson",
//		Password: "",
//		DBName:   "cmix_server",
//		Address:  "0.0.0.0",
//		Port:     "5433",
//	})
//
//	if err != nil {
//		t.Error(err)
//	}
//
//	uid := id.NewIdFromString("zezima", id.User, t)
//	ok, err := db.CheckRegStatus(uid)
//	if err != nil {
//		t.Error(err)
//	}
//	if !ok {
//		t.Errorf("User %s is not registered with UD", uid.HexEncode())
//	}
//
//	strResponse := db.Register(uid, "test")
//	t.Error(strResponse)
//}
