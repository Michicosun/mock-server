package util

import (
	"fmt"
	"strings"
)

const LOAD_ARGS = `
import json
with open("data.json") as data:
    args = json.load(data)`

const INVOKE_DYN_HANDLE = `
func(**args)`
const INVOKE_ESB = `
func(args["msgs"])
`

func WrapCodeForDynHandle(code string) []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%s", LOAD_ARGS, code, INVOKE_DYN_HANDLE))
}

func UnwrapCodeForDynHandle(code string) string {
	splitted := strings.Split(code, "\n")
	splitted = splitted[1 : len(splitted)-1]
	return strings.Join(splitted, "\n")
}

func WrapCodeForEsb(code string) []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%s", LOAD_ARGS, code, INVOKE_ESB))
}

// Example:
//
//	[][]byte{
//		[]byte('msg1'),
//		[]byte('msg2'),
//		[]byte('msg3'),
//	}
//
// converts to
// {"msgs": ["msg1", "msg2", "msg3"]}
func WrapArgsForEsb(msgs [][]byte) []byte {
	jsonStrings := make([]string, len(msgs))
	for i, msg := range msgs {
		jsonStrings[i] = fmt.Sprintf("\"%s\"", msg)
	}
	wrappedArray := fmt.Sprintf("{\"msgs\":[%s]}\n", strings.Join(jsonStrings, ","))

	return []byte(wrappedArray)
}
