package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func parseProtoFile(filePath string) (classPath string, neededImports []string, protoFields []ProtoField, err error) {
	// read the file
	file, err := os.ReadFile(filePath)
	if err != nil {
		return classPath, neededImports, protoFields, err
	}
	lines := strings.Split(string(file), "\n")

	// find encoder function
	functionIdx := 0
	for i, line := range lines {
		if line == protoEncode {
			functionIdx = i
			break
		}
	}

	// resolve the class path
	classPath = ""
	for classPath == "" {
		if strings.Contains(lines[functionIdx], "check-cast") {
			classPath = strings.Split(lines[functionIdx], ", L")[1]
			classPath = classPath[:len(classPath)-1]
		}
		functionIdx++
	}

	// update functionIdx to the point where the tags get read
	for functionIdx < len(lines) {
		if strings.Contains(lines[functionIdx], "return-void") {
			break
		}
		functionIdx++
	}

	registers := map[string]string{}
	for functionIdx < len(lines) {
		line := lines[functionIdx]
		// we are assigning the tag value
		if strings.Contains(line, "const/") {
			// parse const line
			parts := strings.Split(line, ", ")
			register := parts[0][strings.LastIndex(parts[0], " ")+1:]
			hexInt := parts[1]
			val, err := strconv.ParseInt(hexInt[2:], 16, 64)
			if err != nil {
				return classPath, neededImports, protoFields, err
			}
			registers[register] = strconv.Itoa(int(val))
		}

		// we are assigning the field type
		if strings.Contains(line, "sget-object") {
			// parse sget-object line
			parts := strings.Split(line, ", ")
			register := parts[0][strings.Index(parts[0], "sget-object ")+12:]
			if strings.Contains(parts[1], "->ADAPTER:") {
				// add an import
				importClass := strings.Split(strings.Split(parts[1], "L")[1], ";->")[0]
				neededImports = append(neededImports, importClass)
				classSplit := strings.Split(strings.Split(parts[1], ";->")[0], "/")
				registers[register] = classSplit[len(classSplit)-1]
			} else {
				// generic type
				registers[register] = strings.ToLower(strings.Split(strings.Split(parts[1], ":L")[0], ";->")[1])
			}
		}

		// we are assigning the field name
		if strings.Contains(line, "iget-object") {
			// parse iget-object line
			parts := strings.Split(line, ", ")
			register := parts[0][strings.Index(parts[0], "iget-object ")+12:]
			// set register to hold field name
			registers[register] = strings.Split(strings.Split(parts[2], ":L")[0], ";->")[1]
		}

		// handle edge case where the type is moved to a register
		if strings.Contains(line, "Lcom/squareup/wire/ProtoAdapter;->asRepeated()Lcom/squareup/wire/ProtoAdapter;") {
			// parse invoke-virtual line
			sourceRegister := strings.Split(strings.Split(line, "{")[1], "}")[0]

			// wait for move-result-object
			for !strings.Contains(lines[functionIdx], "move-result-object") {
				functionIdx++
			}

			// parse move-result-object line
			targetRegister := strings.Split(lines[functionIdx], "move-result-object ")[1]

			// relocate value
			registers[targetRegister] = registers[sourceRegister]
		}

		// we are encoding a whole field
		if strings.Contains(line, "Lcom/squareup/wire/ProtoAdapter;->encodeWithTag(Lcom/squareup/wire/ProtoWriter;ILjava/lang/Object;)V") {
			// parse invoke-virtual line
			parts := strings.Split(line, ", ")

			// we only care about the first, third, and fourth arguments
			typeName := registers[parts[0][strings.Index(parts[0], "{")+1:]]
			tagId, _ := strconv.Atoi(registers[parts[2]])
			fieldName := registers[parts[3][:len(parts[3])-1]]

			// create a new proto field
			newField := ProtoField{
				Name:  fieldName,
				Value: tagId,
				Type:  typeName,
			}
			protoFields = append(protoFields, newField)
			if strings.Contains(filePath, "tyc.smali") {
				fmt.Println(classPath, newField)
			}
		}

		functionIdx++
	}

	// return the assigned values
	return classPath, neededImports, protoFields, nil
}
