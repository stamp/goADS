package goADS

import (
	log "github.com/cihub/seelog"

	"bytes"
	    "encoding/hex"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"time"
)

type ADSSymbolUploadSymbol struct { /*{{{*/
	conn *Connection

	Name     string
	DataType string
	Comment  string

	Area   uint32
	Offset uint32
	Length uint32

	Extra1 uint32
	Extra2 uint32

	Childs map[string]ADSSymbolUploadDataType
}                                     /*}}}*/
type ADSSymbolUploadDataType struct { /*{{{*/
	conn *Connection

	Name     string
	Area     uint32
	Offset   uint32
	DataType string
	Comment  string

	Value string

	Childs map[string]ADSSymbolUploadDataType

	In1        uint32
	Decoration uint32
	In3        uint32
	Size       uint32
	//Offset         uint32
	In6 uint32
	In7 uint32
	ArrayLevels uint16
	In8 uint16
}                                 /*}}}*/
type ADSSymbolUploadInfo struct { /*{{{*/
	SymbolCount    uint32
	SymbolLength   uint32
	DataTypeCount  uint32
	DataTypeLength uint32
	ExtraCount     uint32
	ExtraLength    uint32
} /*}}}*/

func (conn *Connection) UploadSymbolInfo() (symbols map[string]ADSSymbolUploadSymbol, structs map[string]ADSSymbolUploadDataType) {

	res, _ := conn.Read(61455, 0, 48) //UploadSymbolInfo;

	// Parse the result and read symbol summarys
	var result ADSSymbolUploadInfo
	buff := bytes.NewBuffer(res.Data)
	binary.Read(buff, binary.LittleEndian, &result)

	// Load and parse data types
	conn.UploadSymbolInfoDataTypes(result.DataTypeLength)

	// Load and parse symbols
	conn.UploadSymbolInfoSymbols(result.SymbolLength)

	// Return the result
	return conn.symbols, conn.datatypes
}

