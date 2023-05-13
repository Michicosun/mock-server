package util

import (
	"fmt"
	"strings"

	zlog "github.com/rs/zerolog/log"
)

const LOAD_ARGS = `
import json
with open("data.json") as data:
    args = json.load(data)`

const INVOKE_DYN_HANDLE = `
print(func(**args))`
const INVOKE_ESB = `
func(args["msgs"])
`

func WrapCodeForDynHandle(code string) []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%s", LOAD_ARGS, code, INVOKE_DYN_HANDLE))
}

func UnwrapCodeForDynHandle(code string) string {
	splitted := strings.Split(code, "\n")
	splitted = splitted[4 : len(splitted)-1]
	return strings.Join(splitted, "\n")
}

func WrapCodeForEsb(code string) []byte {
	return []byte(fmt.Sprintf("%s\n%s\n%s", LOAD_ARGS, code, INVOKE_ESB))
}

func UnwrapCodeForEsb(code string) string {
	splitted := strings.Split(code, "\n")
	splitted = splitted[4 : len(splitted)-1]
	return strings.Join(splitted, "\n")
}

// Example:
//
//		headers: json.Marshal(map[string][]string{
//		 "a":    {"b", "c", "d"},
//	  "nice": {"1"},
//		})
//
//		body: json.Marshal(map[string]interface{}{
//			"A": 7,
//			"B": "9",
//		})
//
// converts to
//
//	{
//		"headers": {
//			"a": ["b", "c", "d"],
//			"nice": ["1"]
//		},
//		"body": {
//			"A": 7,
//			"B": "9"
//		}
//	}
func WrapArgsForDynHandle(headers []byte, body []byte) []byte {
	if len(headers) == 0 {
		headers = []byte(`{}`)
	}
	if len(body) == 0 {
		body = []byte(`{}`)
	}
	wrapped := fmt.Sprintf(`{"headers": %s, "body": %s}`, headers, body)
	zlog.Debug().Str("wrapped args", wrapped).Msg("After wrap")
	return []byte(wrapped)
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
