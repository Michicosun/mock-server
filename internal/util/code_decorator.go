package util

import "fmt"

const LOAD_ARGS = `
import json
with open("data.json") as data:
    args = json.load(data)`
const INVOKE = `
func(**args)`

func DecorateCodeForArgExtraction(code string) string {
	ret := fmt.Sprintf("%s\n%s\n%s", LOAD_ARGS, code, INVOKE)
	fmt.Println("RET: ", ret)
	return ret
}
