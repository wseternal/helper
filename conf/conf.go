// conf abstract configuration interfaces
package conf

type Configurator interface {
	Get(uid string) (val interface{}, err error)
	Set(uid string, val interface{}) error
}

type NumberConfigurator interface {
	GetNumber(uid string) (float64, error)
	GetNumberDef(uid string, def float64) float64
	SetNumber(uid string, val float64) error
}

type StringConfigurator interface {
	GetString(uid string) (string, error)
	GetStringDef(uid string, def string) string
	SetString(uid string, val string) error
}

type SingleValueConfigurator interface {
	NumberConfigurator
	StringConfigurator
}
