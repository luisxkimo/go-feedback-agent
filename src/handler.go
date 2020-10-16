package main

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

const (
	Normal = "Normal"
	Halt   = "Halt"
	Drain  = "Drain"
	Down   = "Down"
)

var (
	initialRun  = true
	autodrained = false
)

func handleClient(conn net.Conn) {
	defer conn.Close()
	conn.Write(GetResponseForMode())
	conn.Close()
}

func GetResponseForMode() (response []byte) {
	ramThresholdValue := GlobalConfig.Ram.ThresholdValue.ToFloat()
	cpuThresholdValue := GlobalConfig.Cpu.ThresholdValue.ToFloat()
	cpuImportance := GlobalConfig.Cpu.ImportanceFactor.ToFloat()
	ramImportance := GlobalConfig.Ram.ImportanceFactor.ToFloat()
	returnIdle, _ := strconv.ParseBool(strings.ToLower(GlobalConfig.ReturnIdleInsteadLoad.Value))

	switch GlobalConfig.AgentStatus.Value {
	case Normal:
		usedRam := 0.0
		averageCpuLoad := 0.0
		utilization := 0.0
		divider := 0.0

		// Calculate CPU
		if cpuImportance > 0 {
			cpuLoad, err := cpu.Percent(0, false)
			if err != nil {
				return []byte("0%\n")
			}
			averageCpuLoad = cpuLoad[0]
			divider++
		}

		// Calculate RAM
		if ramImportance > 0 {
			v, err := mem.VirtualMemory()
			if err != nil {
				return []byte("0%\n")
			}
			usedRam = v.UsedPercent
			divider++
		}

		// If any resource is important and utilized 100% then everything else is not important
		if averageCpuLoad > cpuThresholdValue && cpuThresholdValue > 0 || (usedRam > ramThresholdValue && ramThresholdValue > 0) {
			response = []byte("0% drain\n")
			autodrained = true
			return
		}

		for _, tcpService := range GlobalConfig.TCPService {
			// Make sure our importance factor is greater than 0 otherwise ignore
			if tcpService.ImportanceFactor.ToFloat() > 0 {
				// Get session occupied
				sessionOccupied := GetSessionUtilized(tcpService.IPAddress.Value, tcpService.Port.Value, tcpService.MaxConnections.ToInt())

				// Calculate utilization
				utilization = utilization + sessionOccupied*tcpService.ImportanceFactor.ToFloat()

				// increase our divider
				divider++

				if sessionOccupied > 99 && tcpService.ImportanceFactor.ToFloat() == 1 {
					response = []byte("0%\n")
					return
				}
			}
		}

		utilization = utilization + averageCpuLoad*cpuImportance
		utilization = utilization + usedRam*ramImportance

		utilization = utilization / divider

		// Account for utilization less than 0
		if utilization < 0 {
			utilization = 0
		}

		// Account for utilization more than 0
		if utilization > 100 {
			utilization = 100
		}

		if returnIdle {
			if autodrained {
				initialRun = true
			}
			response = []byte(fmt.Sprintf("%v%%\n", math.Ceil(100-utilization)))
			autodrained = false
		} else {
			// Branch not used. returnIdle variable ever is true
			response = []byte(fmt.Sprintf("%v%%\n", math.Ceil(utilization)))
			autodrained = false
		}

		if initialRun {
			response = append([]byte("up ready "), response...)
		}
	case Drain:
		response = []byte("drain\n")
	case Down:
		response = []byte("down\n")
	case Halt:
		response = []byte("down\n")
	default:
		response = []byte("error\n")
	}
	return
}

func GetSessionUtilized(IPAddress, servicePort string, maxNumberOfSessionsPerService int) (result float64) {
	numberOfEstablishedConnections := getNumberOfLocalEstablishedConnections(IPAddress, servicePort)
	if numberOfEstablishedConnections > 0 && maxNumberOfSessionsPerService > 0 {
		result = float64(numberOfEstablishedConnections) / float64(maxNumberOfSessionsPerService) * 100
	}
	return
}

func getNumberOfLocalEstablishedConnections(ipAddress string, port string) int {
	if ipAddress == "*" {
		ipAddress = ""
	}
	result := runcmd("netstat -nt | findstr " + ipAddress + ":" + port + "  | findstr ESTABLISHED ")
	count := len(strings.Split(result, "\n"))
	if count == 0 {
		return count
	}
	return count - 1
}
