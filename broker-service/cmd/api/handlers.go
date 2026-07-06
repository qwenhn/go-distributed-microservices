package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/rpc"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"broker/lib/event"
	"broker/logs"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type RPCPayload struct {
	Name string
	Data string
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

	case "log":
		app.log(w, requestPayload.Log)

	case "mail":
		app.mail(w, requestPayload.Mail)

	case "log-mq":
		app.logViaRabbitMQ(w, requestPayload.Log)

	case "log-rpc":
		app.logViaRPC(w, requestPayload.Log)

	case "log-grpc":
		app.logViaGRPC(w, requestPayload.Log)

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

func (app *Application) log(w http.ResponseWriter, p LogPayload) {
	jsonData, _ := json.MarshalIndent(p, "", "\t")

	request, err := http.NewRequest("POST", "http://logger:8080/log", bytes.NewBuffer(jsonData))
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

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, err)
		return
	}

	payload := JsonResponse{
		Error:   false,
		Message: "Logged",
	}

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (app *Application) mail(w http.ResponseWriter, p MailPayload) {
	jsonData, _ := json.MarshalIndent(p, "", "\t")

	request, err := http.NewRequest("POST", "http://mailer:8080/send", bytes.NewBuffer(jsonData))
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

	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling mailer service"))
		return
	}

	var payload JsonResponse
	payload.Error = false
	payload.Message = "Message sent to " + p.To

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (app *Application) logViaRabbitMQ(w http.ResponseWriter, p LogPayload) {
	err := app.pushToAMQP(p.Name, p.Data)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload JsonResponse
	payload.Error = false
	payload.Message = "Logged via RabbitMQ"

	err = app.writeJSON(w, http.StatusAccepted, payload)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (app *Application) pushToAMQP(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.RabbitConn)
	if err != nil {
		return err
	}

	payload := LogPayload{
		Name: name,
		Data: msg,
	}

	jsonData, _ := json.MarshalIndent(payload, "", "\t")

	err = emitter.Push(string(jsonData), "log.INFO")
	if err != nil {
		return err
	}

	return nil
}

func (app *Application) logViaRPC(w http.ResponseWriter, p LogPayload) {
	client, err := rpc.Dial("tcp", "logger:5001")
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	rpcPayload := RPCPayload{
		Name: p.Name,
		Data: p.Data,
	}

	var result string
	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := JsonResponse{
		Error:   false,
		Message: result,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Application) logViaGRPC(w http.ResponseWriter, p LogPayload) {
	conn, err := grpc.NewClient("logger:50001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer conn.Close()

	client := logs.NewLogServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: p.Name,
			Data: p.Data,
		},
	}

	response, err := client.WriteLog(ctx, request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := JsonResponse{
		Error:   false,
		Message: response.Result,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
