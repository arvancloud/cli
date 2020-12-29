package api

import (
	"encoding/json"
	"errors"
	"github.com/arvancloud/cli/pkg/config"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"net/http"
)

var Version string

const (
	apiPrefix       = "/paas/v1/regions/"
	defaultRegion   = "ir-thr-at1"
	regionsEndpoint = "/g/regions"
	userEndpoint    = "/g/user"
	updateEndpoint  = "/update"
	updateServer = "https://cli.arvan.run"
)

//GetUserInfo returns a dictionary of user info if authentication credentials is valid.
func GetUserInfo(apikey string) (map[string]string, error) {
	arvanConfig := config.GetConfigInfo()
	arvanServer := arvanConfig.GetServer()
	httpReq, err := http.NewRequest("GET", arvanServer+apiPrefix+defaultRegion+userEndpoint, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Authorization", apikey)
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode != http.StatusOK {
		if httpResp.StatusCode >= 400 && httpResp.StatusCode < 500 {
			return nil, errors.New("invalid authorization credentials")
		} else {
			return nil, errors.New("server error. try again later")
		}
	}

	user := make(map[string]string)
	// parse response
	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, err
	}

	return user, nil
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
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return regions, err
	}
	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return regions, err
	}
	// parse response
	err = json.Unmarshal(body, &regions)
	if err != nil {
		return regions, err
	}

	return regions, nil
}

type Update struct {
	URL     string
	Version string
}

func CheckUpdate() (*Update, error) {
	httpReq, err := http.NewRequest("GET", updateServer+updateEndpoint, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("accept", "application/json")
	httpReq.Header.Add("User-Agent", rest.DefaultKubernetesUserAgent())
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if httpResp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, errors.New("server error. try again later")
	}

	// read body
	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	// parse response
	var update Update
	err = json.Unmarshal(body, &update)
	if err != nil {
		return nil, err
	}
	return &update, nil
}
