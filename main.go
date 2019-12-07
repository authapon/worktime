package main

import (
	"fmt"
	mw "github.com/authapon/mwebserv"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		showUsage()
		return
	}

	startOK := false

	if len(os.Args) > 1 {
		if os.Args[1] == "exampleconfig" {
			showExampleConfig()
			return
		}

		if os.Args[1] == "work" {
			if len(os.Args) > 2 {
				makeConfig(os.Args[2])
			} else {
				makeConfig("/etc/worktime.toml")
			}
			startOK = true

		}
	}

	if !startOK {
		fmt.Printf(`
Command Error !!!


`)
		return
	}

	app := mw.New()
	app.SetAsset(Asset, AssetNames)
	app.ViewBindata("template")
	app.StaticBindata("static")

	app.Get("/", indexPage)
	app.Post("/checkin", checkinPage)
	app.Get("/today", todayPage)
	app.Get("/report", reportPage)
	app.Post("/genReport", genReportPage)
	app.Get("/personReport/:epassport/:month/:year", personReportPage)
	app.Get("/personReport2/:epassport/:day1/:month1/:year1/:day2/:month2/:year2", personReportRangePage)

	app.Get("/service/getTime", getTimeService)
	app.Post("/service/epassport", epassportService)
	app.Get("/service/canChkIn", canChkInService)

	app.Serve(portConf)
}

func showUsage() {
	fmt.Printf(`
worktime [command] [data]

command: exampleconfig   <- for print the example config ; example = worktime exampleconfig
command: work            <- for start the worktime system and data is file config ; example = worktime work /etc/worktime.toml 
                         <- if no file config provide the default is /etc/worktime.toml


`)
}

func showExampleConfig() {
	fmt.Printf(`
port   = ":80"
#cpu   = 4
mysql  = "user:password@tcp(localhost:3306)/worktime"
start  = 600
late   = 830
stop   = 930
elogin = "https://elogin.rmutsv.ac.th"


`)
}
