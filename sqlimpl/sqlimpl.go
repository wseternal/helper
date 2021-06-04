// Package sqlimpl is the sql implementor for storing information on underline DB
package sqlimpl

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	`os`
	"strings"
	"time"

	"github.com/wseternal/helper/fastjson"

	// add mysql driver implement
	_ "github.com/go-sql-driver/mysql"
)

// SQLImpl encapsulates database related information and action
type SQLImpl struct {
	DB     *sql.DB
	Driver string
	DSN    string
}

const (
	DriverlMysql = "mysql"
)

var (
	jsonNullBytes = []byte("null")
)

// ConnectDB connect to specific database using given driver and dsn
// dns is in format: username:password@protocol(address)/dbname,
// e.g.: username:password@tcp(127.0.0.1:3306)/testdb
func ConnectDB(driver string, dsn string) (*SQLImpl, error) {
	DB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err = DB.Ping(); err != nil {
		return nil, err
	}
	DB.SetConnMaxLifetime(time.Minute * 2)
	return &SQLImpl{
		DB:     DB,
		Driver: driver,
		DSN:    dsn,
	}, err
}

// DataTable encapsulate informat on database table
type DataTable struct {
	Name            string
	CreateStatement string
	impl            *SQLImpl
}

type NullString struct {
	sql.NullString
}

func init() {
}

func (v NullString) MarshalJSON() ([]byte, error) {
	if v.Valid {
		return json.Marshal(v.String)
	}
	return jsonNullBytes, nil
}

func (v *NullString) UnmarshalJSON(data []byte) error {
	var tmp *string
	var err error
	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	if tmp == nil {
		v.Valid = false
	} else {
		v.String = *tmp
		v.Valid = true
	}
	return nil
}

func (impl *SQLImpl) Close() error {
	return impl.DB.Close()
}

func (impl *SQLImpl) GetTable(name string) (*DataTable, error) {
	query := fmt.Sprintf("SHOW CREATE TABLE %s", name)

	stmt, err := impl.DB.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var tmp, create string
	if err = stmt.QueryRow().Scan(&tmp, &create); err != nil {
		return nil, err
	}

	return &DataTable{
		Name:            name,
		CreateStatement: create,
		impl:            impl,
	}, nil
}

// InitTables initialize the tables
func (impl *SQLImpl) InitTables(tables []*DataTable) error {
	tx, err := impl.DB.Begin()
	if err != nil {
		return err
	}

	rollback := false
	for _, s := range tables {
		fmt.Printf("create table %s %s\n", s.Name, s.CreateStatement)
		_, err = tx.Exec(s.CreateStatement)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create table %s %s failed\n", s.Name, s.CreateStatement)
			rollback = true
			goto out
		}
		s.impl = impl
	}
out:
	if rollback {
		tableNames := make([]string, len(tables))
		for i, t := range tables {
			tableNames[i] = t.Name
		}
		fmt.Fprintf(os.Stderr, "roll back creating table actions as error occurred\n")
		switch impl.Driver {
		case "mysql":
			tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", strings.Join(tableNames, ",")))
		default:
			tx.Rollback()
		}
	} else {
		err = tx.Commit()
	}
	return err
}

func (t *DataTable) InnerJoin(tbAnother *DataTable, joinCond string, whereCond string, columns []string, vals ...interface{}) error {
	if len(columns) != len(vals) {
		return fmt.Errorf("selected column length %d is not equal to length of vals %d", len(columns), len(vals))
	}
	if joinCond == "" {
		return errors.New("join condition is required")
	}
	columnString := strings.Join(columns, ",")
	queryString := fmt.Sprintf("select %s from %s inner join %s on %s %s;", columnString, t.Name, tbAnother.Name, joinCond, whereCond)
	stmt, err := t.impl.DB.Prepare(queryString)
	if err != nil {
		return err
	}
	defer stmt.Close()
	err = stmt.QueryRow().Scan(vals...)
	if err == sql.ErrNoRows {
		err = fmt.Errorf("%s", "no result")
	}
	return err
}

// Find get column value of table and place it in the val,
// val must be a valid pointer value.
func (t *DataTable) Find(column, condition string, val interface{}) error {
	if len(column) == 0 {
		return fmt.Errorf("empty column name")
	}
	queryString := fmt.Sprintf("select %s from %s %s;", column, t.Name, condition)
	stmt, err := t.impl.DB.Prepare(queryString)
	if err != nil {
		return err
	}
	defer stmt.Close()
	return stmt.QueryRow().Scan(val)
}

func (t *DataTable) Delete(condition string) (sql.Result, error) {
	execStr := fmt.Sprintf("delete from %s %s;", t.Name, condition)
	stmt, err := t.impl.DB.Prepare(execStr)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Exec()
}

