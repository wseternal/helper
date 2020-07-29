package sqlimpl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// StructScan scan sql row into destVal, which must be an valid pointer to a struct
// the columns selected from the data table will be deduce from the all the fields of destVal
// Pay Attention: destVal of onRow is reused, you must deep copy that variable if you need store it
func (dt *DataTable) StructScan(ctx context.Context, condOrderLimit string, destValType reflect.Type, onRow func(destVal interface{}), args ...interface{}) error {
	if destValType.Kind() != reflect.Struct {
		return fmt.Errorf("invalid destValType %v, require struct", destValType)
	}

	if destValType.NumField() == 0 {
		return errors.New("no fields found for destVal struct object")
	}

	if ctx == nil {
		ctx = context.Background()
	}
	fieldNames := make([]string, 0)
	valPtrs := make([]interface{}, 0)
	var fn string
	var tmp reflect.StructField

	// v: pointer to an instance of destValType
	v := reflect.New(destValType)

	for i := 0; i < destValType.NumField(); i++ {
		tmp = destValType.Field(i)
		name := tmp.Name
		if len(tmp.Tag) > 0 {
			fn = strings.SplitN(tmp.Tag.Get("sql"), ",", 2)[0]
			if len(fn) > 0 {
				if fn == "-" {
					continue
				}
				name = fn
			}
		}
		fieldNames = append(fieldNames, name)
		valPtrs = append(valPtrs, v.Elem().Field(i).Addr().Interface())
	}

	query := fmt.Sprintf("select %s from %s %s", strings.Join(fieldNames, ","), dt.Name, condOrderLimit)

	stmt, err := dt.impl.DB.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return err
	}

	empty := true
	for rows.Next() {
		empty = false
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = rows.Scan(valPtrs...); err != nil {
			return err
		}
		onRow(v.Interface())
	}
	if empty {
		return sql.ErrNoRows
	}
	return err
}
