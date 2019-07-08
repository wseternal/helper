package iohelper

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

type StrElem struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type IntElem struct {
	Name  string `xml:"name,attr"`
	Value int    `xml:"value,attr"`
}

type BoolElem struct {
	Name  string `xml:"name,attr"`
	Value bool    `xml:"value,attr"`
}


type SharedPreference struct {
	XMLName xml.Name   `xml:"map"`
	SElems  []*StrElem `xml:"string"`
	IElems  []*IntElem `xml:"int"`
	BElems []*BoolElem `xml:"boolean"`
}

var (
	Header = `<?xml version='1.0' encoding='utf-8' standalone='yes' ?>` + "\n"
)

func (sp *SharedPreference) Data() ([]byte, error) {
	return xml.MarshalIndent(sp, "", "    ")
}

func (sp *SharedPreference) ToFile(fn string) error {
	f, err := os.OpenFile(fn, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0660)
	defer f.Close()

	if err != nil {
		return err
	}
	_, err = f.WriteString(Header)
	if err != nil {
		return fmt.Errorf("write shared prefrence header failed, %s", err)
	}
	enc := xml.NewEncoder(f)
	enc.Indent("", "    ")
	err = enc.Encode(sp)
	if err != nil {
		return err
	}
	f.WriteString("\n")
	return nil
}

func (sp *SharedPreference) FromFile(fn string) error {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		fmt.Fprintf(os.Stderr, "SharedPreference FromFile: skip empty file: %s\n", fn)
		return nil
	}
	return xml.Unmarshal(data, sp)
}

func (sp *SharedPreference) GetString(name string) *string {
	for _, elem := range sp.SElems {
		if elem.Name == name {
			return &elem.Value
		}
	}
	return nil
}

func (sp *SharedPreference) PutString(name string, val string) {
	found := false
	for _, elem := range sp.SElems {
		if elem.Name == name {
			elem.Value = val
			found = true
			break
		}
	}
	if !found {
		sp.SElems = append(sp.SElems, &StrElem{
			Name:  name,
			Value: val,
		})
	}
}

func (sp *SharedPreference) GetInt(name string) *int {
	for _, elem := range sp.IElems {
		if elem.Name == name {
			return &elem.Value
		}
	}
	return nil
}

func (sp *SharedPreference) PutInt(name string, val int) {
	found := false
	for _, elem := range sp.IElems {
		if elem.Name == name {
			elem.Value = val
			found = true
			break
		}
	}
	if !found {
		sp.IElems = append(sp.IElems, &IntElem{
			Name:  name,
			Value: val,
		})
	}
}

func (sp *SharedPreference) GetBool(name string) *bool {
	for _, elem := range sp.BElems {
		if elem.Name == name {
			return &elem.Value
		}
	}
	return nil
}

func (sp *SharedPreference) PutBool(name string, val bool) {
	found := false
	for _, elem := range sp.BElems {
		if elem.Name == name {
			elem.Value = val
			found = true
			break
		}
	}
	if !found {
		sp.BElems = append(sp.BElems, &BoolElem{
			Name:  name,
			Value: val,
		})
	}
}
