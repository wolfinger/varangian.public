// Package config holds config and other app / api level settings
package config

type apiFormats struct {
	DateFmt string
}

var (
	// APIFormats struct containing all formats related to interacting with the Varangian API
	APIFormats = apiFormats{
		DateFmt: "2006-01-02"}
)
