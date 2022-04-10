package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
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
		fmt.Println(img)
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
	fmt.Println("Starting server")
	http.ListenAndServe(":8080", nil)
}