func (conn *Connection) UploadSymbolInfoSymbols(length uint32) { /*{{{*/

	// Make a read at 
	res, e := conn.Read(61451, 0, length)
	if e != nil {
		log.Critical(e)
		return
	}

	if conn.symbols == nil {
		conn.symbols = map[string]ADSSymbolUploadSymbol{}
	}

	var buff bytes.Buffer
	buff.Write(res.Data)

	var header []byte
	var name []byte
	var dt []byte
	var comment []byte

	header = make([]byte, 4)
	buff.Read(header)

	for buff.Len() > 0 {
		header = make([]byte, 26)
		buff.Read(header)

		name = make([]byte, binary.LittleEndian.Uint16(header[20:22]))
		dt = make([]byte, binary.LittleEndian.Uint16(header[22:24]))
		comment = make([]byte, binary.LittleEndian.Uint16(header[24:26]))

		buff.Read(name)
		buff.Next(1)
		buff.Read(dt)
		buff.Next(1)
		buff.Read(comment)
		buff.Next(1)

		var item ADSSymbolUploadSymbol
		item.conn = conn
		item.Name = string(name)
		item.DataType = string(dt)
		item.Comment = string(comment)
		item.Area = binary.LittleEndian.Uint32(header[0:4])
		item.Offset = binary.LittleEndian.Uint32(header[4:8])
		item.Length = binary.LittleEndian.Uint32(header[8:12])
		item.Extra1 = binary.LittleEndian.Uint32(header[12:16])
		item.Extra2 = binary.LittleEndian.Uint32(header[16:20])

		if len(item.DataType) > 6 {
			if item.DataType[:6] == "STRING" {
				item.DataType = "STRING"
			}
		}

		conn.addSymbol(item)

		buff.Next(4)

		//log.Warn(hex.Dump(header));
	}
}                                                                  /*}}}*/
func (conn *Connection) UploadSymbolInfoDataTypes(length uint32) { /*{{{*/

	// Make a read at 
	res, e := conn.Read(61454, 0, length)
	if e != nil {
		log.Critical(e)
		return
	}
	buff := bytes.NewBuffer(res.Data)

	if conn.symbols == nil {
		conn.datatypes = map[string]ADSSymbolUploadDataType{}
	}

	//l := buff.Len()
	//var last = l
	//log.Warn("Bytes: ", l)

	for buff.Len() > 0 {
		header := decodeSymbolUploadDataType(buff, "")
		header.conn = conn

		//log.Warn("ITEM (",(l-buff.Len()),"|",(last-buff.Len()),"): ",header.Name,"|",header.DataType,"|",header.Comment)
		//last = buff.Len()

		//header.Index = l - buff.Len()
		//header.Size = last - buff.Len()

		conn.datatypes[header.Name] = header
	}
	//   log.Warn(hex.Dump(header));
}                                                                                                     /*}}}*/
type arrayInfo struct {
	Start uint32
	Length uint32
}
func decodeSymbolUploadDataType(data *bytes.Buffer, parent string) (header ADSSymbolUploadDataType) { /*{{{*/
	type headerStruct struct {
		LenTotal		uint32
		In1				uint32
		Decoration		uint32
		In3				uint32
		Size			uint32
		Offset			uint32
		In6				uint32
		In7				uint32
		LenName			uint16
		LenDataType		uint16
		LenComment		uint16
		ArrayLevels		uint16
		In8				uint16
	}


	var result headerStruct
	header = ADSSymbolUploadDataType{}

	totalSize := data.Len()

	if totalSize < 48 {
		log.Error(parent," - Wrong size <48 byte");
		log.Error(hex.Dump(data.Bytes()));
	}

	binary.Read(data, binary.LittleEndian, &result)

	name := make([]byte, result.LenName)
	dt := make([]byte, result.LenDataType)
	comment := make([]byte, result.LenComment)

	binary.Read(data, binary.LittleEndian, name)
	data.Next(1)
	binary.Read(data, binary.LittleEndian, dt)
	data.Next(1)
	binary.Read(data, binary.LittleEndian, comment)
	data.Next(1)

	header.Name = string(name)
	header.DataType = string(dt)
	header.Comment = string(comment)

	header.In1 = result.In1
	header.Decoration = result.Decoration
	header.In3 = result.In3
	header.Size = result.Size
	header.Offset = result.Offset
	header.In6 = result.In6
	header.In7 = result.In7
	header.Value = "nil"
	header.ArrayLevels = result.ArrayLevels

	if len(header.DataType) > 6 {
		if header.DataType[:6] == "STRING" {
			header.DataType = "STRING"
		}
	}

	//log.Warn("ITEM ",header.Name,"|",header.DataType,"|",header.Comment, " array:",header.ArrayLevels," [",header.In1,"|",header.In3,"|",header.In6,"|",header.In7,"]")

	childLen := int(result.LenTotal) - (totalSize - data.Len())
	if childLen <= 0 {
		return
	}

	childs := make([]byte, childLen)
	data.Read(childs)

	if len(childs) == 0 {
		return
	}

	buff := bytes.NewBuffer(childs)

	if header.ArrayLevels > 0 {
		// Childs is an array
		var result arrayInfo
		arrayLevels := []arrayInfo{}

		for i := 0; i < int(header.ArrayLevels); i++ {
			binary.Read(buff, binary.LittleEndian, &result)

			arrayLevels = append(arrayLevels,result)
		}

		//log.Warn(arrayLevels)

		header.Childs = makeArrayChilds(arrayLevels,header.DataType,header.Size);

	} else {
		// Childs is standard variables
		for buff.Len() > 0 {
			if header.Childs == nil {
				header.Childs = map[string]ADSSymbolUploadDataType{}
			}

			child := decodeSymbolUploadDataType(buff, header.Name)
			header.Childs[child.Name] = child
		}
	}

	return
} /*}}}*/
func makeArrayChilds(levels []arrayInfo, dt string, size uint32 ) (childs map[string]ADSSymbolUploadDataType) {
	childs = map[string]ADSSymbolUploadDataType{}

	if len(levels) < 1 {
		return
	}

	level := levels[:1][0]
	subChilds := makeArrayChilds(levels[1:],dt,size);

	//log.Warn("Make childs [",level.Start,"..",level.Start+level.Length-1,"] and ",levels[1:])
	var offset uint32 = 0

	for i:=level.Start;i<level.Start+level.Length;i++ {
		name := fmt.Sprint("[",i,"]")

		child := ADSSymbolUploadDataType{}
		child.Name = name
		child.DataType = dt
		child.Offset = offset
		child.Size = size/level.Length
		child.Childs = subChilds

		//child.Walk("")

		childs[name] = child
		offset += size/level.Length
	}


	return
}

