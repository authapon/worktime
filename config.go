package main

import (
	"fmt"
	"gopkg.in/BurntSushi/toml.v0"
	"io/ioutil"
	"os"
	"runtime"
)

var (
	config     map[string]interface{}
	portConf   string
	cpuConf    int
	mysqlConf  string
	startConf  int
	lateConf   int
	stopConf   int
	eloginConf string
)

func makeConfig(fconfig string) {
	config = make(map[string]interface{})
	config["port"] = ":80"
	config["cpu"] = int64(runtime.NumCPU())
	config["mysql"] = "user:password@tcp(localhost:3306)/worktime"
	config["start"] = int64(600)
	config["late"] = int64(830)
	config["stop"] = int64(930)
	config["elogin"] = "https://elogin.rmutsv.ac.th"

	configdata, err := ioutil.ReadFile(fconfig)
	if err != nil {
		fmt.Printf("Error to read " + fconfig)
		os.Exit(0)
	}
	err = toml.Unmarshal([]byte(configdata), &config)
	if err != nil {
		fmt.Printf("Error in " + fconfig)
		os.Exit(0)
	}

	portConf = config["port"].(string)
	cpuConf = int(config["cpu"].(int64))
	mysqlConf = config["mysql"].(string)
	startConf = int(config["start"].(int64))
	lateConf = int(config["late"].(int64))
	stopConf = int(config["stop"].(int64))
	eloginConf = config["elogin"].(string)

	runtime.GOMAXPROCS(cpuConf)

	fmt.Printf(`Run with this config:
port = %s
cpu = %d
mysql = %s
start = %d
late = %d
stop = %d
elogin = %s


`, portConf, cpuConf, mysqlConf, startConf, lateConf, stopConf, eloginConf)
}
