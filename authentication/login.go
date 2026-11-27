package authentication

import (
	"encoding/json"
	"fmt"
	"io"
	"myapp/models"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const redirect_uri string = "http://localhost:8080/auth/callback"

func TokenHandler(w http.ResponseWriter, r *http.Request) {
	apiURL := "https://api.upstox.com/v2/login/authorization/token"

	// Set the request data
	data := url.Values{}
	data.Set("code", r.URL.Query().Get("code"))
	data.Set("client_id", os.Getenv("API_KEY"))
	data.Set("client_secret", os.Getenv("API_SECRET"))
	data.Set("redirect_uri", redirect_uri)
	data.Set("grant_type", "authorization_code")

	// Create HTTP client
	client := &http.Client{}

	// Create HTTP request
	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set request headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	// Perform the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error performing request:", err)
		return 
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return 
	}

	// Parse the response body
	var tokenResp models.TokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		fmt.Println("Error parsing:", err)
		return 
	}

	// Save the access token in the environment
	fmt.Println("Access token:", tokenResp.AccessToken)
}


func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	url := "https://api.upstox.com/v2/logout"
	method := "DELETE"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	access_token := os.Getenv("ACCESS_TOKEN")
	req.Header.Add("Authorization", "Bearer " + access_token)
	req.Header.Add("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
	
	print("Access token was revoked successfully!")
}