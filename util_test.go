package main

import (
	"reflect"
	"testing"
)

func TestSet_EmptySlice(t *testing.T) {
	var empty []string
	s := set(empty)
	if len(s) != 0 {
		t.Errorf(`set(%v) = %v, expected %v`, empty, s, empty)
	}
}

func TestSet_SliceWithNonRepeatingValues(t *testing.T) {
	testCases := [][]string{
		{"a", "b", "c"},
		{"1", "2", "3"},
		{"a"},
	}

	for _, testCase := range testCases {
		s := set(testCase)
		if !reflect.DeepEqual(testCase, s) {
			t.Errorf(`set(%v) = %v, expected %v`, testCase, s, testCase)
		}
	}
}

func TestSet_SliceWithRepeatingValues(t *testing.T) {
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
		s := set(inCase)
		if !reflect.DeepEqual(outCases[idx], s) {
			t.Errorf(`set(%v) = %v, expected %v`, inCase, s, outCases[idx])
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
