package main

import (
	"fmt"
	"time"
)

func main() {
	now := time.Now().UTC()

	fmt.Printf("untracated:\t%v\n",
		now.Format(time.RFC3339Nano))

	fmt.Printf("nearest 100 us:\t%v\n",
		now.Truncate(100*time.Microsecond).Format(time.RFC3339Nano))

	fmt.Printf("nearest 500 ms:\t%v\n",
		now.Truncate(500*time.Millisecond).Format(time.RFC3339Nano))

	fmt.Printf("nearest 1 sec:\t%v\n",
		now.Truncate(time.Second).Format(time.RFC3339Nano))

	fmt.Printf("nearest 30 min:\t%v\n",
		now.Truncate(30*time.Minute).Format(time.RFC3339Nano))
}
