package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	imgui "github.com/AllenDang/giu"
	openapiclient "github.com/netbox-community/go-netbox/v4"
	"github.com/sjwhitworth/golearn/base"
	"github.com/sjwhitworth/golearn/evaluation"
	"github.com/sjwhitworth/golearn/trees"
	excel "github.com/xuri/excelize/v2"
)

type Tenant struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Display     string `json:"display"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type Family struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

type Status struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Prefix struct {
	ID           int                    `json:"id"`
	URL          string                 `json:"url"`
	DisplayURL   string                 `json:"display_url"`
	Display      string                 `json:"display"`
	Family       Family                 `json:"family"`
	Prefix       string                 `json:"prefix"`
	Site         *Site                  `json:"site"`
	VRF          *VRF                   `json:"vrf"`
	Tenant       Tenant                 `json:"tenant"`
	VLAN         *VLAN                  `json:"vlan"`
	Status       Status                 `json:"status"`
	Role         *Role                  `json:"role"`
	IsPool       bool                   `json:"is_pool"`
	MarkUtilized bool                   `json:"mark_utilized"`
	Description  string                 `json:"description"`
	Comments     string                 `json:"comments"`
	Tags         []string               `json:"tags"`
	CustomFields map[string]interface{} `json:"custom_fields"`
	Created      string                 `json:"created"`
	LastUpdated  string                 `json:"last_updated"`
	Children     int                    `json:"children"`
	Depth        int                    `json:"_depth"`
}

type Site struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Display     string `json:"display"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type VRF struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Display     string `json:"display"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type VLAN struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Display     string `json:"display"`
	Name        string `json:"name"`
	Vid         int    `json:"vid"`
	Description string `json:"description"`
}

type Role struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Display     string `json:"display"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type ApiResponse struct {
	Count    int      `json:"count"`
	Next     *string  `json:"next"`
	Previous *string  `json:"previous"`
	Results  []Prefix `json:"results"`
}

type Device struct {
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Display string `json:"display"`
}

type DeviceListResponse struct {
	Count    int      `json:"count"`
	Next     string   `json:"next"`
	Previous string   `json:"previous"`
	Results  []Device `json:"results"`
}

type DeviceDetails struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	DeviceRole struct {
		Display string `json:"display"`
	} `json:"device_role"`
	DeviceType struct {
		Display      string `json:"display"`
		Manufacturer struct {
			Display string `json:"display"`
		} `json:"manufacturer"`
	} `json:"device_type"`
	Status struct {
		Value string `json:"value"`
	} `json:"status"`
	Serial string `json:"serial"`
	Tenant struct {
		Display string `json:"display"`
	} `json:"tenant"`
	Site struct {
		Display string `json:"display"`
	} `json:"site"`
}

type DeviceRequest struct {
	Name         string `json:"name"`
	DeviceType   int    `json:"device_type"`      // ID of the device type
	DeviceRole   int    `json:"role"`             // ID of the device role
	Site         int    `json:"site"`             // ID of the site
	Tenant       int    `json:"tenant,omitempty"` // ID of the tenant (optional)
	Manufacturer int    `json:"manufacturer"`     // ID of the manufacturer
	Status       string `json:"status"`           // Status, e.g., "active"
	Serial       string `json:"serial,omitempty"` // Serial number (optional)
}

var apiClient *openapiclient.APIClient
var ctx context.Context
var rows []*imgui.TableRowWidget
var timer float32 = 10.0
var showEnterVLANWindow bool = false
var showEnterDeviceWindow bool = false
var showLoggedIn bool = true
var showDeviceScreen bool = false
var inputVLANName string
var inputVLANDesc string
var inputVLANVid int32
var inputDeviceSerialNumber string = ""
var inputDeviceName string = ""
var inputIPAddressToSearchString string = ""
var inputDeviceToSearchString string = ""
var inputDomainLogIn string = "https://demo.netbox.dev"
var inputAPITokenLogIn string = ""
var listOfTenant []openapiclient.Tenant = make([]openapiclient.Tenant, 0)
var listOfTenantName []string = make([]string, 0)
var listOfDevice []int = make([]int, 0)
var listOfDeviceName []string = make([]string, 0)
var listOfDeviceType []int = make([]int, 0)
var listOfDeviceTypeName []string = make([]string, 0)
var listOfDeviceManufacturer []int = make([]int, 0)
var listOfDeviceManufacturerName []string = make([]string, 0)
var listOfDeviceSite []int = make([]int, 0)
var listOfDeviceSiteName []string = make([]string, 0)
var listOfDeviceRole []int = make([]int, 0)
var listOfDeviceRoleName []string = make([]string, 0)
var tenantChoice int32 = 0
var deviceChoice int32 = 0
var deviceTypeChoice int32 = 0
var deviceManufacturerChoice int32 = 0
var deviceSiteChoice int32 = 0
var deviceRoleChoice int32 = 0

