//go:build linux

package dpdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/influxdata/telegraf/filter"
)

func commandWithParams(command string, params string) string {
	if params != "" {
		return command + "," + params
	}
	return command
}

func stripParams(command string) string {
	index := strings.IndexRune(command, ',')
	if index == -1 {
		return command
	}
	return command[:index]
}

// Since DPDK is an open-source project, developers can use their own format of params
// so it could "/command,1,3,5,123" or "/command,userId=1, count=1234".
// To avoid issues with different formats of params, all params are returned as single string
func getParams(command string) string {
	index := strings.IndexRune(command, ',')
	if index == -1 {
		return ""
	}
	return command[index+1:]
}

// Checks if the provided filePath contains in-memory socket
func isInMemorySocketPath(filePath, socketPath string) bool {
	if filePath == socketPath {
		return true
	}

	socketPathPrefix := fmt.Sprintf("%s:", socketPath)
	if strings.HasPrefix(filePath, socketPathPrefix) {
		suffix := filePath[len(socketPathPrefix):]
		if number, err := strconv.Atoi(suffix); err == nil {
			if number > 0 {
				return true
			}
		}
	}

	return false
}

// Checks if provided path points to socket
func isSocket(path string) error {
	pathInfo, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("provided path does not exist: %q", path)
	}

	if err != nil {
		return fmt.Errorf("cannot get system information of %q file: %w", path, err)
	}

	if pathInfo.Mode()&os.ModeSocket != os.ModeSocket {
		return fmt.Errorf("provided path does not point to a socket file: %q", path)
	}

	return nil
}

// Converts JSON array containing devices identifiers from DPDK response to string slice
func jsonToArray(input []byte, command string) ([]string, error) {
	if len(input) == 0 {
		return nil, errors.New("got empty object instead of json")
	}

	var rawMessage map[string]json.RawMessage
	err := json.Unmarshal(input, &rawMessage)
	if err != nil {
		return nil, err
	}

	var intArray []int64
	err = json.Unmarshal(rawMessage[command], &intArray)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal json response: %w", err)
	}

	stringArray := make([]string, 0, len(intArray))
	for _, value := range intArray {
		stringArray = append(stringArray, strconv.FormatInt(value, 10))
	}

	return stringArray, nil
}

func removeSubset(elements []string, excludedFilter filter.Filter) []string {
	if excludedFilter == nil {
		return elements
	}

	var result []string
	for _, element := range elements {
		if !excludedFilter.Match(element) {
			result = append(result, element)
		}
	}

	return result
}

func uniqueValues(values []string) []string {
	in := make(map[string]bool)
	result := make([]string, 0, len(values))

	for _, value := range values {
		if !in[value] {
			in[value] = true
			result = append(result, value)
		}
	}
	return result
}

func isEmpty(value interface{}) bool {
	return value == nil || (reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil())
}
