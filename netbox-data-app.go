package main

import (
	"context"
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"

	imgui "github.com/AllenDang/giu"
	openapiclient "github.com/netbox-community/go-netbox/v4"
	//excel "github.com/xuri/excelize/v2"
)

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
	subnetList, _, err := apiClient.IpamAPI.IpamPrefixesList(context.Background()).Limit(1000).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching subnets: %v\n", err)
		return
	}

	for _, subnet := range subnetList.Results {
		fmt.Printf("Allocated Subnet: %s\n", subnet.Prefix)
	}
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
