package main

import "flag"
import "fmt"
import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

const Version = "0.0.1"
const Date = "2017-09-24"

const Red = "\033[0;31m"
const Blue = "\033[0;34m"
const Cyan = "\033[0;36m"
const Green = "\033[0;32m"
const Purple = "\033[0;35m"
const Restore = "\033[0m"
const LightRed = "\033[1;31m"

/* Not currently used
const Black = "\033[0;30m"
const Brown = "\033[0;33m"
const White = "\033[1;37m"
const Yellow = "\033[1;33m"
const DarkGray = "\033[1;30m"
const LightGray = "\033[0;37m"
const LightBlue = "\033[1;34m"
const LightCyan = "\033[1;36m"
const LightGreen = "\033[1;32m"
const LightPurple = "\033[1;35m"
*/

var FILE_PROCESSING_COMPLETE error = nil

/* Shared flags */
var Debug bool = false
var IncludeHidden bool = false

/* Shared regular expressions */
var matchRegex *regexp.Regexp = nil
var fileFilter *regexp.Regexp = regexp.MustCompile(".*")
var hiddenFileRegex *regexp.Regexp = regexp.MustCompile(`(^|\/)\.`)

func usageAndExit() {
	flag.Usage()
	os.Exit(1)
}

func debug(a ...interface{}) {
	if Debug {
		fmt.Println(a...)
	}
}

func printMatch(path string, lineNumber int, line []byte, match []int) {
	//fmt.Println(Purple + path + Restore + Green + ":" + strconv.Itoa(lineNumber) + ":" + Restore + string(line[:match[0]]) + LightRed + string(line[match[0]:match[1]]) + Restore + string(line[match[1]:]))
	fmt.Printf("%s%s%s%s:%s:%s%s%s%s%s%s",
		Purple,
		path,
		Restore,
		Green,
		strconv.Itoa(lineNumber),
		Restore,
		string(line[:match[0]]),
		LightRed,
		string(line[match[0]:match[1]]),
		Restore,
		string(line[match[1]:]),
	)
}

func passesFileFilter(path string) bool {
	return fileFilter.MatchString(path)
}

func isHidden(path string) bool {
	// Ignore hidden files unless the IncludeHidden flag is set
	return !IncludeHidden && hiddenFileRegex.MatchString(path)
}

func containsNullByte(line []byte) bool {
	for _, el := range line {
		if el == 0 {
			return true
		}
	}
	return false
}

func checkForMatches(path string) error {
	debug(Blue+"Checking file for matches:"+Restore, path)

	file, err := os.Open(path)
	if err != nil {
		debug(Red+"Error opening file at '"+path+"'.  It might be a directory.  Err: "+Restore, err)
		return FILE_PROCESSING_COMPLETE
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lineNumber int = 0
	for scanner.Scan() {
		line := scanner.Bytes()
		lineNumber++
		if containsNullByte(line) {
			// This is a binary file.  Skip it!
			debug(Blue+"Not processing binary file:"+Restore, path)
			return FILE_PROCESSING_COMPLETE
		}
		if matchIndex := matchRegex.FindIndex(line); matchIndex != nil {
			// we have a match! loc == nil means no match so just ignore that case
			printMatch(path, lineNumber, line, matchIndex)
			return FILE_PROCESSING_COMPLETE
		}
	}

	if err := scanner.Err(); err != nil {
		debug(Red+"Error scanning line from file '"+path+"'. File will be skipped.  Err: "+Restore, err)
		return FILE_PROCESSING_COMPLETE
	}
	return FILE_PROCESSING_COMPLETE
}

func processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		debug("filepath.Walk encountered error with path '"+path+"'", err)
		return FILE_PROCESSING_COMPLETE
	}

	if info.IsDir() {
		if isHidden(path) {
			return filepath.SkipDir // skip the whole sub-contents of this hidden directory
		} else {
			return FILE_PROCESSING_COMPLETE
		}
	}

	if passesFileFilter(path) {
		if isHidden(path) {
			debug(Blue + "Hidden file '" + Restore + path + Blue + "' not processed")
			return FILE_PROCESSING_COMPLETE
		}
		return checkForMatches(path)
	} else {
		debug(Blue + "Ignoring file cause it doesn't match filter: " + Restore + path)
	}
	return FILE_PROCESSING_COMPLETE
}

func getMatchRegex(ignoreCase bool, matchCase bool, usersRegex string) *regexp.Regexp {
	// If ignore case is set, ignore the case of the regex.
	// if match-case is not set, use smart case which means if it's all lower case be case-insensitive,
	// but if there's capitals then be case-sensitive
	if ignoreCase || (!matchCase && !regexp.MustCompile("[A-Z]").MatchString(usersRegex)) {
		debug(Blue + "Match regex will be case-insensitive" + Restore)
		return regexp.MustCompile("(?i)" + usersRegex)
	} else {
		debug(Blue + "Match regex will be exactly as user provided" + Restore)
		return regexp.MustCompile(usersRegex)
	}
}

func determineMatchCase(matchCasePtr *bool, mcPtr *bool) {
	if *mcPtr {
		*matchCasePtr = true
	}
}

func determineIgnoreCase(ignoreCasePtr *bool, icPtr *bool) {
	if *icPtr {
		*ignoreCasePtr = true
	}
}

func printVersionAndExit() {
	fmt.Printf("%s%s%s%s%s%s", Cyan, "findref version ", Version, " released on ", Date, Restore)
}

func main() {
	vPtr := flag.Bool("v", false, "Alias for --version")
	mcPtr := flag.Bool("mc", false, "Alias for --match-case")
	icPtr := flag.Bool("ic", false, "Alias for --ignore-case")
	debugPtr := flag.Bool("debug", false, "Enable debug mode")
	hiddenPtr := flag.Bool("hidden", false, "Include hidden files and files in hidden directories")
	versionPtr := flag.Bool("version", false, "Print current version and exit")
	matchCasePtr := flag.Bool("match-case", false, "Match regex case (if unset smart-case is used)")
	ignoreCasePtr := flag.Bool("ignore-case", false, "Ignore case in regex (overrides smart-case)")

	flag.Parse()

	if *vPtr || *versionPtr {
		printVersionAndExit()
		os.Exit(0)
	}

	determineMatchCase(matchCasePtr, mcPtr)
	determineIgnoreCase(ignoreCasePtr, icPtr)

	IncludeHidden = *hiddenPtr
	Debug = *debugPtr

	debug("match-case set: ", *matchCasePtr)
	debug("ignore-case set: ", *ignoreCasePtr)
	debug("hidden: ", IncludeHidden)
	debug("debug: ", Debug)
	debug("tail: ", flag.Args())

	rootDir := "."

	if len(flag.Args()) < 1 {
		fmt.Println("Must specify regex to match against files")
		usageAndExit()
	} else if len(flag.Args()) > 3 {
		fmt.Println("Too many args")
		usageAndExit()
	} else {
		matchRegex = getMatchRegex(*ignoreCasePtr, *matchCasePtr, flag.Args()[0])

		if len(flag.Args()) >= 2 {
			rootDir = flag.Args()[1]
		}
		if len(flag.Args()) == 3 {
			fileFilter = regexp.MustCompile(flag.Args()[2])
		}
	}

	// TODO: Switch to powerwalk for performance:  https://github.com/stretchr/powerwalk
	filepath.Walk(rootDir, processFile)
}