// Insert insert given values into table, element of keys is mapped with element of args
// one by one.
func (t *DataTable) Insert(keys []string, args ...interface{}) (sql.Result, error) {
	if len(keys) != len(args) {
		return nil, fmt.Errorf("length of key %d not equal length of args: %d", len(keys), len(args))
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("empty keys and values")
	}
	cols := strings.Join(keys, ",")
	placeHolders := strings.Repeat("?,", len(keys)-1) + "?"
	insertString := fmt.Sprintf("insert into %s (%s) values (%s);", t.Name, cols, placeHolders)
	stmt, err := t.impl.DB.Prepare(insertString)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	result, err := stmt.Exec(args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execute %s with args %v, failed, error: %s\n", insertString, args, err)
	}
	return result, err
}

// Update update row satisfied with condition with given values,
// element of keys is mapped with element of args one by one.
func (t *DataTable) Update(condition string, keys []string, args ...interface{}) (sql.Result, error) {
	if len(condition) == 0 {
		return nil, fmt.Errorf("empty condition string")
	}
	if len(keys) != len(args) {
		return nil, fmt.Errorf("length of key %d not equal length of args: %d", len(keys), len(args))
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("empty keys and values")
	}
	sets := make([]string, len(keys))
	for k, v := range keys {
		sets[k] = fmt.Sprintf("%s=?", v)
	}
	setStr := strings.Join(sets, ",")
	updateString := fmt.Sprintf("update %s set %s %s;", t.Name, setStr, condition)
	result, err := t.impl.DB.Exec(updateString, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execute %s failed, error: %s\n", updateString, err)
	}
	return result, err
}

func scanToJSONObject(columns []string, rows *sql.Rows) (*fastjson.JSONObject, error) {
	obj := fastjson.NewObject()

	cnt := len(columns)
	valPtrs := make([]interface{}, cnt)

	for i := 0; i < cnt; i++ {
		valPtrs[i] = new(NullString)
	}

	var err error
	if err = rows.Scan(valPtrs...); err != nil {
		return nil, err
	}
	for idx, col := range columns {
		v := valPtrs[idx].(*NullString)
		if v.Valid {
			obj.Put(col, v.String)
		} else {
			obj.Put(col, nil)
		}
	}
	return obj, nil
}

func (impl *SQLImpl) FetchOne(query string, args ...interface{}) (*fastjson.JSONObject, error) {
	list, err := impl.FetchAll(query, args...)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return list[0], nil
	}
	return nil, nil
}

func (impl *SQLImpl) FetchAll(query string, args ...interface{}) ([]*fastjson.JSONObject, error) {
	stmt, err := impl.DB.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, err := stmt.Query(args...)
	var ret []*fastjson.JSONObject

	columns, _ := rows.Columns()
	var obj *fastjson.JSONObject
	for rows.Next() {
		obj, err = scanToJSONObject(columns, rows)
		if err != nil {
			return nil, err
		}
		ret = append(ret, obj)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func (impl *SQLImpl) Insert(table string, replace bool, args *fastjson.JSONObject) (sql.Result, error) {
	if args == nil || len(args.Keys()) == 0 {
		return nil, errors.New("non-empty args is required to insert into table")
	}
	var act string
	if replace {
		act = "replace"
	} else {
		act = "insert"
	}
	keys, values := args.Entries()

	cols := strings.Join(keys, ",")
	argPlaceHolders := strings.Repeat("?,", len(keys)-1) + "?"
	insertString := fmt.Sprintf("%s into %s (%s) values (%s);", act, table, cols, argPlaceHolders)
	stmt, err := impl.DB.Prepare(insertString)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	result, err := stmt.Exec(values...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Execute %s with args %v, failed, error: %s\n", insertString, args, err)
	}
	return result, err
}

func (impl *SQLImpl) Update(table string, args *fastjson.JSONObject, where string, whereArgs ...interface{}) (sql.Result, error) {
	if args == nil || len(args.Keys()) == 0 {
		return nil, errors.New("non-empty args is required to update table")
	}
	keys, values := args.Entries()
	var updateSets  []string

	for _, key := range keys {
		updateSets = append(updateSets, fmt.Sprintf("%s=?", key))
	}

	updateString := fmt.Sprintf("update %s set %s where %s", table, strings.Join(updateSets, ","), where)
	stmt, err := impl.DB.Prepare(updateString)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	values = append(values, whereArgs...)

	result, err := stmt.Exec(values...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "execute %s with args %v failed, error: %s\n", updateString, values, err)
	}
	return result, err
}
