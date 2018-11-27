# eurekaclient
eureka version = 1
## for Service server
1. Register：Register()  
2. Renew：every 30s Heartbeat after Register()   
3. Cancel：handleSigterm() (os.Interrupt or syscall.SIGTERM) after Register() 

## for Service client
1. Fetch Registries：update in GetServiceInstances(), get address in GetRandomServerAddress()

## TODO
1. cache instance, GetServiceInstances() delta update every 30s
2. choice register center