// Add symbols to the list and read the data type
func (conn *Connection) addSymbol(symbol ADSSymbolUploadSymbol) { /*{{{*/
	dt, ok := conn.datatypes[symbol.DataType]

	if ok {
		symbol.Childs = dt.addOffset(conn, symbol.Area, symbol.Offset)
	}

	conn.symbols[symbol.Name] = symbol
}                                                                                                                                          /*}}}*/
func (data *ADSSymbolUploadDataType) addOffset(conn *Connection, area uint32, offset uint32) (childs map[string]ADSSymbolUploadDataType) { /*{{{*/
	childs = map[string]ADSSymbolUploadDataType{}

	// Make a copy of the datatype
	for k, v := range data.Childs {
		childs[k] = v
	}

	for key, segment := range childs {
		// Uppdate with area and offset
		segment.Area = area
		segment.Offset = segment.Offset + offset

		// Check if subitems exist
		dt, ok := conn.datatypes[segment.DataType]
		if ok {
			//log.Warn("Found sub ",segment.DataType);
			segment.Childs = dt.addOffset(conn, segment.Area, segment.Offset)
		}

		// Save the change
		childs[key] = segment
	}

	return
} /*}}}*/

// Print the whole symbol and data type lists
func (data *ADSSymbolUploadSymbol) DebugWalk() { /*{{{*/
	log.Warn("SYMBOL (", data.Area, ":", data.Offset, "): ", data.Name, " - ", data.DataType, "[", data.Length, "] ", data.Comment)

	for _, segment := range data.Childs {
		segment.DebugWalk()
	}
}                                                  /*}}}*/
func (data *ADSSymbolUploadDataType) DebugWalk() { /*{{{*/
	log.Warn("TYPE (", data.Area, ":", data.Offset, "): ", data.Name, " [", data.In1, "|", data.In3, "|", data.In6, "|", data.In7, "]  ", data.DataType, "[", data.Size, "] ", data.Comment);

	for _, segment := range data.Childs {
		segment.DebugWalk()
	}
} /*}}}*/

// Print the whole symbol and data type lists
func (data *ADSSymbolUploadSymbol) Walk() { /*{{{*/
	log.Info("SYMBOL (", data.Area, ":", data.Offset, "): ", data.Name)

	for _, segment := range data.Childs {
		segment.Walk(data.Name)
	}
}                                                          /*}}}*/
func (data *ADSSymbolUploadDataType) Walk(parent string) { /*{{{*/
	var path string

	if ( data.Name[0:1] != "[" ) {
		path = fmt.Sprint(parent, ".", data.Name)
	} else {
		path = fmt.Sprint(parent, data.Name)
	}

	if len(data.Childs) == 0 {
		if data.Value != "nil" {
			log.Info("TYPE (", data.Area, ":", data.Offset, "): ", path, " [", data.DataType, "|",data.Size,"] = ", data.Value)
		} else {
			log.Warn("TYPE (", data.Area, ":", data.Offset, "): ", path, " [", data.DataType, "|",data.Size,"] = ", data.Value)
		}
	} else {
		//log.Error("TYPE (", data.Area, ":", data.Offset, "): ", path, " [", data.DataType, "] = ", data.Value)
		for _, segment := range data.Childs {
			segment.Walk(path)
		}
	}
} /*}}}*/


