package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// the magic string that identifies a protobuf class
	protoEncode = ".method public final encode(Lcom/squareup/wire/ProtoWriter;Ljava/lang/Object;)V"
	protoEnum   = ".implements Lcom/squareup/wire/WireEnum;"
	protoHeader = "/* Generated using https://github.com/lumaaaaaa/protoextract\n * Definitions may contain inaccuracies. */\nsyntax = \"proto2\";"
)

func generateProtoFileContent(classPath string, neededImports []string, protoFields []ProtoField, enum bool) string {
	// initialize the content with the proto3 header
	content := fmt.Sprintf("%s\n\n", protoHeader)

	if !enum {
		// remove duplicates from the needed imports
		neededImports = removeDuplicates(neededImports)

		// add the needed imports
		for _, neededImport := range neededImports {
			content += fmt.Sprintf("import \"%s.proto\";\n", neededImport)
		}

		// add a newline if there are imports
		if len(neededImports) > 0 {
			content += "\n"
		}

		// add the message definition
		content += fmt.Sprintf("message %s {\n", classPath[strings.LastIndex(classPath, "/")+1:])

		// add the fields
		for _, protoField := range protoFields {
			if protoField.Repeated {
				content += fmt.Sprintf("  %s %s %s = %d;\n", "repeated", protoField.Type, protoField.Name, protoField.Value)
			} else if protoField.Packed {
				content += fmt.Sprintf("  %s %s %s = %d [packed=true];\n", "repeated", protoField.Type, protoField.Name, protoField.Value)
			} else {
				content += fmt.Sprintf("  %s %s %s = %d;\n", "optional", protoField.Type, protoField.Name, protoField.Value)
			}
		}

		// close the message definition
		content += "}\n"
	} else {
		// add the enum definition
		content += fmt.Sprintf("enum %s {\n", classPath[strings.LastIndex(classPath, "/")+1:])

		// add the fields
		for _, protoField := range protoFields {
			content += fmt.Sprintf("  %s = %d;\n", protoField.Name, protoField.Value)
		}

		// close the enum definition
		content += "}\n"
	}

	// return the generated content
	return content
}

func removeDuplicates(imports []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range imports {
		if encountered[imports[v]] == false {
			// record this element as an encountered element
			encountered[imports[v]] = true
			// append to result slice
			result = append(result, imports[v])
		}
	}

	// return the new slice.
	return result
}

// findProtoMessageClasses walks the directory tree and identifies .smali files that contain the protobuf encode method
func findProtoMessageClasses(relativePath string) error {
	err := filepath.WalkDir(relativePath, checkEncode)
	if err != nil {
		return err
	}

	return nil
}

// findProtoEnumClasses walks the directory tree and identifies .smali files that implement WireEnum
func findProtoEnumClasses(relativePath string) error {
	err := filepath.WalkDir(relativePath, checkEnum)
	if err != nil {
		return err
	}

	return nil
}

// checkEncode identifies .smali files that contain the square/wire protobuf encode declaration
func checkEncode(path string, di fs.DirEntry, _ error) error {
	// check if the file is a .smali file
	if strings.Contains(di.Name(), ".smali") && !di.IsDir() {
		// read the file
		file, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("[x] Error reading file: %s\n", err)
			return err
		}

		// check if the file contains the protobuf decode method
		if strings.Contains(string(file), protoEncode) {
			protoMessageFiles = append(protoMessageFiles, path)
		}
	}
	return nil
}

// checkEnum identifies .smali files that contain the square/wire protobuf encode declaration
func checkEnum(path string, di fs.DirEntry, _ error) error {
	// check if the file is a .smali file
	if strings.Contains(di.Name(), ".smali") && !di.IsDir() {
		// read the file
		file, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("[x] Error reading file: %s\n", err)
			return err
		}

		// check if the file contains the protobuf decode method
		if strings.Contains(string(file), protoEnum) {
			protoEnumFiles = append(protoEnumFiles, path)
		}
	}
	return nil
}