func buildRows() []*imgui.TableRowWidget {

	if timer <= 0.0 && !showDeviceScreen {

		//Tenant
		nulTenant := openapiclient.Tenant{
			Name: "None",
		}

		listOfTenantName = append(listOfTenantName, "None")
		listOfTenant = append(listOfTenant, nulTenant)

		tenantList, _, err := apiClient.TenancyAPI.TenancyTenantsList(ctx).Execute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching tenants: %v\n", err)
		}

		for _, tenant := range tenantList.Results {
			listOfTenant = append(listOfTenant, tenant)
			listOfTenantName = append(listOfTenantName, tenant.Name)
		}

		// Fetch all VLANs
		availableVLANs, _, err := apiClient.IpamAPI.IpamVlansList(ctx).Limit(6000).Execute()

		if err != nil {
			log.Fatal(err)
		}

		// Set headers for VLAN table
		headers := []string{"ID", "Name", "Vid", "Prefix", "Tenant", "Description"}

		var total = 0
		for _, vlan := range availableVLANs.Results {
			if strings.Contains(vlan.Name, inputIPAddressToSearchString) {
				total++
			}
		}

		// Initialize row lists
		listOfDevice = listOfDevice[:0]
		listOfDeviceName = listOfDeviceName[:0]

		// Create rows for the VLANs
		rows = make([]*imgui.TableRowWidget, total+1)

		// Insert table headers
		rows[0] = imgui.TableRow(
			imgui.Label(headers[0]),
			imgui.Label(headers[1]),
			imgui.Label(headers[2]),
			imgui.Label(headers[3]),
			imgui.Label(headers[4]),
			imgui.Label(headers[5]),
		)
		rows[0].BgColor(&(color.RGBA{200, 100, 100, 255}))

		// Fill table with VLAN data
		var i = 1
		for _, vlan := range availableVLANs.Results {
			if strings.Contains(vlan.Name, inputIPAddressToSearchString) {

				// Extract data for each column
				id := fmt.Sprintf("%d", vlan.Id)
				name := vlan.Name
				vid := fmt.Sprintf("%d", vlan.Vid) // Numeric VLAN ID (1-4094)
				description := "None"
				if vlan.Description != nil {
					description = *vlan.Description
				}

				// Fetch tenant for VLAN
				tenantName := fetchTenantForVLAN(vlan.Id)

				// Fetch prefixes for VLAN
				prefixes := fetchPrefixesForVLAN(vlan.Id)

				// Insert row data
				rows[i] = imgui.TableRow(
					imgui.Label(id),
					imgui.Label(name),
					imgui.Label(vid),
					imgui.Label(strings.Join(prefixes, ", ")), // Concatenate prefixes
					imgui.Label(tenantName),
					imgui.Label(description),
				)
				i++
			}
		}

		timer = 50.0
	}

	return rows
}

// Function to fetch tenant for a VLAN
func fetchTenantForVLAN(vlanID int32) string {
	// Build the URL to fetch VLAN details
	vlanURL := fmt.Sprintf(inputDomainLogIn+"/api/ipam/vlans/%d/", vlanID)

	// Create the GET request
	req, err := http.NewRequest("GET", vlanURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return "Error"
	}

	// Add headers (e.g., API token)
	req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return "Error"
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error fetching VLAN: %s\n", string(body))
		return "Error"
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return "Error"
	}

	// Parse the response JSON to get tenant info
	var vlanData struct {
		Tenant struct {
			Name string `json:"name"`
		} `json:"tenant"`
	}

	err = json.Unmarshal(body, &vlanData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling VLAN data: %v\n", err)
		return "Error"
	}

	// Return tenant name or "None" if not assigned
	if vlanData.Tenant.Name != "" {
		return vlanData.Tenant.Name
	}
	return "None"
}

