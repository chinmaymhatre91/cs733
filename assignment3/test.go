package main

import (
	"fmt"
	//"github.com/cs733-iitb/log"
	//"time"
)

type abc struct {
	a int
	b string
}

func main() {
	// var t *time.Timer
	// t = time.NewTimer(0)

	// go func() {
	// 	for {
	// 		<-t.C
	// 		fmt.Print("Timer expired -- ")
	// 		fmt.Println(time.Now().UTC())
	// 	}
	// }()

	// fmt.Println(time.Now().UTC())
	// t.Reset(time.Millisecond * 10)
	// t.Reset(time.Millisecond * 5)
	// //t = time.NewTimer(time.Second * 2)
	// time.Sleep(time.Second * 3)

	//t.Stop()

	var aa []abc

	t := abc{}

	aa = Append(aa, t)

	fmt.Println(aa[0].a)

}
