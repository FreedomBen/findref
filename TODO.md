Things To DO:

- Security and code scanning tools
- Create a Brew package for Mac.  It should also be compatible with Brew for Linux.

- Create findref config file support.  Include excludes and any other common settings

- I'd like to add a config file option to this tool.  It should look in the current working directory first for ./.findref.yaml and then in the user's home directory under the appropriate XDG locations, and finally ~/.findref.yaml.  If the config file is not found, just operate like normal, but if the config file is found, it should be read and any options it includes should be applied just as if the user passed those options as flags to the CLI.  It should be in YAML and include the ability to set each possible value that can be passed on the command line.
- Add a flag or command to generate a base config file.  It should allow specifying a flag for local (in the current working directory) or user global (in the XDG data location or ~/.findref.yaml).  It should be initially set to the default settings, but include comments telling the user what each setting does.

- (Maybe) Add a flag for searching "source code files" that will apply a filename matcher that matches source code files only
