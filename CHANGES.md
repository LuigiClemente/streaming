# Audio Streaming Between an iPhone and a Browser JavaScript App via a VPS

To enable audio streaming between an iPhone and a browser JavaScript app via a VPS, we need to modify the Go code. Below are the necessary steps:

## 1. Update the TCP Socket Connection

Update the TCP socket connection code to connect to the VPS IP and port instead of localhost.

```go
// File: transmission/tunnel.go

var tcpAddress = "VPS_IP:VPS_PORT"

func Tunnel() net.Conn {
  conn, err := net.Dial(tcpPort, tcpAddress)
  // Handle error if any
}
```

## 2. Modify Main Code to Listen for Audio

In the main code, listen on a local port for audio sent from the JavaScript app. Pass the received data to the USB send code.

```go
// File: main.go

ln, _ := net.Listen("tcp", ":LOCAL_PORT")
go func() {
  for {
    conn, _ := ln.Accept()
    // Get audio data from conn
    // Pass to usb.SendData()
  }
}()
```

## 3. Forward Audio Data from iPhone to VPS

In the USB receive callback, forward the audio data received from iPhone to the VPS port.

```go
// File: USB/connected_devices.go

func (d ConnectedDeviceDelegate) USBDeviceDidReceiveData(data []byte) {

  conn, _ := net.Dial("tcp", "VPS_IP:VPS_PORT")

  // Send data to VPS
}
```

## 4. Audio Input via JavaScript

In JavaScript, obtain the audio input using the Web Audio API and send it to the Go code listening on the VPS port.

```javascript
// Get input stream from microphone

const source = audioContext.createMediaStreamSource(stream)

// Send audio data to VPS port
const socket = new WebSocket('ws://VPS_IP:VPS_PORT') 
source.connect(socket)
```

## 5. Receive and Play Audio Data

Finally, receive the audio data on the VPS port and play it using the Web Audio API.

```javascript
// Get audio stream from VPS port

const socket = new WebSocket('ws://VPS_IP:VPS_PORT')

// Play received audio
socket.onmessage = function(e) {
  audioContext.decodeAudioData(e.data, function(buffer) {
    const source = audioContext.createBufferSource()
    source.buffer = buffer
    source.connect(audioContext.destination)
    source.start()
  })
}
```
