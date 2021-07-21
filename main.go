package main

import (
	"fmt"

	cm "github.com/easierway/concurrent_map"
)

func main() {
	fmt.Println(cm.CreateConcurrentMap(9))
}
