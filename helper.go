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
	protoHeader = "/* Generated using https://github.com/lumaaaaaa/protoextract\n * Definitions may contain inaccuracies. */\nsyntax = \"proto3\";"
)

func generateProtoFileContent(classPath string, neededImports []string, protoFields []ProtoField) string {
	// initialize the content with the proto3 header
	content := fmt.Sprintf("%s\n\n", protoHeader)

	// remove duplicates from the needed imports
	neededImports = removeDuplicates(neededImports)

	// add the needed imports
	for _, neededImport := range neededImports {
		content += fmt.Sprintf("import \"%s.proto\";\n", neededImport)
	}
	if len(neededImports) > 0 {
		content += "\n"
	}

	// add the message definition
	content += fmt.Sprintf("message %s {\n", classPath[strings.LastIndex(classPath, "/")+1:])

	// add the fields
	for _, protoField := range protoFields {
		// TODO: handle other tags
		content += fmt.Sprintf("  %s %s = %d;\n", protoField.Type, protoField.Name, protoField.Value)
	}

	// close the message definition
	content += "}\n"

	// return the generated content
	return content
}

func removeDuplicates(imports []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range imports {
		if encountered[imports[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[imports[v]] = true
			// Append to result slice.
			result = append(result, imports[v])
		}
	}
	// Return the new slice.
	return result
}

// findProtoRequestClasses walks the directory tree and identifies .smali files that contain the protobuf decode method
func findProtoRequestClasses(relativePath string) error {
	err := filepath.WalkDir(relativePath, visit)
	if err != nil {
		return err
	}

	return nil
}

// visit identifies .smali files that contain the square/wire protobuf field declaration
func visit(path string, di fs.DirEntry, err error) error {
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
			protoFiles = append(protoFiles, path)
		}
	}
	return nil
}
