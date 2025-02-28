package main

import (
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

var (
	originalFileName      string   = fileName
	originalVerboseLevel  bool     = verboseLevel
	originalResourceBlock string   = resourceBlock
	originalOsArgs        []string = os.Args
)

// GlobalVarsTeardown sets the parsed values of CLI flags to the default
// value at the end of the tests to allow a clean "teardown"
func GlobalVarsTeardown() {
	command = "list"
	fileName = originalFileName
	verboseLevel = originalVerboseLevel
	resourceBlock = originalResourceBlock
	resourceHashMap = map[string]interface{}{}
	os.Args = originalOsArgs
	logger = logrus.New()
}

// FileTeardown removes any file that is created during the unit test
func FileTeardown(fileName string) {
	if _, err := os.Stat(fileName); err == nil {
		os.Remove(fileName)
	}
}

func TestCheckValidExtensionInvalid(t *testing.T) {
	var fileName string = "test.txt"
	var valid bool = CheckValidExtension(fileName)
	if valid {
		t.Errorf("Expected false, got true")
	}
}

func TestCheckValidExtensionValid(t *testing.T) {
	var fileName string = "test.tf"
	var valid bool = CheckValidExtension(fileName)
	if !valid {
		t.Errorf("Expected true, got false")
	}
}

func TestCheckFileExistsExists(t *testing.T) {
	var fileName string = "main_test.go"
	var exists bool = CheckFileExists(fileName)
	if !exists {
		t.Errorf("Expected true, got false")
	}
}

func TestCheckFileExistsDoesNotExists(t *testing.T) {
	var fileName string = "bain_test.go"
	var exists bool = CheckFileExists(fileName)
	if exists {
		t.Errorf("Expected false, got true")
	}
}

func TestDetermineResourceLocalResource(t *testing.T) {
	var resourceLine string = "locals {"

	resourceType, resourceName := DetermineResource(resourceLine)
	if resourceType != "locals" {
		t.Errorf("Exepcted locals, got %v", resourceType)
	}

	if resourceName != "" {
		t.Errorf("Expected no value, got %v", resourceName)
	}
}

func TestDetermineResourceAwsRouteTablePublic(t *testing.T) {
	var resourceLine string = `resource "aws_route_table" "public" {`

	resourceType, resourceName := DetermineResource(resourceLine)
	if resourceType != "resource" {
		t.Errorf("Exepcted resource, got %v", resourceType)
	}

	if resourceName != "aws_route_table.public" {
		t.Errorf("Expected aws_route_table.public, got %v", resourceName)
	}
}

func TestReadFileToLinesValidFile(t *testing.T) {
	content, _ := ReadFileToLines("test/variables.tf")

	if len(content) != 1668 {
		t.Errorf("Expected 1668 lines, got %v", len(content))
	}
}

func TestReadFileToLinesInvalidFile(t *testing.T) {
	_, err := ReadFileToLines("test/does_not_exist.tf")

	if err.Error() != "open test/does_not_exist.tf: no such file or directory" {
		t.Errorf("Expected no such file or directory error, got %v", err.Error())
	}
}

func TestReadFileToLinesLargeInput(t *testing.T) {
	largeFile, _ := os.CreateTemp("", "temp-buffer-overflow.tf")
	defer os.Remove(largeFile.Name())

	largeFile.WriteString(strings.Repeat("A", 1024*1024))
	_, err := ReadFileToLines(largeFile.Name())

	if err.Error() != "bufio.Scanner: token too long" {
		t.Errorf("Expected bufio.Scanner: token too long error, got %v", err.Error())
	}
}

func TestMapResourceLocalsKey(t *testing.T) {
	hashMap := map[string]interface{}{}

	hm := MapResource(hashMap, "locals", "", []string{"1-line-1", "1-line-2"})
	_, exists := hm["locals"]
	if !exists {
		t.Errorf("Expected `locals` key to be created, was not created")
	}
}

func TestMapResourceResourcesKey(t *testing.T) {
	hashMap := map[string]interface{}{}

	hm := MapResource(hashMap, "resources", "aws_vpc.this", []string{"1-line-1", "1-line-2"})
	_, exists := hm["resources"]
	if !exists {
		t.Errorf("Expected `resources` key to be created, was not created")
	}
}

func TestMapResourceResource(t *testing.T) {
	hashMap := map[string]interface{}{
		"locals": [][]string{
			{"1-line-1", "1-line-2"},
			{"2-line-1", "2-line-2"},
		},
		"resource": map[string][]string{
			"aws_route_table.public": {
				"1-line-1",
				"1-line-2",
				"1-line-3",
			},
			"aws_route.public_internet_gateway": {
				"2-line-1",
				"2-line-2",
			},
		},
	}

	hm := MapResource(hashMap, "resource", "aws_vpc.this", []string{
		"3-line-1", "3-line-2",
	})

	lines := hm["resource"].(map[string][]string)["aws_vpc.this"]
	if lines[0] != "3-line-1" {
		t.Errorf("Expected `3-line-1` in aws_vpc.this, got %v", lines[0])
	}
}

func TestMapResourceLocals(t *testing.T) {
	hashMap := map[string]interface{}{
		"locals": [][]string{
			{"1-line-1", "1-line-2"},
			{"2-line-1", "2-line-2"},
		},
		"resource": map[string][]string{
			"aws_route_table.public": {
				"1-line-1",
				"1-line-2",
				"1-line-3",
			},
			"aws_route.public_internet_gateway": {
				"2-line-1",
				"2-line-2",
			},
		},
	}

	hm := MapResource(hashMap, "locals", "", []string{
		"3-line-1", "3-line-2",
	})

	lines := hm["locals"].([][]string)
	if lines[2][0] != "3-line-1" {
		t.Errorf("Expected `3-line-1` in locals, got %v", lines[2][0])
	}
}

func TestRetrieveResourceBlocks(t *testing.T) {
	content, _ := ReadFileToLines("test/main.tf")

	resourceMap := RetrieveResourceBlocks(content)
	if len(resourceMap["resource"].(map[string][]string)["aws_vpc.this"]) != 24 {
		t.Errorf("Expected 23 lines for aws_vpc.this, got %v", len(resourceMap["resource"].(map[string][]string)["aws_vpc.this"]))
	}
}

func TestExtractResourcesToFile(t *testing.T) {
	defer GlobalVarsTeardown()
	defer FileTeardown("test/locals.tf")

	content, _ := ReadFileToLines("test/main.tf")
	resourceMap := RetrieveResourceBlocks(content)
	ExtractResourcesToFile(resourceMap, "locals", "test/main.tf")

	_, err := os.Stat("test/locals.tf")
	if err != nil {
		t.Errorf("Expected file %v to be created, but not found", "test/locals.tf")
	}
}

func TestMainValidFile(t *testing.T) {
	defer GlobalVarsTeardown()
	fileName = "./test/main.tf"

	main()
}

func TestMainProvidedArgs(t *testing.T) {
	defer GlobalVarsTeardown()
	fileName = "./test/main.tf"

	os.Args = append(os.Args, "list")

	main()
}

func TestMainInvalidCommands(t *testing.T) {
	defer GlobalVarsTeardown()
	fileName = "./test/main.tf"

	os.Args = append(os.Args, "invalid")

	main()
}

func TestMainExtractArg(t *testing.T) {
	defer GlobalVarsTeardown()
	fileName = "./test/main.tf"

	os.Args = append(os.Args, "extract")

	main()
}

func TestMainInvalidFile(t *testing.T) {
	defer GlobalVarsTeardown()
	fileName = "./test/does_not_exist.tf"

	main()
}

func TestMainVerboseLevel(t *testing.T) {
	defer GlobalVarsTeardown()
	verboseLevel = true

	main()
	if logger.Level.String() != "debug" {
		t.Errorf("Expected debug log level, got %v", logger.Level.String())
	}
}
