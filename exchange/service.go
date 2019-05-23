package exchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang/glog"
	"github.com/open-horizon/anax/config"
	"github.com/open-horizon/anax/cutil"
	"github.com/open-horizon/anax/policy"
	"strings"
	"time"
)

// Types and functions used to work with the exchange's service objects.

// This type is used to abstract the various edge node hardware requirements. The schema is left wide open.
type HardwareRequirement map[string]interface{}

func (h HardwareRequirement) String() string {
	res := ""
	for key, val := range h {
		res += fmt.Sprintf("{%v:%v},", key, val)
	}
	if len(res) > 2 {
		res = res[:len(res)-1]
	} else {
		res = "none"
	}
	return res
}

// This type is a tuple used to refer to a specific service that is a dependency for the referencing service.
type ServiceDependency struct {
	URL     string `json:"url"`
	Org     string `json:"org"`
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

func (sd ServiceDependency) String() string {
	return fmt.Sprintf("{URL: %v, Org: %v, Version: %v, Arch: %v}", sd.URL, sd.Org, sd.Version, sd.Arch)
}

// This type is used to describe a configuration variable that the node owner/user has to set before the
// service is able to execute on the edge node.
type UserInput struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	Type         string `json:"type"` // Valid values are "string", "int", "float", "boolean", "list of strings"
	DefaultValue string `json:"defaultValue"`
}

func (ui UserInput) String() string {
	return fmt.Sprintf("{Name: %v, :Label: %v, Type: %v, DefaultValue: %v}", ui.Name, ui.Label, ui.Type, ui.DefaultValue)
}

// This type is used to describe the package that implements the service. A package is a generic idea that can
// be realized in many forms. Initially a docker container is the only supported form. The schema for this
// type is left wide open. There is 1 required key in the map; "storeType" which is used to discriminate what
// other keys should be there. The valid values for storeType are "container" and "imageServer". The map could
// be completely empty if the docker container image in the deployment string is being used as the "package".
const IMPL_PACKAGE_DISCRIMINATOR = "storeType"
const IMPL_PACKAGE_CONTAINER = "dockerRegistry"
const IMPL_PACKAGE_IMAGESERVER = "imageServer"

type ImplementationPackage map[string]interface{}

func (i ImplementationPackage) String() string {
	res := "{"
	for key, val := range i {
		res += fmt.Sprintf("%v:%v, ", key, val)
	}
	if len(res) > 2 {
		res = res[:len(res)-2] + "}"
	} else {
		res = "none"
	}
	return res
}

// This is the structure of the object returned on a GET /service.
// microservice sharing mode
const MS_SHARING_MODE_EXCLUSIVE = "exclusive"
const MS_SHARING_MODE_SINGLE = "single" // deprecated, use singleton instead. but leave it here for backward compatibility
const MS_SHARING_MODE_SINGLETON = "singleton"
const MS_SHARING_MODE_MULTIPLE = "multiple"

type ServiceDefinition struct {
	Owner               string                `json:"owner"`
	Label               string                `json:"label"`
	Description         string                `json:"description"`
	Public              bool                  `json:"public"`
	URL                 string                `json:"url"`
	Version             string                `json:"version"`
	Arch                string                `json:"arch"`
	Sharable            string                `json:"sharable"`
	MatchHardware       HardwareRequirement   `json:"matchHardware"`
	RequiredServices    []ServiceDependency   `json:"requiredServices"`
	UserInputs          []UserInput           `json:"userInput"`
	Deployment          string                `json:"deployment"`
	DeploymentSignature string                `json:"deploymentSignature"`
	ImageStore          ImplementationPackage `json:"imageStore"`
	LastUpdated         string                `json:"lastUpdated"`
}

func (s ServiceDefinition) String() string {
	return fmt.Sprintf("Owner: %v, "+
		"Label: %v, "+
		"Description: %v, "+
		"Public: %v, "+
		"URL: %v, "+
		"Version: %v, "+
		"Arch: %v, "+
		"Sharable: %v, "+
		"MatchHardware: %v, "+
		"RequiredServices: %v, "+
		"UserInputs: %v, "+
		"Deployment: %v, "+
		"DeploymentSignature: %v, "+
		"Package: %v, "+
		"LastUpdated: %v",
		s.Owner, s.Label, s.Description, s.Public, s.URL, s.Version, s.Arch, s.Sharable,
		s.MatchHardware, s.RequiredServices, s.UserInputs, s.Deployment, s.DeploymentSignature,
		s.ImageStore, s.LastUpdated)
}

