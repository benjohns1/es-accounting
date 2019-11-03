package uuid

import (
	"fmt"

	"github.com/google/uuid"
)

func New() (string, error) {
	id, err := uuid.NewUUID()
	idStr := id.String()
	if idStr == "" {
		err = fmt.Errorf("unable to create UUID")
	}
	return idStr, err
}

func Generate(count int) ([]string, error) {
	uuids := []string{}
	for i := 0; i < count; i++ {
		id, err := New()
		if err != nil {
			return uuids, err
		}
		uuids = append(uuids, id)
	}
	return uuids, nil
}
