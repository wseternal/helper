package sqlimpl

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// StructScan scan sql row into destVal, which must be an valid pointer to a struct
// the columns selected from the data table will be deduce from the all the fields of destVal
func (dt *DataTable) StructScan(condOrderLimit string, destValType reflect.Type, onRow func(destVal interface{})) error {
	if destValType.Kind() != reflect.Struct {
		return fmt.Errorf("invalid destValType %v, require struct", destValType)
	}

	if destValType.NumField() == 0 {
		return errors.New("no fields found for destVal struct object")
	}

	cnt := destValType.NumField()

	fieldNames := make([]string, cnt)
	valPtrs := make([]interface{}, cnt)
	var fn string
	var tmp reflect.StructField

	// v: pointer to an instance of destValType
	v := reflect.New(destValType)

	for i := 0; i < destValType.NumField(); i++ {
		tmp = destValType.Field(i)
		fieldNames[i] = tmp.Name
		if len(tmp.Tag) > 0 {
			fn = strings.SplitN(tmp.Tag.Get("sql"), ",", 2)[0]
			if len(fn) > 0 {
				fieldNames[i] = fn
			}
		}
		valPtrs[i] = v.Elem().Field(i).Addr().Interface()
	}

	query := fmt.Sprintf("select %s from %s %s", strings.Join(fieldNames, ","), dt.Name, condOrderLimit)

	stmt, err := dt.impl.DB.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		return err
	}
	for rows.Next() {
		if err = rows.Scan(valPtrs...); err != nil {
			return err
		}
		onRow(v.Interface())
	}

	return err
}