func (s ServiceDefinition) ShortString() string {
	return fmt.Sprintf("URL: %v, "+
		"Version: %v, "+
		"Arch: %v, "+
		"RequiredServices: %v",
		s.URL, s.Version, s.Arch, s.RequiredServices)
}

func (s *ServiceDefinition) GetUserInputName(name string) *UserInput {
	for _, ui := range s.UserInputs {
		if ui.Name == name {
			return &ui
		}
	}
	return nil
}

func (s *ServiceDefinition) NeedsUserInput() bool {
	for _, ui := range s.UserInputs {
		if ui.DefaultValue == "" {
			return true
		}
	}
	return false
}

func (s *ServiceDefinition) PopulateDefaultUserInput(envAdds map[string]string) {
	for _, ui := range s.UserInputs {
		if ui.DefaultValue != "" {
			if _, ok := envAdds[ui.Name]; !ok {
				envAdds[ui.Name] = ui.DefaultValue
			}
		}
	}
}

func (s *ServiceDefinition) GetDeployment() string {
	return s.Deployment
}

func (s *ServiceDefinition) GetDeploymentSignature() string {
	return s.DeploymentSignature
}

func (s *ServiceDefinition) GetTorrent() string {
	return ""
}

func (s *ServiceDefinition) GetImageStore() policy.ImplementationPackage {
	polIP := make(policy.ImplementationPackage)
	cutil.CopyMap(s.ImageStore, polIP)
	return polIP
}

func (s *ServiceDefinition) HasDependencies() bool {
	return len(s.RequiredServices) != 0
}

func (s *ServiceDefinition) GetServiceDependencies() *[]ServiceDependency {
	return &s.RequiredServices
}

func (s *ServiceDefinition) GetVersion() string {
	return s.Version
}

type GetServicesResponse struct {
	Services  map[string]ServiceDefinition `json:"services"`
	LastIndex int                          `json:"lastIndex"`
}

func (w *GetServicesResponse) ShortString() string {
	// get the short string for each MicroserviceDefinition
	wl_a := make(map[string]string)
	for ms_name, wl := range w.Services {
		wl_a[ms_name] = wl.ShortString()
	}

	return fmt.Sprintf("LastIndex: %v, "+
		"Services: %v",
		w.LastIndex, wl_a)
}

type ImageDockerAuth struct {
	DockAuthId  int    `json:"dockAuthId"`
	Registry    string `json:"registry"`
	UserName    string `json:"username"`
	Token       string `json:"token"`
	LastUpdated string `json:"lastUpdated"`
}

func (s ImageDockerAuth) String() string {
	return fmt.Sprintf("DockAuthId: %v, "+
		"Registry: %v, "+
		"UserName: %v, "+
		"Token: %v, "+
		"LastUpdated: %v",
		s.DockAuthId, s.Registry, s.UserName, s.Token, s.LastUpdated)
}

// service configuration states
const SERVICE_CONFIGSTATE_SUSPENDED = "suspended"
const SERVICE_CONFIGSTATE_ACTIVE = "active"

type ServiceConfigState struct {
	Url         string `json:"url"`
	Org         string `json:"org"`
	ConfigState string `json:"configState"`
}

func (s *ServiceConfigState) String() string {
	return fmt.Sprintf("Url: %v, Org: %v, ConfigState: %v", s.Url, s.Org, s.ConfigState)
}

func NewServiceConfigState(url, org, state string) *ServiceConfigState {
	return &ServiceConfigState{
		Url:         url,
		Org:         org,
		ConfigState: state,
	}
}

// check if the 2 given config states are the same.
func SameCongigState(state1 string, state2 string) bool {
	if state1 == state2 {
		return true
	}

	if state1 == "" && state2 == SERVICE_CONFIGSTATE_ACTIVE {
		return true
	}

	if state1 == SERVICE_CONFIGSTATE_ACTIVE && state2 == "" {
		return true
	}

	return false
}