// Helper function to fetch prefixes for a VLAN using REST API
func fetchPrefixesForVLAN(vlanID int32) []string {
	// Construct the URL for fetching prefixes with vlan_id filter
	prefixesURL := fmt.Sprintf("%s/api/ipam/prefixes/?vlan_id=%d", inputDomainLogIn, vlanID)

	// Create a new request
	req, err := http.NewRequest("GET", prefixesURL, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return []string{}
	}

	// Add authorization header with your API token
	req.Header.Add("Authorization", "Token "+inputAPITokenLogIn) // Replace with your token

	// Create an HTTP client and make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		return []string{}
	}
	defer resp.Body.Close()

	// Check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response from NetBox: %s\n", string(bodyBytes))
		return []string{}
	}

	// Parse the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return []string{}
	}

	// Unmarshal the JSON response
	var prefixResponse struct {
		Results []struct {
			Prefix string `json:"prefix"`
		} `json:"results"`
	}
	err = json.Unmarshal(bodyBytes, &prefixResponse)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling response: %v\n", err)
		return []string{}
	}

	// Collect prefixes from the response
	var prefixList []string
	for _, prefix := range prefixResponse.Results {
		prefixList = append(prefixList, prefix.Prefix)
	}

	return prefixList
}

func getManufacturer() {
	apiUrl := inputDomainLogIn + "/api/dcim/manufacturers/"

	// Create an HTTP GET request
	req, err := http.NewRequestWithContext(context.Background(), "GET", apiUrl, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching manufacturers: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read and print the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return
	}

	// Print the raw JSON response (optional)
	fmt.Printf("Response: %s\n", string(body))

	// Parse JSON response to extract manufacturer information
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		return
	}

	listOfDeviceManufacturer = listOfDeviceManufacturer[:0]
	listOfDeviceManufacturerName = listOfDeviceManufacturerName[:0]

	listOfDeviceManufacturer = append(listOfDeviceManufacturer, 0)
	listOfDeviceManufacturerName = append(listOfDeviceManufacturerName, "None")

	if results, ok := result["results"].([]interface{}); ok {
		for _, r := range results {
			if manufacturer, ok := r.(map[string]interface{}); ok {
				if idFloat, ok := manufacturer["id"].(float64); ok {
					id := int(idFloat) // Convert float64 to int32
					listOfDeviceManufacturer = append(listOfDeviceManufacturer, id)
					listOfDeviceManufacturerName = append(listOfDeviceManufacturerName, manufacturer["name"].(string))
				}
			}
		}
	}
}

func getDeviceType() {
	apiUrl := inputDomainLogIn + "/api/dcim/device-types/"

	// Create an HTTP GET request
	req, err := http.NewRequestWithContext(context.Background(), "GET", apiUrl, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching device types: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return
	}

	// Print the raw JSON response (optional)
	fmt.Printf("Response: %s\n", string(body))

	// Parse JSON response to extract device type information
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		return
	}

	listOfDeviceType = listOfDeviceType[:0]
	listOfDeviceTypeName = listOfDeviceTypeName[:0]

	listOfDeviceType = append(listOfDeviceType, 0)
	listOfDeviceTypeName = append(listOfDeviceTypeName, "None")

	// Extract and print device type IDs, models, and manufacturer names
	if results, ok := result["results"].([]interface{}); ok {
		for _, r := range results {
			if deviceType, ok := r.(map[string]interface{}); ok {

				if idFloat, ok := deviceType["id"].(float64); ok {
					id := int(idFloat) // Convert float64 to int32
					listOfDeviceType = append(listOfDeviceType, id)
					listOfDeviceTypeName = append(listOfDeviceTypeName, deviceType["model"].(string))
				}
			}
		}
	}
}

func getDeviceRole() {
	apiUrl := inputDomainLogIn + "/api/dcim/device-roles/"

	// Create an HTTP GET request
	req, err := http.NewRequestWithContext(context.Background(), "GET", apiUrl, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching device roles: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return
	}

	// Print the raw JSON response (optional)
	fmt.Printf("Response: %s\n", string(body))

	// Parse JSON response to extract device role information
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		return
	}

	listOfDeviceRole = listOfDeviceRole[:0]
	listOfDeviceRoleName = listOfDeviceRoleName[:0]

	listOfDeviceRole = append(listOfDeviceRole, 0)
	listOfDeviceRoleName = append(listOfDeviceRoleName, "None")

	// Extract and print device role IDs and names
	if results, ok := result["results"].([]interface{}); ok {
		for _, r := range results {
			if role, ok := r.(map[string]interface{}); ok {

				if idFloat, ok := role["id"].(float64); ok {
					id := int(idFloat) // Convert float64 to int32
					listOfDeviceRole = append(listOfDeviceRole, id)
					listOfDeviceRoleName = append(listOfDeviceRoleName, role["name"].(string))
				}
			}
		}
	}
}

