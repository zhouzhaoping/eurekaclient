package eurekaclient

import (
	"github.com/twinj/uuid"
	"strings"
	"fmt"
	"time"
	"log"
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"syscall"
	"math/rand"
	"strconv"
	"sync"
)

var regHostName = `dmo-${appName}-${uuid}`
var regInstanceId = `${hostName}:${appName}:${port}`

var regTpl =`{
  "instance": {
	"instanceId":"${instanceId}",
    "hostName":"${hostName}",
    "app":"${appName}",
	"ipAddr":"${ipAddress}",
    "vipAddress":"${appName}",
	"secureVipAddress":"${appName}",
    "status":"UP",
    "port": {
      "$":${port},
      "@enabled": true
    },
    "securePort": {
      "$":${securePort},
      "@enabled": false
    },
    "homePageUrl" : "http://${hostName}:${port}/",
    "statusPageUrl": "http://${hostName}:${port}/info",
    "healthCheckUrl": "http://${hostName}:${port}/health",
    "dataCenterInfo" : {
      "@class":"com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
      "name": "MyOwn"
    },
    "metadata": {
      "management.port": ${port}
    }
  }
}`


type EurekaClient struct {
	urlCur int
	discoveryServerUrls []string
	appName string

	// for Service server
	instanceId string

	// for Service client
	mu      sync.Mutex
	instances []EurekaInstance
}

func NewEurekaClient (eurekaUrls []string, appName string )*EurekaClient{
	return &EurekaClient{
		urlCur : 0,
		discoveryServerUrls: eurekaUrls,
		appName : appName,
		instances:nil,
	}
}

/**
 * Tools for update
 */
func (e *EurekaClient) StartUpdateInstance() { //todo sync
	e.GetServiceInstances()
	go func(){
		time.Sleep(30*time.Second)
		fmt.Println("Update instance...")
		e.GetServiceInstances()
	}()
}

/**
 * Register the application at the default eurekaUrl.
 */
func (e *EurekaClient) Register(port string, securePort string) {

	hostName, err := os.Hostname()
	if err != nil {
		hostName := string(regHostName)
		hostName = strings.Replace(hostName, "${appName}", e.appName, -1)
		hostName = strings.Replace(hostName, "${uuid}",getUUID(),-1)
	}

	e.instanceId = string(regInstanceId)
	e.instanceId = strings.Replace(e.instanceId, "${hostName}",hostName, -1)
	e.instanceId = strings.Replace(e.instanceId, "${appName}", e.appName, -1)
	e.instanceId = strings.Replace(e.instanceId, "${port}", port, -1)

	tpl := string(regTpl)
	tpl = strings.Replace(tpl, "${instanceId}",e.instanceId, -1)
	tpl = strings.Replace(tpl, "${hostName}",hostName, -1)
	tpl = strings.Replace(tpl, "${ipAddress}", getLocalIP(), -1)
	tpl = strings.Replace(tpl, "${port}", port, -1)
	tpl = strings.Replace(tpl, "${securePort}", securePort, -1)
	tpl = strings.Replace(tpl, "${appName}", e.appName, -1)

	// Register.
	registerAction := HttpAction{
		Url:         "${discoveryServerUrl}" + "/eureka/apps/" + e.appName,
		Method:      "POST",
		ContentType: "application/json;charset=UTF-8",
		Body:        tpl,
	}

	var result bool
	for _,url := range e.discoveryServerUrls {
		registerAction.Url = url + "/eureka/apps/" + e.appName
		result = doHttpRequest(registerAction)
		if result {
			fmt.Println("Registration OK")
			e.handleSigterm()
			go e.startHeartbeat()
			break
		} else {
			fmt.Println("Registration attempt of " + e.appName + " failed...")
			time.Sleep(time.Second * 5)
		}
	}

}

/**
 * Given the supplied appName, this func queries the Eureka API for instances of the appName and returns
 * them as a EurekaApplication struct.
 */
