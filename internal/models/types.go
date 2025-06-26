package models

// CloudflareResponse represents the API response structure
type CloudflareResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// KVNamespace represents a Cloudflare KV namespace
type KVNamespace struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// KVNamespaceResponse represents the API response for KV namespaces
type KVNamespaceResponse struct {
	Success bool          `json:"success"`
	Result  []KVNamespace `json:"result"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// HetznerLocation represents a Hetzner datacenter location
type HetznerLocation struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Country     string  `json:"country"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// HetznerServerType represents a Hetzner server type/instance
type HetznerServerType struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Cores              int             `json:"cores"`
	Memory             float64         `json:"memory"`
	Disk               int             `json:"disk"`
	Prices             []HetznerPrice  `json:"prices"`
	StorageType        string          `json:"storage_type"`
	CPUType            string          `json:"cpu_type"`
	Architecture       string          `json:"architecture"`
	AvailableLocations map[string]bool `json:"available_locations,omitempty"`
}

// HetznerPrice represents pricing information for a server type
type HetznerPrice struct {
	Location     string             `json:"location"`
	PriceHourly  HetznerPriceDetail `json:"price_hourly"`
	PriceMonthly HetznerPriceDetail `json:"price_monthly"`
}

// HetznerPriceDetail represents price details
type HetznerPriceDetail struct {
	Net   string `json:"net"`
	Gross string `json:"gross"`
}

// HetznerLocationsResponse represents the API response for locations
type HetznerLocationsResponse struct {
	Locations []HetznerLocation `json:"locations"`
}

// HetznerServerTypesResponse represents the API response for server types
type HetznerServerTypesResponse struct {
	ServerTypes []HetznerServerType `json:"server_types"`
}

// HetznerDatacenter represents a Hetzner datacenter with availability info
type HetznerDatacenter struct {
	ID          int                          `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Location    HetznerLocation              `json:"location"`
	ServerTypes HetznerDatacenterServerTypes `json:"server_types"`
}

// HetznerDatacenterServerTypes represents server type availability in a datacenter
type HetznerDatacenterServerTypes struct {
	Supported             []int `json:"supported"`
	Available             []int `json:"available"`
	AvailableForMigration []int `json:"available_for_migration"`
}

// HetznerDatacentersResponse represents the API response for datacenters
type HetznerDatacentersResponse struct {
	Datacenters []HetznerDatacenter `json:"datacenters"`
}

// CloudflareDomain represents a domain zone in Cloudflare
type CloudflareDomain struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Paused     bool   `json:"paused"`
	Type       string `json:"type"`
	Managed    bool   `json:"managed"`
	CreatedOn  string `json:"created_on"`
	ModifiedOn string `json:"modified_on"`
}

// CloudflareDomainsResponse represents the API response for domain zones
type CloudflareDomainsResponse struct {
	Success bool               `json:"success"`
	Result  []CloudflareDomain `json:"result"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

// Application represents a deployed application
type Application struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Subdomain    string `json:"subdomain"`
	Domain       string `json:"domain"`
	VPSID        string `json:"vps_id"`
	VPSName      string `json:"vps_name"`
	ChartName    string `json:"chart_name"`
	ChartVersion string `json:"chart_version"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}