package main

import "flag"
import "fmt"
import "sort"
import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

func Usage() string {
	return fmt.Sprintf(
		`
    %s%s%s

        %sfindref%s is a simple utility that lets you find strings based on regular expressions
        in directories of text files.

        A common example of why you would want to do this, is if you are searching for occurences
        of a particular word in a source code repository.  Using %sfindref%s you can quickly
        find any variable or function that includes a particular string, or any other pattern.%s

    %sUsage of findref:%s

        %sfindref%s %s[options]%s %smatch_regex%s %s[start_dir]%s %s[filename_regex]%s

    %sArguments:%s

        %smatch_regex:  This is an RE2 regular expression that will be matched against lines
                      in each file, with matches being displayed to the user.%s

        %sstart_dir:  This optional argument sets the starting directory to crawl looking
                    for eligible files with lines matching match_regex.  Default value
                    is the current working directory, AKA $PWD or '.'%s

        %sfilename_regex:  This optional argument restricts the set of files checked for
                         matching lines.  Eligible files must match this expression.
                         Default value matches all files%s

    %sOptions:%s
        %s
        -a | --all
              Aggressively search for matches (implies: -i -h)
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
        -l | --max-line-length
              Set maximum line length in characters (default is 2,000)
        -x |  --no-max-line-length
              Remove maximum line length.  Match againt lines of any length
        -s | --stats
              Track basic statistics and print them on exit
        -v | --version
              Print current version and exit
        -- | --
              End of options.  Use when one of the args starts with a '-'
        %s
    %sExamples:%s

        %s// Find all occurences in the current directory (and children) of "getMethodName"%s
        %sfindref%s %sgetMethodName%s

        %s// Find all exec() calls in the src/ directory%s
        %sfindref%s %s--ignore-case%s %s'exec('%s %ssrc/%s

        %s// Find all "hi1" or "hi2" in all C++ files (including hidden) in "~/starting-dir"%s
        %sfindref%s %s--hidden%s %s"hi(1|2)"%s %s"~/starting-dir"%s %s".*\.[hc](pp)?"%s

        %s// Find all "-str[i1]ng.*" in "~/starting-dir" checking C++ files, include stats%s
        %sfindref%s %s-s --%s %s"-str[i1]ng.*"%s %s"~/starting-dir"%s %s".*\.[hc](pp)?"%s

`,
		// Top block
		colors.Red, versionString(false), colors.Restore, // Title
		colors.Brown, colors.LightGray, colors.Brown, colors.LightGray, colors.Restore, // Description

		// Usage block
		colors.Red, colors.Restore, // header
		colors.Brown, colors.Restore, // findref
		colors.Green, colors.Restore, // options
		colors.Cyan, colors.Restore, // match_regex
		colors.Blue, colors.Restore, // start_dir
		colors.Purple, colors.Restore, // filename_regex

		// Arguments block
		colors.Red, colors.Restore, // header
		colors.Cyan, colors.Restore, // match_regex
		colors.Blue, colors.Restore, // start_dir
		colors.Purple, colors.Restore, // filename_regex

		// Options
		colors.Red, colors.Restore, // Start of Options block
		colors.Green, colors.Restore, // end of Options block

		// Examples
		// First Example
		colors.Red, colors.Restore, // Examples: header
		colors.LightGray, colors.Restore, // first example comment
		colors.Brown, colors.Restore, // first example findref
		colors.Cyan, colors.Restore, // first example match_regex

		// Second Example
		colors.LightGray, colors.Restore, // second example comment
		colors.Brown, colors.Restore, // second example findref
		colors.Green, colors.Restore, // second example option
		colors.Cyan, colors.Restore, // second example match_regex
		colors.Blue, colors.Restore, // second example start_dir

		// Third Example
		colors.LightGray, colors.Restore, // third example comment
		colors.Brown, colors.Restore, // third example findref
		colors.Green, colors.Restore, // third example option
		colors.Cyan, colors.Restore, // third example match_regex
		colors.Blue, colors.Restore, // fourth example start_dir
		colors.Purple, colors.Restore, // fourth example filename_regex

		// Fourth Example
		colors.LightGray, colors.Restore, // fourth example comment
		colors.Brown, colors.Restore, // fourth example findref
		colors.Green, colors.Restore, // fourth example options
		colors.Cyan, colors.Restore, // fourth example match_regex
		colors.Blue, colors.Restore, // fourth example start_dir
		colors.Purple, colors.Restore, // fourth example filename_regex
	)
}

