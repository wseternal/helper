package ram

import (
	"sync"
	"fmt"
	"reflect"
	"strconv"
)

type Conf struct {
	elem map[string]interface{}
	sync.RWMutex
}

func NewConf() *Conf {
	return &Conf{
		elem: make(map[string]interface{}),
	}
}

func (conf *Conf) Get(uid string) (interface{}, error) {
	conf.RLock()
	defer conf.RUnlock()
	if elem, found := conf.elem[uid]; found {
		return elem, nil
	} else {
		return nil, fmt.Errorf("no conf entry: %s", uid)
	}
}

func (conf *Conf) Set(uid string, val interface{}) error {
	conf.Lock()
	defer conf.Unlock()
	conf.elem[uid] = val
	return nil
}

func (conf *Conf) GetString(uid string) (string, error) {
	val, err := conf.Get(uid)
	if err != nil {
		return "", err
	}
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.String {
		return "", fmt.Errorf("inivalid type %s for %s, require string", v.Kind().String(), uid)
	}
	return v.String(), nil
}

func (conf *Conf) GetStringDef(uid string, def string) string {
	val, err := conf.GetString(uid)
	if err != nil {
		return def
	}
	return val
}

func (conf *Conf) SetString(uid string, val string) error {
	return conf.Set(uid, val)
}

func (conf *Conf) GetNumber(uid string) (float64, error) {
	val, err := conf.Get(uid)
	if err != nil {
		return 0, err
	}
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return v.Float(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), nil
	case reflect.String:
		return strconv.ParseFloat(v.String(), 64)
	default:
		return 0, fmt.Errorf("invalid type %s for %s", v.Kind().String(), uid)
	}
}

func (conf *Conf) GetNumberDef(uid string, def float64) float64 {
	val, err := conf.GetNumber(uid)
	if err != nil {
		return def
	}
	return val
}

func (conf *Conf) SetNumber(uid string, val float64) error {
	return conf.Set(uid, val)
}
