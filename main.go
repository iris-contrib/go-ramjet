package main

import (
	"fmt"

	"github.com/Laisky/go-utils"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	_ "github.com/Laisky/go-ramjet/tasks/elasticsearch"
	_ "github.com/Laisky/go-ramjet/tasks/fluentd"
	_ "github.com/Laisky/go-ramjet/tasks/heartbeat"
	_ "github.com/Laisky/go-ramjet/tasks/logrotate/backup"
	"github.com/Laisky/go-ramjet/tasks/store"
)

// setupSettings setup arguments restored in viper
func setupSettings() {
	utils.Settings.Setup(utils.Settings.GetString("config"))

	if utils.Settings.GetBool("debug") { // debug mode
		fmt.Println("run in debug mode")
		utils.SetupLogger("debug")
	} else { // prod mode
		fmt.Println("run in prod mode")
		utils.SetupLogger("info")
	}
}

func main() {
	defer fmt.Println("All done")
	defer utils.Logger.Flush()
	fmt.Println("start main...")
	pflag.Bool("debug", false, "run in debug mode")
	pflag.Bool("dry", false, "run in dry mode")
	pflag.String("config", "/etc/go-ramjet/settings", "config file directory path")
	pflag.StringSliceP("task", "t", []string{}, "which tasks want to runnning, like\n ./main -t t1,t2,heartbeat")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	setupSettings()

	// Bind each task here
	store.Start()
	store.Run()
}
