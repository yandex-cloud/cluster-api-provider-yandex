package options

type Config struct {
	YandexCloudSAKey string `env:"YC_SA_KEY,notEmpty"`
}
