package main

import (
	"regexp"
	"testing"
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
	s := NewSettings()
	s.FilenameRegex = regexp.MustCompile("abcd")
	if !s.PassesFileFilter("abcd.txt") {
		t.Fail()
	}
	if s.PassesFileFilter("abc.txt") {
		t.Fail()
	}
	if !s.PassesFileFilter("abcd") {
		t.Fail()
	}
	s.FilenameRegex = regexp.MustCompile(`.*\.txt`)
	if !s.PassesFileFilter("abcd.txt") {
		t.Fail()
	}
	if s.PassesFileFilter("abc.tx") {
		t.Fail()
	}
	if s.PassesFileFilter("abcd") {
		t.Fail()
	}
}

func TestIsHidden(t *testing.T) {
	s := NewSettings()
	if s.IsHidden("/home/ben") {
		t.Fail()
	}
	if s.IsHidden("/home/ben/nothing.txt") {
		t.Fail()
	}
	if !s.IsHidden("/home/.ben/nothing.txt") {
		t.Fail()
	}
	if !s.IsHidden("/home/ben/.nothing.txt") {
		t.Fail()
	}
	if !s.IsHidden("/home/.ben/.nothing.txt") {
		t.Fail()
	}
}

func TestShouldExcludeDirDefaults(t *testing.T) {
	s := NewSettings()
	if !s.ShouldExcludeDir("./.git") {
		t.Fatalf("expected .git to be excluded by default")
	}
	if !s.ShouldExcludeDir("/tmp/project/.svn") {
		t.Fatalf("expected .svn to be excluded by default")
	}
	if s.ShouldExcludeDir("./vendor") {
		t.Fatalf("did not expect vendor to be excluded by default")
	}
}

func TestShouldExcludeDirUserProvided(t *testing.T) {
	s := NewSettings()
	s.AddExcludeDirs("vendor", "./build/")
	if !s.ShouldExcludeDir("/tmp/project/vendor") {
		t.Fatalf("expected vendor directory to be excluded when provided")
	}
	if !s.ShouldExcludeDir("./build") {
		t.Fatalf("expected build directory to be excluded when provided")
	}
	if s.ShouldExcludeDir("/tmp/project/src") {
		t.Fatalf("did not expect src directory to be excluded")
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

func TestCheckForMatches(t *testing.T) {

}

func TestProcessFile(t *testing.T) {

}
