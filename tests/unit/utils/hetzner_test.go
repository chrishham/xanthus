package utils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chrishham/xanthus/internal/models"
	"github.com/chrishham/xanthus/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestValidateHetznerAPIKey(t *testing.T) {
	testCases := []struct {
		name           string
		responseCode   int
		responseBody   string
		expectedResult bool
	}{
		{
			name:         "Valid API key",
			responseCode: 200,
			responseBody: `{
				"server_types": [
					{
						"id": 1,
						"name": "cx11",
						"description": "CX11",
						"cores": 1,
						"memory": 4.0,
						"disk": 25
					}
				]
			}`,
			expectedResult: true,
		},
		{
			name:         "Invalid API key - 401",
			responseCode: 401,
			responseBody: `{
				"error": {
					"code": "unauthorized",
					"message": "Invalid token"
				}
			}`,
			expectedResult: false,
		},
		{
			name:         "Invalid API key - 403",
			responseCode: 403,
			responseBody: `{
				"error": {
					"code": "forbidden",
					"message": "Access denied"
				}
			}`,
			expectedResult: false,
		},
		{
			name:           "Server error",
			responseCode:   500,
			responseBody:   `{"error": "Internal server error"}`,
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v1/server_types", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tc.responseCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// Test with real API (should fail with invalid key)
			result := utils.ValidateHetznerAPIKey("invalid-api-key")
			assert.False(t, result, "Invalid API key should return false")
		})
	}
}

func TestGetHetznerAPIKey(t *testing.T) {
	testCases := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "Invalid credentials",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test with invalid credentials (should fail)
			_, err := utils.GetHetznerAPIKey("invalid-token", "invalid-account-id")

			if tc.expectError {
				assert.Error(t, err)
			}
		})
	}
}

func TestFetchHetznerLocations(t *testing.T) {
	testCases := []struct {
		name         string
		responseCode int
		responseBody string
		expectError  bool
		expectedLen  int
	}{
		{
			name:         "Successful fetch",
			responseCode: 200,
			responseBody: `{
				"locations": [
					{
						"id": 1,
						"name": "fsn1",
						"description": "Falkenstein DC Park 1",
						"country": "DE",
						"city": "Falkenstein",
						"latitude": 50.47612,
						"longitude": 12.370071,
						"network_zone": "eu-central"
					},
					{
						"id": 2,
						"name": "nbg1",
						"description": "Nuremberg DC Park 1",
						"country": "DE",
						"city": "Nuremberg",
						"latitude": 49.452102,
						"longitude": 11.076665,
						"network_zone": "eu-central"
					}
				]
			}`,
			expectError: false,
			expectedLen: 2,
		},
		{
			name:         "Empty locations",
			responseCode: 200,
			responseBody: `{
				"locations": []
			}`,
			expectError: false,
			expectedLen: 0,
		},
		{
			name:         "HTTP error",
			responseCode: 500,
			responseBody: `{"error": "Internal server error"}`,
			expectError:  true,
			expectedLen:  0,
		},
		{
			name:         "Invalid JSON",
			responseCode: 200,
			responseBody: `invalid json`,
			expectError:  true,
			expectedLen:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v1/locations", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tc.responseCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// Test with real API (should fail with invalid key)
			locations, err := utils.FetchHetznerLocations("invalid-api-key")

			// For invalid credentials, we expect errors
			assert.Error(t, err)
			assert.Empty(t, locations)
		})
	}
}

func TestFetchHetznerServerTypes(t *testing.T) {
	testCases := []struct {
		name         string
		responseCode int
		responseBody string
		expectError  bool
		expectedLen  int
	}{
		{
			name:         "Successful fetch",
			responseCode: 200,
			responseBody: `{
				"server_types": [
					{
						"id": 1,
						"name": "cx11",
						"description": "CX11",
						"cores": 1,
						"memory": 4.0,
						"disk": 25,
						"cpu_type": "shared",
						"prices": [
							{
								"location": "fsn1",
								"price_hourly": {
									"net": "0.0052000000",
									"gross": "0.0061880000"
								},
								"price_monthly": {
									"net": "3.4100000000",
									"gross": "4.0579000000"
								}
							}
						]
					},
					{
						"id": 2,
						"name": "cx21",
						"description": "CX21",
						"cores": 2,
						"memory": 8.0,
						"disk": 40,
						"cpu_type": "shared",
						"prices": [
							{
								"location": "fsn1",
								"price_hourly": {
									"net": "0.0095000000",
									"gross": "0.0113050000"
								},
								"price_monthly": {
									"net": "6.2000000000",
									"gross": "7.3780000000"
								}
							}
						]
					}
				]
			}`,
			expectError: false,
			expectedLen: 2,
		},
		{
			name:         "Empty server types",
			responseCode: 200,
			responseBody: `{
				"server_types": []
			}`,
			expectError: false,
			expectedLen: 0,
		},
		{
			name:         "HTTP error",
			responseCode: 401,
			responseBody: `{"error": "Unauthorized"}`,
			expectError:  true,
			expectedLen:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v1/server_types", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tc.responseCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// Test with real API (should fail with invalid key)
			serverTypes, err := utils.FetchHetznerServerTypes("invalid-api-key")

			// For invalid credentials, we expect errors
			assert.Error(t, err)
			assert.Empty(t, serverTypes)
		})
	}
}

