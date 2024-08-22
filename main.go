package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	// the list of protobuf message class files found
	protoMessageFiles []string

	// the list of protobuf enum class files found
	protoEnumFiles []string
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

	fmt.Printf("[!] Beginning extraction of .proto files from message classes in '%s'...\n", baseDir)

	err = findProtoMessageClasses(baseDir)
	if err != nil {
		fmt.Printf("[x] Error finding protobuf message classes: '%s'\n", err)
		os.Exit(1)
	}

	fmt.Printf("[!] Found %d protobuf message classes. Starting parsing process...\n", len(protoMessageFiles))

	// parse all protobuf classes
	for _, filePath := range protoMessageFiles {
		// parse the .proto file
		classPath, neededImports, protoFields, err := parseProtoFile(filePath)
		if err != nil {
			fmt.Printf("[x] Error parsing .proto file: '%s'\n", err)
			continue
		}

		// create the output directory
		err = os.MkdirAll(fmt.Sprintf("%s/%s", outputDir, classPath[:strings.LastIndex(classPath, "/")]), 0755)
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
		_, err = protoFile.WriteString(generateProtoFileContent(classPath, neededImports, protoFields, false))
		if err != nil {
			fmt.Printf("[x] Error writing .proto file: '%s'\n", err)
			continue
		}

		fmt.Printf("[!] Successfully generated message schema '%s'\n", outputProtoFile)
	}

	fmt.Printf("[!] Beginning extraction of .proto files from enum classes in '%s'...\n", baseDir)

	err = findProtoEnumClasses(baseDir)
	if err != nil {
		fmt.Printf("[x] Error finding protobuf enum classes: '%s'\n", err)
		os.Exit(1)
	}

	fmt.Printf("[!] Found %d protobuf enum classes. Starting parsing process...\n", len(protoEnumFiles))

	for _, filePath := range protoEnumFiles {
		// parse the .proto file
		classPath, protoFields, err := parseWireEnum(filePath)
		if err != nil {
			fmt.Printf("[x] Error parsing .proto file: '%s'\n", err)
			continue
		}

		// create the output directory
		err = os.MkdirAll(fmt.Sprintf("%s/%s", outputDir, classPath[:strings.LastIndex(classPath, "/")]), 0755)
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
		_, err = protoFile.WriteString(generateProtoFileContent(classPath, nil, protoFields, true))
		if err != nil {
			fmt.Printf("[x] Error writing .proto file: '%s'\n", err)
			continue
		}

		fmt.Printf("[!] Successfully generated enum schema '%s'\n", outputProtoFile)
	}

	fmt.Printf("[!] Finished generating %d .proto files in %.3fs\n", len(protoMessageFiles)+len(protoEnumFiles), time.Since(startTime).Seconds())
}
