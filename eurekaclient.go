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
	discoveryServerUrl string // = "http://10.16.58.219:9998"// TODO:to be list
	appName string

	// for server
	instanceId string

	// for client
	instances []EurekaInstance // todo update
}

func NewEurekaClient (eurekaUrl string, appName string )*EurekaClient{
	return &EurekaClient{discoveryServerUrl: eurekaUrl, appName : appName, instances:nil}
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
		Url:         e.discoveryServerUrl + "/eureka/apps/" + e.appName,
		Method:      "POST",
		ContentType: "application/json;charset=UTF-8",
		Body:        tpl,
	}

	var result bool
	for {
		result = doHttpRequest(registerAction)
		if result {
			fmt.Println("Registration OK")
			e.handleSigterm()
			go e.startHeartbeat()
			break
		} else {
			fmt.Println("Registration attempt of " + e.appName + " failed...")//todo not always retry, change other url
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
	fmt.Println("Querying eureka for instances of " + e.appName + " at: " + e.discoveryServerUrl + "/eureka/apps/" + e.appName)
	queryAction := HttpAction{
		Url:         e.discoveryServerUrl + "/eureka/apps/" + e.appName,
		Method:      "GET",
		Accept:      "application/json;charset=UTF-8",
		ContentType: "application/json;charset=UTF-8",
	}
	log.Println("Doing queryAction using URL: " + queryAction.Url)
	bytes, err := executeQuery(queryAction)
	if err != nil {
		e.instances = nil
		return err
	} else {
		fmt.Println("Got instances response from Eureka:\n" + string(bytes))
		err := json.Unmarshal(bytes, &m)
		if err != nil {
			fmt.Println("Problem parsing JSON response from Eureka: " + err.Error())
			e.instances = nil
			return err
		}
		e.instances = m.Application.Instance
		return err
	}
}

// unuse
func (e *EurekaClient) GetServices() ([]EurekaApplication, error) {
	var m EurekaApplicationsRootResponse
	fmt.Println("Querying eureka for services at: " + e.discoveryServerUrl + "/eureka/apps")
	queryAction := HttpAction{
		Url:         e.discoveryServerUrl + "/eureka/apps",
		Method:      "GET",
		Accept:      "application/json;charset=UTF-8",
		ContentType: "application/json;charset=UTF-8",
	}
	log.Println("Doing queryAction using URL: " + queryAction.Url)
	bytes, err := executeQuery(queryAction)
	if err != nil {
		return nil, err
	} else {
		fmt.Println("Got services response from Eureka:\n" + string(bytes))
		err := json.Unmarshal(bytes, &m)
		if err != nil {
			fmt.Println("Problem parsing JSON response from Eureka: " + err.Error())
			return nil, err
		}
		return m.Resp.Applications, nil
	}
}

func (e *EurekaClient)GetRandomServerAddress() string{
	e.GetServiceInstances() // todo update

	rand.Seed(time.Now().Unix())
	i := rand.Intn(len(e.instances))
	for _, ins := range(e.instances){
		address := ins.IpAddr + ":" + strconv.Itoa(ins.Port.Port)
		fmt.Println(address)
	}
	address := e.instances[i].IpAddr + ":" + strconv.Itoa(e.instances[i].Port.Port)
	fmt.Println("choice address:" + address + ", hostname:" +  e.instances[i].HostName)
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
		Url:         e.discoveryServerUrl + "/eureka/apps/" + e.appName + "/" + e.instanceId,
		Method:      "PUT",
		ContentType: "application/json;charset=UTF-8",
	}
	fmt.Println("Issuing heartbeat to " + heartbeatAction.Url)
	doHttpRequest(heartbeatAction)
}

func (e *EurekaClient)deregister() {
	fmt.Println("Trying to deregister application " + e.appName + "...")
	// Deregister
	deregisterAction := HttpAction{
		Url:         e.discoveryServerUrl + "/eureka/apps/" + e.appName + "/" + e.instanceId,
		ContentType: "application/json;charset=UTF-8",
		Method:      "DELETE",
	}
	doHttpRequest(deregisterAction)
	fmt.Println("Deregistered application " + e.appName + ", exiting. Check Eureka...")
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
