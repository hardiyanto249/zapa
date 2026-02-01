// backend/telegram/api.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BackendClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewBackendClient(baseURL, apiKey string) *BackendClient {
	return &BackendClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *BackendClient) doRequest(method, endpoint string, body interface{}, authToken string) (*http.Response, error) {
	url := c.BaseURL + endpoint
	
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(jsonBody)
	}
	
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Bot-API-Key", c.APIKey) // Bot authentication
	
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	
	return c.HTTPClient.Do(req)
}

// ==================== AUTHENTICATION ====================

func (c *BackendClient) Login(volunteerCode, password string) (*BackendUser, error) {
	reqBody := LoginRequest{
		VolunteerCode: volunteerCode,
		Password:      password,
	}
	
	resp, err := c.doRequest("POST", "/login", reqBody, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("login failed: %s", string(body))
	}
	
	var user BackendUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}
	
	return &user, nil
}

// ==================== ZAKAT OPERATIONS ====================

func (c *BackendClient) GetZakatRecords(authToken string) ([]BackendZakat, error) {
	resp, err := c.doRequest("GET", "/zakat", nil, authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get zakat records: %d", resp.StatusCode)
	}
	
	var records []BackendZakat
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}
	
	return records, nil
}

func (c *BackendClient) AddZakatRecord(authToken string, req AddZakatRequest) (*BackendZakat, error) {
	resp, err := c.doRequest("POST", "/zakat", req, authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add zakat: %s", string(body))
	}
	
	var record BackendZakat
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, err
	}
	
	return &record, nil
}

func (c *BackendClient) UpdateZakatRecord(authToken string, id int, updates map[string]interface{}) error {
	reqBody := map[string]interface{}{
		"id":      id,
		"updates": updates,
	}
	
	resp, err := c.doRequest("PUT", "/zakat", reqBody, authToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update zakat: %s", string(body))
	}
	
	return nil
}

func (c *BackendClient) DeleteZakatRecord(authToken string, id int) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/zakat/%d", id), nil, authToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete zakat: %s", string(body))
	}
	
	return nil
}

// ==================== USER MANAGEMENT (ADMIN ONLY) ====================

func (c *BackendClient) GetAllUsers(authToken string) ([]BackendUser, error) {
	resp, err := c.doRequest("GET", "/users", nil, authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get users: %d", resp.StatusCode)
	}
	
	var users []BackendUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}
	
	return users, nil
}

func (c *BackendClient) AddUser(authToken string, user BackendUser, password string) error {
	reqBody := map[string]interface{}{
		"volunteerCode": user.VolunteerCode,
		"name":          user.Name,
		"lazName":       user.LazName,
		"description":   user.Description,
		"password":      password,
		"role":          "user",
	}
	
	resp, err := c.doRequest("POST", "/users", reqBody, authToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add user: %s", string(body))
	}
	
	return nil
}

func (c *BackendClient) UpdateUser(authToken string, volunteerCode string, updates map[string]interface{}) error {
	reqBody := map[string]interface{}{
		"volunteerCode": volunteerCode,
		"updates":       updates,
	}
	
	resp, err := c.doRequest("PUT", "/users", reqBody, authToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user: %s", string(body))
	}
	
	return nil
}

// ==================== LIVE CHAT ====================

func (c *BackendClient) RequestLiveChat(authToken string, user BackendUser) (map[string]interface{}, error) {
	reqBody := map[string]interface{}{
		"userVolunteerCode": user.VolunteerCode,
		"userName":          user.Name,
	}
	
	resp, err := c.doRequest("POST", "/chat/request", reqBody, authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to request live chat: %d", resp.StatusCode)
	}
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}

func (c *BackendClient) GetChatMessages(authToken string, sessionID int) ([]map[string]interface{}, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/chat/messages/%d", sessionID), nil, authToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var messages []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		return nil, err
	}
	
	return messages, nil
}

func (c *BackendClient) SendChatMessage(authToken string, sessionID int, message string, user BackendUser) error {
	reqBody := map[string]interface{}{
		"sessionId":  sessionID,
		"message":    message,
		"sender":     user.VolunteerCode,
		"senderName": user.Name,
	}
	
	resp, err := c.doRequest("POST", "/chat/send", reqBody, authToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return nil
}

func (c *BackendClient) CloseChatSession(authToken string, sessionID int) error {
	reqBody := map[string]interface{}{
		"sessionId": sessionID,
	}
	
	resp, err := c.doRequest("POST", "/chat/close", reqBody, authToken)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	return nil
}