func getDeviceSite() {
	apiUrl := inputDomainLogIn + "/api/dcim/sites/"

	// Create an HTTP GET request
	req, err := http.NewRequestWithContext(context.Background(), "GET", apiUrl, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching sites: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return
	}

	// Print the raw JSON response (optional)
	fmt.Printf("Response: %s\n", string(body))

	// Parse JSON response to extract site information
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		return
	}

	listOfDeviceSite = listOfDeviceSite[:0]
	listOfDeviceSiteName = listOfDeviceSiteName[:0]

	listOfDeviceSite = append(listOfDeviceSite, 0)
	listOfDeviceSiteName = append(listOfDeviceSiteName, "None")

	// Extract and print site IDs and names
	if results, ok := result["results"].([]interface{}); ok {
		for _, r := range results {
			if site, ok := r.(map[string]interface{}); ok {
				if idFloat, ok := site["id"].(float64); ok {
					id := int(idFloat) // Convert float64 to int32
					listOfDeviceSite = append(listOfDeviceSite, id)
					listOfDeviceSiteName = append(listOfDeviceSiteName, site["name"].(string))
				}
			}
		}
	}
}

func buildDeviceRows() []*imgui.TableRowWidget {

	if timer <= 0.0 {

		getManufacturer()
		getDeviceSite()
		getDeviceType()
		getDeviceRole()

		// Set headers
		headers := []string{"Name", "Serial Number", "Tenant", "Site", "Manufacturer"}

		//Interface
		listOfDevice = listOfDevice[:0]
		listOfDeviceName = listOfDeviceName[:0]

		//Interface
		listOfDeviceName = append(listOfDeviceName, "None")
		listOfDevice = append(listOfDevice, 0)
		netboxURL := inputDomainLogIn + "/api/dcim/devices/?per_page=1000"

		// Create a new request
		req, err := http.NewRequest("GET", netboxURL, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		}

		// Add authentication header
		req.Header.Add("Authorization", "Token "+inputAPITokenLogIn)

		// Create an HTTP client and set a timeout
		client := &http.Client{}

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		}
		defer resp.Body.Close()

		// Check the response status
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "Error response from NetBox: %v\n", string(bodyBytes))
		}

		// Parse the response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		}

		// Unmarshal the JSON response
		var deviceList DeviceListResponse
		err = json.Unmarshal(bodyBytes, &deviceList)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error unmarshalling response: %v\n", err)
		}

		// Print the list of devices and their IDs
		for _, device := range deviceList.Results {
			listOfDevice = append(listOfDevice, device.Id)
			listOfDeviceName = append(listOfDeviceName, device.Display)
		}

		total := 0

		for i := 1; i < len(listOfDeviceName); i++ {
			if strings.Contains(listOfDeviceName[i], inputDeviceToSearchString) {
				total++
			}
		}

		rows = make([]*imgui.TableRowWidget, total+1)

		rows[0] = imgui.TableRow(
			imgui.Label(headers[0]),
			imgui.Label(headers[1]),
			imgui.Label(headers[2]),
			imgui.Label(headers[3]),
			imgui.Label(headers[4]),
		)

		rows[0].BgColor(&(color.RGBA{200, 100, 100, 255}))

		// Create a new Excel file
		f := excel.NewFile()
		sheetName := "Sheet1"
		index, _ := f.NewSheet(sheetName)

		// Create header row
		sheetHeaders := []string{
			"Name", "Type", "Site", "Tenant", "Label",
		}

		for i, header := range sheetHeaders {
			cell, _ := excel.CoordinatesToCellName(i+1, 1)
			f.SetCellValue(sheetName, cell, header)
		}

		// Fill data
		var i = 1
		for j := 1; j < len(listOfDeviceName); j++ {
			if strings.Contains(listOfDeviceName[j], inputDeviceToSearchString) {
				apiUrl := fmt.Sprintf(inputDomainLogIn+"/api/dcim/devices/%d/", listOfDevice[j])

				// Make the HTTP request
				req, err := http.NewRequestWithContext(context.Background(), "GET", apiUrl, nil)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
					continue
				}

				// Add authentication token (replace with your actual token)
				req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)

				// Perform the request
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error fetching device details: %v\n", err)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					fmt.Fprintf(os.Stderr, "Error: HTTP %v\n", resp.Status)
				}

				// Read the response body
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
				}

				// Parse the JSON response
				var deviceDetails DeviceDetails
				err = json.Unmarshal(body, &deviceDetails)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
				}

				rows[i] = imgui.TableRow(
					imgui.Label(listOfDeviceName[j]),
					imgui.Label(deviceDetails.Serial),
					imgui.Label(deviceDetails.Tenant.Display),
					imgui.Label(deviceDetails.Site.Display),
					imgui.Label(deviceDetails.DeviceType.Manufacturer.Display),
				)

				f.SetCellValue(sheetName, fmt.Sprintf("A%d", i+1), listOfDeviceName[j])
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", i+1), deviceDetails.DeviceType.Display)
				f.SetCellValue(sheetName, fmt.Sprintf("C%d", i+1), deviceDetails.Site)
				f.SetCellValue(sheetName, fmt.Sprintf("D%d", i+1), deviceDetails.Tenant)
				f.SetCellValue(sheetName, fmt.Sprintf("E%d", i+1), "Office")

				i++
			}
		}

		// Set the active sheet
		f.SetActiveSheet(index)

		// Save the file
		if err := f.SaveAs("devices_data.xlsx"); err != nil {
			log.Fatalf("Error saving file: %v\n", err)
		}

		fmt.Println("Excel file created successfully: devices_data.xlsx")

		timer = 50.0
	}

	return rows
}

