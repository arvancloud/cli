package config

import "time"

// Region of PaaS service
type Region struct {
	Zones []Zone `json:"data"`
}

type Zone struct {
	Name          string    `json:"name"`
	Endpoint      string    `json:"endpoint"`
	Active        bool      `json:"active"`
	Status        string    `json:"status"`
	Version       string    `json:"version"`
	Release       string    `json:"release"`
	Default       bool      `json:"default"`
	RegionName    string    `json:"region_name"`
	RegionCity    string    `json:"region_city"`
	RegionCountry string    `json:"region_country"`
	CreatedAt     time.Time `json:"created_at"`
}
