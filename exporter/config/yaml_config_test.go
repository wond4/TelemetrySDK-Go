package config

import "testing"

func TestLoadLogConfig(t *testing.T) {
	//tests := []struct {
	//	name string
	//}{
	//	// TODO: Add test cases.
	//}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		LoadLogConfig()
	//	})
	//}

	LogVP.AddConfigPath("C:\\Users\\frank.liu01\\GolandProjects\\TelemetrySDK-Go\\examples\\api_service\\")
	LogVP.SetConfigName("ob-app-config-log")
	LogVP.SetConfigType("yaml")

	LoadLogConfig()
}
