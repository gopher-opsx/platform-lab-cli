package platform

type Definition struct {
	Name             string   `yaml:"name"`
	Version          string   `yaml:"version"`
	ComposeFiles     []string `yaml:"compose_files"`
	RequiredServices []string `yaml:"required_services"`
}