const Version = "1.2.5"
const Date = "2024-03-10"

const MaxLineLengthDefault = 2000

var FILE_PROCESSING_COMPLETE error = nil

var settings *Settings = NewSettings()
var statistics *Statistics = NewStatistics()
var colors *Colors = NewColors()

var filenameOnlyFiles []string = make([]string, 0, 100)
var filesToScan []FileToScan = make([]FileToScan, 0, 100)

func usageAndExit() {
	flag.Usage()
	os.Exit(1)
}

func usageAndExitErr(errMsg error) {
	flag.Usage()
	fmt.Println(colors.Red + "[error]: " + errMsg.Error() + colors.Restore)
	os.Exit(1)
}

func debug(a ...interface{}) {
	if settings.Debug {
		fmt.Println(a...)
	}
}

func containsNullByte(line []byte) bool {
	for _, el := range line {
		if el == 0 {
			return true
		}
	}
	return false
}

func checkForMatches(path string) []Match {
	debug(colors.Blue+"Checking file for matches:"+colors.Restore, path)
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(colors.Red+"Error opening file at '"+path+"'.  Err: "+colors.Restore, err)
		debug(colors.Red+"Error opening file at '"+path+"'.  It might be a bad symlink.  Err: "+colors.Restore, err)
		return []Match{Match{path, 0, []byte{}, []int{}, 0}}
	}
	defer func() {
		// if path == "src/main/java/com/canopy/service/EFileService.java" {
		// 	fmt.Println("Closing the file: " + path)
		// }
		file.Close()
	}()

	retval := make([]Match, 50)

	// Split function defaults to ScanLines
	scanner := bufio.NewScanner(file)

	const sixtyFourKB = 64 * 1024
	const oneMB = 1024 * 1024
	const tenMB = 10 * oneMB
	const hundredMB = 100 * tenMB

	// TODO: We should check the file size and size the buffer intelligently
	// We open the file above on line 118 so maybe there's a method we can
	// use on the file object, but if not we can use os.Stat()

	// Fix for max token size:  https://stackoverflow.com/a/37455465/2062384
	buf := make([]byte, 0, sixtyFourKB)
	scanner.Buffer(buf, hundredMB) // Files up to 100 MB in size.  Lower if memory becomes a problem

	var lineNumber int = 0
	for scanner.Scan() {
		lineNumber += 1
		line := scanner.Bytes()
		statistics.IncrLineCount()
		if containsNullByte(line) {
			// This is a binary file.  Skip it!
			debug(colors.Blue+"Not processing binary file:"+colors.Restore, path)
			statistics.IncrSkippedNullCount()
			return retval
		}
		if matchIndex := settings.MatchRegex.FindIndex(line); matchIndex != nil {
			// we have a match! loc == nil means no match so just ignore that case
			statistics.IncrMatchCount()
			if settings.FilenameOnly {
				filenameOnlyFiles = append(filenameOnlyFiles, path)
			} else {
				m := Match{path, lineNumber, line, matchIndex, settings.MaxLineLength}
				if !settings.NoMaxLineLength && (len(line) > settings.MaxLineLength) {
					statistics.IncrSkippedLongCount()
					m.printMatchClip()
					// m.printMatchTooLong()
				} else {
					m.printMatch()
				}
				retval = append(retval, m)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		debug(colors.Red+"Error scanning line from file '"+path+"'. File will be skipped.  Err: "+colors.Restore, err)
		statistics.IncrErroredFilesCount()
	}
	return retval
}

func processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		debug("filepath.Walk encountered error with path '"+path+"'", err)
		return FILE_PROCESSING_COMPLETE
	}

	if info.IsDir() {
		if settings.IsHidden(path) {
			debug(colors.Blue, "Directory", path, "is hidden and will be pruned", colors.Restore)
			return filepath.SkipDir // skip the whole sub-contents of this hidden directory
		} else {
			return FILE_PROCESSING_COMPLETE
		}
	}

	if settings.PassesFileFilter(path) {
		debug(colors.Blue+"Passes file filter:", path)
		if settings.IsHidden(path) {
			debug(colors.Blue + "Hidden file '" + colors.Restore + path + colors.Blue + "' not processed")
			return FILE_PROCESSING_COMPLETE
		}
		statistics.IncrFilesToScan()
		defer statistics.IncrFileCount()

		filesToScan = append(filesToScan, FileToScan{Path: path, Info: info, Err: err})
	} else {
		debug(colors.Blue + "Ignoring file cause it doesn't match filter: " + colors.Restore + path)
	}
	return FILE_PROCESSING_COMPLETE
}

