package main

import (
	"io/ioutil"
	"strings"
	"errors"
	"regexp"
	"strconv"
	"bufio"
	"fmt"
	"log"
	"os"
)


func printCpuInfo(){
	log.Printf("Read and print CPU info: Vendor: \033[32m%s\033[0m, Family: \033[32m%s\033[0m, Series: \033[32m%s\033[0m, Cores: \033[32m%s\033[0m, Speed: \033[32m%d\033[0m\n", cpuInfo.Vendor, cpuInfo.Family, cpuInfo.Series, cpuInfo.Cores, cpuInfo.ClockSpeed)
}

func parseConfig(filePath string) (map[string]string, error) {
	config := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

func readConfig(filePath string){
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if err := readCpuInfo(); err != nil {
			log.Printf("Error reading CPU info: %v\n", err)
			os.Exit(1)
		}
		return
	}
	log.Printf("Config file path: %s\n",filePath)
	config, err := parseConfig(filePath)
	if err != nil {
		log.Printf("Error reading ConfigFile: %v\n", err)
		os.Exit(1)
	}

	cpuInfo.Vendor = config["Vendor"]
	cpuInfo.Family = config["Family"]
	cpuInfo.Series = config["Series"]
	cpuInfo.Cores = config["Cores"]
	if clockSpeed, err := strconv.Atoi(config["ClockSpeed"]); err == nil {
		cpuInfo.ClockSpeed = clockSpeed
	} else {
		cpuInfo.ClockSpeed = 0
	}
	printCpuInfo()
}
func readCpuInfo() (error) {
	var cpuInfoContent string

	data, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		return  err
	}
	cpuInfoContent = strings.TrimSpace(string(data))

	if strings.Contains(cpuInfoContent, "GenuineIntel") {
		cpuInfo.Vendor = "Intel"
	} else if strings.Contains(cpuInfoContent, "AuthenticAMD") {
		cpuInfo.Vendor = "AMD"
	} else {
		return errors.New("Unknown Model")
	}
	cpuInfo.Cores = parseCoreInfo(cpuInfoContent)

	modelRegex := regexp.MustCompile(`model name\s+:\s+(.*)`)
	match := modelRegex.FindStringSubmatch(cpuInfoContent)
	if len(match) < 1 {
		cpuInfo.Family = ""
		cpuInfo.Series = ""
		return errors.New("Cpuinfo read failed")
	}
	if cpuInfo.Vendor == "AMD" {
		cpuInfo.Family, cpuInfo.Series = parseAMDModel(match[1])
	} else if cpuInfo.Vendor == "Intel" {
		cpuInfo.Family, cpuInfo.Series = parseIntelModel(match[1])
	} 

	frequencyRegex := regexp.MustCompile(`cpu MHz\s+:\s+(\d+\.\d+)`)
	frequencyMatch := frequencyRegex.FindStringSubmatch(cpuInfoContent)
	if len(frequencyMatch) > 1 {
		frequencyMHz, err := strconv.ParseFloat(frequencyMatch[1], 64)
		if err == nil {
			cpuInfo.ClockSpeed = int(frequencyMHz)
		}
	}
	printCpuInfo()
	return nil
}

func parseAMDModel(model string) (string, string) {
	
	model = strings.TrimSpace(strings.Replace(model, "AMD", "", 1))
	parts := strings.Fields(model)

	
	var series string
	for i := len(parts) - 1; i > 0; i-- {
		if match, _ := regexp.MatchString(`^[0-9]`, parts[i]); match {
			series = strings.Join(parts[i:], " ")
			break
		}
	}
	
	if series == "" {
		for i := len(parts) - 1; i >= 0; i-- {
			if match, _ := regexp.MatchString(`.*-.*`, parts[i]); match {
				series = parts[i]
				break
			}
		}
	}

	
	var family string
	if series != "" {
		family = strings.TrimSpace(strings.Replace(model, series, "", 1))
	}

	return family, strings.TrimSpace(series)
}

func parseIntelModel(model string) (string, string) {

	reBrackets := regexp.MustCompile(`\([^()]*\)`)
	modelWithoutBrackets := reBrackets.ReplaceAllString(model, "")
	reBeforeIntel := regexp.MustCompile(`^[^I]*Intel`)
	modelAfterIntel := reBeforeIntel.ReplaceAllString(modelWithoutBrackets, "")

	parts := strings.Fields(strings.ReplaceAll(modelAfterIntel, " CPU", ""))
	if len(parts) < 2 {
		return "", ""
	}

	
	startIndex := strings.Index(modelAfterIntel, parts[1])
	endIndex := strings.Index(modelAfterIntel, "@")
	if endIndex == -1 {
		endIndex = len(modelAfterIntel)
	}
	series := strings.TrimSpace(modelAfterIntel[startIndex:endIndex])

	return parts[0], series
}

func parseCoreInfo(cpuInfoContent string) string {
	rePhysicalID := regexp.MustCompile(`(?m)^physical id\s+:\s+(\d+)$`)
	physicalIDMatches := rePhysicalID.FindAllStringSubmatch(cpuInfoContent, -1)

	physicalIDs := make(map[string]struct{})
	for _, match := range physicalIDMatches {
		if len(match) > 1 {
			physicalIDs[match[1]] = struct{}{}
		}
	}

	if len(physicalIDs) > 1 {
		cpuCores := make(map[string]string)
		for physicalID := range physicalIDs {
			reCores := regexp.MustCompile(fmt.Sprintf(`(?s)physical id\s*:\s*%s.*?cpu cores\s*:\s*(\d+)`, physicalID))
			match := reCores.FindStringSubmatch(cpuInfoContent)
			if len(match) > 1 {
				cpuCores[physicalID] = match[1]
			}
		}
		coreStrings := []string{}
		for _, cores := range cpuCores {
			coreStrings = append(coreStrings, cores)
		}
		return strings.Join(coreStrings, " + ")
	} else {
		reSingleCores := regexp.MustCompile(`cpu cores\s*:\s*(\d+)`)
		match := reSingleCores.FindStringSubmatch(cpuInfoContent)
		if len(match) > 1 {
			return match[1]
		}
		return "Unknown"
	}
}

func readCPUTemperature() (int, error) {
	tempPath := "/sys/class/thermal/thermal_zone0/temp"

	data, err := ioutil.ReadFile(tempPath)
	if err != nil {
		return 0, err
	}

	tempStr := strings.TrimSpace(string(data))
	tempMilliCelsius, err := strconv.Atoi(tempStr)
	if err != nil {
		return 0, err
	}

	tempCelsius := tempMilliCelsius / 1000

	return tempCelsius, nil
}
