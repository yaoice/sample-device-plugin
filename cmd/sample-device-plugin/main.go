package main

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/yaoice/sample-device-plugin/pkg/server"
	"github.com/yaoice/sample-device-plugin/pkg/utils/ldflags"
	"k8s.io/component-base/cli/flag"
	"os"
	"path/filepath"
	"time"
)

func main() {
	flag.InitFlags()
	// if checking version, print it and exit
	ldflags.PrintAndExitIfRequested()
	log.Info("sample device plugin starting")
	sampleSrv := server.NewSampleServer()
	go sampleSrv.Run()

	// device plugin向kubelet注册
	if err := sampleSrv.RegisterToKubelet(); err != nil {
		log.Fatalf("register to kubelet error: %v", err)
	} else {
		log.Infoln("register to kubelet successfully")
	}

	// 监听kubelet.sock，一旦创建则重启
	devicePluginSocket := filepath.Join(server.DevicePluginPath, server.KubeletSocket)
	log.Info("device plugin socket name:", devicePluginSocket)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error("Failed to created FS watcher.")
		os.Exit(1)
	}
	defer watcher.Close()
	err = watcher.Add(server.DevicePluginPath)
	if err != nil {
		log.Error("watch kubelet error")
		return
	}
	log.Info("watching kubelet.sock")
	for {
		select {
		case event := <-watcher.Events:
			log.Infof("watch kubelet events: %s, event name: %s, isCreate: %v", event.Op.String(), event.Name, event.Op&fsnotify.Create == fsnotify.Create)
			if event.Name == devicePluginSocket && event.Op&fsnotify.Create == fsnotify.Create {
				time.Sleep(time.Second)
				log.Fatalf("inotify: %s created, restarting.", devicePluginSocket)
			}
		case err := <-watcher.Errors:
			log.Fatalf("inotify: %s", err)
		}
	}
}
