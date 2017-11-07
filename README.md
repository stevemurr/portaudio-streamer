# PortAudio Streamer
This allows you to pipe output from an audio device on your system and stream it to the internet.  

Modified https://github.com/bakape/livestreamer-aac to use an audio device as a stream source rather than a video stream.  

# Installation

You will need software to create a virtual audio device in order to use this.  On Mac, you can use `SoundFlower` which is available [here](https://github.com/mattingalls/Soundflower/releases/tag/2.0b2).  

After installing SoundFlower you will likely want t create a `Multi-Output Device` in `Audio MIDI Setup` if you would like to hear what you're streaming on the host system.  

In the main pane, click the `+` sign on the bottom left corner and tick `Soundflower (2ch)` and `Built-in Output` and any other output's you want.  

# Usage

Stream your soundflower sink to the internet  
`go run main.go -name "Soundflower (2ch)"`  

Open a browser and navigate to `localhost:8005` and observe the fat beats.  

