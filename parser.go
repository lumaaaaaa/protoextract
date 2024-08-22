package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func parseWireEnum(filePath string) (classPath string, protoFields []ProtoField, err error) {
	// read the file
	file, err := os.ReadFile(filePath)
	if err != nil {
		return classPath, protoFields, err
	}
	lines := strings.Split(string(file), "\n")

	// find the clinit() method
	functionIdx := 0
	for i, line := range lines {
		if strings.Contains(line, "clinit()V") {
			functionIdx = i
			break
		}
	}

	// resolve the class path
	classPath = ""
	classPathRegex := regexp.MustCompile(`new-instance .{2}, L([^;]+);`)
	for classPath == "" && functionIdx < len(lines) {
		matches := classPathRegex.FindStringSubmatch(lines[functionIdx])
		if matches != nil {
			classPath = matches[1]
		}
		functionIdx++
	}

	registers := map[string]string{}
	constStringRegex := regexp.MustCompile(`const-string (\w+), "([^"]+)"`)
	constRegex := regexp.MustCompile(`const/\d+ (\w+), (0x[0-9a-fA-F]+)`)
	invokeDirectRegex := regexp.MustCompile(fmt.Sprintf(`invoke-direct\s+\{([^}]*)\},\s*L%s;-><init>\((.*?)\)`, classPath))

	// parse the smali line by line
	for functionIdx < len(lines) {
		line := lines[functionIdx]
		if matches := constStringRegex.FindStringSubmatch(line); matches != nil {
			// we are assigning the field name
			register := matches[1]
			fieldName := matches[2]
			registers[register] = fieldName
		} else if matches := constRegex.FindStringSubmatch(line); matches != nil {
			// we are assigning the field value
			register := matches[1]
			hexInt := matches[2]
			val, err := strconv.ParseInt(hexInt[2:], 16, 64)
			if err != nil {
				return classPath, protoFields, err
			}
			registers[register] = strconv.Itoa(int(val))
		} else if matches := invokeDirectRegex.FindStringSubmatch(line); matches != nil {
			argumentRegisters := strings.Split(matches[1], ", ")
			types := strings.Split(matches[2], ";")

			valueRegisterIdx := -1
			for i, s := range types {
				// parse out the value register index
				switch strings.Index(s, "II") {
				case 0:
					valueRegisterIdx = i
					break
				case -1:
					continue
				default:
					valueRegisterIdx = i + 1
					break
				}
			}

			nameRegisterIdx := -1
			for i, s := range types {
				// parse out the name register index
				if s == "Ljava/lang/String" {
					nameRegisterIdx = i
					break
				}
			}

			// shift nameRegisterIdx if valueRegisterIdx is before it
			if nameRegisterIdx > valueRegisterIdx {
				nameRegisterIdx += 2
			}

			// ensure we have both registers
			if nameRegisterIdx == -1 || valueRegisterIdx == -1 {
				return classPath, protoFields, fmt.Errorf("could not find name or value register")
			}

			// we are encoding a whole field
			fieldName := registers[argumentRegisters[nameRegisterIdx+1]]
			value, _ := strconv.Atoi(registers[argumentRegisters[valueRegisterIdx+1]])

			// create a new proto field
			newField := ProtoField{
				Name:  fieldName,
				Value: value,
			}
			protoFields = append(protoFields, newField)
		} else if strings.Contains(line, "return-void") {
			// terminate
			return classPath, protoFields, nil
		}

		functionIdx++
	}

	// return the enum name and values
	return classPath, protoFields, nil
}