// This function is used to figure out what kind of version search to do in the exchange based on the input version string.
func getSearchVersion(version string) (string, error) {
	// The caller could pass a specific version or a version range, in the version parameter. If it's a version range
	// then it must be a full expression. That is, it must be expanded into the full syntax. For example; 1.2.3 is a specific
	// version, and [4.5.6, INFINITY) is the full expression corresponding to the shorthand form of "4.5.6".
	searchVersion := ""
	if version == "" || policy.IsVersionExpression(version) {
		// search for all versions
	} else if policy.IsVersionString(version) {
		// search for a specific version
		searchVersion = version
	} else {
		return "", errors.New(fmt.Sprintf("input version %v is not a valid version string", version))
	}
	return searchVersion, nil
}

// Retrieve service definition metadata from the exchange, by specific version or for all versions.
func GetService(ec ExchangeContext, mURL string, mOrg string, mVersion string, mArch string) (*ServiceDefinition, string, error) {

	glog.V(3).Infof(rpclogString(fmt.Sprintf("getting service definition %v %v %v %v", mURL, mOrg, mVersion, mArch)))

	var resp interface{}
	resp = new(GetServicesResponse)

	// Figure out which version to filter the search with. Could be "".
	searchVersion, err := getSearchVersion(mVersion)
	if err != nil {
		return nil, "", err
	}

	// Search the exchange for the service definition
	targetURL := fmt.Sprintf("%vorgs/%v/services?url=%v&arch=%v", ec.GetExchangeURL(), mOrg, mURL, mArch)
	if searchVersion != "" {
		targetURL = fmt.Sprintf("%vorgs/%v/services?url=%v&version=%v&arch=%v", ec.GetExchangeURL(), mOrg, mURL, searchVersion, mArch)
	}

	for {
		if err, tpErr := InvokeExchange(ec.GetHTTPFactory().NewHTTPClient(nil), "GET", targetURL, ec.GetExchangeId(), ec.GetExchangeToken(), nil, &resp); err != nil {
			glog.Errorf(rpclogString(fmt.Sprintf(err.Error())))
			return nil, "", err
		} else if tpErr != nil {
			glog.Warningf(rpclogString(fmt.Sprintf(tpErr.Error())))
			time.Sleep(10 * time.Second)
			continue
		} else {
			return processGetServiceResponse(mURL, mOrg, mVersion, mArch, searchVersion, resp.(*GetServicesResponse))
		}
	}
}

// When we get a non-error response from the exchange, process the response to return the results based on what the caller
// was searching for (the service tuple and the desired version or version range).
func processGetServiceResponse(mURL string, mOrg string, mVersion string, mArch string, searchVersion string, resp interface{}) (*ServiceDefinition, string, error) {

	glog.V(5).Infof(rpclogString(fmt.Sprintf("found service %v.", resp.(*GetServicesResponse).ShortString())))
	msMetadata := resp.(*GetServicesResponse).Services

	// If the caller wanted a specific version, check for 1 result.
	if searchVersion != "" {
		if len(msMetadata) != 1 {
			glog.Errorf(rpclogString(fmt.Sprintf("expecting 1 service %v %v %v response: %v", mURL, mOrg, mVersion, resp)))
			return nil, "", errors.New(fmt.Sprintf("expecting 1 service %v %v %v, got %v", mURL, mOrg, mVersion, len(msMetadata)))
		} else {
			for msId, msDef := range msMetadata {
				glog.V(3).Infof(rpclogString(fmt.Sprintf("returning service definition %v", msDef.ShortString())))
				return &msDef, msId, nil
			}
			return nil, "", errors.New("should not get here")
		}

	} else {
		if len(msMetadata) == 0 {
			return nil, "", errors.New(fmt.Sprintf("expecting at least 1 service %v %v %v, got %v", mURL, mOrg, mVersion, len(msMetadata)))
		}
		// The caller wants the highest version in the input version range. If no range was specified then
		// they will get the highest of all available versions.
		vRange, _ := policy.Version_Expression_Factory("0.0.0")
		var err error
		if mVersion != "" {
			if vRange, err = policy.Version_Expression_Factory(mVersion); err != nil {
				return nil, "", errors.New(fmt.Sprintf("version range %v in error: %v", mVersion, err))
			}
		}

		highest, resMsDef, resMsId, err := GetHighestVersion(msMetadata, vRange)
		if err != nil {
			glog.Errorf(rpclogString(err))
			return nil, "", err
		}

		if highest == "" {
			// when highest is empty, it means that there were no data in msMetadata, hence return nil.
			glog.V(3).Infof(rpclogString(fmt.Sprintf("returning service definition %v for %v", nil, mURL)))
			return nil, "", nil
		} else {
			glog.V(3).Infof(rpclogString(fmt.Sprintf("returning service definition %v for %v", resMsDef.ShortString(), mURL)))
			return &resMsDef, resMsId, nil
		}
	}
}

