package main

import (
	"context"
	"errors"
	"fmt"
	"main/flow/flow"
)

type Payload struct {
	a int
	b int
}

type I interface {
	AlertTimeout()
}

func Fl() {
	payload := Payload{
		a: 1,
		b: 2,
	}
	a := 5
	c := context.WithValue(context.Background(), flow.PAYLOADKEY, &payload)
	wrapF := flow.WrapFunc(c, func(c context.Context) error {
		handle1(c, a)

		fmt.Println("000000000000000")

		return nil
	})
	wrapF.UseBefore(AlertTimeout)
	wrapF.UseBehind(Sentinel)
	err := wrapF.Handle()
	fmt.Println(err)

}

func main() {
	Fl()

}

func AlertTimeout(c *flow.Context) error {
	pay := c.Value(flow.PAYLOADKEY).(*Payload)
	pay.b = 1000

	fmt.Println("before")

	return nil
}

func Sentinel(c *flow.Context) error {
	fmt.Println("behind")

	return errors.New("ppppppppp")
}

func handle1(c context.Context, a int) int {
	fmt.Println("main1")

	return 67
}

func handle2(a interface{}) {
	fmt.Println("main2")
}
