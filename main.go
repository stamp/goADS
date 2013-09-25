package goADS

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Connection struct {
	connection  net.Conn
	target      AMSAddress
	source      AMSAddress
	sendChannel chan []byte

	symbols   map[string]ADSSymbolUploadSymbol
	datatypes map[string]ADSSymbolUploadDataType
}

type AMSAddress struct {
	netid [6]byte
	port  uint16
}

// List of active requests that waits a response, invokeid is key and value is a channel to the request rutine
var activeRequests = map[uint32]chan []byte{}
var activeNotifications = map[uint32]chan []byte{}
var invokeID uint32 = 0
var invokeIDmutex = &sync.Mutex{}

// Shutdown tools
var shutdown = make(chan bool)
var shutdownFinal = make(chan bool)
var WaitGroup sync.WaitGroup
var WaitGroupFinal sync.WaitGroup

var buf [1024000]byte

func Dial(ip string, netid string, port int) (conn Connection, err error) { /*{{{*/
	defer logger.Flush()

	logger.Infof("Dailing ip: %s NetID: %s", ip, netid)
	conn.connection, err = net.Dial("tcp", fmt.Sprintf("%s:48898", ip))
	//conn.connection, err = net.Dial("tcp", fmt.Sprintf("%s:6666",ip))
	if err != nil {
		return
	}
	logger.Trace("Connected")

	conn.target = stringToNetId(netid)
	conn.target.port = 801

	localhost, _, _ := net.SplitHostPort(conn.connection.LocalAddr().String())
	conn.source = stringToNetId(localhost)
	conn.source.netid[4] = 1
	conn.source.netid[5] = 1
	conn.source.port = 800

	conn.sendChannel = make(chan []byte)

	go reciveWorker(&conn)
	go transmitWorker(&conn)

	return
} /*}}}*/

func stringToNetId(source string) (result AMSAddress) { /*{{{*/
	localhost_split := strings.Split(source, ".")

	for i, a := range localhost_split {
		value, _ := strconv.ParseUint(a, 10, 8)
		result.netid[i] = byte(value)
	}
	return
} /*}}}*/

func reciveWorker(conn *Connection) { /*{{{*/
	WaitGroupFinal.Add(1)

	// Create a buffer so we can join halfdone messages
	var buff bytes.Buffer

	// Create a listner
	read := listen(conn)

loop:
	for {
		select {
		case data := <-read:
			if data == nil {
				logger.Error("Got an error from the socket reader")
				break loop
			}
			logger.Tracef("Got data!: \r\n%s", hex.Dump(data))

			// Add it to the buffer
			buff.Write(data)

			// Decode the AMS header
			for buff.Len() > 38 {
				logger.Tracef("Buffer len: %d bytes", buff.Len())

				// Read the header
				header := make([]byte, 38)
				buff.Read(header)

				command, length, invoke, err := conn.decode(header)
				if err != nil {
					logger.Warnf("Failed to decode AMS header: %s", err)
					continue
				}

				// Read the body
				pack := make([]byte, length)
				n, _ := buff.Read(pack)

				if n != int(length) {
					logger.Tracef("Did not get the whole message, only got %d bytes of %d, adding data back to buffer", n, length)
					buff.Write(header)
					buff.Write(pack[:n])
					break // Wait for more data
				}

				switch command {
				case 8:
					conn.DeviceNotification(pack)
				default:
					// Check if the response channel exists and is open
					_, test := activeRequests[invoke]

					if test {
						// Try to send the response to the waiting request function
						select {
						case activeRequests[invoke] <- pack:
							logger.Debugf("Successfully deliverd answer to invoke %d - command %d", invoke,command)
						default:
						}
					} else {
						logger.Debug("Got broadcast")
						logger.Debug(hex.Dump(pack))
					}
				}
			}
		case <-shutdownFinal:
			logger.Debug("Exit reciveWorker")
			break loop
		}
	}

	WaitGroupFinal.Done()
}                                             /*}}}*/
func listen(conn *Connection) <-chan []byte { /*{{{*/
	c := make(chan []byte)

	go func(conn *Connection) {
		b := make([]byte, 1024)

		for {
			n, err := conn.connection.Read(b)
			if n > 0 {
				res := make([]byte, n)
				copy(res, b[:n])
				c <- res
			}
			if err == io.EOF {
				//fmt.Println("client: Read EOF",n) 
				break
			}
			if err != nil {
				logger.Errorf("Failed to read socket: %s", err)
				c <- nil
				break
			}
		}
	}(conn)

	return c
} /*}}}*/