func getMatchRegex(ignoreCase bool, matchCase bool, usersRegex string) *regexp.Regexp {
	// If ignore case is set, ignore the case of the regex.
	// if match-case is not set, use smart case which means if it's all lower case be case-insensitive,
	// but if there's capitals then be case-sensitive
	if ignoreCase || (!matchCase && !regexp.MustCompile("[A-Z]").MatchString(usersRegex)) {
		debug(colors.Blue, "Match regex will be case-insensitive", colors.Restore)
		return regexp.MustCompile("(?i)" + usersRegex)
	} else {
		debug(colors.Blue, "Match regex will be exactly as user provided", colors.Restore)
		return regexp.MustCompile(usersRegex)
	}
}

func versionString(color bool) string {
	if color {
		return fmt.Sprintf("%s%s%s%s%s%s%s", colors.Cyan, "findref (version ", Version, " released on ", Date, ")", colors.Restore)
	} else {
		return fmt.Sprintf("%s%s%s%s%s", "findref (version ", Version, " released on ", Date, ")")
	}
}

func printVersion() {
	fmt.Println(versionString(true))
}

func uniq(stringSlice []string) []string {
	/* There is no built-in uniq function for slices, so we will use a map */
	stringMap := make(map[string]bool)
	for _, v := range stringSlice {
		stringMap[v] = true
	}
	retval := make([]string, len(stringMap), len(stringMap))
	i := 0
	for key := range stringMap {
		retval[i] = key
		i++
	}
	return retval
}

func finishAndExit() {
	if settings.FilenameOnly {
		filenames := uniq(filenameOnlyFiles)
		sort.Strings(filenames)
		for _, filename := range filenames {
			fmt.Printf("%s%s%s\n", colors.Purple, filename, colors.Restore)
		}
	}

	if settings.TrackStats {
		fmt.Printf("%sElapsed time:%s  %s\n", colors.Cyan, colors.Restore, statistics.ElapsedTime().String())
		fmt.Printf("%sLines scanned:%s %d\n", colors.Cyan, colors.Restore, statistics.LineCount())
		fmt.Printf("%sFiles scanned:%s %d\n", colors.Cyan, colors.Restore, statistics.FileCount())
		fmt.Printf("%sMatches found:%s %d\n", colors.Cyan, colors.Restore, statistics.MatchCount())
		fmt.Printf("%sSkipped Long: %s %d\n", colors.Cyan, colors.Restore, statistics.SkippedLongCount())
		fmt.Printf("%sSkipped Null: %s %d\n", colors.Cyan, colors.Restore, statistics.SkippedNullCount())
		fmt.Printf("%sErrored Files:%s %d\n", colors.Cyan, colors.Restore, statistics.ErroredFilesCount())
	}
}

