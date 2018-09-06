package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gordonklaus/portaudio"
	"github.com/labstack/echo"
)

var (
	port                         = flag.String("port", ":1323", "Port to host on.")
	ffmpegCommand                []string
	encoderReader, encoderWriter = io.Pipe()
	streamReader, streamWriter   = io.Pipe()
	fGetDevices                  = flag.Bool("devices", false, "return the devices")
	fDeviceName                  = flag.String("name", "", "device name")
	fBufferSize                  = flag.Int("bufSize", 8192, "size of the port audio buffer")
	fSampleRate                  = flag.Float64("sr", 48000.0, "samplerate for port audio ")
	fChannels                    = flag.Int("chans", 2, "number of channels - set incorreclty for interesting effects")
	fOutputFormat                = flag.String("of", "aac", "output format")
	// ErrNoDeviceFound --
	ErrNoDeviceFound = errors.New("no port audio device found with that name")
	// ErrCouldNotGetDevices --
	ErrCouldNotGetDevices = errors.New("could not get devices list")
)

// stream reads the stream from port audio and writes it to the supplied writer
func stream(w io.Writer, deviceName string, sizeBuf int, sampleRate float64) {
	d, err := getDeviceByName(deviceName)
	if err != nil {
		panic(err)
	}

	portaudio.Initialize()
	defer portaudio.Terminate()

	in := make([]int32, sizeBuf)
	stream, err := openStream(d, nil, 2, 2, sampleRate, sizeBuf, in)
	if err != nil {
		panic(err)
	}
	defer stream.Close()
	if err := stream.Start(); err != nil {
		panic(err)
	}
	for {
		if err := stream.Read(); err != nil {
			log.Println(err)
		}
		binary.Write(w, binary.LittleEndian, in)
	}
}

func streamTranscoder(r io.Reader, w io.Writer, commands []string) {
	ffmpeg := exec.Command("ffmpeg", commands...)

	ffmpeg.Stdin = r
	ffmpeg.Stderr = os.Stderr
	ffmpeg.Stdout = w

	if err := ffmpeg.Run(); err != nil {
		panic(err)
	}
}

// getDevices returns the audio devices on the system
func getDevices() ([]*portaudio.DeviceInfo, error) {
	portaudio.Initialize()
	defer portaudio.Terminate()
	hs, err := portaudio.HostApis()
	if err != nil {
		return nil, err
	}
	return hs[0].Devices, nil
}

// getDeviceByName returns the portaudio data structure for the named device
func getDeviceByName(name string) (*portaudio.DeviceInfo, error) {
	devices, err := getDevices()
	if err != nil {
		return nil, ErrCouldNotGetDevices
	}
	for _, device := range devices {
		if device.Name == name {
			return device, nil
		}
	}
	return nil, ErrNoDeviceFound
}

// openStream will is similar to OpenDefaultStream however it takes a deviceInfo structure as a parameter
func openStream(inDev *portaudio.DeviceInfo, outDev *portaudio.DeviceInfo, numInputChannels, numOutputChannels int, sampleRate float64, framesPerBuffer int, args ...interface{}) (*portaudio.Stream, error) {
	p := portaudio.LowLatencyParameters(inDev, outDev)
	p.Input.Channels = numInputChannels
	p.Output.Channels = numOutputChannels
	p.SampleRate = sampleRate
	p.FramesPerBuffer = framesPerBuffer
	return portaudio.OpenStream(p, args...)
}

func parseFFMPEGCommand(channels int, outputFormat string, sampleRate float64) []string {
	sr := ""
	switch sampleRate {
	case 16000.0:
		sr = "16k"
	case 48000.0:
		sr = "48k"
	default:
		sr = "48k"
	}
	return []string{
		"-v", "error",
		"-f", "s32le",
		"-ac", strconv.Itoa(channels),
		"-ar", sr,
		"-i", "-",
		"-c:a", outputFormat,
		"-ar", sr,
		"-ac", strconv.Itoa(channels),
		"-f", "adts",
		"-",
	}
}

// Parse and validate CLI arguments
func init() {
	flag.Parse()

	if *fGetDevices {
		devices, err := getDevices()
		if err != nil {
			panic(err)
		}
		for _, device := range devices {
			fmt.Println(device.Name)
		}
		os.Exit(0)
	}

	if *fDeviceName == "" {
		fmt.Println("err: device name was empty")
		os.Exit(0)
	}
	ffmpegCommand = parseFFMPEGCommand(*fChannels, *fOutputFormat, *fSampleRate)
}

func main() {
	// get source stream from portaudio
	go stream(encoderWriter, *fDeviceName, *fBufferSize, *fSampleRate)
	// transcode what was written to encoderWriter and pass it to streamWriter
	go streamTranscoder(encoderReader, streamWriter, ffmpegCommand)
	// read from streamWriter and write it to all connected clients
	e := echo.New()
	e.GET("/", func(c echo.Context) error {
		return c.Stream(http.StatusOK, "audio/aac", streamReader)
	})
	e.Logger.Fatal(e.Start(*port))
}