// Find the highest version service and return it.
func GetHighestVersion(msMetadata map[string]ServiceDefinition, vRange *policy.Version_Expression) (string, ServiceDefinition, string, error) {
	highest := ""
	if vRange == nil {
		vRange, _ = policy.Version_Expression_Factory("0.0.0")
	}
	// resSDef has to be an object instead of pointer to the object because once the pointer points to &sDef,
	// the content of it will get changed when the content of sDef gets changed in the loop.
	var resSDef ServiceDefinition
	var resSId string
	for sId, sDef := range msMetadata {
		if inRange, err := vRange.Is_within_range(sDef.Version); err != nil {
			return "", resSDef, "", errors.New(fmt.Sprintf("unable to verify that %v is within %v, error %v", sDef.Version, vRange, err))
		} else if inRange {
			glog.V(5).Infof(rpclogString(fmt.Sprintf("found service version %v within acceptable range", sDef.Version)))

			// cannot pass in "" in the CompareVersions because it checks for invalid version strings.
			var c int
			var err error

			if highest == "" {
				c, err = policy.CompareVersions("0.0.0", sDef.Version)
			} else {
				c, err = policy.CompareVersions(highest, sDef.Version)
			}
			if err != nil {
				glog.Errorf(rpclogString(fmt.Sprintf("error comparing version %v with version %v. %v", highest, sDef.Version, err)))
			} else if c <= 0 {
				highest = sDef.Version
				resSDef = sDef
				resSId = sId
			}
		}
	}
	return highest, resSDef, resSId, nil
}

// The purpose of this function is to verify that a given service URL, version and architecture, is defined in the exchange
// as well as all of its required services. This function also returns the dependencies converted into policy types so that the caller
// can use those types to do policy compatibility checks if they want to.
func ServiceResolver(wURL string, wOrg string, wVersion string, wArch string, serviceHandler ServiceHandler) (*policy.APISpecList, *ServiceDefinition, error) {

	resolveRequiredServices := true

	glog.V(5).Infof(rpclogString(fmt.Sprintf("resolving service %v %v %v %v", wURL, wOrg, wVersion, wArch)))

	res := new(policy.APISpecList)
	// Get a version specific service definition.
	tlService, _, werr := serviceHandler(wURL, wOrg, wVersion, wArch)
	if werr != nil {
		return nil, nil, werr
	} else if tlService == nil {
		return nil, nil, errors.New(fmt.Sprintf("unable to find service %v %v %v %v on the exchange.", wURL, wOrg, wVersion, wArch))
	} else {

		// We found the service definition. Required services are referred to within a service definition by URL, org, architecture,
		// and version range. Service definitions in the exchange arent queryable by version range, so we will have to do the version
		// filtering.  We're looking for the highest version service definition that is within the range defined by the service.
		// See ./policy/version.go for an explanation of version syntax and version ranges. The GetService() function is smart enough
		// to return the service we're looking for as long as we give it a range to search within.

		if resolveRequiredServices {
			glog.V(5).Infof(rpclogString(fmt.Sprintf("resolving required services for %v %v %v %v", wURL, wOrg, wVersion, wArch)))
			for _, sDep := range tlService.RequiredServices {

				// Make sure the required service has the same arch as the service.
				// Convert version to a version range expression (if it's not already an expression) so that the underlying GetService
				// will return us something in the range required by the service.
				var serviceDef *ServiceDefinition
				if sDep.Arch != wArch {
					return nil, nil, errors.New(fmt.Sprintf("service %v has a different architecture than the top level service.", sDep))
				} else if vExp, err := policy.Version_Expression_Factory(sDep.Version); err != nil {
					return nil, nil, errors.New(fmt.Sprintf("unable to create version expression from %v, error %v", sDep.Version, err))
				} else if apiSpecs, sd, err := ServiceResolver(sDep.URL, sDep.Org, vExp.Get_expression(), sDep.Arch, serviceHandler); err != nil {
					return nil, nil, err
				} else {
					// Add all service dependencies to the running list of API specs.
					serviceDef = sd
					for _, as := range *apiSpecs {
						// If the apiSpec is already in the list, ignore it by ignoring the returned error.
						res.Add_API_Spec(&as)
					}
				}

				// Capture the current service dependency as an API Spec object and add it to the running list of API specs.
				newAPISpec := policy.APISpecification_Factory(sDep.URL, sDep.Org, sDep.Version, sDep.Arch)
				if serviceDef.Sharable == MS_SHARING_MODE_SINGLETON || serviceDef.Sharable == MS_SHARING_MODE_SINGLE {
					newAPISpec.ExclusiveAccess = false
				}
				res.Add_API_Spec(newAPISpec)
			}
			glog.V(5).Infof(rpclogString(fmt.Sprintf("resolved required services for %v %v %v %v", wURL, wOrg, wVersion, wArch)))
		}
		glog.V(5).Infof(rpclogString(fmt.Sprintf("resolved service %v %v %v %v, APISpecs: %v", wURL, wOrg, wVersion, wArch, *res)))
		return res, tlService, nil

	}

}

