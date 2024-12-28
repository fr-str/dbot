package types

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
)

type Aliases []string

func (a *Aliases) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("invalid type in DB, type=%s, value=%s", reflect.TypeOf(src).Kind(), src)
	}

	*a = strings.Split(s, ",")
	return nil
}

func (a Aliases) Value() (driver.Value, error) {
	return strings.Join(a, ","), nil
}
