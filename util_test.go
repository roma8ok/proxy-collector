package main

import (
	"reflect"
	"testing"
)

func TestUniqueArrayOfValues_Empty(t *testing.T) {
	var empty []string
	uniq := uniqueArrayOfValues(empty)
	if len(uniq) != 0 {
		t.Errorf(`uniqueArrayOfValues_Unique(%v) = %v, expected %v`, empty, uniq, empty)
	}
}

func TestUniqueArrayOfValues_Unique(t *testing.T) {
	testCases := [][]string{
		{"a", "b", "c"},
		{"1", "2", "3"},
		{"a"},
	}

	for _, testCase := range testCases {
		uniq := uniqueArrayOfValues(testCase)
		if !reflect.DeepEqual(testCase, uniq) {
			t.Errorf(`uniqueArrayOfValues_Unique(%v) = %v, expected %v`, testCase, uniq, testCase)
		}
	}
}

func TestUniqueArrayOfValues_NonUnique(t *testing.T) {
	inCases := [][]string{
		{"a", "b", "c", "c"},
		{"1", "2", "3", "3"},
		{"a", "a", "a", "a"},
	}
	outCases := [][]string{
		{"a", "b", "c"},
		{"1", "2", "3"},
		{"a"},
	}

	for idx, inCase := range inCases {
		uniq := uniqueArrayOfValues(inCase)
		if !reflect.DeepEqual(outCases[idx], uniq) {
			t.Errorf(`uniqueArrayOfValues_Unique(%v) = %v, expected %v`, inCase, uniq, outCases[idx])
		}
	}
}