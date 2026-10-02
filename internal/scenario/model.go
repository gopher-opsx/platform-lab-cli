package scenario

type Scenario struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	Course int `yaml:"course"`
	Lesson int `yaml:"lesson"`

	Platform PlatformRequirement `yaml:"platform"`

	Preflight Preflight `yaml:"preflight"`

	Inject []Action `yaml:"inject"`

	VerifyFailure []Check `yaml:"verify_failure"`

	Rollback []Action `yaml:"rollback"`

	VerifyRecovery []Check `yaml:"verify_recovery"`
}

type PlatformRequirement struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Preflight struct {
	RequireHealthy []string `yaml:"require_healthy"`
}

type Action struct {
	Type    string `yaml:"type"`
	Target  string `yaml:"target,omitempty"`
	Network string `yaml:"network,omitempty"`
}

type Check struct {
	Type     string `yaml:"type"`
	Target   string `yaml:"target"`
	Expected string `yaml:"expected"`
}
