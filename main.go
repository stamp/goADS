package goADS

import (
    "fmt"
    "net"
    "sync"
    "time"
    "errors"
    "strconv"
    "strings"
    "io"
    "encoding/hex"
    "bytes"
)

type Connection struct {
    connection net.Conn
    target AMSAddress
    source AMSAddress
    sendChannel chan []byte
}

type AMSAddress struct {
    netid [6]byte
    port uint16
}

// List of active requests that waits a response, invokeid is key and value is a channel to the request rutine
var activeRequests = map[uint32] chan []byte{}
var invokeID uint32 = 0
var invokeIDmutex = &sync.Mutex{}

// Shutdown tools
var shutdown = make(chan bool)
var WaitGroup sync.WaitGroup

var buf [1024000]byte


func Dial(ip string,netid string, port int) (conn Connection,err error)  {
    defer logger.Flush()

    logger.Infof("Dailing ip: %s NetID: %s", ip, netid)
    conn.connection, err = net.Dial("tcp", fmt.Sprintf("%s:48898",ip))
    //conn.connection, err = net.Dial("tcp", fmt.Sprintf("%s:6666",ip))
    if err != nil {
        return
    }
    logger.Trace("Connected");

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
}

func stringToNetId(source string) (result AMSAddress) {
    localhost_split := strings.Split(source,".")

    for i, a := range localhost_split {
        value,_ := strconv.ParseUint(a,10,8)
        result.netid[i] = byte(value)
    }
    return
}

func reciveWorker(conn *Connection) {
    WaitGroup.Add(1)

    // Create a buffer so we can join halfdone messages
    var buff bytes.Buffer;

    // Create a listner
    read := listen(conn)

    loop:
    for {
        select {
            case data := <-read:
                if data == nil {
                    logger.Error("Got an error from the socket reader");
                    break loop
                }
                logger.Tracef("Got data!: \r\n%s",hex.Dump(data))

                // Add it to the buffer
                buff.Write(data)

                // Decode the AMS header
                for buff.Len()>38 {
                    logger.Tracef("Buffer len: %d bytes",buff.Len())

                    // Read the header
                    header := make([]byte,38)
                    buff.Read(header)

                    length, invoke, err := conn.decode(header)
                    if err!=nil {
                        logger.Warnf("Failed to decode AMS header: %s",err)
                        continue
                    }

                    // Read the body
                    pack := make([]byte,length)
                    n,_ := buff.Read(pack)

                    if n!=int(length) {
                        logger.Tracef("Did not get the whole message, only got %d bytes of %d, adding data back to buffer",n,length)
                        buff.Write(header)
                        buff.Write(pack[:n])
                        break // Wait for more data
                    }

                    // Check if the response channel exists and is open
                    _, test := activeRequests[invoke]

                    if test {
                        // Try to send the response to the waiting request function
                        select {
                            case activeRequests[invoke] <- pack:
                                logger.Debugf("Successfully recived answer to invoke %d",invoke);
                            default:
                        }
                    }
                }
            case <-shutdown:
                logger.Debug("Exit reciveWorker")
                break loop
        }
    }

    WaitGroup.Done()
}
func listen(conn *Connection) <-chan []byte {/*{{{*/
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
                                logger.Errorf("Failed to read socket: %s",err)
                                c <- nil
                                break
                        }
                 }
        }(conn)

        return c 
}/*}}}*/

func transmitWorker(conn *Connection) {
    WaitGroup.Add(1)

    loop:
    for {
        select {
            case data := <-conn.sendChannel:
                logger.Tracef("Sending %d bytes",len(data))
                conn.connection.Write(data)
            case <-shutdown:
                logger.Debug("Exit reciveWorker");
                break loop
        }
    }

    WaitGroup.Done()
}

func (conn *Connection) Close() {/*{{{*/
    logger.Trace("CLOSE is called")

    if shutdown!=nil {
        logger.Debug("Sending shutdown to workers")
        close(shutdown)
        shutdown = nil

        logger.Debug("Waiting for workers to close")
        WaitGroup.Wait()
    }

    //logger.Critical("Shutdown")
}/*}}}*/

func getNewInvokeId() uint32 {/*{{{*/
    invokeIDmutex.Lock()
    invokeID++
    id := invokeID
    invokeIDmutex.Unlock()

    return id
}/*}}}*/

func (conn *Connection) sendRequest(command uint16, data []byte) (response []byte, err error) {
    WaitGroup.Add(1)

    // First, request a new invoke id
    id := getNewInvokeId()

    // Create a channel for the response
    activeRequests[id] = make(chan []byte)

    pack := conn.encode(command,data,id);

    conn.sendChannel <- pack

    select{
        case response = <-activeRequests[id]:
            WaitGroup.Done()
            return
        case <-time.After(time.Second*2):
            WaitGroup.Done()
            return response, errors.New("Timeout, got no answer in 2sec")
        case <-shutdown:
            WaitGroup.Done()
            return response, errors.New("Request aborted, shutdown initiated")
    }

    return
}