func parseProtoFile(filePath string) (classPath string, neededImports []string, protoFields []ProtoField, err error) {
	// read the file
	file, err := os.ReadFile(filePath)
	if err != nil {
		return classPath, neededImports, protoFields, err
	}
	lines := strings.Split(string(file), "\n")

	// find encoder function
	functionIdx := -1
	for i, line := range lines {
		if line == protoEncode {
			functionIdx = i
			break
		}
	}

	// resolve the class path
	classPath = ""
	classPathRegex := regexp.MustCompile(`check-cast .{2}, L([^;]+);`)
	for classPath == "" && functionIdx < len(lines) {
		matches := classPathRegex.FindStringSubmatch(lines[functionIdx])
		if matches != nil {
			classPath = matches[1]
		}
		functionIdx++
	}

	// update functionIdx to the point where the tags get read
	for functionIdx < len(lines) {
		if strings.Contains(lines[functionIdx], "return-void") {
			functionIdx++
			break
		}
		functionIdx++
	}

	registers := map[string]string{}
	repeated := false
	packed := false
	constRegex := regexp.MustCompile(`const/\d+ (\w+), (0x[0-9a-fA-F]+)`)
	sgetObjectRegex := regexp.MustCompile(`sget-object (\w+), L([^;]+);->(\w+):L[^;]+;`)
	igetObjectRegex := regexp.MustCompile(`iget-object (\w+), \w+, L[^;]+;->(\w+):L[^;]+;`)
	invokeVirtualRegex := regexp.MustCompile(`invoke-virtual {(\w+), \w+, (\w+), (\w+)}, Lcom/squareup/wire/ProtoAdapter;->encodeWithTag\(Lcom/squareup/wire/ProtoWriter;ILjava/lang/Object;\)V`)
	moveResultObjectRegex := regexp.MustCompile(`move-result-object (\w+)`)

	// parse the smali line by line
	for functionIdx < len(lines) {
		line := lines[functionIdx]
		if matches := constRegex.FindStringSubmatch(line); matches != nil {
			// we are assigning the tag value
			register := matches[1]
			hexInt := matches[2]
			val, err := strconv.ParseInt(hexInt[2:], 16, 64)
			if err != nil {
				return classPath, neededImports, protoFields, err
			}
			registers[register] = strconv.Itoa(int(val))
		} else if matches := sgetObjectRegex.FindStringSubmatch(line); matches != nil {
			// we are assigning the field type
			register := matches[1]
			if matches[3] == "ADAPTER" {
				// add an import
				importClass := matches[2]
				neededImports = append(neededImports, importClass)
				classSplit := strings.Split(importClass, "/")
				registers[register] = classSplit[len(classSplit)-1]
			} else {
				// generic type
				registers[register] = strings.ToLower(matches[3])
			}
		} else if matches := igetObjectRegex.FindStringSubmatch(line); matches != nil {
			// we are assigning the field name
			register := matches[1]
			// set register to hold field name
			registers[register] = matches[2]
		} else if strings.Contains(line, "Lcom/squareup/wire/ProtoAdapter;->asRepeated()Lcom/squareup/wire/ProtoAdapter;") {
			// set repeated to true
			repeated = true

			// parse invoke-virtual line
			sourceRegister := strings.Split(strings.Split(line, "{")[1], "}")[0]

			// wait for move-result-object
			for !strings.Contains(lines[functionIdx], "move-result-object") {
				functionIdx++
			}

			// parse move-result-object line
			matches := moveResultObjectRegex.FindStringSubmatch(lines[functionIdx])
			targetRegister := matches[1]

			// relocate value
			registers[targetRegister] = registers[sourceRegister]
		} else if strings.Contains(line, "Lcom/squareup/wire/ProtoAdapter;->asPacked()Lcom/squareup/wire/ProtoAdapter;") {
			// set packed to true
			packed = true

			// parse invoke-virtual line
			sourceRegister := strings.Split(strings.Split(line, "{")[1], "}")[0]

			// wait for move-result-object
			for !strings.Contains(lines[functionIdx], "move-result-object") {
				functionIdx++
			}

			// parse move-result-object line
			matches := moveResultObjectRegex.FindStringSubmatch(lines[functionIdx])
			targetRegister := matches[1]

			// relocate value
			registers[targetRegister] = registers[sourceRegister]
		} else if matches := invokeVirtualRegex.FindStringSubmatch(line); matches != nil {
			// we are encoding a whole field
			// parse invoke-virtual line
			typeName := registers[matches[1]]
			tagId, _ := strconv.Atoi(registers[matches[2]])
			fieldName := registers[matches[3]]

			// create a new proto field
			newField := ProtoField{
				Name:     fieldName,
				Value:    tagId,
				Type:     typeName,
				Repeated: repeated,
				Packed:   packed,
			}
			protoFields = append(protoFields, newField)

			// reset repeated and packed
			repeated = false
			packed = false
			if strings.Contains(filePath, "tyc.smali") {
				fmt.Println(classPath, newField)
			}
		} else if strings.Contains(line, "return-void") {
			// terminate
			return classPath, neededImports, protoFields, nil
		}

		functionIdx++
	}

	// return the assigned values
	return classPath, neededImports, protoFields, nil
}