func worker(id int, jobs <-chan string, results chan<- []Match) {
	for file := range jobs {
		debug(colors.Blue, "Worker number", id, "started file", colors.Restore, file)
		results <- checkForMatches(file)
		debug(colors.Blue, "Worker number", id, "finished file", colors.Restore, file)
	}
}

func main() {
	aPtr := flag.Bool("a", false, "Alias for --all")
	sPtr := flag.Bool("s", false, "Alias for --stats")
	dPtr := flag.Bool("d", false, "Alias for --debug")
	hPtr := flag.Bool("h", false, "Alias for --hidden")
	vPtr := flag.Bool("v", false, "Alias for --version")
	nPtr := flag.Bool("n", false, "Alias for --no-color")
	mPtr := flag.Bool("m", false, "Alias for --match-case")
	iPtr := flag.Bool("i", false, "Alias for --ignore-case")
	fPtr := flag.Bool("f", false, "Alias for --filename-only")
	xPtr := flag.Bool("x", false, "Alias for --no-max-line-length")
	lPtr := flag.Int("l", MaxLineLengthDefault, "Alias for --max-line-length")
	allPtr := flag.Bool("all", false, "Include hidden files and ignore case (implies: -i -h)")
	helpPtr := flag.Bool("help", false, "Show usage")
	statsPtr := flag.Bool("stats", false, "Track and display statistics")
	debugPtr := flag.Bool("debug", false, "Enable debug mode")
	hiddenPtr := flag.Bool("hidden", false, "Include hidden files and files in hidden directories")
	versionPtr := flag.Bool("version", false, "Print current version and exit")
	nocolorPtr := flag.Bool("no-color", false, "Don't use color in output")
	matchCasePtr := flag.Bool("match-case", false, "Match regex case (if unset smart-case is used)")
	ignoreCasePtr := flag.Bool("ignore-case", false, "Ignore case in regex (overrides smart-case)")
	filenameOnlyPtr := flag.Bool("filename-only", false, "Display only filenames with matches")
	maxLineLengthPtr := flag.Int("max-line-length", MaxLineLengthDefault, "Set maximum line length in characters (default is 2,000)")
	noMaxLineLengthPtr := flag.Bool("no-max-line-length", false, "Remove maximum line length.  Match againt lines of any length")

	flag.Usage = func() {
		fmt.Print(Usage())
	}
	flag.Parse()

	if *vPtr || *versionPtr {
		printVersion()
		os.Exit(0)
	}

	if *helpPtr {
		usageAndExit()
	}

	if *nPtr || *nocolorPtr {
		debug("Color output is disabled")
		colors.ZeroColors()
	}

	if *xPtr && (*lPtr != *maxLineLengthPtr || *lPtr != MaxLineLengthDefault) {
		usageAndExitErr(fmt.Errorf("%s", "Explicit -l|--max-line-length contradicts -x|--no-max-line-length"))
	}

	settings.Debug = *debugPtr || *dPtr
	settings.TrackStats = *statsPtr || *sPtr
	settings.FilenameOnly = *filenameOnlyPtr || *fPtr
	settings.IncludeHidden = *hiddenPtr || *hPtr
	settings.IncludeHidden = *allPtr || *aPtr // -a implies -h
	settings.NoMaxLineLength = *noMaxLineLengthPtr || *xPtr
	*matchCasePtr = *matchCasePtr || *mPtr
	*ignoreCasePtr = *ignoreCasePtr || *iPtr
	*ignoreCasePtr = *allPtr || *aPtr // -a implies -i

	if *lPtr != MaxLineLengthDefault {
		settings.MaxLineLength = *lPtr
	}
	if *maxLineLengthPtr != MaxLineLengthDefault {
		settings.MaxLineLength = *maxLineLengthPtr
	}

	if settings.TrackStats {
		statistics.startTime = time.Now()
		debug(colors.Blue, "Start time is:", colors.Restore, statistics.startTime.String())
	}

	debug(colors.Blue, "stats enabled: ", colors.Restore, settings.TrackStats)
	debug(colors.Blue, "match-case enabled: ", colors.Restore, *matchCasePtr)
	debug(colors.Blue, "ignore-case enabled: ", colors.Restore, *ignoreCasePtr)
	debug(colors.Blue, "include hidden files: ", colors.Restore, settings.IncludeHidden)
	debug(colors.Blue, "debug mode: ", colors.Restore, settings.Debug)
	debug(colors.Blue, "filename only: ", colors.Restore, settings.FilenameOnly)
	debug(colors.Blue, "max line length: ", colors.Restore, settings.MaxLineLength)
	debug(colors.Blue, "no max line length enabled: ", colors.Restore, settings.NoMaxLineLength)

	rootDir := "."

	if len(flag.Args()) < 1 {
		usageAndExitErr(fmt.Errorf("%s", "Must specify regex to match against files"))
	} else if len(flag.Args()) > 3 {
		usageAndExitErr(fmt.Errorf("%s", "Too many args (expected 1 <= 3)"))
	} else {
		settings.MatchRegex = getMatchRegex(*ignoreCasePtr, *matchCasePtr, flag.Args()[0])

		if len(flag.Args()) >= 2 {
			rootDir = flag.Args()[1]
		}
		if len(flag.Args()) == 3 {
			settings.FilenameRegex = regexp.MustCompile(flag.Args()[2])
		}
	}

	debug(colors.Blue, "matchRegex: ", colors.Restore, settings.MatchRegex.String())
	debug(colors.Blue, "rootDir: ", colors.Restore, rootDir)
	debug(colors.Blue, "fileRegex: ", colors.Restore, settings.FilenameRegex.String())

	runtime.GOMAXPROCS(runtime.NumCPU())
	filepath.Walk(rootDir, processFile)

	// TODO: set niceness value to low

	jobs := make(chan string, len(filesToScan))
	results := make(chan []Match, 100)

	// two workers for each core
	numWorkers := runtime.NumCPU() * 1
	for w := 0; w < numWorkers; w++ {
		go worker(w, jobs, results)
	}

	// create a job for each file to scan
	for _, val := range filesToScan {
		jobs <- val.Path
	}
	close(jobs)

	for r := 0; r < len(filesToScan); r++ {
		result := <-results
		for _, res := range result {
			if res.hasMatch() {
				//res.printMatch()
			}
		}
	}

	// Repeat settings at the end
	debug(colors.Cyan, "Search settings were:", colors.Restore)
	debug(colors.Blue, "* stats enabled: ", colors.Restore, settings.TrackStats)
	debug(colors.Blue, "* match-case enabled: ", colors.Restore, *matchCasePtr)
	debug(colors.Blue, "* ignore-case enabled: ", colors.Restore, *ignoreCasePtr)
	debug(colors.Blue, "* include hidden files: ", colors.Restore, settings.IncludeHidden)
	debug(colors.Blue, "* debug mode: ", colors.Restore, settings.Debug)
	debug(colors.Blue, "* filename only: ", colors.Restore, settings.FilenameOnly)
	debug(colors.Blue, "* max line length: ", colors.Restore, settings.MaxLineLength)
	debug(colors.Blue, "* no max line length enabled: ", colors.Restore, settings.NoMaxLineLength)
	debug(colors.Blue, "* matchRegex: ", colors.Restore, settings.MatchRegex.String())
	debug(colors.Blue, "* rootDir: ", colors.Restore, rootDir)
	debug(colors.Blue, "* fileRegex: ", colors.Restore, settings.FilenameRegex.String())

	finishAndExit()
}