func predictDevice() {

	file, err := os.Open("devices_data.csv")
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read all the records
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV file: %v", err)
	}

	// Ensure there are at least some records, including the header
	if len(records) == 0 {
		log.Fatal("No records found in the CSV file.")
	}
	header := records[0]
	expectedNumFields := len(header)

	// Ensure there's a header row
	if expectedNumFields == 0 {
		log.Fatal("No header fields found.")
	}

	// Prepare to create a dataset
	var filteredRecords [][]string

	for i, record := range records[1:] { // Skip header
		if len(record) != expectedNumFields {
			log.Printf("Skipping row %d: wrong number of fields", i+2)
			continue
		}
		filteredRecords = append(filteredRecords, record)
	}

	// Check if we have any valid records to parse
	if len(filteredRecords) == 0 {
		log.Fatal("No valid records found after filtering.")
	}

	// Create a DenseInstances object by directly parsing the CSV
	rawData, err := base.ParseCSVToInstances("devices_data.csv", true)
	if err != nil {
		log.Fatalf("Failed to parse CSV to instances: %v", err)
	}

	// Print a summary of the data
	fmt.Println(rawData)

	// Create a new ID3 Decision Tree with a 0.6 pruning factor
	decisionTree := trees.NewID3DecisionTree(0.6)

	// Split data into 80% training and 20% testing
	trainData, testData := base.InstancesTrainTestSplit(rawData, 0.8)

	// Check if the data split succeeded
	if trainData == nil || testData == nil {
		log.Fatal("Failed to split data into training and testing sets.")
	}

	// Train the decision tree model using the training data
	err = decisionTree.Fit(trainData)
	if err != nil {
		log.Fatal("Failed to train the model:", err)
	}

	// Make predictions on the test set
	predictions, err := decisionTree.Predict(testData)
	if err != nil {
		log.Fatal("Failed to predict on test data:", err)
	}

	// Get the confusion matrix to evaluate the model
	confusionMat, err := evaluation.GetConfusionMatrix(testData, predictions)
	if err != nil {
		log.Fatalf("Unable to get confusion matrix: %s", err)
	}

	// Print the evaluation summary (precision, recall, F1 score, etc.)
	fmt.Println(evaluation.GetSummary(confusionMat))
}

