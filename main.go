package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/ai-mastering/aimastering-go"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// parse options
	input := flag.String("input", "", "Input audio file path.")
	output := flag.String("output", "", "Output audio file path.")
	flag.Parse()

	// create API client
	client := aimastering.NewAPIClient(aimastering.NewConfiguration())
	auth := context.WithValue(context.Background(), aimastering.ContextAPIKey, aimastering.APIKey{
		Key: os.Getenv("AIMASTERING_ACCESS_TOKEN"),
	})

	// upload input audio
	inputAudioFile, err := os.Open(*input)
	if err != nil {
		log.Fatal(err)
	}
	defer inputAudioFile.Close()
	inputAudio, _, err := client.AudioApi.CreateAudio(auth, map[string]interface{}{
		"file":  inputAudioFile,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "The input audio was uploaded id %d\n", inputAudio.Id)

	// start the mastering
	mastering, _, err := client.MasteringApi.CreateMastering(auth, inputAudio.Id, map[string]interface{}{
		"mode": "default",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stderr, "The mastering started id %d\n", mastering.Id)

	// wait for the mastering completion
	for mastering.Status == "processing" || mastering.Status == "waiting" {
		mastering, _, err = client.MasteringApi.GetMastering(auth, mastering.Id)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr,
			"waiting for the mastering completion %d%%\n", int(100 * mastering.Progression))
		time.Sleep(5 * time.Second)
	}

	// download output audio
	// notes
	// - client.AudioApi.DownloadAudio cannot be used because swagger-codegen doesn't support binary string response in golang
	// - instead use GetAudioDownloadToken (to get signed url) + HTTP Get
	audioDownloadToken, _, err := client.AudioApi.GetAudioDownloadToken(auth, mastering.OutputAudioId)

	// http get signed url
	resp, err := http.Get(audioDownloadToken.DownloadUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	outputAudioFile, err := os.Create(*output)
	if err != nil  {
		log.Fatal(err)
	}
	defer outputAudioFile.Close()

	// write output
	_, err = io.Copy(outputAudioFile, resp.Body)
	if err != nil  {
		log.Fatal(err)
	}

	fmt.Fprintf(os.Stderr,
		"The Output audio was saved to %s\n", *output)
}
