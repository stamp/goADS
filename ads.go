package goADS

import (
    "fmt"
    "encoding/binary"
    "errors"
    "bytes"
    "encoding/hex"
)

// ADS command id: 1
type ADSReadDeviceInfo struct {
    MajorVersion uint8
    MinorVersion uint8
    BuildVersion uint16
    DeviceName string
}
func (conn *Connection) ReadDeviceInfo() (response ADSReadDeviceInfo, err error) {
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
        err = fmt.Errorf("Got ADS error number %i",result)
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
}

// ADS command id: 2
type ADSRead struct {
    Data []byte
}
func (conn *Connection) Read(group uint32,offset uint32,length uint32) (response ADSRead, err error) {
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

    // Check the response length
    if len(resp)-8!=int(length) {
        return response, errors.New(fmt.Sprintf("Wrong length of response! Got %d bytes and it should be %d",len(resp),length) )
    }

    // Check the result error code
    result := binary.LittleEndian.Uint32(resp[0:4])
    datalength := binary.LittleEndian.Uint32(resp[4:8])
    if result>0 {
        err = fmt.Errorf("Got ADS error number %i",result)
        return
    }

    response.Data = resp[8:datalength+8]
    logger.Debugf("The read data at %d:%d: \r\n%s",group,offset,hex.Dump(response.Data))

    return
}

// ADS command id: 3
func (conn *Connection) Write() {
}

// ADS command id: 4
func (conn *Connection) ReadState() {
}

// ADS command id: 5
func (conn *Connection) WriteControl() {
}

// ADS command id: 6
func (conn *Connection) AddDeviceNotification() {
}

// ADS command id: 7
func (conn *Connection) DeleteDeviceNotification() {
}

// ADS command id: 8
func (conn *Connection) DeviceNotification() {
}

// ADS command id: 9
func (conn *Connection) ReadWrite() {
}

