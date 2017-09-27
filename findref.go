package main

//import "github.com/stretchr/powerwalk"
import "flag"
import "fmt"
import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
	//"runtime"
)

const Usage = `
    Usage of findref:

    findref [options] match_regex [start_dir] [filename_regex]

    Arguments:

        match_regex:  This is an RE2 regular expression that will be matched against lines
                      in each file, with matches being displayed to the user.

        start_dir:  This optional argument sets the starting directory to crawl looking
                    for eligible files with lines matching match_regex.  Default value
                    is the current working directory, AKA $PWD or '.'

        filename_regex:  This optional argument restricts the set of files checked for
                         matching lines.  Eligible files must match this expression.
                         Default value matches all files

    Options:

        -d | --debug
              Enable debug mode
        -h | --hidden
              Include hidden files and files in hidden directories
        -i | --ignore-case
              Ignore case in regex (overrides smart-case)
        -m | --match-case
              Match regex case (if unset smart-case is used)
        -s | --stats
              Track basic statistics and print them on exit
        -v | --version
              Print current version and exit
`

const Version = "0.0.4"
const Date = "2017-09-26"

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
var TrackStats bool = false

/* Shared regular expressions */
var matchRegex *regexp.Regexp = nil
var filenameRegex *regexp.Regexp = regexp.MustCompile(".*")
var hiddenFileRegex *regexp.Regexp = regexp.MustCompile(`(^|\/)\.`)

/* Statistics */
var filesScanned int = 0
var linesScanned int = 0
var matchesFound int = 0
var startTime time.Time

func elapsedTime() time.Duration {
	return time.Now().Sub(startTime)
}

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
	fmt.Printf("%s%s%s%s:%s:%s%s%s%s%s%s\n",
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
	return filenameRegex.MatchString(path)
}

func isHidden(path string) bool {
	// Ignore hidden files unless the IncludeHidden flag is set
	return path != "." && !IncludeHidden && hiddenFileRegex.MatchString(path)
}

func containsNullByte(line []byte) bool {
	for _, el := range line {
		if el == 0 {
			return true
		}
	}
	return false
}

func incrLineCount() {
	if TrackStats {
		linesScanned++
	}
}

func incrFileCount() {

	if TrackStats {
		filesScanned++
	}
}

func incrMatchCount() {
	if TrackStats {
		matchesFound++
	}
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
		incrLineCount()
		if containsNullByte(line) {
			// This is a binary file.  Skip it!
			debug(Blue+"Not processing binary file:"+Restore, path)
			return FILE_PROCESSING_COMPLETE
		}
		if matchIndex := matchRegex.FindIndex(line); matchIndex != nil {
			// we have a match! loc == nil means no match so just ignore that case
			incrMatchCount()
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
	incrFileCount()

	if err != nil {
		debug("filepath.Walk encountered error with path '"+path+"'", err)
		return FILE_PROCESSING_COMPLETE
	}

	if info.IsDir() {
		if isHidden(path) {
			debug(Blue, "Directory", path, "is hidden and will be pruned", Restore)
			return filepath.SkipDir // skip the whole sub-contents of this hidden directory
		} else {
			return FILE_PROCESSING_COMPLETE
		}
	}

	if passesFileFilter(path) {
		debug(Blue+"Passes file filter:", path)
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
		debug(Blue, "Match regex will be case-insensitive", Restore)
		return regexp.MustCompile("(?i)" + usersRegex)
	} else {
		debug(Blue, "Match regex will be exactly as user provided", Restore)
		return regexp.MustCompile(usersRegex)
	}
}

func printVersionAndExit() {
	fmt.Printf("%s%s%s%s%s%s\n", Cyan, "findref version ", Version, " released on ", Date, Restore)
}

func main() {
	sPtr := flag.Bool("s", false, "Alias for --stats")
	dPtr := flag.Bool("d", false, "Alias for --debug")
	hPtr := flag.Bool("h", false, "Alias for --hidden")
	vPtr := flag.Bool("v", false, "Alias for --version")
	mPtr := flag.Bool("m", false, "Alias for --match-case")
	iPtr := flag.Bool("i", false, "Alias for --ignore-case")
	helpPtr := flag.Bool("help", false, "Show usage")
	statsPtr := flag.Bool("stats", false, "Track and display statistics")
	debugPtr := flag.Bool("debug", false, "Enable debug mode")
	hiddenPtr := flag.Bool("hidden", false, "Include hidden files and files in hidden directories")
	versionPtr := flag.Bool("version", false, "Print current version and exit")
	matchCasePtr := flag.Bool("match-case", false, "Match regex case (if unset smart-case is used)")
	ignoreCasePtr := flag.Bool("ignore-case", false, "Ignore case in regex (overrides smart-case)")

	flag.Parse()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", Usage)
	}

	if *vPtr || *versionPtr {
		printVersionAndExit()
		os.Exit(0)
	}

	if *helpPtr {
		usageAndExit()
	}

	*matchCasePtr = *matchCasePtr || *mPtr
	*ignoreCasePtr = *ignoreCasePtr || *iPtr
	TrackStats = *statsPtr || *sPtr
	IncludeHidden = *hiddenPtr || *hPtr
	Debug = *debugPtr || *dPtr

	if TrackStats {
		startTime = time.Now()
		debug(Blue, "Start time is:", Restore, startTime.String())
	}

	debug(Blue, "stats enabled: ", Restore, TrackStats)
	debug(Blue, "match-case enabled: ", Restore, *matchCasePtr)
	debug(Blue, "ignore-case enabled: ", Restore, *ignoreCasePtr)
	debug(Blue, "include hidden files: ", Restore, IncludeHidden)
	debug(Blue, "debug mode: ", Restore, Debug)

	rootDir := "."

	if len(flag.Args()) < 1 {
		fmt.Errorf("%s", "Must specify regex to match against files")
		usageAndExit()
	} else if len(flag.Args()) > 3 {
		fmt.Errorf("%s", "Too many args (expected 1 <= 3)")
		usageAndExit()
	} else {
		matchRegex = getMatchRegex(*ignoreCasePtr, *matchCasePtr, flag.Args()[0])

		if len(flag.Args()) >= 2 {
			rootDir = flag.Args()[1]
		}
		if len(flag.Args()) == 3 {
			filenameRegex = regexp.MustCompile(flag.Args()[2])
		}
	}

	debug(Blue, "matchRegex: ", Restore, matchRegex.String())
	debug(Blue, "rootDir: ", Restore, rootDir)
	debug(Blue, "fileRegex: ", Restore, filenameRegex.String())

	filepath.Walk(rootDir, processFile)

	// TODO: Switch to powerwalk for performance:  https://github.com/stretchr/powerwalk
	//runtime.GOMAXPROCS(runtime.NumCPU())
	//powerwalk.Walk(rootDir, processFile)

	if TrackStats {
		fmt.Printf("%sElapsed time:%s  %s\n", Cyan, Restore, elapsedTime().String())
		fmt.Printf("%sLines scanned:%s %d\n", Cyan, Restore, linesScanned)
		fmt.Printf("%sFiles scanned:%s %d\n", Cyan, Restore, filesScanned)
		fmt.Printf("%sMatches found:%s %d\n", Cyan, Restore, matchesFound)
	}
}