func addVLANConfirmation() {
	imgui.Msgbox("Confirmation", "Are you sure you want to add this VLAN?").Buttons(imgui.MsgboxButtonsYesNo).ResultCallback(func(result imgui.DialogResult) {
		switch result {
		case imgui.DialogResultYes:
			// Prepare the request body as a JSON payload
			vlanData := map[string]interface{}{
				"vid":         inputVLANVid,  // Numeric VLAN ID (1-4094)
				"name":        inputVLANName, // VLAN Name
				"description": inputVLANDesc, // Optional Description
			}

			// Add tenant if selected
			if tenantChoice != 0 {
				vlanData["tenant"] = listOfTenant[tenantChoice].Id
			}

			// Add site if selected
			/*if siteChoice != 0 {
				vlanData["site"] = listOfSite[siteChoice].Id
			}*/

			// Convert the data to JSON
			jsonData, err := json.Marshal(vlanData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshaling VLAN data: %v\n", err)
				return
			}

			// Create the HTTP request
			req, err := http.NewRequest("POST", inputDomainLogIn+"/api/ipam/vlans/", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
				return
			}

			// Add headers
			req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)
			req.Header.Set("Content-Type", "application/json")

			// Send the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			// Handle the response
			if resp.StatusCode != http.StatusCreated {
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stderr, "Error creating VLAN: %s\n", string(body))
				return
			}

			fmt.Println("VLAN successfully created")

			resetRefreshTimer() // Reset any timers if applicable

		case imgui.DialogResultNo:
			fmt.Println("No clicked")
		}

		showEnterVLANWindow = false // Close the VLAN input window after confirmation
	})
}

func addDeviceConfirmation() {
	imgui.Msgbox("Confirmation", "Are you sure?").Buttons(imgui.MsgboxButtonsYesNo).ResultCallback(func(result imgui.DialogResult) {
		switch result {
		case imgui.DialogResultYes:

			deviceData := DeviceRequest{
				Name:         inputDeviceName,
				DeviceType:   listOfDeviceType[deviceTypeChoice],
				DeviceRole:   listOfDeviceRole[deviceRoleChoice],
				Site:         listOfDeviceSite[deviceSiteChoice],
				Tenant:       int(listOfTenant[tenantChoice].Id),
				Manufacturer: listOfDeviceManufacturer[deviceManufacturerChoice],
				Status:       "active",                // Device status
				Serial:       inputDeviceSerialNumber, // Serial number
			}

			// Convert the device data to JSON
			jsonData, err := json.Marshal(deviceData)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error marshalling device data: %v\n", err)
				return
			}

			// NetBox API URL to create a new device
			apiUrl := inputDomainLogIn + "/api/dcim/devices/"

			// Create an HTTP POST request with the device data
			req, err := http.NewRequestWithContext(context.Background(), "POST", apiUrl, bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
				return
			}

			req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)
			req.Header.Set("Content-Type", "application/json")

			// Send the request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating device: %v\n", err)
				return
			}
			defer resp.Body.Close()

			// Check if the device was created successfully
			if resp.StatusCode != http.StatusCreated {
				// Print detailed error message from the response
				body, _ := io.ReadAll(resp.Body)
				fmt.Fprintf(os.Stderr, "Error: HTTP %v\nResponse: %s\n", resp.Status, string(body))
				return
			}

			fmt.Println("Device created successfully!")

			resetRefreshTimer()
		case imgui.DialogResultNo:
			fmt.Println("No clicked")
		}

		showEnterDeviceWindow = false
	})
}

