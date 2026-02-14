package registry

var providers = make(map[string]interface{})

func RegisterProvider(name string, factory interface{}) {
	providers[name] = factory
}

func GetProviders() map[string]interface{} {
	return providers
}