// This function gets the image docker auths for a service.
func GetServiceDockerAuths(ec ExchangeContext, url string, org string, version string, arch string) ([]ImageDockerAuth, error) {

	glog.V(3).Infof(rpclogString(fmt.Sprintf("getting docker auths for service %v %v %v %v", url, org, version, arch)))

	if version == "" || !policy.IsVersionString(version) {
		return nil, errors.New(rpclogString(fmt.Sprintf("GetServiceDockerAuths got wrong version string %v. The version string should be a non-empy single version string.", version)))
	}

	// get the service id
	s_resp, s_id, err := GetService(ec, url, org, version, arch)
	if err != nil {
		return nil, errors.New(rpclogString(fmt.Sprintf("failed to get the service %v %v %v %v.%v", url, org, version, arch, err)))
	} else if s_resp == nil {
		return nil, errors.New(rpclogString(fmt.Sprintf("unable to find the service %v %v %v %v.", url, org, version, arch)))
	}

	return GetServiceDockerAuthsWithId(ec, s_id)
}

// This function gets the image docker auths for the service by the given service id
func GetServiceDockerAuthsWithId(ec ExchangeContext, service_id string) ([]ImageDockerAuth, error) {

	glog.V(3).Infof(rpclogString(fmt.Sprintf("getting docker auths for service %v.", service_id)))

	// get all the docker auths for the service
	var resp_DockAuths interface{}
	resp_DockAuths = ""
	docker_auths := make([]ImageDockerAuth, 0)

	targetURL := fmt.Sprintf("%vorgs/%v/services/%v/dockauths", ec.GetExchangeURL(), GetOrg(service_id), GetId(service_id))
	for {
		if err, tpErr := InvokeExchange(ec.GetHTTPFactory().NewHTTPClient(nil), "GET", targetURL, ec.GetExchangeId(), ec.GetExchangeToken(), nil, &resp_DockAuths); err != nil {
			glog.Errorf(rpclogString(fmt.Sprintf(err.Error())))
			return nil, err
		} else if tpErr != nil {
			glog.Warningf(rpclogString(fmt.Sprintf(tpErr.Error())))
			time.Sleep(10 * time.Second)
			continue
		} else {
			if resp_DockAuths.(string) != "" {
				if err := json.Unmarshal([]byte(resp_DockAuths.(string)), &docker_auths); err != nil {
					return nil, errors.New(fmt.Sprintf("Unable to demarshal service docker auth response %v, error: %v", resp_DockAuths, err))
				}
			}
			break
		}
	}

	glog.V(5).Infof(rpclogString(fmt.Sprintf("returning service docker auths %v for service %v.", docker_auths, service_id)))
	return docker_auths, nil
}

func GetServicesConfigState(httpClientFactory *config.HTTPClientFactory, dev_id string, dev_token string, exchangeUrl string) ([]ServiceConfigState, error) {
	service_cs := []ServiceConfigState{}

	pDevice, err := GetExchangeDevice(httpClientFactory, dev_id, dev_token, exchangeUrl)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unable to retrieve node resource for %v from the exchange, error %v", dev_id, err))
	}

	for _, service := range pDevice.RegisteredServices {
		// service.Url is in the form of org/url
		org, url := cutil.SplitOrgSpecUrl(service.Url)

		// set to default if empty
		config_state := service.ConfigState
		if config_state == "" {
			config_state = SERVICE_CONFIGSTATE_ACTIVE
		}

		mcs := NewServiceConfigState(url, org, config_state)
		service_cs = append(service_cs, *mcs)
	}

	glog.V(5).Infof(rpclogString(fmt.Sprintf("returning service configuration states:  %v.", service_cs)))

	return service_cs, nil
}

