package model

import (
	"database/sql/driver"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	stringArrayCodec   = pgtype.NewMap()
	stringArrayCodecMu sync.Mutex
)

type StringArray []string

func NewStringArray(items []string) StringArray {
	if len(items) == 0 {
		return StringArray{}
	}
	cloned := make(StringArray, len(items))
	copy(cloned, items)
	return cloned
}

func (a StringArray) Slice() []string {
	if len(a) == 0 {
		return []string{}
	}
	cloned := make([]string, len(a))
	copy(cloned, a)
	return cloned
}

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	stringArrayCodecMu.Lock()
	defer stringArrayCodecMu.Unlock()
	buf, err := stringArrayCodec.Encode(
		pgtype.TextArrayOID,
		pgtype.TextFormatCode,
		pgtype.FlatArray[string](a),
		nil,
	)
	if err != nil {
		return nil, err
	}
	return string(buf), nil
}

func (a *StringArray) Scan(src any) error {
	if a == nil {
		return fmt.Errorf("model.StringArray: Scan on nil pointer")
	}

	switch value := src.(type) {
	case nil:
		*a = StringArray{}
		return nil
	case []string:
		*a = NewStringArray(value)
		return nil
	case string:
		return a.scanText([]byte(value))
	case []byte:
		return a.scanText(value)
	default:
		return fmt.Errorf("model.StringArray: unsupported Scan type %T", src)
	}
}

func (a *StringArray) scanText(src []byte) error {
	if len(src) == 0 {
		*a = StringArray{}
		return nil
	}

	var values pgtype.FlatArray[string]
	stringArrayCodecMu.Lock()
	defer stringArrayCodecMu.Unlock()
	if err := stringArrayCodec.Scan(
		pgtype.TextArrayOID,
		pgtype.TextFormatCode,
		src,
		&values,
	); err != nil {
		return err
	}

	*a = NewStringArray([]string(values))
	return nil
}
