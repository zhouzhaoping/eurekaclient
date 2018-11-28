# eurekaclient
eureka version = 1

## usage
1. go get github.com/zhouzhaoping/eurekaclient
2. go run example/test.go

## for Service server
1. Register：Register()  
2. Renew：every 30s Heartbeat after Register()   
3. Cancel：handleSigterm() (os.Interrupt or syscall.SIGTERM) after Register() 

## for Service client
1. Fetch Registries：update in StartUpdateInstance(), get address in GetRandomServerAddress()

## TODO
1. cache instance, StartUpdateInstance() update every 30s [11.27]
2. choice register center, in order [11.28]
3. instance need sync [11.28]