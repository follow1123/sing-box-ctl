package provider

type SingBoxCtlConfig struct {
	DefaultProvider string           `json:"default_provider,omitempty"`
	Providers       []ProviderConfig `json:"providers"`
}

type ProviderConfig struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}
