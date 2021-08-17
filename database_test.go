package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

const (
	testTableName = "table_name"

	testEqKey     = "active"
	testEqValue   = true
	testGtOrEqKey = "ts"
)

var (
	testWhereEqEmpty     = WhereCondition{}
	testWhereGtOrEqEmpty = WhereCondition{}

	testGtOrEqValue = time.Unix(946684800, 0)
)

func TestNewSQLBuilder(t *testing.T) {
	expected := SQLBuilder{table: testTableName}
	got := newSQLBuilder(testTableName)
	if !reflect.DeepEqual(expected, got) {
		t.Errorf(`newSQLBuilder(%s) = %v, expected %v`, testTableName, got, expected)
	}
}

func TestSQLBuilder_Count_WithoutWhere(t *testing.T) {
	expectedSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", testTableName)
	sqlBuilder := newSQLBuilder(testTableName)

	s, args, err := sqlBuilder.count(testWhereEqEmpty, testWhereGtOrEqEmpty)
	if err != nil {
		t.Errorf(` SQLBuilder.count(%v, %v); err = %s, expected nil`, testWhereEqEmpty, testWhereGtOrEqEmpty, err)
	}
	if s != expectedSQL {
		t.Errorf(` SQLBuilder.count(%v, %v); sql = %s, expected %s`, testWhereEqEmpty, testWhereGtOrEqEmpty, s, expectedSQL)
	}
	if len(args) != 0 {
		t.Errorf(` SQLBuilder.count(%v, %v); args = %s, expected empty`, testWhereEqEmpty, testWhereGtOrEqEmpty, args)
	}
}

func TestSQLBuilder_Count_WithinWhere(t *testing.T) {
	var (
		whereEq     = WhereCondition{testEqKey: testEqValue}
		whereGtOrEq = WhereCondition{testGtOrEqKey: testGtOrEqValue}

		expectedSQL  = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE (%s = $1 AND %s >= $2)", testTableName, testEqKey, testGtOrEqKey)
		expectedArgs = []interface{}{testEqValue, testGtOrEqValue}
	)

	sqlBuilder := newSQLBuilder(testTableName)

	s, args, err := sqlBuilder.count(whereEq, whereGtOrEq)
	if err != nil {
		t.Errorf(` SQLBuilder.count(%v, %v); err = %s, expected nil`, whereEq, whereGtOrEq, err)
	}
	if s != expectedSQL {
		t.Errorf(` SQLBuilder.count(%v, %v); sql = %s, expected %s`, whereEq, whereGtOrEq, s, expectedSQL)
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf(` SQLBuilder.count(%v, %v); args = %v, expected %v`, whereEq, whereGtOrEq, args, expectedArgs)
	}
}

func TestSQLBuilder_Count_WithinWhereMultiple(t *testing.T) {
	var (
		eqKey2   = "order"
		eqValue2 = 2

		whereEq     = WhereCondition{testEqKey: testEqValue, eqKey2: eqValue2}
		whereGtOrEq = WhereCondition{testGtOrEqKey: testGtOrEqValue}

		expectedSQL = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE (%s = $1 AND %s = $2 AND %s >= $3)",
			testTableName, testEqKey, eqKey2, testGtOrEqKey)
		expectedArgs = []interface{}{testEqValue, eqValue2, testGtOrEqValue}
	)

	sqlBuilder := newSQLBuilder(testTableName)

	s, args, err := sqlBuilder.count(whereEq, whereGtOrEq)
	if err != nil {
		t.Errorf(` SQLBuilder.count(%v, %v); err = %s, expected nil`, whereEq, whereGtOrEq, err)
	}
	if s != expectedSQL {
		t.Errorf(` SQLBuilder.count(%v, %v); sql = %s, expected %s`, whereEq, whereGtOrEq, s, expectedSQL)
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf(` SQLBuilder.count(%v, %v); args = %v, expected %v`, whereEq, whereGtOrEq, args, expectedArgs)
	}
}

func TestSQLBuilder_Count_WithinWhereEq(t *testing.T) {
	var (
		whereEq = WhereCondition{testEqKey: testEqValue}

		expectedSQL  = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", testTableName, testEqKey)
		expectedArgs = []interface{}{testEqValue}
	)

	sqlBuilder := newSQLBuilder(testTableName)

	s, args, err := sqlBuilder.count(whereEq, testWhereGtOrEqEmpty)
	if err != nil {
		t.Errorf(` SQLBuilder.count(%v, %v); err = %s, expected nil`, whereEq, testWhereGtOrEqEmpty, err)
	}
	if s != expectedSQL {
		t.Errorf(` SQLBuilder.count(%v, %v); sql = %s, expected %s`, whereEq, testWhereGtOrEqEmpty, s, expectedSQL)
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf(` SQLBuilder.count(%v, %v); args = %v, expected %v`, whereEq, testWhereGtOrEqEmpty, args, expectedArgs)
	}
}

