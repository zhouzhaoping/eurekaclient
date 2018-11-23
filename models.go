package eurekaclient

/**
  Defines a graph of structs that conforms to a part of the return type of the Eureka "get instances for appId", e.g:
  GET /eureka/v2/apps/appID
  The root is the EurekaServiceResponse which contains a single EurekaApplication, which in its turn contains an array
  of EurekaInstance instances.
*/

// Response for /eureka/apps/{appName}
type EurekaServiceResponse struct {
	Application EurekaApplication `json:"application"`
}

// Response for /eureka/apps
type EurekaApplicationsRootResponse struct {
	Resp EurekaApplicationsResponse `json:"applications"`
}

type EurekaApplicationsResponse struct {
	Version      string              `json:"versions__delta"`
	AppsHashcode string              `json:"versions__delta"`
	Applications []EurekaApplication `json:"application"`
}

type EurekaApplication struct {
	Name     string           `json:"name"`
	Instance []EurekaInstance `json:"instance"`
}

type EurekaInstance struct {
	InstanceId string `json:"instanceId"`
	HostName string     `json:"hostName"`
	Port     EurekaPort `json:"port"`
	IpAddr	string 		`json:"ipAddr"`
}

type EurekaPort struct {
	Port int `json:"$"`
}