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

var discoveryServerUrl = "http://10.16.58.219:9998"// TODO:to be setting

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

/**
 * Registers this application at the Eureka server at @eurekaUrl as @appName running on port(s) @port and/or @securePort.
 */
func RegisterAt(eurekaUrl string, appName string, port string, securePort string) {
	discoveryServerUrl = eurekaUrl
	Register(appName, port, securePort)
}

/**
  Register the application at the default eurekaUrl.
*/
func Register(appName string, port string, securePort string) {

	hostName, err := os.Hostname()
	if err != nil {
		hostName := string(regHostName)
		hostName = strings.Replace(hostName, "${appName}", appName, -1)
		hostName = strings.Replace(hostName, "${uuid}",getUUID(),-1)
	}

	instanceId := string(regInstanceId)
	instanceId = strings.Replace(instanceId, "${hostName}",hostName, -1)
	instanceId = strings.Replace(instanceId, "${appName}", appName, -1)
	instanceId = strings.Replace(instanceId, "${port}", port, -1)

	tpl := string(regTpl)
	tpl = strings.Replace(tpl, "${instanceId}",instanceId, -1)
	tpl = strings.Replace(tpl, "${hostName}",hostName, -1)
	tpl = strings.Replace(tpl, "${ipAddress}", getLocalIP(), -1)
	tpl = strings.Replace(tpl, "${port}", port, -1)
	tpl = strings.Replace(tpl, "${securePort}", securePort, -1)
	tpl = strings.Replace(tpl, "${appName}", appName, -1)

	// Register.
	registerAction := HttpAction{
		Url:         discoveryServerUrl + "/eureka/apps/" + appName,
		Method:      "POST",
		ContentType: "application/json;charset=UTF-8",
		Body:        tpl,
	}

	var result bool
	for {
		result = doHttpRequest(registerAction)
		if result {
			fmt.Println("Registration OK")
			handleSigterm(appName,instanceId)
			go startHeartbeat(appName,instanceId)
			break
		} else {
			fmt.Println("Registration attempt of " + appName + " failed...")
			time.Sleep(time.Second * 5)
		}
	}

}

/**
 * Given the supplied appName, this func queries the Eureka API for instances of the appName and returns
 * them as a EurekaApplication struct.
 */
func GetServiceInstances(appName string) ([]EurekaInstance, error) {
	var m EurekaServiceResponse
	fmt.Println("Querying eureka for instances of " + appName + " at: " + discoveryServerUrl + "/eureka/apps/" + appName)
	queryAction := HttpAction{
		Url:         discoveryServerUrl + "/eureka/apps/" + appName,
		Method:      "GET",
		Accept:      "application/json;charset=UTF-8",
		ContentType: "application/json;charset=UTF-8",
	}
	log.Println("Doing queryAction using URL: " + queryAction.Url)
	bytes, err := executeQuery(queryAction)
	if err != nil {
		return nil, err
	} else {
		fmt.Println("Got instances response from Eureka:\n" + string(bytes))
		err := json.Unmarshal(bytes, &m)
		if err != nil {
			fmt.Println("Problem parsing JSON response from Eureka: " + err.Error())
			return nil, err
		}
		return m.Application.Instance, nil
	}
}

// Experimental, untested.
func GetServices() ([]EurekaApplication, error) {
	var m EurekaApplicationsRootResponse
	fmt.Println("Querying eureka for services at: " + discoveryServerUrl + "/eureka/apps")
	queryAction := HttpAction{
		Url:         discoveryServerUrl + "/eureka/apps",
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

func GetRandomServerAddress(url string, appName string) string{
	discoveryServerUrl = url
	eurekaInstances, err := GetServiceInstances(appName)
	if err != nil {
		println(err.Error())
		return ""
	}

	rand.Seed(time.Now().Unix())
	i := rand.Intn(len(eurekaInstances))
	for _, ins := range(eurekaInstances){
		address := ins.IpAddr + ":" + strconv.Itoa(ins.Port.Port)
		fmt.Println(address)
	}
	address := eurekaInstances[i].IpAddr + ":" + strconv.Itoa(eurekaInstances[i].Port.Port)
	fmt.Println("choice address:" + address + ", hostname:" +  eurekaInstances[i].HostName)
	return address
}

// Start as goroutine, will loop indefinitely until application exits.
func startHeartbeat(appName string, instanceId string) {
	for {
		time.Sleep(time.Second * 30)
		heartbeat(appName, instanceId)
	}
}

func heartbeat(appName string, instanceId string) {
	heartbeatAction := HttpAction{
		Url:         discoveryServerUrl + "/eureka/apps/" + appName + "/" + instanceId,
		Method:      "PUT",
		ContentType: "application/json;charset=UTF-8",
	}
	fmt.Println("Issuing heartbeat to " + heartbeatAction.Url)
	doHttpRequest(heartbeatAction)
}

func deregister(appName string, instanceId string) {
	fmt.Println("Trying to deregister application " + appName + "...")
	// Deregister
	deregisterAction := HttpAction{
		Url:         discoveryServerUrl + "/eureka/apps/" + appName + "/" + instanceId,
		ContentType: "application/json;charset=UTF-8",
		Method:      "DELETE",
	}
	doHttpRequest(deregisterAction)
	fmt.Println("Deregistered application " + appName + ", exiting. Check Eureka...")
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

func handleSigterm(appName string, instanceId string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		deregister(appName, instanceId)
		os.Exit(1)
	}()
}
