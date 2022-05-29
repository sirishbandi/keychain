package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const bucket = "keychainbucket"

var img []byte
var client *storage.Client
var it *storage.ObjectIterator
var imgLock sync.Mutex

// listFiles lists objects within specified bucket.
func listFiles() error {
	// bucket := "bucket-name"
	ctx := context.Background()

	it = client.Bucket(bucket).Objects(ctx, nil)
	return nil
}

func listFunc(w http.ResponseWriter, req *http.Request){
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
        defer cancel()

        it := client.Bucket(bucket).Objects(ctx, nil)
        for {
                attrs, err := it.Next()
                if err == iterator.Done {
                        break
                }
                if err != nil {
                        return
                }
                fmt.Fprintln(w, attrs.Name)
        }
        return
}

func getFunc(w http.ResponseWriter, req *http.Request) {
	if it == nil {
		listFiles()
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fmt.Println("Data being sent")

	imgLock.Lock()
	for _, d := range img {
		fmt.Fprintf(w, "%c", d)
	}
	imgLock.Unlock()

	// Get the next image to allow faster serving
	go func() {
		attrs, err := it.Next()
		if err == iterator.Done {
			listFiles()
			attrs, _ = it.Next()
		}

		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Second*50)
		defer cancel()

		rc, err := client.Bucket(bucket).Object(attrs.Name).NewReader(ctx)
		if err != nil {
			fmt.Println("Object().NewReader:", attrs, err)
			return
		}
		defer rc.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(rc)
		t := buf.Bytes()

		imgLock.Lock()
		img = make([]byte, base64.StdEncoding.EncodedLen(len(t)))
		base64.StdEncoding.Encode(img, t)
		//fmt.Println(img)
		imgLock.Unlock()

	}()

}

func postFunc(w http.ResponseWriter, req *http.Request) {
	//var err error
	object := "img_b64_" + time.Now().String()
	data, _ := ioutil.ReadAll(req.Body)
	img := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
	base64.StdEncoding.Decode(img, data)
	fmt.Println("Starting file uplaod")

	go func() {
		buf := bytes.NewBuffer(img)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
		defer cancel()

		// Upload an object with storage.Writer.
		wc := client.Bucket(bucket).Object(object).NewWriter(ctx)
		wc.ChunkSize = 0 // note retries are not supported for chunk size 0.

		if _, err := io.Copy(wc, buf); err != nil {
			fmt.Println("Unable to upload img, err:", err)
			return
		}
		// Data can continue to be added to the file until the writer is closed.
		if err := wc.Close(); err != nil {
			fmt.Println("Data end not found, err:", err)
			return
		}
		fmt.Println("File uploaded")
	}()
}

func runScript() string {
	cmd, err := exec.Command("/bin/sh", "script.sh").Output()
	if err != nil {
		fmt.Println("error running script err:", err)
	}
	output := string(cmd)
	return output
}

func youtubeChannel() {
	ctx := context.Background()
	youtubeService, err := youtube.NewService(ctx, option.WithAPIKey("AIzaSyBp62WYnrV5dXAzdv8LkZ7K2zmXNqcuDCo"))
	if err != nil {
		fmt.Println("Error creating YouTube API service:", err)
		return
	}

	for {
		channel := youtube.NewChannelsService(youtubeService)
		channelService := channel.List([]string{"statistics"})
		response, err := channelService.Id("UCXQJydP8GCBSaB7XpFTIoXg").Do()
		if err != nil {
			fmt.Println("Error making YouTube API call:", err)
			return
		}
		stats := response.Items[0].Statistics
		fmt.Println(response.Items[0].Statistics)

		const S = 200

		fmt.Println("Updating Youtube channel img.")
		im, err := gg.LoadJPG("channel.jpg")
		if err != nil {
			fmt.Println("Could not read image template,", err)
			return
		}
		dc := gg.NewContext(S, S)
		dc.DrawImage(im, 0, 0)
		dc.SetRGB(0, 0, 0)
		font, err := truetype.Parse(goregular.TTF)
		if err != nil {
			fmt.Println("Could not set font,", err)
			return
		}
		face := truetype.NewFace(font, &truetype.Options{
			Size: 35,
		})
		dc.SetFontFace(face)

		// Subs count
		dc.DrawStringAnchored(strconv.Itoa(int(stats.SubscriberCount)), 150, 60, 0.5, 0.5)

		face = truetype.NewFace(font, &truetype.Options{
			Size: 25,
		})
		dc.SetFontFace(face)
		// Views count
		dc.DrawStringAnchored(strconv.Itoa(int(stats.ViewCount)), 145, 130, 0.5, 0.5)
		// Videos Coumt
		dc.DrawStringAnchored(strconv.Itoa(int(stats.VideoCount)), 145, 160, 0.5, 0.5)

		err = dc.SavePNG("channel.png")
		if err != nil {
			fmt.Println("Error saving image:", err)
			return
		}

		fmt.Println("Script output:", runScript())

		time.Sleep(time.Minute * 15)
	}

}

func main() {
	img = []byte{}
	//paste()

	ctx := context.Background()
	var err error
	client, err = storage.NewClient(ctx)
	if err != nil {
		fmt.Println("Unable to conenct to GCS, err:", err)
		return
	}
	defer client.Close()

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/keychain/get", getFunc)
	http.HandleFunc("/keychain/post", postFunc)
	http.HandleFunc("/keychain/lsit", listFunc)
	fmt.Println("Starting server")

	go func() {
		for {
			youtubeChannel()
			fmt.Println("Youtube Channel func exited, restarting")
		}
	}()

	if err = http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Could not start server:", err)
	}
}
