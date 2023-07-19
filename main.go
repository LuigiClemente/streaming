package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-usbmuxd/USB"
	"go-usbmuxd/frames"
	"go-usbmuxd/transmission"

	"nhooyr.io/websocket"
)

// some global vars
var connectHandle USB.ConnectedDevices
var port = 29173
var pluggedUSBDevices map[int]frames.USBDeviceAttachedDetachedFrame
var connectedUSB int // only stores the device id
var scanningInstance USB.Scan
var self USBDeviceDelegate

var websocketProxyHost string = "ws://49.13.56.241:6969/usbmux/client/subscribe"

func main() {
	// inti section
	connectedUSB = -1
	pluggedUSBDevices = map[int]frames.USBDeviceAttachedDetachedFrame{}
	scanningInstance = USB.Scan{}
	self = USBDeviceDelegate{}

	// logger
	logFile, err := os.OpenFile("kusb_ios.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Println(err)
	}
	defer logFile.Close()
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	var (
		listenConnection net.Conn
	)
	defer func() {
		if r := recover(); r != nil {
			listenConnection.Close()
			connectHandle.Connection.Close()
			scanningInstance.Stop()
			log.Printf("Error %v", r)
		}
	}()

	// create a USB.Listen(USBDeviceDelegate) instance. Pass a delegate to resolve the attached and detached callbacks
	// then on device added save ot to array/ map and send connect to a port with proper tag
	listenConnection = USB.Listen(transmission.Tunnel(), self)
	// defer listenConnection.Close()

	// connect to a random usb device, if Number == 0 then
	connectHandle = USB.ConnectedDevices{Delegate: self, Connection: transmission.Tunnel()}
	// defer connectHandle.Connection.Close()

	// scan defer
	// defer scanningInstance.Stop()

	// go func() {
	// 	for {
	// 		if len(pluggedUSBDevices) != 0 {
	// 			log.Printf("DEBUG %v", pluggedUSBDevices)
	// 		}
	// 	}
	// }()

	go func() {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, websocketProxyHost, nil)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		for {
			_, message, err := conn.Read(context.TODO())
			if err != nil {
				log.Printf("Read Error: %v", err)
				return
			}
			connectHandle.SendData(message, 106)
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	// run loop
	select {
	case <-quit:
		listenConnection.Close()
		connectHandle.Connection.Close()
		scanningInstance.Stop()
	}
}

// USBDeviceDelegate - USB Delegate Methods
type USBDeviceDelegate struct{}

// USBDeviceDidPlug - device plugged callback
func (usb USBDeviceDelegate) USBDeviceDidPlug(frame frames.USBDeviceAttachedDetachedFrame) {
	// usb has been plugged DO: startScanning
	log.Printf("[USB-INFO] : Device Plugged %s ID: %d\n", frame.Properties.SerialNumber, frame.DeviceID)
	pluggedUSBDevices[frame.DeviceID] = frame
	scanningInstance.Start(&connectHandle, frame, port)
	connectHandle.SendData([]byte("audio data"), 1)
}

// USBDeviceDidUnPlug - device unplugged callback
func (usb USBDeviceDelegate) USBDeviceDidUnPlug(frame frames.USBDeviceAttachedDetachedFrame) {
	// usb has been unplugged
	// stop scan
	log.Printf("[USB-INFO] : Device UnPlugged %s ID: %d\n", pluggedUSBDevices[frame.DeviceID].Properties.SerialNumber, frame.DeviceID)
	delete(pluggedUSBDevices, frame.DeviceID)
	scanningInstance.Stop()
}

// USBDidReceiveErrorWhilePluggingOrUnplugging - device plugging/unplugging callback
func (usb USBDeviceDelegate) USBDidReceiveErrorWhilePluggingOrUnplugging(err error, stringResponse string) {
	// plug or unplug error
	// stop scan
	if stringResponse != "" {
		//some unresolved message came
		//TODO - Implement some resolver to understand message received
	}
	log.Println("[USB-EM-1] : Some error encountered wile pluging and unpluging. ", err.Error())
	scanningInstance.Stop()
}

// USBDeviceDidSuccessfullyConnect - device successful connection callback
func (usb USBDeviceDelegate) USBDeviceDidSuccessfullyConnect(device USB.ConnectedDevices, deviceID int, toPort int) {
	// successfully connected to the port mentioned
	// stop the scan
	connectedUSB = deviceID
	log.Printf("HERE STOP SCAN")
	scanningInstance.Stop()
}

// USBDeviceDidFailToConnect - device connection failure callback
func (usb USBDeviceDelegate) USBDeviceDidFailToConnect(device USB.ConnectedDevices, deviceID int, toPort int, err error) {
	// error while communication in the socket
	// start scan
	connectedUSB = -1
	pluggedDeviceID := getFirstPluggedDeviceId()
	if pluggedDeviceID != -1 {
		scanningInstance.Start(&connectHandle, pluggedUSBDevices[pluggedDeviceID], port)
	}

}

// USBDeviceDidReceiveData - data received callback
func (usb USBDeviceDelegate) USBDeviceDidReceiveData(device USB.ConnectedDevices, deviceID int, messageTAG uint32, data []byte) {
	// received data from the device
	log.Println(string(data))
	//device.SendData(data[20:], 106)
}

// USBDeviceDidDisconnect - device disconnect callback
func (usb USBDeviceDelegate) USBDeviceDidDisconnect(devices USB.ConnectedDevices, deviceID int, toPort int) {
	// socket disconnect
	// start scan
	connectedUSB = -1
	pluggedDeviceID := getFirstPluggedDeviceId()
	if pluggedDeviceID != -1 {
		scanningInstance.Start(&connectHandle, pluggedUSBDevices[pluggedDeviceID], port)
	}
}

// MARK - helper functions here
// Needs restructuring, removal or other implementation
func getFirstPluggedDeviceId() int {
	var deviceID int = -1
	for deviceID, _ = range pluggedUSBDevices {
		break
	}
	return deviceID
}
