package main

import (
	"bufio"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	logger          *logrus.Logger = logrus.New()
	resourceHashMap map[string]interface{}

	command string = "list"

	// flag-related variables
	fileName      string
	verboseLevel  bool
	resourceBlock string

	VALID_COMMANDS []string = []string{"list", "extract"}
)

func init() {
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	flag.StringVar(&fileName, "f", "", "path of terraform file")
	flag.BoolVar(&verboseLevel, "v", false, "enable verbose logging")
	flag.StringVar(&resourceBlock, "b", "default", "type of resource block")
}

func main() {
	flag.Parse()

	if verboseLevel {
		logger.SetLevel(logrus.DebugLevel)
	}

	if fileName == "" {
		logger.Error("No file path provided, check usage with -h")
		return
	} else if !CheckFileExists(fileName) || !CheckValidExtension(fileName) {
		logger.Error("Check that file exists and is of .tf type")
		return
	}

	if len(flag.Args()) > 0 {
		command = flag.Args()[0]
	}

	if !slices.Contains(VALID_COMMANDS, command) {
		logger.Errorf("Invalid command provided: %v", command)
		return
	}

	logger.Info("Starting terraform file parser...")
	content, _ := ReadFileToLines(fileName)

	resourceHashMap = RetrieveResourceBlocks(content)

	switch command {
	case "list":
		for key := range resourceHashMap {
			if key != "locals" {
				for subKey := range resourceHashMap[key].(map[string][]string) {
					logger.Debug(subKey)
				}
			}
		}
	case "extract":
		ExtractResourcesToFile(resourceHashMap, resourceBlock, fileName)
	}

	logger.Info("Parsing completed")
}

func ExtractResourcesToFile(resourceMap map[string]interface{}, resourceBlock string, fileName string) error {
	logger.Infof("Extracting file from %v", fileName)
	var targetDir string = filepath.Dir(fileName)
	var targetFile string = targetDir + "/" + resourceBlock + ".tf"

	file, err := os.Create(targetFile)
	if err != nil {
		logger.Errorf("Err is: %v", err)
		return err
	}
	defer file.Close()

	if resourceBlock == "locals" {
		for _, local := range resourceMap["locals"].([][]string) {
			for _, line := range local {
				_, err := file.WriteString(line + string('\n'))
				if err != nil {
					logger.Errorf("Error writing to file: %v", err)
				}
			}
			_, err := file.WriteString(string('\n'))
			if err != nil {
				logger.Errorf("Error writing to file: %v", err)
			}
		}
	}

	return nil
}

func CheckFileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return !os.IsNotExist(err)
}

func CheckValidExtension(fileName string) bool {
	if !strings.HasSuffix(fileName, ".tf") {
		logger.Errorf("Target file %v is not a terraform file", fileName)
		return false
	}
	return true
}

func ReadFileToLines(fileName string) ([]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func DetermineResource(line string) (string, string) {
	var resourceText string
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == '{' {
			resourceText = line[0 : i-1]
			break
		}
	}
	var resourceType string = strings.Split(resourceText, " ")[0]
	var resourceName string = ""
	if len(strings.Split(resourceText, " ")) > 1 {
		resourceName = strings.ReplaceAll(strings.Join(strings.Split(resourceText, " ")[1:], "."), `"`, "")
	}

	return resourceType, resourceName
}

func MapResource(hm map[string]interface{}, resourceType string, resourceName string, lines []string) map[string]interface{} {
	_, exists := hm["locals"]
	if !exists {
		hm["locals"] = make([][]string, 0)
	}

	_, exists = hm[resourceType]
	if !exists {
		hm[resourceType] = make(map[string][]string)
	}

	if resourceType == "locals" {
		hm["locals"] = append(hm["locals"].([][]string), lines)
	} else {
		hm[resourceType].(map[string][]string)[resourceName] = lines
	}

	return hm
}

func RetrieveResourceBlocks(lines []string) map[string]interface{} {
	var parsingResource bool = false

	var resourceType, resourceName string

	var startLine int = -1
	var endLine int = -1

	// each '{' adds 1 value, each '}' reduces 1 value
	// when value is 0, we have reached the end of the block
	var bracketValue int = 0

	var resourceHashMap map[string]interface{} = make(map[string]interface{})

	for i := 0; i < len(lines); i++ {
		if !parsingResource {
			if strings.Contains(lines[i], "{") {
				startLine = i
				parsingResource = true
				resourceType, resourceName = DetermineResource(lines[i])
			}

			if !parsingResource {
				continue
			}
		}

		for j := 0; j < len(lines[i]); j++ {
			if lines[i][j] == '{' {
				bracketValue++
			}

			if lines[i][j] == '}' {
				bracketValue--
			}
		}

		if bracketValue == 0 {
			parsingResource, endLine = false, i
			logger.Debugf("Found %v block :: %v at lines %v to %v", resourceType, resourceName, startLine+1, endLine+1)
			resourceHashMap = MapResource(resourceHashMap, resourceType, resourceName, lines[startLine:endLine+1])

			startLine, endLine = -1, -1
		}
	}
	return resourceHashMap
}
