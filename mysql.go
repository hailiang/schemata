package schemata

import (
	"database/sql"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type MySQL struct {
	DB           *sql.DB
	ForceFloat32 bool
}

func (d MySQL) Schema(source string) (*Schema, error) {
	if isSelectStmt(source) {
		return d.schemaFromSelect(source)
	}
	return d.schema(source)
}
func isSelectStmt(source string) bool {
	return strings.ToUpper(strings.Fields(source)[0]) == "SELECT"
}

func (d MySQL) schemaFromSelect(stmt string) (*Schema, error) {
	view := "view_" + strconv.Itoa(rand.Int())
	createStmt := fmt.Sprintf("CREATE VIEW %s AS %s", view, stmt)
	if _, err := d.DB.Exec(createStmt); err != nil {
		return nil, err
	}
	defer d.DB.Exec(fmt.Sprintf("DROP VIEW %s", view))
	return d.schema(view)
}

func (d MySQL) schema(table string) (*Schema, error) {
	rows, err := d.DB.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", table))
	if err != nil {
		return nil, err
	}
	schema := Schema{Name: table}
	for rows.Next() {
		var field, type_, null, key, extra string
		var default_ *string
		if err := rows.Scan(&field, &type_, &null, &key, &default_, &extra); err != nil {
			return nil, err
		}
		schema.Fields = append(schema.Fields, d.parseField(field, type_, null, key))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &schema, nil
}

func (d MySQL) parseField(field, type_, null, key string) Field {
	return Field{
		Name:     field,
		Primary:  key == "PRI",
		Nullable: null == "YES",
		Type:     type_,
	}
}
func (d MySQL) ParseType(type_ string) reflect.Type {
	ss := strings.Split(type_, "(")
	switch ss[0] {
	case "tinyint", "int", "integer", "smallint", "mediumint", "bigint":
		return reflect.TypeOf(int(0))
	case "boolean", "bool":
		return reflect.TypeOf(bool(false))
	case "decimal", "float":
		return reflect.TypeOf(float32(0))
	case "double":
		if d.ForceFloat32 {
			return reflect.TypeOf(float32(0))
		}
		return reflect.TypeOf(float64(0))
	case "datetime", "timestamp", "date", "time", "year":
		return reflect.TypeOf("")
	case "char", "varchar", "text", "tinytext":
		return reflect.TypeOf("")
	}
	return nil
}
