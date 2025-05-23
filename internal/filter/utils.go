package filter

import "github.com/aashari/go-generative-api-router/internal/config"

// CredentialsByVendor filters credentials by vendor platform
func CredentialsByVendor(creds []config.Credential, vendor string) []config.Credential {
	var result []config.Credential
	for _, c := range creds {
		if c.Platform == vendor {
			result = append(result, c)
		}
	}
	return result
}

// ModelsByVendor filters models by vendor
func ModelsByVendor(models []config.VendorModel, vendor string) []config.VendorModel {
	var result []config.VendorModel
	for _, m := range models {
		if m.Vendor == vendor {
			result = append(result, m)
		}
	}
	return result
}
