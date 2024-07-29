package storage

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"database/sql"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var bearerToken = os.Getenv("API_TOKEN")

func GetKeys(name string) (string, string) {
	userID, err := GetUserIdByName(name)
	if err != nil {
		fmt.Println(err)
		return "", ""
	}

	url := "https://api.clo.ru/v2/s3/users/" + userID + "/credentials"

	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", ""
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", ""
	}

	type Result struct {
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
	}

	type Response struct {
		Count  int      `json:"count"`
		Result []Result `json:"result"`
	}

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return "", ""
	}

	return response.Result[0].AccessKey, response.Result[0].SecretKey
}

func GetProjectId() (string, error) {
	url := "https://api.clo.ru/v2/projects"

	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	type Instance struct {
		StoppingReason *string `json:"stopping_reason"`
		Name           string  `json:"name"`
		Status         string  `json:"status"`
		HasAbuse       bool    `json:"has_abuse"`
		ID             string  `json:"id"`
		CreatedIn      string  `json:"created_in"`
	}

	type Response struct {
		Result []Instance `json:"result"`
		Count  int        `json:"count"`
	}

	// Parse the JSON response
	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Check if we have results and return the first ID
	if len(response.Result) > 0 {
		return response.Result[0].ID, nil
	}

	return "", fmt.Errorf("no project found in response")
}

func GetUserIdByName(projectName string) (string, error) {
	projectID, err := GetProjectId()
	if err != nil {
		fmt.Println(err)
	}
	url := "https://api.clo.ru/v2/projects/" + projectID + "/s3/users"

	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Set the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearerToken)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}
	type Quota struct {
		Type       string `json:"type"`
		MaxObjects *int   `json:"max_objects"`
		MaxSize    int    `json:"max_size"`
	}

	type Project struct {
		CanonicalName string  `json:"canonical_name"`
		Tenant        *string `json:"tenant"`
		MaxBuckets    int     `json:"max_buckets"`
		Name          string  `json:"name"`
		Quotas        []Quota `json:"quotas"`
		Status        string  `json:"status"`
		ID            string  `json:"id"`
	}

	type ProjectResponse struct {
		Result []Project `json:"result"`
		Count  int       `json:"count"`
	}
	// Parse the JSON response
	var response ProjectResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("error unmarshalling response: %v", err)
	}

	// Loop through the results and find the project by name
	for _, project := range response.Result {
		if project.Name == projectName {
			return project.ID, nil
		}
	}

	return "", fmt.Errorf("project with name %s not found", projectName)
}

func CheckUser(login, token string) bool {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	connect_db := "host=" + viper.GetString("db.host") + " " + "user=" + viper.GetString("db.username") + " " + "port=" + viper.GetString("db.port") + " " + "password=" + viper.GetString("db.password") + " " + "dbname=" + viper.GetString("db.dbname") + " " + "sslmode=" + viper.GetString("db.sslmode")
	db, err := sql.Open("postgres", connect_db)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		fmt.Println(err)
		return false
	}

	var tok string
	err = db.QueryRow("SELECT token FROM Person WHERE login = $1", login).Scan(&tok)
	if err != nil {
		return false
	}

	if token == tok {
		return true
	} else {
		return false
	}
}
