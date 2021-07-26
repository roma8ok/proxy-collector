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

func TestIsExist_NonExist(t *testing.T) {
	testCases := [][]string{
		{},
		{"a", "b", "c"},
		{"a", "a", "a"},
	}
	nonExist := "nonExist"

	for _, testCase := range testCases {
		if exist := isExist(nonExist, testCase); exist {
			t.Errorf(`isExist(%s, %v) = %t, expected false`, nonExist, testCase, exist)
		}
	}
}

func TestIsExist_Exist(t *testing.T) {
	testCases := [][]string{
		{"a", "b", "c"},
		{"a", "a", "a"},
	}
	exist := "a"

	for _, testCase := range testCases {
		if found := isExist(exist, testCase); !found {
			t.Errorf(`isExist(%s, %v) = %t, expected true`, exist, testCase, found)
		}
	}
}

func TestURLsHaveSameDomain_DifferentDomains(t *testing.T) {
	u1 := "https://google.com"
	u2 := "https://duckduckgo.com"

	if same := urlsHaveSameDomain(u1, u2); same {
		t.Errorf(`urlsHaveSameDomain(%s, %v) = %t, expected false`, u1, u2, same)
	}
}

func TestURLsHaveSameDomain_SameDomains(t *testing.T) {
	u1 := "https://google.com:56789/"
	u2 := "http://google.com/hello?q=123"

	if same := urlsHaveSameDomain(u1, u2); !same {
		t.Errorf(`urlsHaveSameDomain(%s, %v) = %t, expected true`, u1, u2, same)
	}
}
