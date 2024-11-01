package rancher

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type WorkloadResp struct {
	Name            string
	NamespaceID     string
	Containers      []Container
	PublicEndpoints []Endpoint
}

type Container struct {
	Image string
}

type Endpoint struct {
	Port int
}

func makeRequest(environment Environment, method, url string, payload []byte) (*http.Response, error) {
	baseURL := environment.BaseURL
	project := environment.Project
	fullURL := fmt.Sprintf("%s/project/%s/%s", baseURL, project, url)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	var body io.Reader
	if payload != nil {
		body = strings.NewReader(string(payload))
	}
	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(environment.username, environment.password)
	return client.Do(req)
}

func Scale(environment Environment, namespace string, workload string, replicas int) bool {

	service := workload
	if colonIndex := strings.LastIndex(workload, ":"); colonIndex > 0 {
		service = workload[colonIndex+1:]
	}

	fmt.Printf("%s\tscale:%d\t", service, replicas)

	payload := map[string]int{"scale": replicas}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := makeRequest(environment, "PUT", fmt.Sprintf("workloads/deployment:%s:%s", namespace, service), jsonPayload)

	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func Redeploy(environment Environment, namespace string, workload string) bool {

	service := workload
	if colonIndex := strings.LastIndex(workload, ":"); colonIndex > 0 {
		service = workload[colonIndex+1:]
	}

	fmt.Printf("%s\tredeploy\t", service)

	resp, err := makeRequest(environment, "POST", fmt.Sprintf("workloads/deployment:%s:%s?action=redeploy", namespace, service), nil)
	if err != nil {
		fmt.Println("失败")
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func GetConfigMaps(environment Environment, confPath string) (string, error) {
	resp, err := makeRequest(environment, "GET", fmt.Sprintf("configMaps/%s", confPath), nil)
	if err != nil {
		log.Printf("Error fetching nginx config: %v", err)
		return "", err
	}

	var configMap struct {
		Data struct {
			DefaultConf string `json:"default.conf"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&configMap); err != nil {
		log.Printf("Error decoding nginx config: %v", err)
		return "", err
	}
	resp.Body.Close()
	return configMap.Data.DefaultConf, nil
}

func GetWorkloadsList(environment Environment) ([]WorkloadResp, error) {

	resp, err := makeRequest(environment, "GET", "workloads?limit=-1", nil)
	if err != nil {
		log.Printf("Error fetching workloads: %v", err)
		return nil, err
	}

	var workloadsResponse struct {
		Data []WorkloadResp `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&workloadsResponse); err != nil {
		log.Printf("Error decoding workloads: %v", err)
		return nil, err
	}
	resp.Body.Close()
	return workloadsResponse.Data, nil
}