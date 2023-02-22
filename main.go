package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/alphagov/paas-nginx-hosts-reload/utils"
)

const (
	defaultProcessCheckInterval time.Duration = time.Second * 5
)

type NginxHostsReload struct {
	stop         chan bool
	Interval     time.Duration
	Logger       lager.Logger
	Utils        utils.Utils
	ProcessName  string
	CmdSubstring string
}

func NewNginxHostsReload(interval time.Duration, logger lager.Logger, utils utils.Utils, processName string, CmdSubstring string) *NginxHostsReload {
	hm := &NginxHostsReload{stop: make(chan bool)}
	hm.Logger = logger
	hm.Utils = utils
	hm.ProcessName = processName
	hm.CmdSubstring = CmdSubstring
	hm.Interval = interval
	return hm
}

func (hm *NginxHostsReload) Monitor() error {

	hm.Logger.Info("Monitoring /etc/hosts for changes...")
	initialModTime, err := hm.Utils.ModTime()
	if err != nil {
		return err
	}
	for {
		select {
		case <-hm.stop:
			return nil
		default:
			// Check the current state of the hosts file
			hm.Logger.Debug("check-host-file-mod-time", lager.Data{
				"message": "Checking hosts file for changes...",
			})
			currentModTime, err := hm.Utils.ModTime()
			if err != nil {
				return err
			}

			// Compare the modification times
			if currentModTime != initialModTime {
				pid, err := hm.Utils.FindNginxPID(hm.ProcessName, hm.CmdSubstring)
				initialModTime = currentModTime
				if err != nil {
					return err
				}
				hm.Logger.Info("hosts-modified", lager.Data{
					"message": fmt.Sprintf("Hosts file has been modified, sending SIGHUP to nginx %d.", pid),
				})
				if pid != 0 {
					err = hm.Utils.SigHup(pid)
					if err != nil {
						return err
					}
				} else {
					hm.Logger.Error("find-nginx-pid", errors.New("nginx process not found"))
				}
			}

			time.Sleep(hm.Interval)
		}
	}
}

func (hm *NginxHostsReload) Stop() {
	hm.stop <- true
}

func main() {
	logLevel := flag.String("log-level", "info", "debug level")
	interval := flag.Duration("interval", defaultProcessCheckInterval, "interval to check for changes")
	processName := flag.String("process-name", "nginx", "process name to check")
	cmdSubstring := flag.String("cmd-substring", "master process", "cmd substring to check")
	flag.Parse()

	lagerLogLevel, err := lager.LogLevelFromString(strings.ToLower(*logLevel))
	if err != nil {
		log.Fatal(err)
	}

	logger := lager.NewLogger("nginx-hosts-reload")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lagerLogLevel))

	utils := utils.NewUtils()

	hm := NewNginxHostsReload(*interval, logger, utils, *processName, *cmdSubstring)
	if err := hm.Monitor(); err != nil {
		fmt.Println(err)
	}
}
