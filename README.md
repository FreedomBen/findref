# findref

`findref` helps you find strings, or match regular expressions, in a directory of files.  It is inspired by `git grep` which is a good tool, but has limitations that made writing a replacement a worthy endeavor.

`findref` regular expressions are the same as `grep` regular expressions (the ones you use with `grep -E` or `egrep`).  `findref` respects the `.gitignore` file if in a `git` repo, unless you tell it not to.

Usage:

    findref [-f|--fast (skip git ignore list)] [-i|--ignore-case] "what text (RegEx) to look for" "[starting location (root dir)]" "[filenames to check (must match pattern)]"

How it compares to other tools (or why it is better in my opinion):
-------------------------------------------------------------------

**grep**:  `findref` adds much simpler recursive search that doesn't require a bunch of tedious flags to get pleasant output.  It also ignores files that git ignores, which helps you avoid a lot of junk matches.  Speed is a little slower but views as a good tradeoff for the more useful results.

**git grep**:  `findref` output looks very similar, but adds colorization, which makes reading it *much* easier.  `findref` also works on non-git repos, unlike git grep.  Speed is about the same.

**Ag (or the silver searcher)**:  `findref` is a little slower, but has much better formatting and coloring.  `findref` does not currently have a vim plugin tho, so for searching within vim, [ag](https://github.com/vim-scripts/ag.vim) is the way to go.

Examples:
---------

Let's say we are looking for the string "getMethodName":

Simple usage that starts looking recursively in the current directory, and checks all files (excluding binary files) for the string (which could also be a regular expression)

    findref getMethodName

To go case insensitive, simply add -i or --ignore-case as the first arg:

    findref -i getMethodName

We could also get case-insensitive by using "smart-case" by just using lower case letters in our regex pattern:

    findref getmethodname

Or search by regex:

    findref "str[i1]ng.*"

You can add a starting directory, which defaults to the current directory if not specified:

    findref "str[i1]ng.*" "/home/ben/my-starting-directory"

If you want to restrict which files are searched, you can pass a glob pattern as a file argument.  For example, to only search cpp files:

    findref "str[i1]ng.*" "~/my-starting-directory" "*.cpp"

Or to restrict the search to C++ code files (.h and .cpp):

    findref "str[i1]ng.*" "~/my-starting-directory" "*.[hc]*"

If you are searching in a git repository, and it is simply too slow, you can disable the parsing of the git ignore list by passing the -f or --fast flag as the first argument.  This makes `findref` behave like it does when you are not inside of a git repository (which is a little faster)

    findref --fast "str[i1]ng.*" "~/my-starting-directory" "*.[hc]*"
