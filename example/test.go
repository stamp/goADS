package main

import (
	"flag"
	"fmt"
	"sync"

	"github.com/stamp/goADS"

	"os"
	"os/signal"
	"syscall"

	//   "time"

	log "github.com/cihub/seelog"
	//    "bytes"
	//    "encoding/hex"
	//    "encoding/binary"
)

var WaitGroup sync.WaitGroup

func main() {
	defer log.Flush()

	// Flags/*{{{*/
	debug := flag.Bool("debug", false, "print debugging messages.")
	ip := flag.String("ip", "172.16.21.10", "the address to the AMS router")
	netid := flag.String("netid", "172.16.21.10.1.1", "AMS NetID of the target")
	port := flag.Int("port", 801, "AMS Port of the target")

	flag.Parse()
	fmt.Println(*debug, *ip, *netid, *port) /*}}}*/

	// Start the logger/*{{{*/
	logger, err := log.LoggerFromConfigAsFile("logconfig.xml")
	if err != nil {
		panic(err)
	}
	log.ReplaceLogger(logger)
	goADS.UseLogger(logger) /*}}}*/
	// Startup the connection/*{{{*/
	connection, e := goADS.NewConnection(*ip, *netid, *port)
	connection.Connect()
	defer connection.Close() // Close the connection when we are done
	if e != nil {
		logger.Critical(e)
		os.Exit(1)
	} /*}}}*/
	// Add a handler for Ctrl^C,  soft shutdown/*{{{*/
	go shutdownRoutine(connection) /*}}}*/

	// Check what device are we connected to/*{{{*/
	data, e := connection.ReadDeviceInfo()
	if e != nil {
		log.Critical(e)
		os.Exit(1)
	}
	log.Infof("Successfully conncected to \"%s\" version %d.%d (build %d)", data.DeviceName, data.MajorVersion, data.MinorVersion, data.BuildVersion) /*}}}*/

	symbols, _ := connection.UploadSymbolInfo()

	//for _, segment := range symbols {
	//segment.Walk()
	//}
	log.Warn("Count: ", len(symbols))

	//pd,ok := symbols[".RECIPE"]
	pd, ok := symbols[".PD"]
	//pd,ok := symbols[".SYSTEMTASKINFOARR"]
	if ok {
		pd.AddDeviceNotification(func(symbol *goADS.ADSSymbol) {
			val := connection.Value(".PD.TEST1BOOL")
			if val == "True" {
				connection.Set(".PD.TEST1WORD", "1")
			} else {
				connection.Set(".PD.TEST1WORD", "0")
			}
			symbol.Walk()
		})
		//		pd.DebugWalk()
	}

	// Do some work/*{{{*/
	/*for i := 0; i < 100; i++ {
	    WaitGroup.Add(1)
	    go func() {
	        _, e = connection.ReadDeviceInfo();
	        if e != nil {
	            log.Critical(e)
	            //connection.Close()
	        }
	        WaitGroup.Done()
	    }()
	}*/ /*}}}*/

	// Wait for all routines to finish/*{{{*/
	WaitGroup.Wait()
	connection.Wait()

	log.Info("MAIN Done :)") /*}}}*/
}

func shutdownRoutine(conn *goADS.Connection) { /*{{{*/
	sigchan := make(chan os.Signal, 2)
	signal.Notify(sigchan, os.Interrupt)
	signal.Notify(sigchan, syscall.SIGTERM)
	<-sigchan

	conn.Close()
} /*}}}*/