func importDeviceFromCSV() {
	f, err := excel.OpenFile("DeviceToImport.xlsx")

	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	rows, err := f.GetRows("Sheet1")
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, row := range rows {

		deviceTypeIndex := 0

		for i := 0; i < len(listOfDeviceTypeName); i++ {
			//fmt.Println("Checking Type: " + listOfDeviceTypeName[i] + " == " + row[6])
			if strings.Contains(listOfDeviceTypeName[i], row[6]) {
				deviceTypeIndex = i
				break
			}
		}

		deviceRoleIndex := 0

		for i := 0; i < len(listOfDeviceRoleName); i++ {
			//fmt.Println("Checking Role: " + listOfDeviceRoleName[i] + " == " + row[4])
			if strings.Contains(listOfDeviceRoleName[i], row[4]) {
				deviceRoleIndex = i
				break
			}
		}

		deviceSiteIndex := 0

		for i := 0; i < len(listOfDeviceSiteName); i++ {
			//fmt.Println("Checking Site: " + listOfDeviceSiteName[i] + " == " + row[5])
			if strings.Contains(listOfDeviceSiteName[i], row[5]) {
				deviceSiteIndex = i
				break
			}
		}

		deviceTenantIndex := 0

		for i := 0; i < len(listOfTenantName); i++ {
			//fmt.Println("Checking Tenant: " + listOfTenantName[i] + " == " + row[2])
			if strings.Contains(listOfTenantName[i], row[2]) {
				deviceTenantIndex = i
				break
			}
		}

		deviceManufacturerIndex := 0

		for i := 0; i < len(listOfDeviceManufacturerName); i++ {
			//fmt.Println("Checking Manu: " + listOfDeviceManufacturerName[i] + " == " + row[3])
			if strings.Contains(listOfDeviceManufacturerName[i], row[3]) {
				deviceManufacturerIndex = i
				break
			}
		}

		if deviceManufacturerIndex == 0 || deviceRoleIndex == 0 || deviceSiteIndex == 0 || deviceTenantIndex == 0 || deviceTypeIndex == 0 {
			continue
		}

		deviceData := DeviceRequest{
			Name:         row[0],
			DeviceType:   listOfDeviceType[deviceTypeIndex],
			DeviceRole:   listOfDeviceRole[deviceRoleIndex],
			Site:         listOfDeviceSite[deviceSiteIndex],
			Tenant:       int(listOfTenant[deviceTenantIndex].Id),
			Manufacturer: listOfDeviceManufacturer[deviceManufacturerIndex],
			Status:       "active", // Device status
			Serial:       row[1],   // Serial number
		}

		// Convert the device data to JSON
		jsonData, err := json.Marshal(deviceData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshalling device data: %v\n", err)
			return
		}

		// NetBox API URL to create a new device
		apiUrl := inputDomainLogIn + "/api/dcim/devices/"

		// Create an HTTP POST request with the device data
		req, err := http.NewRequestWithContext(context.Background(), "POST", apiUrl, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
			return
		}

		req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating device: %v\n", err)
			return
		}
		defer resp.Body.Close()

		// Check if the device was created successfully
		if resp.StatusCode != http.StatusCreated {
			// Print detailed error message from the response
			body, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "Error: HTTP %v\nResponse: %s\n", resp.Status, string(body))
			return
		}

		fmt.Println("Device created successfully!")
	}

	resetRefreshTimer()
}

func logIn() {
	apiClient = openapiclient.NewAPIClientFor(inputDomainLogIn, inputAPITokenLogIn)
	//apiClient = openapiclient.NewAPIClientFor("https://netbox.cit.insea.io", "e3d318664caba8355bcea30a00237ae38c02b357")
	resp, _, err := apiClient.StatusAPI.StatusRetrieve(ctx).Execute()
	if err == nil {
		showLoggedIn = false
		resetRefreshTimer()
	}
	// response from `StatusRetrieve`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `StatusAPI.StatusRetrieve`: %v\n", resp)
}

func resetRefreshTimer() {
	timer = 0.0
}

func checkSubnet() {
	url := inputDomainLogIn + "/api/ipam/prefixes/?limit=3000"

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		return
	}

	// Add the Authorization header with your API token
	req.Header.Set("Authorization", "Token "+inputAPITokenLogIn)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making GET request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response body: %v\n", err)
		return
	}

	var apiResponse ApiResponse

	// Unmarshal JSON data
	err = json.Unmarshal([]byte(string(body)), &apiResponse)
	if err != nil {
		log.Fatalf("Error parsing JSON: %v\n", err)
		return
	}

	// Create a new Excel file
	f := excel.NewFile()
	sheetName := "Prefixes"
	index, _ := f.NewSheet(sheetName)

	// Create header row
	headers := []string{
		"ID", "URL", "Display URL", "Display", "Family Value",
		"Family Label", "Prefix", "Tenant Name", "Created", "Last Updated",
	}

	for i, header := range headers {
		cell, _ := excel.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Populate the sheet with data
	for rowIndex, prefix := range apiResponse.Results {
		row := rowIndex + 2 // Start from the second row
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), prefix.ID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), prefix.URL)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), prefix.DisplayURL)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), prefix.Display)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), prefix.Family.Value)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), prefix.Family.Label)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), prefix.Prefix)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), prefix.Tenant.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), prefix.Created)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), prefix.LastUpdated)
	}

	// Set the active sheet
	f.SetActiveSheet(index)

	// Save the file
	if err := f.SaveAs("prefixes.xlsx"); err != nil {
		log.Fatalf("Error saving file: %v\n", err)
		return
	}

	fmt.Println("Excel file created successfully: prefixes.xlsx")
	resetRefreshTimer()
}

