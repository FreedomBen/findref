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
    %sUsage of %s:%s

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
%s
`,
		colors.Red, versionString(false), colors.Restore,
		colors.Brown, colors.Restore,
		colors.Green, colors.Restore,
		colors.Cyan, colors.Restore,
		colors.Blue, colors.Restore,
		colors.Purple, colors.Restore,
		colors.Red, colors.Restore,
		colors.Cyan, colors.Restore,
		colors.Blue, colors.Restore,
		colors.Purple, colors.Restore,
		colors.Red, colors.Restore,
		colors.Green, colors.Restore,
	)
}

const Version = "0.0.9"
const Date = "2017-10-04"

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
		fmt.Println(colors.Red+"Error opening file at '"+path+"'.  It might be a directory.  Err: "+colors.Restore, err)
		debug(colors.Red+"Error opening file at '"+path+"'.  It might be a directory.  Err: "+colors.Restore, err)
		return []Match{Match{path, 0, []byte{}, []int{}}}
	}
	defer func() {
		if path == "src/main/java/com/canopy/service/EFileService.java" {
			fmt.Println("Closing the file: " + path)
		}
		file.Close()
	}()

	retval := make([]Match, 50)

	scanner := bufio.NewScanner(file)
	var lineNumber int = 0
	for scanner.Scan() {
		lineNumber += 1
		line := scanner.Bytes()
		statistics.IncrLineCount()
		if containsNullByte(line) {
			// This is a binary file.  Skip it!
			debug(colors.Blue+"Not processing binary file:"+colors.Restore, path)
			return retval
		}
		if matchIndex := settings.MatchRegex.FindIndex(line); matchIndex != nil {
			// we have a match! loc == nil means no match so just ignore that case
			statistics.IncrMatchCount()
			if settings.FilenameOnly {
				filenameOnlyFiles = append(filenameOnlyFiles, path)
			} else {
				retval = append(retval, Match{path, lineNumber, line, matchIndex})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		debug(colors.Red+"Error scanning line from file '"+path+"'. File will be skipped.  Err: "+colors.Restore, err)
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
	}
}

func worker(id int, jobs <-chan string, results chan<- []Match) {
	for file := range jobs {
		debug(colors.Blue, "Worker number", id, "started file", file, colors.Restore)
		results <- checkForMatches(file)
		debug(colors.Blue, "Worker number", id, "finished file", file, colors.Restore)
	}
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

	settings.Debug = *debugPtr || *dPtr
	settings.TrackStats = *statsPtr || *sPtr
	settings.FilenameOnly = *filenameOnlyPtr || *fPtr
	settings.IncludeHidden = *hiddenPtr || *hPtr
	*matchCasePtr = *matchCasePtr || *mPtr
	*ignoreCasePtr = *ignoreCasePtr || *iPtr

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

	rootDir := "."

	if len(flag.Args()) < 1 {
		fmt.Errorf("%s", "Must specify regex to match against files")
		usageAndExit()
	} else if len(flag.Args()) > 3 {
		fmt.Errorf("%s", "Too many args (expected 1 <= 3)")
		usageAndExit()
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
	numWorkers := runtime.NumCPU() * 2
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
				res.printMatch()
			}
		}
	}

	fmt.Println(colors.Blue, "All jobs finished.  Ready to exit", colors.Restore)

	finishAndExit()
}