func (symbol *ADSSymbolUploadSymbol) AddDeviceNotification(callback func(*ADSSymbolUploadSymbol)) { /*{{{*/
	log.Info("AddDeviceNotification (", symbol.Area, ":", symbol.Offset, "): ", symbol.Name)

	s := symbol
	c := callback

	symbol.conn.AddDeviceNotification(symbol.Area,symbol.Offset,symbol.Length,ADS_ServerOnChange,2000,100,func(data []byte) {
		s.notification(data)
		c(s)
	});

}                                                          /*}}}*/

// Read a symbol and all sub values
func (symbol *ADSSymbolUploadSymbol) Read() { /*{{{*/
	log.Warn("Read (", symbol.Area, ":", symbol.Offset, "): ", symbol.Name)

	res, _ := symbol.conn.Read(symbol.Area, symbol.Offset, symbol.Length)
	//buff := bytes.NewBuffer(res.Data)

	//log.Error(hex.Dump(buff.Bytes()));

	for i, segment := range symbol.Childs {
		segment.parse(symbol.Offset, res.Data)
		symbol.Childs[i] = segment
	}
}                                                                      /*}}}*/
func (dt *ADSSymbolUploadDataType) parse(offset uint32, data []byte) { /*{{{*/
	start := dt.Offset - offset
	stop := start + dt.Size

	//log.Debug(dt.Name, ".parse(", start, ":", stop, ",", len(data), ")")

	for i, segment := range dt.Childs {
		segment.parse(offset, data)
		dt.Childs[i] = segment
	}

	if len(dt.Childs) == 0 {

		switch dt.DataType {
		case "BOOL":
			if stop-start != 1 {return}
			if data[start:stop][0] > 0 {
				dt.Value = "True"
			} else {
				dt.Value = "False"
			}
		case "BYTE","USINT": // Unsigned Short INT 0 to 255
			if stop-start != 1 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i uint8
			binary.Read(buf, binary.LittleEndian, &i)
			dt.Value = strconv.FormatInt(int64(i), 10)
		case "SINT": // Short INT -128 to 127
			if stop-start != 1 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i int8
			binary.Read(buf, binary.LittleEndian, &i)
			dt.Value = strconv.FormatInt(int64(i), 10)
		case "UINT","WORD":
			if stop-start != 2 {return}
			i := binary.LittleEndian.Uint16(data[start:stop])
			dt.Value = strconv.FormatUint(uint64(i), 10)
		case "UDINT","DWORD":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			dt.Value = strconv.FormatUint(uint64(i), 10)
		case "INT":
			if stop-start != 2 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i int16
			binary.Read(buf, binary.LittleEndian, &i)
			dt.Value = strconv.FormatInt(int64(i), 10)
		case "DINT":
			if stop-start != 4 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i int32
			binary.Read(buf, binary.LittleEndian, &i)
			dt.Value = strconv.FormatInt(int64(i), 10)
		case "REAL":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			f := math.Float32frombits(i)
			dt.Value = strconv.FormatFloat(float64(f), 'f', -1, 32)
		case "LREAL":
			if stop-start != 8 {return}
			i := binary.LittleEndian.Uint64(data[start:stop])
			f := math.Float64frombits(i)
			dt.Value = strconv.FormatFloat(f, 'f', -1, 64)
		case "STRING":
			dt.Value = string(data[start:stop])
		case "TIME":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

			dt.Value = t.Truncate(time.Millisecond).Format("15:04:05.999999999")
		case "TOD":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

			dt.Value = t.Truncate(time.Millisecond).Format("15:04")
		case "DATE":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second)) )

			dt.Value = t.Truncate(time.Millisecond).Format("2006-01-02")
		case "DT":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second))-int64(time.Hour) )

			dt.Value = t.Truncate(time.Millisecond).Format("2006-01-02 15:04:05")
		default:
			dt.Value = "nil"
		}
	}

	//buff := bytes.NewBuffer(res.Data)

	//log.Error(hex.Dump(buff.Bytes()));

	//for _, segment := range data.Childs {
	//segment.Walk(data.Name)
	//}
} /*}}}*/
func (symbol *ADSSymbolUploadSymbol) notification(data []byte) {
	for i, segment := range symbol.Childs {
		segment.parse(symbol.Offset, data)
		symbol.Childs[i] = segment
	}
}

