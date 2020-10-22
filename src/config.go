package main

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var currentAgentStatus string = ""
var GlobalConfig *XMLConfig

type ValueAttr struct {
	Value string `xml:"value,attr"`
}

func (va ValueAttr) ToInt() int {
	val, err := strconv.Atoi(va.Value)
	if err != nil {
		panic(err)
	}
	return val
}

func (va ValueAttr) ToFloat() float64 {
	val, err := strconv.ParseFloat(va.Value, 64)
	if err != nil {
		panic(err)
	}
	return val
}
func (va ValueAttr) ToString() string {
	return va.Value
}

type TCPService struct {
	Name             ValueAttr
	IPAddress        ValueAttr
	Port             ValueAttr
	MaxConnections   ValueAttr
	ImportanceFactor ValueAttr
}

type CPU struct {
	ImportanceFactor ValueAttr
	ThresholdValue   ValueAttr
}

type RAM struct {
	ImportanceFactor ValueAttr
	ThresholdValue   ValueAttr
}

type XMLConfig struct {
	XMLName                           xml.Name `xml:"xml"`
	Cpu                               CPU
	Ram                               RAM
	TCPService                        []TCPService
	ReadAgentStatusFromConfig         ValueAttr
	ReadAgentStatusFromConfigInterval ValueAttr
	ReturnIdleInsteadLoad             ValueAttr
	AgentStatus                       ValueAttr
	Interval                          ValueAttr
	Port                              ValueAttr
}

func readConfig(configFilePath string) {
	xmlFile, err := os.Open(configFilePath)
	if err != nil {
		panic(err)
	}
	defer xmlFile.Close()
	content, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		panic(err)
	}

	err = xml.Unmarshal(content, &GlobalConfig)
	if err != nil {
		panic(err)
	}
}

func InitConfig(logFilePath string, configFilePath string) {
	f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	mw := io.MultiWriter(os.Stdout, f)
	log.SetOutput(mw)
	readConfig(configFilePath)

	intervalTicker := time.NewTicker(time.Second * time.Duration(GlobalConfig.Interval.ToInt()))
	go func() {
		for {
			select {
			case <-intervalTicker.C:
				initialRun = false
			}
		}
	}()

	if strings.ToLower(GlobalConfig.ReadAgentStatusFromConfig.Value) == "true" {
		statusTicker := time.NewTicker(time.Second * time.Duration(GlobalConfig.ReadAgentStatusFromConfigInterval.ToInt()))
		go func() {
			for {
				select {
				case <-statusTicker.C:
					readConfig(configFilePath)
					// If status changed, send 'up ready' for a full interval
					if currentAgentStatus != GlobalConfig.AgentStatus.Value {
						initialRun = true
						intervalTicker.Stop()
						intervalTicker = time.NewTicker(time.Second * time.Duration(GlobalConfig.Interval.ToInt()))
						go func() {
							for {
								select {
								case <-intervalTicker.C:
									initialRun = false
								}
							}
						}()
					}
					currentAgentStatus = GlobalConfig.AgentStatus.Value
				}
			}
		}()
	}
}
