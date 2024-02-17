package options

import "time"

// Config describes all environment variables required for controllers
type Config struct {
	YandexCloudSAKey string        `env:"YC_SA_KEY,notEmpty"`
	ReconcileTimeout time.Duration `env:"CAPY_RECONCILE_TIMEOUT" envDefault:"1m"`
}
