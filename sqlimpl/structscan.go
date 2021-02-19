package sqlimpl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
)


func (dt *DataTable) StructScanOne(ctx context.Context, condOrderLimit string, destValType reflect.Type, args ...interface{}) (destVal interface{}, err error) {
	var res interface{}
	if len(condOrderLimit) == 0 {
		condOrderLimit = "limit 1"
	}
	onRow := func(v interface{}) {
		if res != nil {
			return
		}
		res = v
	}

	if err = dt.StructScan(ctx, condOrderLimit, destValType, onRow, args...); err != nil {
		return nil, err
	}
	return res, nil
}

// StructScan scan sql row into destVal, which must be an valid pointer to a struct
// the columns selected from the data table will be deduce from the all the fields of destVal
// descValType shall be: reflect.TypeOf(T{})
// destVal in the onRow callback is: *T
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

	distinctExist := false
	for i := 0; i < destValType.NumField(); i++ {
		tmp = destValType.Field(i)
		name := tmp.Name
		if len(tmp.Tag) > 0 {
			fields := strings.SplitN(tmp.Tag.Get("sql"), ",", 2)
			fn = fields[0]
			if len(fn) > 0 {
				if fn == "-" {
					continue
				}
				if len(fields) == 2 && fields[1] == "distinct" {
					if distinctExist {
						return fmt.Errorf("distct can only be specifed on one column")
					}
					name = fmt.Sprintf("distinct(%s)", fn)
					distinctExist = true
				} else {
					name = fn
				}
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
