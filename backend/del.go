package main

import (
	"fmt"
	"time"
)

func main() {
	now := time.Now()
	mes := now.Format("01")
	fmt.Println("месяц ", mes)
	var Mon = map[string]string{}
	var mon string
	for i := 0; i < 4; i++ {
		mon = "--" + fmt.Sprint(i) + "--"
		Mon[mon] = mes
	}
	fmt.Println(Mon)
	fmt.Println(Mon["--1--"])
}
