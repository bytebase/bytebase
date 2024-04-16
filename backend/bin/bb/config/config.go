package config

type Config struct {
	URL   string
	Token string
}

func New() (*Config, error) {
	return &Config{
		URL:   "",
		Token: "",
	}, nil
}