func loop() {
	imgui.SingleWindow().Layout(
		imgui.PrepareMsgbox(),
		imgui.Row(
			imgui.Button("Devices").OnClick(func() {
				showDeviceScreen = true
				resetRefreshTimer()
			}),
			imgui.Button("Check Subnet Used").OnClick(checkSubnet),
			imgui.Button("Add New VLAN").OnClick(func() {
				showEnterVLANWindow = true
			}),
			imgui.Button("Refresh VLAN List").OnClick(resetRefreshTimer),
			imgui.InputText(&inputIPAddressToSearchString).Label("Input VLAN name To Search").Size(300),
		),
		imgui.Row(
			imgui.Label("IP Addresses"),
			imgui.Table().Freeze(0, 1).FastMode(true).Rows(buildRows()...),
		),
	)

	if showDeviceScreen {
		imgui.SingleWindow().IsOpen(&showDeviceScreen).Flags(imgui.WindowFlagsNone).Layout(
			imgui.Row(
				imgui.Button("IP Addresses").OnClick(func() {
					showDeviceScreen = false
					resetRefreshTimer()
				}),
				imgui.Button("Add New Device").OnClick(func() {
					showEnterDeviceWindow = true
				}),
				imgui.Button("Predict New Device Location").OnClick(predictDevice),
				imgui.Button("Import New Devices From CSV").OnClick(importDeviceFromCSV),
				imgui.Button("Refresh Device List").OnClick(resetRefreshTimer),
				imgui.InputText(&inputDeviceToSearchString).Label("Input Device To Search").Size(300),
			),
			imgui.Row(
				imgui.Label("Devices"),
				imgui.Table().Freeze(0, 1).FastMode(true).Rows(buildDeviceRows()...),
			),
		)
	}

	if showLoggedIn {
		imgui.SingleWindow().IsOpen(&showLoggedIn).Flags(imgui.WindowFlagsNone).Layout(
			imgui.InputText(&inputDomainLogIn).Label("Input Domain Address").Size(300),
			imgui.InputText(&inputAPITokenLogIn).Label("Input API Token").Size(300),
			imgui.Button("Log In").OnClick(logIn),
		)
	}

	if showEnterVLANWindow {
		imgui.Window("VLAN Input Window").IsOpen(&showEnterVLANWindow).Flags(imgui.WindowFlagsNone).Layout(
			imgui.InputText(&inputVLANName).Label("Input VLAN Name").Size(300),
			imgui.InputInt(&inputVLANVid).Label("Input VLAN ID").Size(300),
			imgui.InputText(&inputVLANDesc).Label("Input Description").Size(700),
			imgui.Combo("Tenants", listOfTenantName[tenantChoice], listOfTenantName, &tenantChoice).Size(300),
			//imgui.Combo("Sites", listOfSiteName[siteChoice], listOfSiteName, &siteChoice).Size(300),
			imgui.Button("Add VLAN").OnClick(addVLANConfirmation),
		)

	}

	if showEnterDeviceWindow {
		imgui.Window("Device Input Window").IsOpen(&showEnterDeviceWindow).Flags(imgui.WindowFlagsNone).Layout(
			imgui.InputText(&inputDeviceName).Label("Input Device Name").Size(300),
			imgui.InputText(&inputDeviceSerialNumber).Label("Input Serial Number").Size(300),
			imgui.Combo("Tenants", listOfTenantName[tenantChoice], listOfTenantName, &tenantChoice).Size(300),
			imgui.Combo("Manufacturer", listOfDeviceManufacturerName[deviceManufacturerChoice], listOfDeviceManufacturerName, &deviceManufacturerChoice).Size(300),
			imgui.Combo("Device Role", listOfDeviceRoleName[deviceRoleChoice], listOfDeviceRoleName, &deviceRoleChoice).Size(300),
			imgui.Combo("Device Site", listOfDeviceSiteName[deviceSiteChoice], listOfDeviceSiteName, &deviceSiteChoice).Size(300),
			imgui.Combo("Device Type", listOfDeviceTypeName[deviceTypeChoice], listOfDeviceTypeName, &deviceTypeChoice).Size(300),
			imgui.Button("Add Device").OnClick(addDeviceConfirmation),
		)
	}
}

func main() {
	ctx = context.Background()
	wnd := imgui.NewMasterWindow("IP Storage System", 1280, 720, imgui.MasterWindowFlagsFloating)
	wnd.Run(loop)
}
