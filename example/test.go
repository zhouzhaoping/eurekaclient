package main

import (
	"../../eurekaclient"
	"time"
)

func main(){

	// Service server
	e := eurekaclient.NewEurekaClient([]string{"http://10.16.58.219:9998","http://10.18.37.71:9998"},"zzp-go-test")
	e.Register("6565","443")

	// Service client
	e.StartUpdateInstance()
	for {
		e.GetRandomServerAddress()
		time.Sleep(time.Minute)
	}
}
