package api

import (
	"errors"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"git.arvan.me/arvan/cli/pkg/config"
)

const (
	apiPrefix = "/paas/v1/regions/"
	defaultRegion = "ir-thr-mn1"
	regionsEndpoint = "/g/regions"
)

//GetUserInfo #TODO implement GetUserInfo
func GetUserInfo(apikey string) (map[string]string, error) {
	if len(apikey) > 0 {
		return make(map[string]string), nil
	}
	return nil, errors.New("No api key provided.")
}

// Region of PaaS service
type Region struct {
	Name   string
	Active bool
}

//GetRegions from PaaS API
func GetRegions() ([]Region, error) {
	var regions []Region
	arvanConfig := config.GetConfigInfo()
	arvanServer := arvanConfig.GetServer()
	httpReq, err := http.NewRequest("GET", arvanServer+apiPrefix+defaultRegion+regionsEndpoint, nil)
	if err != nil {
		return regions, err
	}
	httpReq.Header.Add("accept", "application/json")

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return regions, err
	}
	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)

	// parse response
	err = json.Unmarshal(body, &regions)
	if err != nil {
		return regions, err
	}

	return regions, nil
}