func TestSQLBuilder_Count_WithinWhereGtOrEq(t *testing.T) {
	var (
		whereGtOrEq = WhereCondition{testGtOrEqKey: testGtOrEqValue}

		expectedSQL  = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s >= $1", testTableName, testGtOrEqKey)
		expectedArgs = []interface{}{testGtOrEqValue}
	)

	sqlBuilder := newSQLBuilder(testTableName)

	s, args, err := sqlBuilder.count(testWhereEqEmpty, whereGtOrEq)
	if err != nil {
		t.Errorf(` SQLBuilder.count(%v, %v); err = %s, expected nil`, testWhereEqEmpty, whereGtOrEq, err)
	}
	if s != expectedSQL {
		t.Errorf(` SQLBuilder.count(%v, %v); sql = %s, expected %s`, testWhereEqEmpty, whereGtOrEq, s, expectedSQL)
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf(` SQLBuilder.count(%v, %v); args = %v, expected %v`, testWhereEqEmpty, whereGtOrEq, args, expectedArgs)
	}
}

func TestSQLBuilder_Insert_EmptyData(t *testing.T) {
	var data = map[string]interface{}{}

	sqlBuilder := newSQLBuilder(testTableName)
	_, _, err := sqlBuilder.insert(data)
	if err == nil {
		t.Errorf(` SQLBuilder.insert(empty data); err = "%v", expected "%v"`, err, errDBInsertEmptyData)
	}
}

func TestSQLBuilder_Insert_SingleData(t *testing.T) {
	var (
		colName  = "field"
		colValue = "field_value"

		data = map[string]interface{}{
			colName: colValue,
		}

		expectedSQL  = fmt.Sprintf("INSERT INTO %s (%s) VALUES ($1)", testTableName, colName)
		expectedArgs = []interface{}{colValue}
	)

	sqlBuilder := newSQLBuilder(testTableName)
	s, args, err := sqlBuilder.insert(data)

	if err != nil {
		t.Errorf(` SQLBuilder.insert(%v); err = "%v", expected "%v"`, data, err, errDBInsertEmptyData)
	}
	if s != expectedSQL {
		t.Errorf(` SQLBuilder.insert(%v); sql = %s, expected %s`, data, s, expectedSQL)
	}
	if !reflect.DeepEqual(args, expectedArgs) {
		t.Errorf(` SQLBuilder.insert(%v); args = %v, expected %v`, data, args, expectedArgs)
	}
}

func TestSQLBuilder_Insert_MultipleData(t *testing.T) {
	var (
		colName1  = "field"
		colValue1 = "field_value"
		colName2  = "field2"
		colValue2 = "field_value2"

		data = map[string]interface{}{
			colName1: colValue1,
			colName2: colValue2,
		}

		expectedSQL1  = fmt.Sprintf("INSERT INTO %s (%s,%s) VALUES ($1,$2)", testTableName, colName1, colName2)
		expectedArgs1 = []interface{}{colValue1, colValue2}
		expectedSQL2  = fmt.Sprintf("INSERT INTO %s (%s,%s) VALUES ($1,$2)", testTableName, colName2, colName1)
		expectedArgs2 = []interface{}{colValue2, colValue1}
	)

	sqlBuilder := newSQLBuilder(testTableName)
	s, args, err := sqlBuilder.insert(data)

	if err != nil {
		t.Errorf(` SQLBuilder.insert(%v); err = "%v", expected "%v"`, data, err, errDBInsertEmptyData)
	}
	if s != expectedSQL1 && s != expectedSQL2 {
		t.Errorf(` SQLBuilder.insert(%v); sql = %s, expected %s or %s`, data, s, expectedSQL1, expectedSQL2)
	}
	if !reflect.DeepEqual(args, expectedArgs1) && !reflect.DeepEqual(args, expectedArgs2) {
		t.Errorf(` SQLBuilder.insert(%v); args = %v, expected %v or %s`, data, args, expectedArgs1, expectedArgs2)
	}
}
