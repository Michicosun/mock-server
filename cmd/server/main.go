package main

import "mock-server/internal/control"

func main() {
	control.Components.Start()
	defer control.Components.Stop()
	control.Components.Wait()
}
