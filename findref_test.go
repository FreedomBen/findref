package main

import (
    "testing"
    "regexp"
)

func TestContainsNullByte(t *testing.T) {
    if containsNullByte([]byte("Here is a string....")) {
        t.Fail()
    }
    if !containsNullByte([]byte("Here is a \x00string....")) {
        t.Fail()
    }
}

func TestPassesFileFilter(t *testing.T) {
    fileFilter = regexp.MustCompile("abcd")
    if !passesFileFilter("abcd.txt") {
        t.Fail()
    }
    if passesFileFilter("abc.txt") {
        t.Fail()
    }
    if !passesFileFilter("abcd") {
        t.Fail()
    }
    fileFilter = regexp.MustCompile(`.*\.txt`)
    if !passesFileFilter("abcd.txt") {
        t.Fail()
    }
    if passesFileFilter("abc.tx") {
        t.Fail()
    }
    if passesFileFilter("abcd") {
        t.Fail()
    }
}

func TestIsHidden(t *testing.T) {
    if isHidden("/home/ben") {
        t.Fail()
    }
    if isHidden("/home/ben/nothing.txt") {
        t.Fail()
    }
    if !isHidden("/home/.ben/nothing.txt") {
        t.Fail()
    }
    if !isHidden("/home/ben/.nothing.txt") {
        t.Fail()
    }
    if !isHidden("/home/.ben/.nothing.txt") {
        t.Fail()
    }
}

func TestGetMatchRegex(t *testing.T) {
    r1 := getMatchRegex(false, false, "HEllo")
    if !r1.MatchString("HEllo") {
        t.Fail()
    }
    if r1.MatchString("hello") {
        t.Fail()
    }
    r2 := getMatchRegex(false, false, "hello")
    //  verify smart case works
    if !r2.MatchString("abcHEllo") {
        t.Fail()
    }
    if !r2.MatchString("abchello") {
        t.Fail()
    }
    if !r2.MatchString("abcHELLO") {
        t.Fail()
    }
    if r2.MatchString("abc") {
        t.Fail()
    }
    r3 := getMatchRegex(true, false, "hello")
    if !r3.MatchString("HEllo") {
        t.Fail()
    }
    if !r3.MatchString("abchello") {
        t.Fail()
    }
    if !r3.MatchString("abcHELLO") {
        t.Fail()
    }
    if r3.MatchString("abc") {
        t.Fail()
    }
    r4 := getMatchRegex(false, true, "hello")
    if r4.MatchString("HEllo") {
        t.Fail()
    }
    if !r4.MatchString("abchello") {
        t.Fail()
    }
    if r4.MatchString("abcHELLO") {
        t.Fail()
    }
    if r4.MatchString("abc") {
        t.Fail()
    }
}

func TestDetermineIgnoreCase(t *testing.T) {
    ignoreCase := false
    ic := false
    determineIgnoreCase(&ignoreCase, &ic)
    if ignoreCase {
        t.Fail()
    }
    ignoreCase = true
    determineIgnoreCase(&ignoreCase, &ic)
    if !ignoreCase {
        t.Fail()
    }
    ic = true
    determineIgnoreCase(&ignoreCase, &ic)
    if !ignoreCase {
        t.Fail()
    }
    ignoreCase = false
    determineIgnoreCase(&ignoreCase, &ic)
    if !ignoreCase {
        t.Fail()
    }
}

func TestCheckForMatches(t *testing.T) {

}

func TestProcessFile(t *testing.T) {

}