// check the registered services to see if the given service is suspended or not
// returns (found, suspended)
func ServiceSuspended(registered_services []Microservice, service_url string, service_org string) (bool, bool) {
	if registered_services == nil {
		return false, false
	}
	for _, svc := range registered_services {
		if svc.Url == cutil.FormOrgSpecUrl(service_url, service_org) || svc.Url == service_url {
			if svc.ConfigState == SERVICE_CONFIGSTATE_SUSPENDED {
				return true, true
			} else {
				return true, false
			}
		}
	}

	return false, false
}

// modify the the configuration state for the registeredServices for a device.
func PostDeviceServicesConfigState(httpClientFactory *config.HTTPClientFactory, deviceId string, deviceToken string, exchangeUrl string, svcs_configstate *ServiceConfigState) error {
	// create POST body
	targetURL := exchangeUrl + "orgs/" + GetOrg(deviceId) + "/nodes/" + GetId(deviceId) + "/services_configstate"
	var resp interface{}
	resp = ""

	for {
		if err, tpErr := InvokeExchange(httpClientFactory.NewHTTPClient(nil), "POST", targetURL, deviceId, deviceToken, svcs_configstate, &resp); err != nil {
			return err
		} else if tpErr != nil {
			glog.Warningf(tpErr.Error())
			time.Sleep(10 * time.Second)
			continue
		} else {
			glog.V(3).Infof(rpclogString(fmt.Sprintf("post service configuration states %v for device %v to the exchange.", svcs_configstate, deviceId)))
			return nil
		}
	}
}

// This function gets the service policy for a service.
// It returns nil if there is no service policy for this service
func GetServicePolicy(ec ExchangeContext, url string, org string, version string, arch string) (*ExchangePolicy, string, error) {

	glog.V(3).Infof(rpclogString(fmt.Sprintf("getting service policy for service %v %v %v %v", url, org, version, arch)))

	if version == "" || !policy.IsVersionString(version) {
		return nil, "", errors.New(rpclogString(fmt.Sprintf("GetServicePolicy got wrong version string %v. The version string should be a non-empy single version string.", version)))
	}

	// get the service id
	s_resp, s_id, err := GetService(ec, url, org, version, arch)
	if err != nil {
		return nil, "", errors.New(rpclogString(fmt.Sprintf("failed to get the service %v %v %v %v.%v", url, org, version, arch, err)))
	} else if s_resp == nil {
		return nil, "", errors.New(rpclogString(fmt.Sprintf("unable to find the service %v %v %v %v.", url, org, version, arch)))
	}

	pol, err := GetServicePolicyWithId(ec, s_id)
	return pol, s_id, err
}

// Retrieve the service policy object from the exchange. The service_id is prefixed with the org name.
// It returns nil if there is no service policy for this service
func GetServicePolicyWithId(ec ExchangeContext, service_id string) (*ExchangePolicy, error) {
	glog.V(3).Infof(rpclogString(fmt.Sprintf("getting service policy for %v.", service_id)))

	// Get the service policy object. There should only be 1.
	var resp interface{}
	resp = new(ExchangePolicy)

	targetURL := fmt.Sprintf("%vorgs/%v/services/%v/policy", ec.GetExchangeURL(), GetOrg(service_id), GetId(service_id))
	for {
		if err, tpErr := InvokeExchange(ec.GetHTTPFactory().NewHTTPClient(nil), "GET", targetURL, ec.GetExchangeId(), ec.GetExchangeToken(), nil, &resp); err != nil {
			glog.Errorf(rpclogString(fmt.Sprintf(err.Error())))
			return nil, err
		} else if tpErr != nil {
			glog.Warningf(rpclogString(fmt.Sprintf(tpErr.Error())))
			time.Sleep(10 * time.Second)
			continue
		} else {
			glog.V(3).Infof(rpclogString(fmt.Sprintf("returning service policy for %v.", service_id)))
			servicePolicy := resp.(*ExchangePolicy)
			if servicePolicy.GetLastUpdated() == "" {
				return nil, nil
			} else {
				return servicePolicy, nil
			}
		}
	}
}

