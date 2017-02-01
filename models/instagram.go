package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/hieven/go-instagram/constants"
	"github.com/hieven/go-instagram/utils"
	"github.com/parnurzeal/gorequest"
)

type Instagram struct {
	Username      string
	Password      string
	LoginInterval int
	loggedInUser
	AgentPool    *utils.SuperAgentPool
	Inbox        *Inbox
	TimelineFeed *TimelineFeed
}

type DefaultResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type loggedInUser struct {
	Pk int64 `json:"pk"`
}

type loginRequest struct {
	SignedBody      string `json:"signed_body"`
	IgSigKeyVersion string `json:"ig_sig_key_version"`
}

type loginResponse struct {
	LoggedInUser loggedInUser `json:"logged_in_user"`
	DefaultResponse
}

type likeRequest struct {
	MediaID string `json:"media_id"`
	Src     string `json:"src"`
	loginRequest
}

type likeResponse struct {
	DefaultResponse
}

func (ig *Instagram) Login() error {
	for i := 0; i < ig.AgentPool.Capacity; i++ {
		igSigKeyVersion, signedBody := ig.CreateSignature()

		payload := loginRequest{
			IgSigKeyVersion: igSigKeyVersion,
			SignedBody:      signedBody,
		}

		jsonData, _ := json.Marshal(payload)

		url := constants.GetURL("Login", nil)

		agent := ig.AgentPool.Get()
		defer ig.AgentPool.Put(agent)

		_, body, _ := ig.SendRequest(agent.Post(url).
			Type("multipart").
			Send(string(jsonData)))

		var resp loginResponse
		json.Unmarshal([]byte(body), &resp)

		if resp.Status == "fail" {
			return errors.New(resp.Message)
		}

		// store user info
		ig.Pk = resp.LoggedInUser.Pk

		time.Sleep(time.Duration(ig.LoginInterval) * time.Millisecond)
	}

	return nil
}

func (ig *Instagram) Like(mediaID string) error {
	url := constants.GetURL("Like", struct{ ID string }{ID: mediaID})

	igSigKeyVersion, signedBody := ig.CreateSignature()

	payload := likeRequest{
		MediaID: mediaID,
		Src:     "profile",
	}
	payload.IgSigKeyVersion = igSigKeyVersion
	payload.SignedBody = signedBody

	jsonData, _ := json.Marshal(payload)

	agent := ig.AgentPool.Get()
	defer ig.AgentPool.Put(agent)

	_, body, _ := ig.SendRequest(agent.Post(url).
		Type("multipart").
		Send(string(jsonData)))

	var resp loginResponse
	json.Unmarshal([]byte(body), &resp)

	if resp.Status == "fail" {
		return errors.New(resp.Message)
	}

	return nil
}

func (ig *Instagram) Unlike(mediaID string) error {
	url := constants.GetURL("Unlike", struct{ ID string }{ID: mediaID})

	igSigKeyVersion, signedBody := ig.CreateSignature()

	payload := likeRequest{
		MediaID: mediaID,
		Src:     "profile",
	}
	payload.IgSigKeyVersion = igSigKeyVersion
	payload.SignedBody = signedBody

	jsonData, _ := json.Marshal(payload)

	agent := ig.AgentPool.Get()
	defer ig.AgentPool.Put(agent)

	_, body, _ := ig.SendRequest(agent.Post(url).
		Type("multipart").
		Send(string(jsonData)))

	var resp loginResponse
	json.Unmarshal([]byte(body), &resp)

	if resp.Status == "fail" {
		return errors.New(resp.Message)
	}

	return nil
}

func (ig *Instagram) CreateSignature() (sigVersion string, signedBody string) {
	data := struct {
		Csrftoken         string `json:"_csrftoken"`
		DeviceID          string `json:"device_id"`
		UUID              string `json:"_uuid"`
		UserName          string `json:"username"`
		Password          string `json:"password"`
		LoginAttemptCount int    `json:"login_attempt_count"`
	}{
		Csrftoken:         "missing",
		DeviceID:          "android-b256317fd493b848",
		UUID:              utils.GenerateUUID(),
		UserName:          ig.Username,
		Password:          ig.Password,
		LoginAttemptCount: 0,
	}

	jsonData, _ := json.Marshal(data)

	return utils.GenerateSignature(jsonData)
}

func (ig *Instagram) SendRequest(agent *gorequest.SuperAgent) (gorequest.Response, string, []error) {
	return agent.
		Set("Connection", "close").
		Set("Accept", "*/*").
		Set("X-IG-Connection-Type", "WIFI").
		Set("X-IG-Capabilities", "3QI=").
		Set("Accept-Language", "en-US").
		Set("Host", constants.HOSTNAME).
		Set("User-Agent", "Instagram "+constants.APP_VERSION+" Android (21/5.1.1; 401dpi; 1080x1920; Oppo; A31u; A31u; en_US)").
		End()
}
