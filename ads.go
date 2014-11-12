package goADS

import (
    "fmt"
    "encoding/binary"
    "errors"
    "bytes"
    "strconv"
//   "encoding/hex"
)

type ADSNotification struct {
	handle uint32
}

// ReadDeviceInfo				- ADS command id: 1/*{{{*/
type ADSReadDeviceInfo struct {
    MajorVersion uint8
    MinorVersion uint8
    BuildVersion uint16
    DeviceName string
}
func (conn *Connection) ReadDeviceInfo() (response ADSReadDeviceInfo, err error) {/*{{{*/
    var resp []byte

    // Try to send the request
    resp,err = conn.sendRequest(1,[]byte{})
    if err!=nil {
        return
    }

    // Check the response length
    if len(resp)!=24 {
        return response, errors.New(fmt.Sprintf("Wrong length of response! Got %d bytes and it should be 24",len(resp)) )
    }

    // Check the result error code
    result := binary.LittleEndian.Uint32(resp[0:4])
    if result>0 {
        err = errors.New("Got ADS error number "+strconv.FormatUint(uint64(result),10)+" in ReadDeviceInfo")
        return
    }

    // Parse the response
    response.MajorVersion = resp[4]
    response.MinorVersion = resp[5]
    response.BuildVersion = binary.LittleEndian.Uint16(resp[6:8])
    response.DeviceName   = string(resp[8:24])

    logger.Debugf("response.majorVersion=%d",response.MajorVersion);
    logger.Debugf("response.minorVersion=%d",response.MinorVersion);
    logger.Debugf("response.buildVersion=%d",response.BuildVersion);
    logger.Debugf("response.deviceName=%s",response.DeviceName);

    return
}/*}}}*/
/*}}}*/
// Read							- ADS command id: 2/*{{{*/
type ADSRead struct {
    Data []byte
}
func (conn *Connection) Read(group uint32,offset uint32,length uint32) (response ADSRead, err error) {/*{{{*/
    request := new(bytes.Buffer)
    var content = []interface{}{
        group,
        offset,
        length,
    }

    for _, v := range content {
        err := binary.Write(request, binary.LittleEndian, v)
        if err != nil {
            logger.Errorf("binary.Write failed: %s", err)
        }
    }

    // Try to send the request
    resp,err := conn.sendRequest(2,request.Bytes())
    if err!=nil {
        return
    }

    // Check the result error code
    result := binary.LittleEndian.Uint32(resp[0:4])
    datalength := binary.LittleEndian.Uint32(resp[4:8])
    if result>0 {
        err = errors.New("Got ADS error number "+strconv.FormatUint(uint64(result),10)+" in Read")
        return
    }

    if len(resp)-8!=int(datalength) {
        return response, errors.New(fmt.Sprintf("Wrong length of response! Got %d bytes and it should be %d",len(resp),datalength) )
	}

    response.Data = resp[8:datalength+8]
    //logger.Debugf("The read data at %d:%d: \r\n%s",group,offset,hex.Dump(response.Data))

    return
}/*}}}*/
/*}}}*/
// Write						- ADS command id: 3/*{{{*/
func (conn *Connection) Write(group uint32,offset uint32,data []byte) {
	if conn==nil {
		logger.Error("Failed to Write, connection is nil pointer");
		return
	}

    request := new(bytes.Buffer)

	length := uint32(len(data))
    var content = []interface{}{
        group,
		offset,
		length,
    }

    for _, v := range content {
        err := binary.Write(request, binary.LittleEndian, v)
        if err != nil {
            logger.Errorf("binary.Write failed: %s", err)
        }
    }

    _,err := request.Write(data)
    if err != nil {
        logger.Errorf("bytes.Write failed: %s", err)
	}

    // Try to send the request
    resp,err := conn.sendRequest(3,request.Bytes())
    if err!=nil {
        return
    }

    // Check the result error code
    result := binary.LittleEndian.Uint32(resp[0:4])
    if result>0 {
        err = errors.New("Got ADS error number "+strconv.FormatUint(uint64(result),10)+" in Write")
        return
    }

    return
}
/*}}}*/
// ReadState					- ADS command id: 4/*{{{*/
type ADSReadState struct {
    ADSState uint16
    ADSDeviceState uint16
}
func (conn *Connection) ReadState() (response ADSReadState, err error) {/*{{{*/
    var resp []byte

    // Try to send the request
    resp,err = conn.sendRequest(4,[]byte{})
    if err!=nil {
        return
    }

    // Check the response length
    if len(resp)!=24 {
        return response, errors.New(fmt.Sprintf("Wrong length of response! Got %d bytes and it should be 24",len(resp)) )
    }

    // Check the result error code
    result := binary.LittleEndian.Uint32(resp[0:4])
    if result>0 {
        err = errors.New("Got ADS error number "+strconv.FormatUint(uint64(result),10)+" in ReadState")
        return
    }

    // Parse the response
    response.ADSState = binary.LittleEndian.Uint16(resp[4:6])
    response.ADSDeviceState = binary.LittleEndian.Uint16(resp[6:8])

    logger.Debugf("response.ADSState=%d",response.ADSState);
    logger.Debugf("response.ADSDeviceState=%d",response.ADSDeviceState);

    return
}/*}}}*/
/*}}}*/
// WriteControl TODO			- ADS command id: 5/*{{{*/
func (conn *Connection) WriteControl() {
}
/*}}}*/
// AddDeviceNotification		- ADS command id: 6/*{{{*/
const (
	ADS_NoTransmission	= 0
	ADS_ClientCycle		= 1
	ADS_ClientOnChange	= 2
	ADS_ServerCycle		= 3
	ADS_ServerOnChange	= 4
	ADS_Client1Reqest	= 5
)
type ADSAddDeviceNotification struct {
    Handle uint32
}
func (conn *Connection) AddDeviceNotification(group uint32,offset uint32,length uint32,transmissionMode uint32, maxDelay uint32, cycleTime uint32,callback func([]byte)) (response ADSAddDeviceNotification, err error) {/*{{{*/
    request := new(bytes.Buffer)
	reserved := make([]byte,16)
    var content = []interface{}{
        group,
        offset,
        length,
        transmissionMode,
        maxDelay, // 1 = 1ms (alt 100ns?)
		cycleTime, // 1 = 1ms
		reserved,
    }

    for _, v := range content {
        err := binary.Write(request, binary.LittleEndian, v)
        if err != nil {
            logger.Errorf("binary.Write failed: %s", err)
        }
    }

    // Try to send the request
    handle,err := conn.createNotificationWorker(request.Bytes(),callback)
    if err!=nil {
		logger.Debug("Added notification handler, FAILED: ",err)
        return
    }

    response.Handle = handle

	logger.Debug("Added notification handler: ",handle)

    return
}/*}}}*/
/*}}}*/
// DeleteDeviceNotification		- ADS command id: 7/*{{{*/
func (conn *Connection) DeleteDeviceNotification(handle uint32) {/*{{{*/
    request := new(bytes.Buffer)
    var content = []interface{}{
        handle,
    }

    for _, v := range content {
        err := binary.Write(request, binary.LittleEndian, v)
        if err != nil {
            logger.Errorf("binary.Write failed: %s", err)
        }
    }

    // Try to send the request
    resp,err := conn.sendRequest(7,request.Bytes())
    if err!=nil {
        return
    }

    // Check the result error code
    result := binary.LittleEndian.Uint32(resp[0:4])
    if result>0 {
        err = errors.New("Got ADS error number "+strconv.FormatUint(uint64(result),10)+" in DeleteDeviceNotification")
        return
    }

    return
}/*}}}*/
/*}}}*/
// DeviceNotification			- ADS command id: 8/*{{{*/
func (conn *Connection) DeviceNotification(in []byte) {
	type ADSNotificationStream struct {
		Length		uint32
		Stamps		uint32
	}
	type ADSStampHeader struct {
		Timestamp	uint64
		Samples		uint32
	}
	type ADSNotificationSample struct {
		Handle		uint32
		Size		uint32
	}

	var stream ADSNotificationStream
	var header ADSStampHeader
	var sample ADSNotificationSample
	var content []byte

	data := bytes.NewBuffer(in)

	// Read stream header
	binary.Read(data, binary.LittleEndian, &stream)

	for i:=0;i<int(stream.Stamps);i++ {
		// Read stamp header
		binary.Read(data, binary.LittleEndian, &header)

		for j:=0;j<int(header.Samples);j++ {
			binary.Read(data, binary.LittleEndian, &sample)

			content = make([]byte,sample.Size)
			data.Read(content);

			_, test := conn.activeNotifications[sample.Handle]

			if test {
				// Try to send the response to the waiting request function
				select {
					case conn.activeNotifications[sample.Handle] <- content:
						logger.Debugf("Successfully delived notification for handle %d",sample.Handle);
					default:
						logger.Errorf("Failed to deliver notification for handle %d, deleting device notification",sample.Handle);
                        conn.DeleteDeviceNotification(sample.Handle)
				}
			}
		}
	}



}
/*}}}*/
// ReadWrite TODO				- ADS command id: 9/*{{{*/
func (conn *Connection) ReadWrite() {
}
/*}}}*/
