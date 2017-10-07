package main

type Colors struct {
	Red         string
	Blue        string
	Cyan        string
	Green       string
	Black       string
	Brown       string
	White       string
	Yellow      string
	Purple      string
	Restore     string
	LightRed    string
	DarkGray    string
	LightGray   string
	LightBlue   string
	LightCyan   string
	LightGreen  string
	LightPurple string
}

func NewColors() *Colors {
	return (&Colors{}).RestoreColors()
}

func (zc *Colors) ZeroColors() {
	zc.Red = ""
	zc.Blue = ""
	zc.Cyan = ""
	zc.Green = ""
	zc.Black = ""
	zc.Brown = ""
	zc.White = ""
	zc.Yellow = ""
	zc.Purple = ""
	zc.Restore = ""
	zc.LightRed = ""
	zc.DarkGray = ""
	zc.LightGray = ""
	zc.LightBlue = ""
	zc.LightCyan = ""
	zc.LightGreen = ""
	zc.LightPurple = ""
}

func (zc *Colors) RestoreColors() *Colors {
	zc.Red = "\033[0;31m"
	zc.Blue = "\033[0;34m"
	zc.Cyan = "\033[0;36m"
	zc.Green = "\033[0;32m"
	zc.Black = "\033[0;30m"
	zc.Brown = "\033[0;33m"
	zc.White = "\033[1;37m"
	zc.Yellow = "\033[1;33m"
	zc.Purple = "\033[0;35m"
	zc.Restore = "\033[0m"
	zc.LightRed = "\033[1;31m"
	zc.DarkGray = "\033[1;30m"
	zc.LightGray = "\033[0;37m"
	zc.LightBlue = "\033[1;34m"
	zc.LightCyan = "\033[1;36m"
	zc.LightGreen = "\033[1;32m"
	zc.LightPurple = "\033[1;35m"
	return zc
}