// This function updates the service policy for a service.
func PutServicePolicy(ec ExchangeContext, url string, org string, version string, arch string, ep *ExchangePolicy) (*PutDeviceResponse, error) {

	glog.V(3).Infof(rpclogString(fmt.Sprintf("updating service policy for service %v %v %v %v", url, org, version, arch)))

	if version == "" || !policy.IsVersionString(version) {
		return nil, errors.New(rpclogString(fmt.Sprintf("PutServicePolicy got wrong version string %v. The version string should be a non-empy single version string.", version)))
	}

	// get the service id
	s_resp, s_id, err := GetService(ec, url, org, version, arch)
	if err != nil {
		return nil, errors.New(rpclogString(fmt.Sprintf("failed to get the service %v %v %v %v.%v", url, org, version, arch, err)))
	} else if s_resp == nil {
		return nil, errors.New(rpclogString(fmt.Sprintf("unable to find the service %v %v %v %v.", url, org, version, arch)))
	}

	return PutServicePolicyWithId(ec, s_id, ep)
}

// Write an updated service policy to the exchange.
func PutServicePolicyWithId(ec ExchangeContext, service_id string, ep *ExchangePolicy) (*PutDeviceResponse, error) {
	// create PUT body
	var resp interface{}
	resp = new(PutDeviceResponse)
	targetURL := fmt.Sprintf("%vorgs/%v/services/%v/policy", ec.GetExchangeURL(), GetOrg(service_id), GetId(service_id))

	for {
		if err, tpErr := InvokeExchange(ec.GetHTTPFactory().NewHTTPClient(nil), "PUT", targetURL, ec.GetExchangeId(), ec.GetExchangeToken(), ep, &resp); err != nil {
			return nil, err
		} else if tpErr != nil {
			glog.Warningf(tpErr.Error())
			time.Sleep(10 * time.Second)
			continue
		} else {
			glog.V(3).Infof(rpclogString(fmt.Sprintf("put service policy for %v to exchange %v", service_id, ep)))
			return resp.(*PutDeviceResponse), nil
		}
	}
}

// This function deletes the service policy for a service.
// it returns nil if the policy is deleted or does not exist.
func DeleteServicePolicy(ec ExchangeContext, url string, org string, version string, arch string) error {

	glog.V(3).Infof(rpclogString(fmt.Sprintf("deleting service policy for service %v %v %v %v", url, org, version, arch)))

	if version == "" || !policy.IsVersionString(version) {
		return errors.New(rpclogString(fmt.Sprintf("DeleteServicePolicy got wrong version string %v. The version string should be a non-empy single version string.", version)))
	}

	// get the service id
	s_resp, s_id, err := GetService(ec, url, org, version, arch)
	if err != nil {
		return errors.New(rpclogString(fmt.Sprintf("failed to get the service %v %v %v %v.%v", url, org, version, arch, err)))
	} else if s_resp == nil {
		return errors.New(rpclogString(fmt.Sprintf("unable to find the service %v %v %v %v.", url, org, version, arch)))
	}

	return DeleteServicePolicyWithId(ec, s_id)
}

// Delete service policy from the exchange.
// It returns nil if the policy is deleted or does not exist.
func DeleteServicePolicyWithId(ec ExchangeContext, service_id string) error {
	// create PUT body
	var resp interface{}
	resp = new(PostDeviceResponse)
	targetURL := fmt.Sprintf("%vorgs/%v/services/%v/policy", ec.GetExchangeURL(), GetOrg(service_id), GetId(service_id))

	for {
		if err, tpErr := InvokeExchange(ec.GetHTTPFactory().NewHTTPClient(nil), "DELETE", targetURL, ec.GetExchangeId(), ec.GetExchangeToken(), nil, &resp); err != nil && !strings.Contains(err.Error(), "status: 404") {
			return err
		} else if tpErr != nil {
			glog.Warningf(tpErr.Error())
			time.Sleep(10 * time.Second)
			continue
		} else {
			glog.V(3).Infof(rpclogString(fmt.Sprintf("deleted device policy for %v from the exchange.", service_id)))
			return nil
		}
	}
}