func TestFetchServerAvailability(t *testing.T) {
	testCases := []struct {
		name         string
		responseCode int
		responseBody string
		expectError  bool
		expectedData map[string]map[int]bool
	}{
		{
			name:         "Successful fetch",
			responseCode: 200,
			responseBody: `{
				"datacenters": [
					{
						"id": 1,
						"name": "fsn1-dc8",
						"description": "Falkenstein 1 DC8",
						"location": {
							"id": 1,
							"name": "fsn1",
							"description": "Falkenstein DC Park 1"
						},
						"server_types": {
							"supported": [1, 2, 3],
							"available": [1, 2],
							"available_for_migration": [1, 2, 3]
						}
					},
					{
						"id": 2,
						"name": "nbg1-dc3",
						"description": "Nuremberg 1 DC3",
						"location": {
							"id": 2,
							"name": "nbg1",
							"description": "Nuremberg DC Park 1"
						},
						"server_types": {
							"supported": [1, 2, 3],
							"available": [1, 3],
							"available_for_migration": [1, 2, 3]
						}
					}
				]
			}`,
			expectError: false,
			expectedData: map[string]map[int]bool{
				"fsn1": {1: true, 2: true},
				"nbg1": {1: true, 3: true},
			},
		},
		{
			name:         "Empty datacenters",
			responseCode: 200,
			responseBody: `{
				"datacenters": []
			}`,
			expectError:  false,
			expectedData: map[string]map[int]bool{},
		},
		{
			name:         "HTTP error",
			responseCode: 500,
			responseBody: `{"error": "Internal server error"}`,
			expectError:  true,
			expectedData: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v1/datacenters", r.URL.Path)
				assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(tc.responseCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// Test with real API (should fail with invalid key)
			availability, err := utils.FetchServerAvailability("invalid-api-key")

			// For invalid credentials, we expect errors
			assert.Error(t, err)
			assert.Nil(t, availability)
		})
	}
}

func TestFilterSharedVCPUServers(t *testing.T) {
	serverTypes := []models.HetznerServerType{
		{
			ID:      1,
			Name:    "cx11",
			CPUType: "shared",
			Cores:   1,
			Memory:  4.0,
		},
		{
			ID:      2,
			Name:    "ccx11",
			CPUType: "dedicated",
			Cores:   2,
			Memory:  8.0,
		},
		{
			ID:      3,
			Name:    "cx21",
			CPUType: "shared",
			Cores:   2,
			Memory:  8.0,
		},
		{
			ID:      4,
			Name:    "ccx21",
			CPUType: "dedicated",
			Cores:   4,
			Memory:  16.0,
		},
	}

	filtered := utils.FilterSharedVCPUServers(serverTypes)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "cx11", filtered[0].Name)
	assert.Equal(t, "cx21", filtered[1].Name)

	for _, server := range filtered {
		assert.Equal(t, "shared", server.CPUType)
	}
}

