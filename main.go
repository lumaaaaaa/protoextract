package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	// the list of .proto files found
	protoFiles []string
)

func main() {
	// parse arguments
	if len(os.Args) < 2 {
		fmt.Println("[x] Usage: protoextract <decompiled apk dir>")
		os.Exit(1)
	}

	// store start time
	startTime := time.Now()

	// create the output directory
	baseDir := os.Args[1]
	outputDir := baseDir + "_protoextract/proto"
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		fmt.Printf("[x] Error creating output directory: '%s'\n", err)
		os.Exit(1)
	}

	fmt.Printf("[!] Beginning extraction of .proto files from classes in '%s'...\n", baseDir)

	err = findProtoRequestClasses(baseDir)
	if err != nil {
		fmt.Printf("[x] Error finding protobuf classes: '%s'\n", err)
		os.Exit(1)
	}

	fmt.Printf("[!] Found %d protobuf classes. Starting parsing process...\n", len(protoFiles))

	// parse all protobuf classes
	for _, filePath := range protoFiles {
		// parse the .proto file
		classPath, neededImports, protoFields, err := parseProtoFile(filePath)
		if err != nil {
			fmt.Printf("[x] Error parsing .proto file: '%s'\n", err)
			continue
		}

		// create the output directory
		err = os.MkdirAll(fmt.Sprintf("%s/%s", outputDir, classPath[:strings.Index(classPath, "/")]), 0755)
		if err != nil {
			fmt.Printf("[x] Error creating output directory: '%s'\n", err)
			continue
		}

		// create the output .proto file
		outputProtoFile := fmt.Sprintf("%s/%s.proto", outputDir, classPath)
		protoFile, err := os.Create(outputProtoFile)
		if err != nil {
			fmt.Printf("[x] Error creating .proto file: '%s'\n", err)
			continue
		}

		// write the .proto file
		_, err = protoFile.WriteString(generateProtoFileContent(classPath, neededImports, protoFields))
		if err != nil {
			fmt.Printf("[x] Error writing .proto file: '%s'\n", err)
			continue
		}

		fmt.Printf("[!] Successfully generated '%s'\n", outputProtoFile)
	}

	fmt.Printf("[!] Finished generating %d .proto files in %.4fs\n", len(protoFiles), time.Since(startTime).Seconds())
}