func (e *EurekaClient) GetServiceInstances() (error) {

	var m EurekaServiceResponse
	fmt.Println("Querying eureka for instances of " + e.appName + " at: " + "${discoveryServerUrl}" + "/eureka/apps/" + e.appName)
	queryAction := HttpAction{
		Url:         "${discoveryServerUrl}" + "/eureka/apps/" + e.appName,
		Method:      "GET",
		Accept:      "application/json;charset=UTF-8",
		ContentType: "application/json;charset=UTF-8",
	}
	var err error
	var bytes []byte
	for _,url := range e.discoveryServerUrls {
		queryAction.Url = url + "/eureka/apps/" + e.appName
		log.Println("Doing queryAction using URL: " + queryAction.Url)
		bytes, err = executeQuery(queryAction)
		if err != nil {
			continue
		} else {
			fmt.Println("Got instances response from Eureka:\n" + string(bytes))
			err = json.Unmarshal(bytes, &m)
			if err != nil {
				fmt.Println("Problem parsing JSON response from Eureka: " + err.Error())
				continue
			}
			e.mu.Lock()
			e.instances = m.Application.Instance
			e.mu.Unlock()
			return err
		}
	}
	e.mu.Lock()
	e.instances = nil
	e.mu.Unlock()
	return err
}

// unuse
func (e *EurekaClient) GetServices() ([]EurekaApplication, error) {
	var m EurekaApplicationsRootResponse
	fmt.Println("Querying eureka for services at: " + "${discoveryServerUrl}"+ "/eureka/apps")
	queryAction := HttpAction{
		Url:        "${discoveryServerUrl}" + "/eureka/apps",
		Method:      "GET",
		Accept:      "application/json;charset=UTF-8",
		ContentType: "application/json;charset=UTF-8",
	}

	var err error
	var bytes []byte
	for _,url := range e.discoveryServerUrls {
		queryAction.Url = url + "/eureka/apps"
		log.Println("Doing queryAction using URL: " + queryAction.Url)
		bytes, err = executeQuery(queryAction)
		if err != nil {
			continue
		} else {
			fmt.Println("Got services response from Eureka:\n" + string(bytes))
			err = json.Unmarshal(bytes, &m)
			if err != nil {
				fmt.Println("Problem parsing JSON response from Eureka: " + err.Error())
				continue
			}
			return m.Resp.Applications, nil
		}
	}
	return nil, err
}

func (e *EurekaClient)GetRandomServerAddress() string {
	rand.Seed(time.Now().Unix())

	e.mu.Lock()
	if e.instances == nil {
		e.mu.Unlock()
		fmt.Println("can not get address")
		return ""
	}
	i := rand.Intn(len(e.instances))
	for _, ins := range(e.instances){
		address := ins.IpAddr + ":" + strconv.Itoa(ins.Port.Port)
		fmt.Println(address)
	}
	address := e.instances[i].IpAddr + ":" + strconv.Itoa(e.instances[i].Port.Port)
	fmt.Println("choice address:" + address + ", hostname:" +  e.instances[i].HostName)
	e.mu.Unlock()

	return address
}

// Start as goroutine, will loop indefinitely until application exits.
func (e *EurekaClient)startHeartbeat() {
	for {
		time.Sleep(time.Second * 30)
		e.heartbeat()
	}
}

func (e *EurekaClient)heartbeat() {

	heartbeatAction := HttpAction{
		Url:         "${discoveryServerUrl}" + "/eureka/apps/" + e.appName + "/" + e.instanceId,
		Method:      "PUT",
		ContentType: "application/json;charset=UTF-8",
	}

	var result bool
	for _,url := range e.discoveryServerUrls {
		heartbeatAction.Url = url + "/eureka/apps/" + e.appName + "/" + e.instanceId
		result = doHttpRequest(heartbeatAction)
		if result {
			fmt.Println("Issuing heartbeat to " + heartbeatAction.Url)
			break
		} else {
			fmt.Println("Retry heartbeat...")
		}
	}
}

func (e *EurekaClient)deregister() {

	fmt.Println("Trying to deregister application " + e.appName + "...")
	// Deregister
	deregisterAction := HttpAction{
		Url:         "${discoveryServerUrl}" + "/eureka/apps/" + e.appName + "/" + e.instanceId,
		ContentType: "application/json;charset=UTF-8",
		Method:      "DELETE",
	}
	var result bool
	for _,url := range e.discoveryServerUrls {
		deregisterAction.Url = url + "/eureka/apps/" + e.appName + "/" + e.instanceId
		result = doHttpRequest(deregisterAction)
		if result {
			fmt.Println("Deregistered application " + e.appName + ", exiting. Check Eureka...")
			break
		} else {
			fmt.Println("Retry deregister...")
		}
	}
}

// get Intranet Ip
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	/*for _, address := range addrs {
		fmt.Println("get ip:",address.String(), address.Network())
	}*/
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it

		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	panic("Unable to determine local IP address (non loopback). Exiting.")
}

func getUUID() string {
	return uuid.NewV4().String()
}

func (e *EurekaClient)handleSigterm() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		e.deregister()
		os.Exit(1)
	}()
}
