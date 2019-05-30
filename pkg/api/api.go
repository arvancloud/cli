package api

import "errors"

//GetUserInfo #TODO implement GetUserInfo
func GetUserInfo(apikey string) (map[string]string, error) {
	if len(apikey) > 0 {
		return make(map[string]string), nil
	}
	return nil, errors.New("No api key provided.")
}

//GetRegions #TODO implement GetRegions
func GetRegions() ([]string, error) {
	return []string{"ir-thr-mn1", "ir-thr-mn1"}, nil
}