func transmitWorker(conn *Connection) { /*{{{*/
	WaitGroupFinal.Add(1)

loop:
	for {
		select {
		case data := <-conn.sendChannel:
			logger.Tracef("Sending %d bytes", len(data))
			conn.connection.Write(data)
		case <-shutdownFinal:
			logger.Debug("Exit reciveWorker")
			break loop
		}
	}

	WaitGroupFinal.Done()
} /*}}}*/

func (conn *Connection) Close() { /*{{{*/
	logger.Trace("CLOSE is called")

	if shutdown != nil {
		logger.Debug("Sending shutdown to workers")
		close(shutdown)
		shutdown = nil

		logger.Debug("Waiting for workers to close")
		WaitGroup.Wait()

		logger.Debug("Sending shutdown to connection")
		close(shutdownFinal)
		shutdownFinal = nil

		logger.Debug("Waiting for connection to close")
		WaitGroupFinal.Wait()
	}

	//logger.Critical("Shutdown")
} /*}}}*/
func (conn *Connection) Wait() {
	WaitGroup.Wait()
	WaitGroupFinal.Wait()
}

func getNewInvokeId() uint32 { /*{{{*/
	invokeIDmutex.Lock()
	invokeID++
	id := invokeID
	invokeIDmutex.Unlock()

	return id
} /*}}}*/

func (conn *Connection) sendRequest(command uint16, data []byte) (response []byte, err error) { /*{{{*/
	WaitGroup.Add(1)

	// First, request a new invoke id
	id := getNewInvokeId()

	// Create a channel for the response
	activeRequests[id] = make(chan []byte)

	pack := conn.encode(command, data, id)

	conn.sendChannel <- pack

	select {
	case response = <-activeRequests[id]:
		WaitGroup.Done()
		return
	case <-time.After(time.Second * 4):
		WaitGroup.Done()
		return response, errors.New("Timeout, got no answer in 4sec")
	case <-shutdown:
		WaitGroup.Done()
		return response, errors.New("Request aborted, shutdown initiated")
	}

	return
}                                                                                          /*}}}*/
func (conn *Connection) createNotificationWorker(data []byte,callback func([]byte)) (handle uint32, err error) { /*{{{*/
	WaitGroup.Add(1)

	// First, request a new invoke id
	id := getNewInvokeId()

	// Create a channel for the response
	activeRequests[id] = make(chan []byte)

	pack := conn.encode(uint16(6), data, id)

	conn.sendChannel <- pack

	select {
	case response := <-activeRequests[id]:
		result := binary.LittleEndian.Uint32(response[0:4])
		handle = binary.LittleEndian.Uint32(response[4:8])
		if result > 0 {
			err = fmt.Errorf("Got ADS error number %i", result)
			WaitGroup.Done()
			return
		}

		go func() {
			WaitGroup.Add(1)
			logger.Debug("Started notification reciver for ", handle)
			activeNotifications[handle] = make(chan []byte)

		Label:
			for {
				select {
				case response = <-activeNotifications[handle]:
					//logger.Warn(hex.Dump(response))
					callback(response)
				case <-shutdown:
					break Label
				}
			}

			conn.DeleteDeviceNotification(handle)
			close(activeNotifications[handle])
			logger.Debug("Closed notification reciver for ", handle)
			WaitGroup.Done()
		}()

	case <-time.After(time.Second * 4):
		WaitGroup.Done()
		return handle, errors.New("Timeout, got no answer in 4sec")
	case <-shutdown:
		WaitGroup.Done()
		return handle, errors.New("Request aborted, shutdown initiated")
	}

	return
} /*}}}*/
