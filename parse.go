package goADS;

import (
	//"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	log "github.com/cihub/seelog"
	"strconv"
)

type plcProjectInfo struct {
	ProjectInfo struct {
		Path string
		ChangeDate string
	}
	RoutingInfo struct {
		AdsInfo	struct {
			NetId string
			Port string
		}
	}
	DataTypes struct {
		DataType []dataType
	}
	Functions struct {
		Function []function
	}
	Programs struct {
		Program []program
	}
	Symbols struct {
		Symbol []symbol
	}
}

type dataType struct {
	Name string
	Type string
	BitSize uint32
	SubItem []dataTypeSubItem
	ArrayInfo struct {
		//LBound string
		//Elements string
		LBound uint32
		Elements uint32
	}
}

type dataTypeSubItem struct {
	Name string
	Type string
	BitSize string
	BitOffs string
	nBitSize int
	nBitOffs int
}

type function struct {
	Name string
}

type program struct {
	Name string
	PrgInfo struct {
		Symbol []prgInfoSymbol
	}
}

type prgInfoSymbol struct {
	Name string
}

type symbol struct {
	Name string
	Type string
	IGroup uint32
	IOffset uint32
	BitSize uint32
}


func (conn *Connection) ParseTPY (path string) (symbols map[string]ADSSymbol) {
	v := plcProjectInfo{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	err = xml.Unmarshal([]byte(data), &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}


    log.Warn("DAtatypes: ",len(v.DataTypes.DataType));
    log.Warn("Symbols: ",len(v.Symbols.Symbol));

	// Loop thru data types/*{{{*/
	if conn.symbols == nil {
		conn.datatypes = map[string]ADSSymbolUploadDataType{}
	}
	for _, val := range(v.DataTypes.DataType) {
		dt := ADSSymbolUploadDataType{}

		dt.Name = val.Name
		dt.DataType = val.Type
		//dt.Comment
		//dt.Size
		dt.Offset = val.BitSize/8;

		if len(dt.DataType) > 6 {
			if dt.DataType[:6] == "STRING" {
				dt.DataType = "STRING"
			}
		}

		if val.ArrayInfo.LBound != 0 || val.ArrayInfo.Elements != 0 {
			if dt.Childs == nil {
				dt.Childs = map[string]ADSSymbolUploadDataType{}
			}
			log.Info("Found array: start: ",val.ArrayInfo.LBound, ", num:", val.ArrayInfo.Elements);

			size := uint32(val.BitSize/8/val.ArrayInfo.Elements)
			for i := 0; i < int(val.ArrayInfo.Elements); i++ {
				child := ADSSymbolUploadDataType{}
				child.Name = "["+strconv.Itoa(i+int(val.ArrayInfo.LBound))+"]";
				child.DataType = val.Type
				child.Size = size
				child.Offset = size*uint32(i)

				dt.Childs[child.Name] = child

				log.Warn(dt.Childs)
			}
		}

		if val.SubItem != nil {
			if dt.Childs == nil {
				dt.Childs = map[string]ADSSymbolUploadDataType{}
			}

			for _, val2 := range(val.SubItem) {
				val2.nBitSize, _ = strconv.Atoi(val2.BitSize);
				val2.nBitOffs, _ = strconv.Atoi(val2.BitOffs);

				child := ADSSymbolUploadDataType{}
				child.Name = val2.Name
				child.DataType = val2.Type
				child.Size = uint32(val2.nBitSize/8)
				child.Offset = uint32(val2.nBitOffs/8)

				dt.Childs[child.Name] = child
			}
		}

		//if val.Name == "dtFlash" {
			//text,_ := json.MarshalIndent(val, "", "    ");
			//log.Info(string(text));
			//text,_ = json.MarshalIndent(dt, "", "    ");
			//log.Info(string(text));
		//}

		//if header.ArrayLevels > 0 {
			//// Childs is an array
			//var result arrayInfo
			//arrayLevels := []arrayInfo{}

			//for i := 0; i < int(header.ArrayLevels); i++ {
				//binary.Read(buff, binary.LittleEndian, &result)

				//arrayLevels = append(arrayLevels,result)
			//}

			////log.Warn(arrayLevels)

			//header.Childs = makeArrayChilds(arrayLevels,header.DataType,header.Size);

		//} else {
			//// Childs is standard variables
			//for buff.Len() > 0 {
				//if header.Childs == nil {
					//header.Childs = map[string]ADSSymbolUploadDataType{}
				//}

				//child := decodeSymbolUploadDataType(buff, header.Name)
				//header.Childs[child.Name] = child
			//}
		//}

		dt.conn = conn
		conn.datatypes[dt.Name] = dt
	}/*}}}*/

	// Loop thru all symbols/*{{{*/
	if conn.symbols == nil {
		conn.symbols = map[string]ADSSymbol{}
	}
	for _, val := range(v.Symbols.Symbol) {

		var item ADSSymbolUploadSymbol
		item.conn = conn
		item.Name = val.Name
		item.DataType = val.Type
		//item.Comment = ""
		item.Area = val.IGroup
		item.Offset = val.IOffset
		item.Length = val.BitSize/8
		//item.Extra1 = ""
		//item.Extra2 = ""

		if len(item.DataType) > 6 {
			if item.DataType[:6] == "STRING" {
				item.DataType = "STRING"
			}
		}

		conn.addSymbol(item)
	}/*}}}*/

	//ADSSymbolUploadDataType
	//conn.datatypes[header.Name] = header

	return conn.symbols
}
