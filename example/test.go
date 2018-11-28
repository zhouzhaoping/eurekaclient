package main

import (
	"../../eurekaclient"
)

func main(){

	// Service server
	e := eurekaclient.NewEurekaClient([]string{"http://123.45.45.31:1234","http://10.16.58.219:9998","http://10.18.37.71:9998"},"zzp-go-test")
	e.Register("6565","443")

	// Service client
	e.StartUpdateInstance()
	for i:= 0; i <10; i++ {
		e.GetRandomServerAddress()
	}
}
