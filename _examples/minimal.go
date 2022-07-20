package main

import (
	"fmt"
	"kittla"
)

func main() {
	k := kittla.New()
	res, _, _ := k.Execute("set sum 0; set i 0; while {$i < 50} {inc i; set sum [eval $i+$sum]};")
	fmt.Println(string(res))
}
