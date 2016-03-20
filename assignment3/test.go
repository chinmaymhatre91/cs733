package main

import (
	"errors"
	"fmt"
	//"strconv"
	//"github.com/cs733-iitb/log"
	//"time"
)

type abc struct {
	a int
	b string
}

func temp() (abc, error) {
	var t abc
	t.a = 100
	t.b = "100"

	return t, errors.New("errrrr")
}

type big_struct struct {
	arr []abc
	id  int
	str string
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

	//var ax []abc

	zz, err := temp()
	if err == nil {
		fmt.Println("Error Nil")
	}
	fmt.Println(err)

	fmt.Println(zz.a)
	fmt.Println(zz.b)

	var ch chan int

	ch = make(chan int)

	ch = nil

	ch = make(chan int)

	go func() {
		ch <- 10
	}()

	fmt.Println(<-ch)

	var xx, yy big_struct

	xx = big_struct{arr: []abc{{a: 10, b: "10"}, {a: 11, b: "11"}}, id: 1, str: "1111"}
	yy = xx

	fmt.Println(yy.arr[1].b)

}
