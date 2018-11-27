package main

import (
	"../../eurekaclient"
	"time"
)

func main(){

	// Service server
	e := eurekaclient.NewEurekaClient("http://10.16.58.219:9998","zzp-go-test")
	e.Register("6565","443")

	// Service client
	for {
		e.GetRandomServerAddress()
		time.Sleep(time.Minute)
	}
}