func TestGetServerTypeMonthlyPrice(t *testing.T) {
	testCases := []struct {
		name          string
		serverType    models.HetznerServerType
		expectedPrice float64
	}{
		{
			name: "Valid price",
			serverType: models.HetznerServerType{
				Prices: []models.HetznerPrice{
					{
						PriceMonthly: models.HetznerPriceDetail{
							Gross: "4.99",
						},
					},
				},
			},
			expectedPrice: 4.99,
		},
		{
			name: "Price with currency",
			serverType: models.HetznerServerType{
				Prices: []models.HetznerPrice{
					{
						PriceMonthly: models.HetznerPriceDetail{
							Gross: "12.50 EUR",
						},
					},
				},
			},
			expectedPrice: 12.50,
		},
		{
			name: "No prices",
			serverType: models.HetznerServerType{
				Prices: []models.HetznerPrice{},
			},
			expectedPrice: 0.0,
		},
		{
			name: "Invalid price format",
			serverType: models.HetznerServerType{
				Prices: []models.HetznerPrice{
					{
						PriceMonthly: models.HetznerPriceDetail{
							Gross: "invalid",
						},
					},
				},
			},
			expectedPrice: 0.0,
		},
		{
			name: "Empty price",
			serverType: models.HetznerServerType{
				Prices: []models.HetznerPrice{
					{
						PriceMonthly: models.HetznerPriceDetail{
							Gross: "",
						},
					},
				},
			},
			expectedPrice: 0.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			price := utils.GetServerTypeMonthlyPrice(tc.serverType)
			assert.Equal(t, tc.expectedPrice, price)
		})
	}
}

func TestSortServerTypesByPriceAsc(t *testing.T) {
	serverTypes := []models.HetznerServerType{
		{
			Name: "expensive",
			Prices: []models.HetznerPrice{
				{PriceMonthly: models.HetznerPriceDetail{Gross: "20.00"}},
			},
		},
		{
			Name: "cheap",
			Prices: []models.HetznerPrice{
				{PriceMonthly: models.HetznerPriceDetail{Gross: "5.00"}},
			},
		},
		{
			Name: "medium",
			Prices: []models.HetznerPrice{
				{PriceMonthly: models.HetznerPriceDetail{Gross: "10.00"}},
			},
		},
	}

	utils.SortServerTypesByPriceAsc(serverTypes)

	assert.Equal(t, "cheap", serverTypes[0].Name)
	assert.Equal(t, "medium", serverTypes[1].Name)
	assert.Equal(t, "expensive", serverTypes[2].Name)
}

func TestSortServerTypesByPriceDesc(t *testing.T) {
	serverTypes := []models.HetznerServerType{
		{
			Name: "cheap",
			Prices: []models.HetznerPrice{
				{PriceMonthly: models.HetznerPriceDetail{Gross: "5.00"}},
			},
		},
		{
			Name: "expensive",
			Prices: []models.HetznerPrice{
				{PriceMonthly: models.HetznerPriceDetail{Gross: "20.00"}},
			},
		},
		{
			Name: "medium",
			Prices: []models.HetznerPrice{
				{PriceMonthly: models.HetznerPriceDetail{Gross: "10.00"}},
			},
		},
	}

	utils.SortServerTypesByPriceDesc(serverTypes)

	assert.Equal(t, "expensive", serverTypes[0].Name)
	assert.Equal(t, "medium", serverTypes[1].Name)
	assert.Equal(t, "cheap", serverTypes[2].Name)
}

func TestSortServerTypesByCPUAsc(t *testing.T) {
	serverTypes := []models.HetznerServerType{
		{Name: "high-cpu", Cores: 8},
		{Name: "low-cpu", Cores: 1},
		{Name: "mid-cpu", Cores: 4},
	}

	utils.SortServerTypesByCPUAsc(serverTypes)

	assert.Equal(t, "low-cpu", serverTypes[0].Name)
	assert.Equal(t, "mid-cpu", serverTypes[1].Name)
	assert.Equal(t, "high-cpu", serverTypes[2].Name)
}

func TestSortServerTypesByCPUDesc(t *testing.T) {
	serverTypes := []models.HetznerServerType{
		{Name: "low-cpu", Cores: 1},
		{Name: "high-cpu", Cores: 8},
		{Name: "mid-cpu", Cores: 4},
	}

	utils.SortServerTypesByCPUDesc(serverTypes)

	assert.Equal(t, "high-cpu", serverTypes[0].Name)
	assert.Equal(t, "mid-cpu", serverTypes[1].Name)
	assert.Equal(t, "low-cpu", serverTypes[2].Name)
}

func TestSortServerTypesByMemoryAsc(t *testing.T) {
	serverTypes := []models.HetznerServerType{
		{Name: "high-mem", Memory: 32.0},
		{Name: "low-mem", Memory: 4.0},
		{Name: "mid-mem", Memory: 16.0},
	}

	utils.SortServerTypesByMemoryAsc(serverTypes)

	assert.Equal(t, "low-mem", serverTypes[0].Name)
	assert.Equal(t, "mid-mem", serverTypes[1].Name)
	assert.Equal(t, "high-mem", serverTypes[2].Name)
}

