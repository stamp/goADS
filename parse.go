package main;

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
)

type plcProjectInfo struct {
	ProjectInfo projectInfo
	RoutingInfo routringInfo
	DataTypes dataTypes
	Functions functions
	Programs programs
	Symbols symbols
}

type projectInfo struct {
	Path string
	ChangeDate string
}

type routringInfo struct {
	AdsInfo adsInfo
}

type adsInfo struct {
	NetId string
	Port string
}

type dataTypes struct {
	DataType []dataType
}

type dataType struct {
	Name string
	BitSize string
	SubItems []dataTypeSubItem
}

type dataTypeSubItem struct {
	Name string
	Type string
	BitSize string
	BitOffs string
}

type functions struct {
	Function []function
}

type function struct {
	Name string
}

type programs struct {
	Program []program
}

type program struct {
	Name string
	PrgInfo prgInfo
}

type prgInfo struct {
	Symbol []prgInfoSymbol
}

type prgInfoSymbol struct {
	Name string
}

type symbols struct {
	Symbol []symbol
}

type symbol struct {
	Name string
	Type string
	IGroup string
	IOffset string
	BitSize string
}

func main() {
	v := plcProjectInfo{}

	data, err := ioutil.ReadFile("/beckhoff/Hanaslöv.tpy")
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	err = xml.Unmarshal([]byte(data), &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	text,_ := json.MarshalIndent(v, "", "    ");
	fmt.Print(string(text));
}
