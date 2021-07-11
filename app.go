package main

import (
	"flag"
	"fmt"
	"github.com/Jguer/go-alpm/v2"
	paconf "github.com/Morganamilo/go-pacmanconf"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"os"
	"os/exec"
)

func SendMessage(updateCount int) error {
	api, err := tgbotapi.NewBotAPI(*botTokenFlag)
	if err != nil {
		return err
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	text := fmt.Sprintf("%d updates available for host %s", updateCount, hostname)
	msg := tgbotapi.NewMessage(*channelIdFlag, text)
	_, err = api.Send(msg)

	return err
}

func checkForUpgrades(h *alpm.Handle) (int, error) {
	localDb, err := h.LocalDB()
	if err != nil {
		return 0, err
	}

	syncDbs, err := h.SyncDBs()
	if err != nil {
		return 0, err
	}

	newPkgCount := 0
	for _, pkg := range localDb.PkgCache().Slice() {
		newPkg := pkg.SyncNewVersion(syncDbs)
		if newPkg != nil {
			newPkgCount++
		}
	}

	return newPkgCount, nil
}

var (
	channelIdFlag = flag.Int64("channel-id", 0, "Channel id for the telegram channel")
	botTokenFlag  = flag.String("bot-token", "", "Token for the telegram bot")
)

func main() {
	flag.Parse()
	err := exec.Command("/usr/bin/pacman", "-Syy").Run()
	if err != nil {
		fmt.Println(err)
		return
	}

	h, err := alpm.Initialize("/", "/var/lib/pacman")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer h.Release()
	pacmanConfig, _, err := paconf.ParseFile("/etc/pacman.conf")
	if err != nil {
		fmt.Println(err)
		return
	}
	/*
	   We have to configure alpm with pacman configuration
	   to load the repositories and other stuff
	*/
	for _, repo := range pacmanConfig.Repos {
		db, err := h.RegisterSyncDB(repo.Name, 0)
		if err != nil {
			fmt.Println(err)
			return
		}
		db.SetServers(repo.Servers)

		/*
		   Configure repository usage to match with
		   the alpm library provided formats
		*/
		if len(repo.Usage) == 0 {
			db.SetUsage(alpm.UsageAll)
		}
		for _, usage := range repo.Usage {
			switch usage {
			case "Sync":
				db.SetUsage(alpm.UsageSync)
			case "Search":
				db.SetUsage(alpm.UsageSearch)
			case "Install":
				db.SetUsage(alpm.UsageInstall)
			case "Upgrade":
				db.SetUsage(alpm.UsageUpgrade)
			case "All":
				db.SetUsage(alpm.UsageAll)
			}
		}
	}
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	upgradablePackages, err := checkForUpgrades(h)
	if upgradablePackages > 0 {
		fmt.Printf("%d packages have an upgrade", upgradablePackages)
		err = SendMessage(upgradablePackages)
		if err != nil {
			fmt.Printf(err.Error())
		}
	}
}
