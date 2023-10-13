package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Type struct {
	Id string `json:"id"`
}
type Profile struct {
	LastName       string `json:"lastName"`
	ZipCode        string `json:"zipCode"`
	Manager        string `json:"manager"`
	City           string `json:"city"`
	DisplayName    string `json:"displayName"`
	NickName       string `json:"nickName"`
	SecondEmail    string `json:"secondEmail"`
	managerId      string `json:"managerId"`
	Title          string `json:"title"`
	Login          string `json:"login"`
	EmployeeNumber string `json:"employeeNumber"`
	Division       string `json:"division"`
	FirstName      string `json:"firstName"`
	PrimaryPhone   string `json:"primaryPhone"`
	PostalAddress  string `json:"postalAddress"`
	MobilePhone    string `json:"mobilePhone"`
	CountryCode    string `json:"countryCode"`
	MiddleName     string `json:"middleName"`
	UserType       string `json:"userType"`
	State          string `json:"state"`
	Department     string `json:"department"`
	Email          string `json:"email"`
}

type Credentials struct {
	Emails struct {
		Value  string `json:"value"`
		Status string `json:"status"`
		Type   string `json:"type"`
	} `json:"email"`
	Provider struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"provider"`
}

type _Links struct {
	Self struct {
		Href string `json:"href"`
	} `json:"self"`
}

type User struct {
	Id              string `json:"id"`
	Status          string `json:"status"`
	Created         string `json:"created"`
	Activated       string `json:"activated"`
	StatusChanged   string `json:"statusChanged"`
	LastLogin       string `json:"lastLogin"`
	LastUpdated     string `json:"lastUpdated"`
	PasswordChanged string `json:"passwordChanged"`
	Type            Type
	Profile         Profile
	Credentials     Credentials
	_Links          _Links
}

func fetchApiToken() string {
	if r := os.Getenv("OKTA_API_TOKEN"); r != "" {
		return r
	}
	return "nil"
}

type ResponseData struct {
	body    []byte
	nextUrl string
}

func oktaGetUsers(url string) (*ResponseData, error) {
	httpRequest, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	hosts := strings.SplitAfter(url, "/")
	host := hosts[2]

	apiKey := fetchApiToken()

	httpRequest.Header = http.Header{
		"Host":          {host},
		"Accept":        {"application/json"},
		"Content-Type":  {"application/json"},
		"Authorization": {"SSWS " + apiKey},
	}

	client := &http.Client{}
	httpResponse, err := client.Do(httpRequest)
	if err != nil {
		log.Fatalln(err)
	}
	defer httpResponse.Body.Close()

	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if !status2xx(httpResponse.StatusCode) {
		return nil, fmt.Errorf("Status %d %s %s", httpResponse.StatusCode, httpRequest.URL, body)
	}

	resp := &ResponseData{body: body}

	linkHeaderValue := httpResponse.Header["Link"]
	for k, _ := range linkHeaderValue {
		if linkHeaderValue[k] != "" {
			resp.nextUrl = getNextUrl(linkHeaderValue[k])
		}
	}

	return resp, nil
}

func status2xx(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

func getNextUrl(linkHeaderVal string) string {
	links := strings.Split(linkHeaderVal, ",")
	for i := range links {
		if strings.HasSuffix(links[i], `rel="next"`) {
			return strings.Trim(strings.Split(links[i], "; ")[0], "<>")
		}
	}

	return ""
}

func GetAllResponseData(initialUrl, limit string) ([][]byte, error) {
	var allData [][]byte

	url := initialUrl + limit

	for "" != url {
		// oktaGetUsers returns a ResponseData{body []byte, nextUrl string}
		resp, err := oktaGetUsers(url)
		if err != nil {
			return nil, err
		}

		allData = append(allData, resp.body)
		fmt.Println(url)
		url = resp.nextUrl
	}

	return allData, nil
}

func main() {
	var users []User
	resp, err := GetAllResponseData("https://auth.isovalent.com/api/v1/users?limit=", "200")
	if err != nil {
		log.Fatalln(err)
	}

	outputFile, err := os.Create("users.csv")
	if err != nil {
		log.Fatalln(err)
	}

	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	header := []string{"firstName", "lastName", "email", "status"}
	if err := writer.Write(header); err != nil {
		log.Fatalf("Error writing CSV header: %v", err)
	}
	for _, body := range resp {
		err := json.Unmarshal(body, &users)
		if err != nil {
			log.Fatalln(err)
		}
		for _, user := range users {
			if !strings.HasSuffix(user.Profile.Email, "isovalent.com") {
				record := []string{user.Profile.FirstName, user.Profile.LastName, user.Profile.Email, user.Status}
				if err := writer.Write(record); err != nil {
					log.Fatalf("Error writing CSV record: %v", err)
				}
			}

		}
	}
	writer.Flush()
	err = writer.Error()
	if err != nil {
		log.Fatalf("Error flushing CSV Writer: &v", err)
	}

}