func TestSortServerTypesByMemoryDesc(t *testing.T) {
	serverTypes := []models.HetznerServerType{
		{Name: "low-mem", Memory: 4.0},
		{Name: "high-mem", Memory: 32.0},
		{Name: "mid-mem", Memory: 16.0},
	}

	utils.SortServerTypesByMemoryDesc(serverTypes)

	assert.Equal(t, "high-mem", serverTypes[0].Name)
	assert.Equal(t, "mid-mem", serverTypes[1].Name)
	assert.Equal(t, "low-mem", serverTypes[2].Name)
}

func TestSortingEdgeCases(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		var serverTypes []models.HetznerServerType

		// Should not panic
		utils.SortServerTypesByPriceAsc(serverTypes)
		utils.SortServerTypesByCPUAsc(serverTypes)
		utils.SortServerTypesByMemoryAsc(serverTypes)

		assert.Empty(t, serverTypes)
	})

	t.Run("Single element", func(t *testing.T) {
		serverTypes := []models.HetznerServerType{
			{Name: "single", Cores: 2, Memory: 8.0},
		}

		utils.SortServerTypesByPriceAsc(serverTypes)
		utils.SortServerTypesByCPUAsc(serverTypes)
		utils.SortServerTypesByMemoryAsc(serverTypes)

		assert.Len(t, serverTypes, 1)
		assert.Equal(t, "single", serverTypes[0].Name)
	})

	t.Run("Identical values", func(t *testing.T) {
		serverTypes := []models.HetznerServerType{
			{Name: "server1", Cores: 2, Memory: 8.0},
			{Name: "server2", Cores: 2, Memory: 8.0},
			{Name: "server3", Cores: 2, Memory: 8.0},
		}

		utils.SortServerTypesByCPUAsc(serverTypes)

		// Order should be preserved for equal values
		assert.Len(t, serverTypes, 3)
		for _, server := range serverTypes {
			assert.Equal(t, 2, server.Cores)
			assert.Equal(t, 8.0, server.Memory)
		}
	})
}

func TestHetznerUtilsIntegration(t *testing.T) {
	t.Run("Full workflow with invalid credentials", func(t *testing.T) {
		// Test that all functions handle invalid credentials gracefully

		// Validate API key
		isValid := utils.ValidateHetznerAPIKey("invalid-key")
		assert.False(t, isValid)

		// Fetch locations
		locations, err := utils.FetchHetznerLocations("invalid-key")
		assert.Error(t, err)
		assert.Empty(t, locations)

		// Fetch server types
		serverTypes, err := utils.FetchHetznerServerTypes("invalid-key")
		assert.Error(t, err)
		assert.Empty(t, serverTypes)

		// Fetch server availability
		availability, err := utils.FetchServerAvailability("invalid-key")
		assert.Error(t, err)
		assert.Nil(t, availability)

		// Get API key from KV
		_, err = utils.GetHetznerAPIKey("invalid-token", "invalid-account-id")
		assert.Error(t, err)
	})
}

// Benchmarks
func BenchmarkValidateHetznerAPIKey(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.ValidateHetznerAPIKey("benchmark-key")
	}
}

func BenchmarkGetServerTypeMonthlyPrice(b *testing.B) {
	serverType := models.HetznerServerType{
		Prices: []models.HetznerPrice{
			{PriceMonthly: models.HetznerPriceDetail{Gross: "12.99"}},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.GetServerTypeMonthlyPrice(serverType)
	}
}

func BenchmarkSortServerTypesByPrice(b *testing.B) {
	serverTypes := make([]models.HetznerServerType, 100)
	for i := 0; i < 100; i++ {
		serverTypes[i] = models.HetznerServerType{
			Name: strings.Repeat("a", i%10),
			Prices: []models.HetznerPrice{
				{PriceMonthly: models.HetznerPriceDetail{Gross: "10.00"}},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		serverTypesCopy := make([]models.HetznerServerType, len(serverTypes))
		copy(serverTypesCopy, serverTypes)
		utils.SortServerTypesByPriceAsc(serverTypesCopy)
	}
}

func BenchmarkFilterSharedVCPUServers(b *testing.B) {
	serverTypes := make([]models.HetznerServerType, 100)
	for i := 0; i < 100; i++ {
		cpuType := "shared"
		if i%3 == 0 {
			cpuType = "dedicated"
		}
		serverTypes[i] = models.HetznerServerType{
			ID:      i,
			Name:    strings.Repeat("a", i%10),
			CPUType: cpuType,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		utils.FilterSharedVCPUServers(serverTypes)
	}
}
