package heroku

type Config struct {
	Build BuildConfig
}

type BuildConfig struct {
	Buildpacks []string
	Packages   []string
	Config     map[string]string
}

