package helper

import "strings"

type StringList []string

func (l *StringList) String() string {
	return strings.Join(*l, " ")
}

func (l *StringList) Set(v string) error {
	*l = append(*l, v)
	return nil
}
