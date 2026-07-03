package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (app *Application) broker(w http.ResponseWriter, r *http.Request) {
	payload := JsonResponse{
		Error:   false,
		Message: "Broker Service is running",
	}

	err := app.writeJSON(w, http.StatusOK, payload)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (app *Application) handleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth-login":
		app.login(w, requestPayload.Auth)
	default:
		app.errorJSON(w, errors.New("Invalid action"))
	}
}

func (app *Application) login(w http.ResponseWriter, p AuthPayload) {
	jsonData, _ := json.MarshalIndent(p, "", "\t")

	request, err := http.NewRequest("POST", "http://auth:8080/auth/login", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("Invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("Error calling auth service"))
		return
	}

	var serviceResponse JsonResponse
	err = json.NewDecoder(response.Body).Decode(&serviceResponse)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if serviceResponse.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	payload := JsonResponse{
		Error:   false,
		Message: "Authorized",
		Data:    serviceResponse.Data,
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
