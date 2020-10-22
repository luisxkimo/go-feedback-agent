package main

import (
	"log"
	"os"

	"github.com/howeyc/fsnotify"
	"github.com/kardianos/service"
)

var logger service.Logger

var (
	downTicker = 0.0
)

type program struct{}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *program) run() {
	logFilePath := os.Args[1]
	configFilePath := os.Args[2]
	InitConfig(logFilePath, configFilePath)
	srv := InitServer()
	go prepareConfigFileWatcher(configFilePath)
	for {
		conn, err := srv.server.Accept()
		if err != nil {
			log.Println(err)
		}
		go handleClient(conn)
	}
}

func prepareConfigFileWatcher(configFilePath string) {
	log.Println("Preparing config file watcher")
	watcher, errWatcher := fsnotify.NewWatcher()
	if errWatcher != nil {
		log.Println(errWatcher)
	}

	defer watcher.Close()
	done := make(chan bool)
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if ev.IsModify() {
					log.Println("New FileWatcher event:", ev)
					return
				}
			case err := <-watcher.Error:
				log.Println("Error on config FileWatcher:", err)
			}
		}
	}()

	if err := watcher.WatchFlags(configFilePath, fsnotify.FSN_MODIFY); err != nil {
		log.Println("Error on definition of WatchFlags:", err)
	}
	log.Println("Finished configuration file watcher setup")
	<-done

}

func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "FeedBackService",
		DisplayName: "TCP Feedback Service",
		Description: "This is a go service to provide system stats",
	}
	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
