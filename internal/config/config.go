package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Credential struct {
	Platform string `json:"platform"`
	Type     string `json:"type"`
	Value    string `json:"value"`
}

type VendorModel struct {
	Vendor string `json:"vendor"`
	Model  string `json:"model"`
}

func LoadCredentials(filePath string) ([]Credential, error) {
	filePath = filepath.Clean(filePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var creds []Credential
	err = json.Unmarshal(data, &creds)
	return creds, err
}

func LoadVendorModels(filePath string) ([]VendorModel, error) {
	filePath = filepath.Clean(filePath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var models []VendorModel
	err = json.Unmarshal(data, &models)
	return models, err
}
