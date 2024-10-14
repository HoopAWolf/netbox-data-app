package main

import (
	"context"
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
    ID          int    `json:"id"`
    Name        string `json:"name"`
    DeviceRole  struct {
        Display string `json:"display"`
    } `json:"device_role"`
    DeviceType struct {
        Display     string `json:"display"`
        Manufacturer struct {
            Display string `json:"display"`
        } `json:"manufacturer"`
    } `json:"device_type"`
    Status struct {
        Value string `json:"value"`
    } `json:"status"`
    Serial      string `json:"serial"`
    Tenant      struct {
        Display string `json:"display"`
    } `json:"tenant"`
    Site        struct {
        Display string `json:"display"`
    } `json:"site"`
}

var apiClient *openapiclient.APIClient
var ctx context.Context
var rows []*imgui.TableRowWidget
var timer float32 = 10.0
var showEnterIPAddressWindow bool = false
var showLoggedIn bool = true
var showDeviceScreen bool = false
var inputIPAddressString string
var inputIPAddressDesc string
var inputIPAddressDNSName string = ""
var inputIPAddressToSearchString string = ""
var inputDeviceToSearchString string = ""
var inputDomainLogIn string = "https://demo.netbox.dev"
var inputAPITokenLogIn string = ""
var listOfTenant []openapiclient.Tenant = make([]openapiclient.Tenant, 0)
var listOfTenantName []string = make([]string, 0)
var listOfDevice []int = make([]int, 0)
var listOfDeviceName []string = make([]string, 0)
var tenantChoice int32 = 0
var deviceChoice int32 = 0

func buildRows() []*imgui.TableRowWidget {

	if timer <= 0.0 && !showDeviceScreen {

		// Fetch all IP addresses
		availableIPs, _, err := apiClient.IpamAPI.IpamIpAddressesList(ctx).Limit(6000).Execute()

		if err != nil {
			log.Fatal(err)
		}
		// Set headers
		headers := []string{"ID", "Address", "Role", "Status", "Tenant", "Assigned", "DNS Name"}

		var total = 0
		for _, ip := range availableIPs.Results {
			if strings.Contains(ip.Address, inputIPAddressToSearchString) {
				total++
			}
		}

		listOfDevice = listOfDevice[:0]
		listOfDeviceName = listOfDeviceName[:0]
		listOfTenant = listOfTenant[:0]
		listOfTenantName = listOfTenantName[:0]

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

		rows = make([]*imgui.TableRowWidget, total+1)

		rows[0] = imgui.TableRow(
			imgui.Label(headers[0]),
			imgui.Label(headers[1]),
			imgui.Label(headers[2]),
			imgui.Label(headers[3]),
			imgui.Label(headers[4]),
			imgui.Label(headers[5]),
			imgui.Label(headers[6]),
		)

		rows[0].BgColor(&(color.RGBA{200, 100, 100, 255}))

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

		// Fill data
		var i = 1
		for _, ip := range availableIPs.Results {
			if strings.Contains(ip.Address, inputIPAddressToSearchString) {
				ipData, _ := ip.ToMap()

				id := fmt.Sprintf("%d", ip.Id)
				status := fmt.Sprintf("%d", ipData[`status`])
				tenant := fmt.Sprintf("%d", ipData[`tenant`])
				assigned := fmt.Sprintf("%d", ipData[`assigned_object`])
				dnsName := fmt.Sprintf("%d", ipData[`dns_name`])
				roleName := fmt.Sprintf("%d", ipData[`role`])

				if strings.Contains(status, "Active") {
					status = "Active"
				} else if strings.Contains(status, "Deprecated") {
					status = "Deprecated"
				} else {
					status = "Reserved"
				}

				if strings.Contains(tenant, "nil") {
					tenant = "Nil"
				} else {
					tenant = strings.Trim(tenant, "map[%!d(string=description):%!d(string=) %!d(string=display):%!d(string=) %!d(string=id):%!d(float64=5) %!d(string=name):%!d(string=) %!d(string=slug):%!d(string=dunder-mifflin) %!d(string=url):%!d(string=https://demo.netbox.dev/api/tenancy/tenants/5/)]")
					temp := strings.Split(tenant, ")")
					tenant = temp[0]
				}

				if strings.Contains(roleName, "nil") {
					roleName = "Nil"
				} else {
					roleName = strings.Trim(roleName, "map[%!d(string=label):%!d(string=) %!d(string=value):%!d(string=)]")
					temp := strings.Split(roleName, ")")
					roleName = temp[0]
				}

				if strings.Contains(assigned, "true") {
					assigned = "True"
				} else {
					assigned = "False"
				}

				if len(dnsName) > 12 {
					dnsName = strings.Trim(dnsName, "%!d(string=")
					dnsName = strings.Trim(dnsName, ")")
				} else {
					dnsName = "Nil"
				}

				rows[i] = imgui.TableRow(
					imgui.Label(id),
					imgui.Label(ip.Address),
					imgui.Label(roleName),
					imgui.Label(status),
					imgui.Label(tenant),
					imgui.Label(assigned),
					imgui.Label(dnsName),
				)
				i++
			}
		}

		timer = 50.0
	}

	return rows
}

