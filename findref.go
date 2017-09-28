package main

//import "github.com/stretchr/powerwalk"
import "flag"
import "fmt"
import "sort"
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
        -f | --filename-only
              Display only filenames with matches, not the matches themselves
        -h | --hidden
              Include hidden files and files in hidden directories
        -i | --ignore-case
              Ignore case in regex (overrides smart-case)
        -m | --match-case
              Match regex case (if unset smart-case is used)
        -n | --no-color
              Disable colorized output
        -s | --stats
              Track basic statistics and print them on exit
        -v | --version
              Print current version and exit
`

const Version = "0.0.7"
const Date = "2017-09-27"

/* Colors */
var Red string = "\033[0;31m"
var Blue string = "\033[0;34m"
var Cyan string = "\033[0;36m"
var Green string = "\033[0;32m"
var Black string = "\033[0;30m"
var Brown string = "\033[0;33m"
var White string = "\033[1;37m"
var Yellow string = "\033[1;33m"
var Purple string = "\033[0;35m"
var Restore string = "\033[0m"
var LightRed string = "\033[1;31m"
var DarkGray string = "\033[1;30m"
var LightGray string = "\033[0;37m"
var LightBlue string = "\033[1;34m"
var LightCyan string = "\033[1;36m"
var LightGreen string = "\033[1;32m"
var LightPurple string = "\033[1;35m"

var FILE_PROCESSING_COMPLETE error = nil

/* Shared flags */
var Debug bool = false
var TrackStats bool = false
var FilenameOnly bool = false
var IncludeHidden bool = false

/* Shared regular expressions */
var matchRegex *regexp.Regexp = nil
var filenameRegex *regexp.Regexp = regexp.MustCompile(".*")
var hiddenFileRegex *regexp.Regexp = regexp.MustCompile(`(^|\/)\.`)

/* Statistics */
var filesScanned int = 0
var linesScanned int = 0
var matchesFound int = 0
var startTime time.Time

var filenameOnlyFiles []string = make([]string, 0, 100)

func zeroColors() {
	Red = ""
	Blue = ""
	Cyan = ""
	Green = ""
	Black = ""
	Brown = ""
	White = ""
	Yellow = ""
	Purple = ""
	Restore = ""
	LightRed = ""
	DarkGray = ""
	LightGray = ""
	LightBlue = ""
	LightCyan = ""
	LightGreen = ""
	LightPurple = ""
}

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
			if FilenameOnly {
				filenameOnlyFiles = append(filenameOnlyFiles, path)
			} else {
				printMatch(path, lineNumber, line, matchIndex)
			}
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

func uniq(stringSlice []string) []string {
	/* There is no built-in uniq function for slices, so we will use a map */
	stringMap := make(map[string]bool)
	for _, v := range stringSlice {
		stringMap[v] = true
	}
	retval := make([]string, 0, len(stringMap))
	for key, _ := range stringMap {
		retval = append(retval, key)
	}
	return retval
}

func main() {
	sPtr := flag.Bool("s", false, "Alias for --stats")
	dPtr := flag.Bool("d", false, "Alias for --debug")
	hPtr := flag.Bool("h", false, "Alias for --hidden")
	vPtr := flag.Bool("v", false, "Alias for --version")
	nPtr := flag.Bool("n", false, "Alias for --no-color")
	mPtr := flag.Bool("m", false, "Alias for --match-case")
	iPtr := flag.Bool("i", false, "Alias for --ignore-case")
	fPtr := flag.Bool("f", false, "Alias for --filename-only")
	helpPtr := flag.Bool("help", false, "Show usage")
	statsPtr := flag.Bool("stats", false, "Track and display statistics")
	debugPtr := flag.Bool("debug", false, "Enable debug mode")
	hiddenPtr := flag.Bool("hidden", false, "Include hidden files and files in hidden directories")
	versionPtr := flag.Bool("version", false, "Print current version and exit")
	nocolorPtr := flag.Bool("no-color", false, "Don't use color in output")
	matchCasePtr := flag.Bool("match-case", false, "Match regex case (if unset smart-case is used)")
	ignoreCasePtr := flag.Bool("ignore-case", false, "Ignore case in regex (overrides smart-case)")
	filenameOnlyPtr := flag.Bool("filename-only", false, "Display only filenames with matches")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n", Usage)
	}
	flag.Parse()

	if *vPtr || *versionPtr {
		printVersionAndExit()
		os.Exit(0)
	}

	if *helpPtr {
		usageAndExit()
	}

	if *nPtr || *nocolorPtr {
		debug("Color output is disabled")
		zeroColors()
	}

	Debug = *debugPtr || *dPtr
	TrackStats = *statsPtr || *sPtr
	FilenameOnly = *filenameOnlyPtr || *fPtr
	IncludeHidden = *hiddenPtr || *hPtr
	*matchCasePtr = *matchCasePtr || *mPtr
	*ignoreCasePtr = *ignoreCasePtr || *iPtr

	if TrackStats {
		startTime = time.Now()
		debug(Blue, "Start time is:", Restore, startTime.String())
	}

	debug(Blue, "stats enabled: ", Restore, TrackStats)
	debug(Blue, "match-case enabled: ", Restore, *matchCasePtr)
	debug(Blue, "ignore-case enabled: ", Restore, *ignoreCasePtr)
	debug(Blue, "include hidden files: ", Restore, IncludeHidden)
	debug(Blue, "debug mode: ", Restore, Debug)
	debug(Blue, "filename only: ", Restore, FilenameOnly)

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

	if FilenameOnly {
		filenames := uniq(filenameOnlyFiles)
		sort.Strings(filenames)
		for _, filename := range filenames {
			fmt.Printf("%s%s%s\n", Purple, filename, Restore)
		}
	}

	if TrackStats {
		fmt.Printf("%sElapsed time:%s  %s\n", Cyan, Restore, elapsedTime().String())
		fmt.Printf("%sLines scanned:%s %d\n", Cyan, Restore, linesScanned)
		fmt.Printf("%sFiles scanned:%s %d\n", Cyan, Restore, filesScanned)
		fmt.Printf("%sMatches found:%s %d\n", Cyan, Restore, matchesFound)
	}
}
