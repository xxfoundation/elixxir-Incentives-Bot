package storage

import "gitlab.com/xx_network/primitives/id"

func (m *MapImpl) CheckUser(id string) (string, error) {
	return "", nil
}

func (m *MapImpl) UseCode(id, code string) error {
	return nil
}

func (m *MapImpl) CheckRegStatus(id *id.ID) (bool, error) {
	return true, nil
}
