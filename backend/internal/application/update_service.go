package application

import (
	"encoding/json"
	"net/http"
	"time"

	"calendaradvanced/internal/infrastructure/config"
)

type UpdateService struct {
	Config config.Config
}

type UpdateInfo struct {
	CurrentVersion   string `json:"currentVersion"`
	AvailableVersion string `json:"availableVersion,omitempty"`
	ReleaseNotes     string `json:"releaseNotes,omitempty"`
	ReleaseURL       string `json:"releaseUrl,omitempty"`
	UpdateCheckURL   string `json:"updateCheckUrl,omitempty"`
	UpdateAvailable  bool   `json:"updateAvailable"`
	CheckError       string `json:"checkError,omitempty"`
}

func (s *UpdateService) Check() UpdateInfo {
	info := UpdateInfo{CurrentVersion: s.Config.Version, UpdateCheckURL: s.Config.UpdateCheckURL}
	if s.Config.UpdateCheckURL == "" {
		info.CheckError = "update check url is not configured"
		return info
	}
	client := http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(s.Config.UpdateCheckURL)
	if err != nil {
		info.CheckError = err.Error()
		return info
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		info.CheckError = resp.Status
		return info
	}
	var release struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		info.CheckError = err.Error()
		return info
	}
	info.AvailableVersion = release.TagName
	info.ReleaseNotes = release.Body
	info.ReleaseURL = release.HTMLURL
	info.UpdateAvailable = release.TagName != "" && release.TagName != s.Config.Version
	return info
}
