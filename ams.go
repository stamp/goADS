package goADS

import (
    "fmt"
    "bytes"
    "encoding/binary"
    "encoding/hex"
)

func (conn *Connection) encode(command uint16,pack []byte,invoke uint32) (header []byte) {
    logger.Tracef("Starting encoding of AMS header command: %d",command)

	if conn==nil {
		logger.Error("Failed to encode header, connection is nil pointer");
		return
	}

    buf := new(bytes.Buffer)
    var data = []interface{}{
        uint16(0),
        uint32(32+len(pack)),
        conn.target.netid,
        conn.target.port,
        conn.source.netid,
        conn.source.port,
        uint16(command),
        uint16(4),
        uint32(len(pack)),
        uint32(0),
        invoke,
        pack,
    }

    for _, v := range data {
        err := binary.Write(buf, binary.LittleEndian, v)
        if err != nil {
            logger.Errorf("binary.Write failed: %s", err)
        }
    }

    header = buf.Bytes()

    /*fmt.Println("Len: ",header[2:6])
    fmt.Println("Target netid: ",header[6:12])
    fmt.Println("Target port: ",header[12:14])
    fmt.Println("Source netid: ",header[14:20])
    fmt.Println("Source port: ",header[20:22])
    fmt.Println("Command: ",header[22:24])
    fmt.Println("Flags: ",header[24:26])
    fmt.Println("Len: ",header[26:30])
    fmt.Println("Error: ",header[30:34])
    fmt.Println("Invoke: ",header[34:38])
    fmt.Println("DATA: ",header[38:])*/

    logger.Tracef("The encoded AMS header: \r\n%s",hex.Dump(header[6:]))

    return
}

func (conn *Connection) decode(in []byte) (command uint16,length uint32,invoke uint32,err error) {
    logger.Trace("Starting decoding of AMS header\n\r",hex.Dump(in[6:]))

    if len(in) < 38 {
        err = fmt.Errorf("Not a full AMS header (to small, %d < 38byte)",len(in));
        return
    }

    command = binary.LittleEndian.Uint16(in[22:24])
    length = binary.LittleEndian.Uint32(in[26:30])
    error := binary.LittleEndian.Uint32(in[30:34])
    invoke = binary.LittleEndian.Uint32(in[34:38])

    logger.Tracef("cmd: %d len: %d error: %d invoke: %d",command,length,error,invoke )

    if error > 0 {
        err = fmt.Errorf("Got ADS error code: %s in AMS decode",error)
        logger.Error(err)
        return
    }

/*
    header := buf.Bytes()
    fmt.Println("Len: ",header[2:6])
    fmt.Println("Target netid: ",header[6:12])
    fmt.Println("Target port: ",header[12:14])
    fmt.Println("Source netid: ",header[14:20])
    fmt.Println("Source port: ",header[20:22])
    fmt.Println("Command: ",header[22:24])
    fmt.Println("Flags: ",header[24:26])
    fmt.Println("Len: ",header[26:30])
    fmt.Println("Error: ",header[30:34])
    fmt.Println("Invoke: ",header[34:38])
    fmt.Println("DATA: ",header[38:])

    fmt.Println(data) */

    return
}
