package uci

import (
	"fmt"
	"github.com/wseternal/helper/conf"
	"strconv"
	"strings"
	"uci"
)

type Conf struct {
	conf.Configurator
	conf.SingleValueConfigurator
	Ctx *uci.Context
}

// NewConf create new uci based configurator
// param ctx: using specified uci context, if nil, create a new one
func NewConf(ctx *uci.Context) (*Conf, error) {
	var err error
	if ctx == nil {
		ctx, err = uci.Cursor(uci.DefaultConfDir, uci.DefaultSaveDir)
		if err != nil {
			return nil, err
		}
	}
	conf := &Conf{
		Ctx: ctx,
	}
	return conf, nil
}

func (conf *Conf) Get(path string) (interface{}, error) {
	opt, err := conf.Ctx.GetOption(path)
	if err != nil {
		return nil, err
	}
	return opt.Value(), nil
}

func (conf *Conf) Set(path string, val interface{}) error {
	fields := strings.SplitN(path, ".", 2)
	if len(fields) != 2 {
		return fmt.Errorf("invalid path: %s", path)
	}
	updated, err := conf.Ctx.SetOption(path, val)
	if err != nil {
		return err
	}
	if updated {
		conf.Ctx.Commit(fields[0])
	}
	return nil
}

func (conf *Conf) GetString(uid string) (string, error) {
	opt, err := conf.Ctx.GetOption(uid)
	if err != nil {
		return "", err
	}
	switch opt.T {
	case uci.OptionTypeString:
		return opt.ValString, nil
	case uci.OptionTypeList:
		return strings.Join(opt.ValStringList, ","), nil
	default:
		return "", fmt.Errorf("unsupported option type %d for %s", opt.T, uid)
	}
}

func (conf *Conf) GetStringDef(uid string, def string) string {
	val, err := conf.GetString(uid)
	if err != nil {
		return def
	}
	return val
}

func (conf *Conf) SetString(uid string, val string) error {
	_, err := conf.Ctx.SetOption(uid, val)
	return err
}

func (conf *Conf) GetNumber(uid string) (float64, error) {
	var err error
	var opt *uci.Option
	var val float64

	opt, err = conf.Ctx.GetOption(uid)
	if err != nil {
		return 0, err
	}
	switch opt.T {
	case uci.OptionTypeString:
		val, err = strconv.ParseFloat(opt.ValString, 64)
		if err != nil {
			return 0, err
		}
		return val, nil
	default:
		return 0, fmt.Errorf("unsupported option type %d for %s", opt.T, uid)
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
	_, err := conf.Ctx.SetOption(uid, val)
	return err
}
