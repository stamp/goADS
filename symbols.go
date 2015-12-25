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

	Name        string
	Area        uint32
	Offset      uint32
	DataType    string
	Comment     string

	Value       string
    Changed     bool

	Childs      map[string]ADSSymbolUploadDataType

	In1         uint32
	Decoration  uint32
	In3         uint32
	Size        uint32
	In6         uint32
	In7         uint32
	ArrayLevels uint16
	In8         uint16
}                                 /*}}}*/
type ADSSymbol struct { /*{{{*/
	conn *Connection

	Self		*ADSSymbol
	FullName    string
	Name        string
	DataType    string
	Comment     string

	Area        uint32
	Offset      uint32
	Length		uint32

	Value       string
	Valid       bool
    Changed     bool

	Childs      map[string]ADSSymbol
}                                 /*}}}*/
type ADSSymbolUploadInfo struct { /*{{{*/
	SymbolCount    uint32
	SymbolLength   uint32
	DataTypeCount  uint32
	DataTypeLength uint32
	ExtraCount     uint32
	ExtraLength    uint32
} /*}}}*/

func (conn *Connection) UploadSymbolInfo() (symbols map[string]ADSSymbol, structs map[string]ADSSymbolUploadDataType) {
	log.Debug("Start UploadSymbolInfo")

	res, e := conn.Read(61455, 0, 24) //UploadSymbolInfo;
	if e != nil {
		log.Critical(e)
		return
	}

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
		conn.symbols = map[string]ADSSymbol{}
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
func makeArrayChilds(levels []arrayInfo, dt string, size uint32 ) (childs map[string]ADSSymbolUploadDataType) {/*{{{*/
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
}/*}}}*/

// Add symbols to the list and read the data type
func (conn *Connection) addSymbol(symbol ADSSymbolUploadSymbol) { /*{{{*/
	sym := ADSSymbol{}

	sym.Self = &sym
	sym.conn = conn
	sym.Name = symbol.Name
	sym.FullName = symbol.Name
	sym.DataType = symbol.DataType
	sym.Comment = symbol.Comment
	sym.Length = symbol.Length

	sym.Area = symbol.Area
	sym.Offset = symbol.Offset

	dt, ok := conn.datatypes[symbol.DataType]
	if ok {
		sym.Childs = dt.addOffset(conn, sym.Name, symbol.Area, symbol.Offset)
	}

	conn.symbols[symbol.Name] = sym

	return
}                                                                                                                                          /*}}}*/
func (data *ADSSymbolUploadDataType) addOffset(conn *Connection, parent string, area uint32, offset uint32) (childs map[string]ADSSymbol) { /*{{{*/
	childs = map[string]ADSSymbol{}

	var path string

	// Make a copy of the datatype
	for key,segment := range data.Childs {

		if ( segment.Name[0:1] != "[" ) {
			path = fmt.Sprint(parent, ".", segment.Name)
		} else {
			path = fmt.Sprint(parent, segment.Name)
		}

		child := ADSSymbol{}
		child.Self = &child

		child.conn = conn
		child.Name = segment.Name
		child.FullName = path
		child.DataType = segment.DataType
		child.Comment = segment.Comment
		child.Length = segment.Size

		// Uppdate with area and offset
		child.Area = area
		child.Offset = segment.Offset + offset

		// Check if subitems exist
		dt, ok := conn.datatypes[segment.DataType]
		if ok {
			//log.Warn("Found sub ",segment.DataType);
			child.Childs = dt.addOffset(conn, path, child.Area, child.Offset)
		}

		childs[key] = child
	}

	return
} /*}}}*/

// Print the whole symbol and data type lists
func (data *ADSSymbolUploadSymbol) DebugWalk() { /*{{{*/
	log.Warn("DebugWalk (", data.Area, ":", data.Offset, "): ", data.Name, " - ", data.DataType, "[", data.Length, "] ", data.Comment)

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
//func (data *ADSSymbolUploadSymbol) Walk() { [>{{{<]
	//log.Info("Walk (", data.Area, ":", data.Offset, "): ", data.Name," - have ",len(data.Childs)," childs")

	//for _, segment := range data.Childs {
		//segment.Walk(data.Name)
	//}
//}                                                          [>}}}<]
func (data *ADSSymbol) Walk() { /*{{{*/
	if len(data.Childs) == 0 {
		if !data.Valid {
			log.Warn("TYPE (", data.Area, ":", data.Offset, "): ", data.FullName, " [", data.DataType, "|",data.Length,"] = INVALID:",data.Value)
		} else {
			log.Info("TYPE (", data.Area, ":", data.Offset, "): ", data.FullName, " [", data.DataType, "|",data.Length,"] = ", data.Value)
		}
	} else {
		//log.Error("TYPE (", data.Area, ":", data.Offset, "): ", path, " [", data.DataType, "] = ", data.Value)
		for i,_  := range data.Childs {
			data.Childs[i].Self.Walk()
		}
	}
} /*}}}*/

//func (data *ADSSymbolUploadSymbol) FindChanged() (changed []*ADSSymbolUploadDataType) { [>{{{<]
	//log.Info("FindChanged (", data.Area, ":", data.Offset, "): ", data.Name)

	//for _, segment := range data.Childs {
		//c := segment.FindChanged(segment.Name)
        //for _, cc := range c {
            //changed = append(changed,cc)
        //}
	//}

    //return
//}                                                          [>}}}<]
func (data *ADSSymbol) FindChanged() (changed []*ADSSymbol) { /*{{{*/

	if len(data.Childs) == 0 {
        if data.Changed && data.Valid {
			//log.Debug("Changed: ",data.FullName,"|",data.Value,"|",data.PrevValue,"=",data.Changed)
            changed = append(changed,data)
        /*} else {
			log.Debug("NOT: ",path,"|",data.Value,"|",data.PrevValue)*/
		}
        data.Changed = false
	} else {
		for i,_ := range data.Childs {
			c := data.Childs[i].Self.FindChanged()
            for _, cc := range c {
                changed = append(changed,cc)
            }
		}
	}

    return
} /*}}}*/

func (data *ADSSymbol) Find(name string) (list []*ADSSymbol) { /*{{{*/
	if len(data.Childs) == 0 {
        if len(data.FullName)>=len(name)&&data.FullName[:len(name)]==name {
            list = append(list,data)
		}
	} else {
		for i,_ := range data.Childs {
			c := data.Childs[i].Self.Find(name)
            for _,cc  := range c {
                list = append(list,cc)
            }
		}
	}

    return
} /*}}}*/

func (symbol *ADSSymbol) AddDeviceNotification(callback func(*ADSSymbol)) { /*{{{*/
	log.Info("AddDeviceNotification (", symbol.Area, ":", symbol.Offset, "): ", symbol.Name)

	s := symbol
	c := callback

	symbol.conn.AddDeviceNotification(symbol.Area,symbol.Offset,symbol.Length,ADS_ServerOnChange,1000,1000,func(data []byte) {
		s.notification(data)
		c(s)
	});

}                                                          /*}}}*/

// Read a symbol and all sub values
func (symbol *ADSSymbol) Read() { /*{{{*/
	log.Debug("Read (", symbol.Area, ":", symbol.Offset, "): ", symbol.FullName)

	res, _ := symbol.conn.Read(symbol.Area, symbol.Offset, symbol.Length)

	for i, _ := range symbol.Childs {
		segment := symbol.Childs[i].Self
		segment.parse(symbol.Offset, res.Data)
	}
}
/*}}}*/
func (symbol *ADSSymbol) Write(value string) (error) { /*{{{*/
	log.Debug("Write (", symbol.Area, ":", symbol.Offset, "): ", symbol.FullName)

	if len(symbol.Childs) != 0 {
		e := fmt.Errorf("Cannot write to a whole struct at once!")
		log.Error(e)
		return e
	}

	buf := bytes.NewBuffer([]byte{})

		switch symbol.DataType {
		case "BOOL":
			v,e := strconv.ParseBool(value)
			if e!=nil { return e }

			if v {
				buf.Write([]byte{1});
			} else {
				buf.Write([]byte{0});
			}
		case "BYTE","USINT": // Unsigned Short INT 0 to 255
			v,e := strconv.ParseUint(value,10,8)
			if e!=nil { return e }

			v8 := uint8(v)
			binary.Write(buf,binary.LittleEndian, &v8 )
		case "UINT16","UINT","WORD":
			v,e := strconv.ParseUint(value,10,16)
			if e!=nil { return e }

			v16 := uint16(v)
			binary.Write(buf,binary.LittleEndian, &v16 )
		case "UDINT","DWORD":
			v,e := strconv.ParseUint(value,10,32)
			if e!=nil { return e }

			v32 := uint32(v)
			binary.Write(buf,binary.LittleEndian, &v32 )

		case "SINT": // Short INT -128 to 127
			v,e := strconv.ParseInt(value,10,8)
			if e!=nil { return e }

			v8 := int8(v)
			binary.Write(buf,binary.LittleEndian, &v8 )
		case "INT":
			v,e := strconv.ParseInt(value,10,16)
			if e!=nil { return e }

			v16 := int16(v)
			binary.Write(buf,binary.LittleEndian, &v16 )
		case "DINT":
			v,e := strconv.ParseInt(value,10,32)
			if e!=nil { return e }

			v32 := int32(v)
			binary.Write(buf,binary.LittleEndian, &v32 )

		case "REAL":
			v,e := strconv.ParseFloat(value,32)
			if e!=nil { return e }

			v32 := math.Float32bits(float32(v))
			binary.Write(buf,binary.LittleEndian, &v32 )
		case "LREAL":
			v,e := strconv.ParseFloat(value,64)
			if e!=nil { return e }

			v64 := math.Float64bits(v)
			binary.Write(buf,binary.LittleEndian, &v64 )
		case "STRING":
			buf.Write([]byte(value));
		/*case "TIME":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

			newValue = t.Truncate(time.Millisecond).Format("15:04:05.999999999")
		case "TOD":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

			newValue = t.Truncate(time.Millisecond).Format("15:04")
		case "DATE":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second)) )

			newValue = t.Truncate(time.Millisecond).Format("2006-01-02")
		case "DT":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second))-int64(time.Hour) )

			newValue = t.Truncate(time.Millisecond).Format("2006-01-02 15:04:05")*/
		default:
			e := fmt.Errorf("Datatype '%s' write is not implemented yet!",symbol.DataType)
			log.Error(e)
			return e
		}


	symbol.Self.conn.Write(symbol.Area, symbol.Offset, buf.Bytes())

	return nil

}
/*}}}*/

func (dt *ADSSymbol) parse(offset uint32, data []byte) { /*{{{*/
	start := dt.Offset - offset
	stop := start + dt.Length

	for i,_ := range dt.Childs {
		dt.Childs[i].Self.parse(offset, data)
	}

	if len(dt.Childs) == 0 {
        var newValue = "nil"

		if int(start)<0||int(stop)>len(data) {
			log.Errorf("Incomming data is to small, !0<%d<%d<%d",start,stop,len(data));
			return
		}

		switch dt.DataType {
		case "BOOL":
			if stop-start != 1 {return}
			if data[start:stop][0] > 0 {
				newValue = "True"
			} else {
				newValue = "False"
			}
		case "BYTE","USINT": // Unsigned Short INT 0 to 255
			if stop-start != 1 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i uint8
			binary.Read(buf, binary.LittleEndian, &i)
			newValue = strconv.FormatInt(int64(i), 10)
		case "SINT": // Short INT -128 to 127
			if stop-start != 1 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i int8
			binary.Read(buf, binary.LittleEndian, &i)
			newValue = strconv.FormatInt(int64(i), 10)
		case "UINT","WORD":
			if stop-start != 2 {return}
			i := binary.LittleEndian.Uint16(data[start:stop])
			newValue = strconv.FormatUint(uint64(i), 10)
		case "UDINT","DWORD":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			newValue = strconv.FormatUint(uint64(i), 10)
		case "INT":
			if stop-start != 2 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i int16
			binary.Read(buf, binary.LittleEndian, &i)
			newValue = strconv.FormatInt(int64(i), 10)
		case "DINT":
			if stop-start != 4 {return}
			buf := bytes.NewBuffer(data[start:stop])
			var i int32
			binary.Read(buf, binary.LittleEndian, &i)
			newValue = strconv.FormatInt(int64(i), 10)
		case "REAL":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			f := math.Float32frombits(i)
			newValue = strconv.FormatFloat(float64(f), 'f', -1, 32)
		case "LREAL":
			if stop-start != 8 {return}
			i := binary.LittleEndian.Uint64(data[start:stop])
			f := math.Float64frombits(i)
			newValue = strconv.FormatFloat(f, 'f', -1, 64)
		case "STRING":
			newValue = string( bytes.TrimSpace(data[start:stop]) )
		case "TIME":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

			newValue = t.Truncate(time.Millisecond).Format("15:04:05.999999999")
		case "TOD":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Millisecond))-int64(time.Hour) )

			newValue = t.Truncate(time.Millisecond).Format("15:04")
		case "DATE":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second)) )

			newValue = t.Truncate(time.Millisecond).Format("2006-01-02")
		case "DT":
			if stop-start != 4 {return}
			i := binary.LittleEndian.Uint32(data[start:stop])
			t := time.Unix(0, int64(uint64(i)*uint64(time.Second))-int64(time.Hour) )

			newValue = t.Truncate(time.Millisecond).Format("2006-01-02 15:04:05")
		default:
			newValue = "nil"
		}

        if strcmp(dt.Value,newValue)!=0 {
			dt.Value = newValue
			dt.Changed = dt.Valid
        }

		dt.Valid = true
    }
}
func strcmp(a, b string) int {
	min := len(b)
	if len(a) < len(b) {
		min = len(a)
	}
	diff := 0
	for i := 0; i < min && diff == 0; i++ {
		diff = int(a[i]) - int(b[i])
	}
	if diff == 0 {
		diff = len(a) - len(b)
	}
	return diff
}
/*}}}*/
func (symbol *ADSSymbol) notification(data []byte) {
	symbol.parse(symbol.Offset, data)
}

