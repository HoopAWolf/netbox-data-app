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

var apiClient *openapiclient.APIClient
var ctx context.Context
var rows []*imgui.TableRowWidget
var timer float32 = 10.0
var showEnterIPAddressWindow bool = false
var showLoggedIn bool = false
var loggedIn bool = false
var inputIPAddressString string
var inputIPAddressDesc string
var inputIPAddressDNSName string = ""
var inputIPAddressToSearchString string = ""
var inputDomainLogIn string = "https://demo.netbox.dev"
var inputAPITokenLogIn string = ""

func buildRows() []*imgui.TableRowWidget {
	if !showEnterIPAddressWindow && loggedIn {
		timer -= 0.1
	}

	if timer <= 0.0 {

		// Fetch all IP addresses
		availableIPs, _, err := apiClient.IpamAPI.IpamIpAddressesList(ctx).Limit(6000).Execute()

		if err != nil {
			log.Fatal(err)
		}
		// Set headers
		headers := []string{"ID", "Address", "Status", "Tenant", "Assigned", "DNS Name"}

		var total = 0
		for _, ip := range availableIPs.Results {
			if strings.Contains(ip.Address, inputIPAddressToSearchString) {
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
			imgui.Label(headers[5]),
		)

		rows[0].BgColor(&(color.RGBA{200, 100, 100, 255}))

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

func addIPAddressConfirmation() {
	imgui.Msgbox("Confirmation", "Are you sure?").Buttons(imgui.MsgboxButtonsYesNo).ResultCallback(func(result imgui.DialogResult) {
		switch result {
		case imgui.DialogResultYes:
			statusOfNewIP := openapiclient.PATCHEDWRITABLEIPADDRESSREQUESTSTATUS_ACTIVE

			ipAddressRequest := openapiclient.WritableIPAddressRequest{
				Address:     inputIPAddressString,   // IP address with CIDR notation
				Status:      &statusOfNewIP,         // Status of the IP address
				DnsName:     &inputIPAddressDNSName, //DNS name
				Description: &inputIPAddressDesc,    // Optional description
			}

			resp, _, _ := apiClient.IpamAPI.IpamIpAddressesCreate(context.Background()).WritableIPAddressRequest(ipAddressRequest).Execute()

			fmt.Println(resp)
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
		loggedIn = true
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
	url := "https://demo.netbox.dev/api/ipam/prefixes/?limit=3000"

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

	// Print the response (list of prefixes)
	fmt.Println(string(body))

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
}

func loop() {
	imgui.MainMenuBar().Layout(
		imgui.Menu("File").Layout(
			imgui.MenuItem("Open").OnClick(func() {
				showLoggedIn = true
			}),
			imgui.Separator(),
			imgui.MenuItem("Exit"),
		),
	).Build()

	imgui.SingleWindow().Layout(
		imgui.PrepareMsgbox(),
		imgui.Row(
			imgui.Button("Log In").OnClick(func() {
				showLoggedIn = true
			}),
			imgui.Button("Add New IP Address").OnClick(func() {
				showEnterIPAddressWindow = true
			}),
			imgui.Button("Check Subnet Used").OnClick(checkSubnet),
			imgui.Button("Refresh IP Address List").OnClick(resetRefreshTimer),
			imgui.InputText(&inputIPAddressToSearchString).Label("Input IP Address To Search").Size(300),
		),
		imgui.Row(
			imgui.Label("IP Addresses"),
			imgui.Table().Freeze(0, 1).FastMode(true).Rows(buildRows()...),
		),
	)

	if showLoggedIn {
		imgui.Window("Log In Window").IsOpen(&showLoggedIn).Flags(imgui.WindowFlagsNone).Layout(
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
			imgui.Button("Add IP Address").OnClick(addIPAddressConfirmation),
		)
	}
}

func main() {
	ctx = context.Background()
	wnd := imgui.NewMasterWindow("IP Storage System", 1280, 720, imgui.MasterWindowFlagsFloating)
	wnd.Run(loop)
}