func buildDeviceRows() []*imgui.TableRowWidget {

	if timer <= 0.0 {

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

		rows = make([]*imgui.TableRowWidget, total + 1)

		rows[0] = imgui.TableRow(
			imgui.Label(headers[0]),
			imgui.Label(headers[1]),
			imgui.Label(headers[2]),
			imgui.Label(headers[3]),
			imgui.Label(headers[4]),
		)

		rows[0].BgColor(&(color.RGBA{200, 100, 100, 255}))

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
				i++
			}
		}

		timer = 50.0
	}

	return rows
}

func addIPAddressConfirmation() {
	imgui.Msgbox("Confirmation", "Are you sure?").Buttons(imgui.MsgboxButtonsYesNo).ResultCallback(func(result imgui.DialogResult) {
		switch result {
		case imgui.DialogResultYes:
			statusOfNewIP := openapiclient.PATCHEDWRITABLEIPADDRESSREQUESTSTATUS_ACTIVE

			if tenantChoice != 0 {
				tenantRequest := openapiclient.NewNullableTenantRequest(&openapiclient.TenantRequest{
					Name: listOfTenant[tenantChoice].Name,
					Slug: listOfTenant[tenantChoice].Slug,
				})

				ipAddressRequest := openapiclient.WritableIPAddressRequest{
					Address:     inputIPAddressString,   // IP address with CIDR notation
					Tenant:      *tenantRequest,         // Tenant Name
					Status:      &statusOfNewIP,         // Status of the IP address
					DnsName:     &inputIPAddressDNSName, //DNS name
					Description: &inputIPAddressDesc,    // Optional description

				}

				_, httpResp, err := apiClient.IpamAPI.IpamIpAddressesCreate(context.Background()).WritableIPAddressRequest(ipAddressRequest).Execute()

				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating IP address: %v\n", err)
					if httpResp != nil {
						body, _ := io.ReadAll(httpResp.Body)
						fmt.Printf("HTTP Response: %s\n", string(body))
					}

					return
				}
			} else {
				ipAddressRequest := openapiclient.WritableIPAddressRequest{
					Address:     inputIPAddressString,   // IP address with CIDR notation
					Status:      &statusOfNewIP,         // Status of the IP address
					DnsName:     &inputIPAddressDNSName, //DNS name
					Description: &inputIPAddressDesc,    // Optional description
				}

				resp, _, _ := apiClient.IpamAPI.IpamIpAddressesCreate(context.Background()).WritableIPAddressRequest(ipAddressRequest).Execute()
				fmt.Println(resp)
			}

			resetRefreshTimer()
		case imgui.DialogResultNo:
			fmt.Println("No clicked")
		}

		showEnterIPAddressWindow = false
	})
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
			imgui.Button("Add New IP Address").OnClick(func() {
				showEnterIPAddressWindow = true
			}),
			imgui.Button("Refresh IP Address List").OnClick(resetRefreshTimer),
			imgui.InputText(&inputIPAddressToSearchString).Label("Input IP Address To Search").Size(300),
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
					showEnterIPAddressWindow = true
				}),
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

	if showEnterIPAddressWindow {
		imgui.Window("IPAddress Input Window").IsOpen(&showEnterIPAddressWindow).Flags(imgui.WindowFlagsNone).Layout(
			imgui.InputText(&inputIPAddressString).Label("Input IP Address").Size(300),
			imgui.InputText(&inputIPAddressDNSName).Label("Input DNS Name").Size(300),
			imgui.InputText(&inputIPAddressDesc).Label("Input Desceiption").Size(700),
			imgui.Combo("Tenants", listOfTenantName[tenantChoice], listOfTenantName, &tenantChoice).Size(300),
			imgui.Combo("Interface", listOfDeviceName[deviceChoice], listOfDeviceName, &deviceChoice).Size(300),
			imgui.Button("Add IP Address").OnClick(addIPAddressConfirmation),
		)
	}
}

func main() {
	ctx = context.Background()
	wnd := imgui.NewMasterWindow("IP Storage System", 1280, 720, imgui.MasterWindowFlagsFloating)
	wnd.Run(loop)
}
