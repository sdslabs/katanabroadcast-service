package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
)

func sendFile(file []byte, params map[string]string, filename, uri string) error {
	client := &http.Client{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(os.Getenv("CHALLENGE_ARTIFACT"), filename)
	if err != nil {
		return err
	}

	part.Write(file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}

	if err = writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client.Do(req)
	return nil
}

func upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(2 << 30)
	var fileName string
	for k := range r.MultipartForm.File {
		fileName = k
		break
	}

	if fileName == "" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: no challenge file provided")
		return
	}

	file, handler, err := r.FormFile(fileName)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s", err.Error())
		return
	}
	defer file.Close()

	challName := r.MultipartForm.Value["challenge_name"][0]
	targets := []string{}
	json.Unmarshal([]byte(r.MultipartForm.Value["targets"][0]), &targets)

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s", err.Error())
		return
	}

	params := make(map[string]string)
	params["challenge_name"] = challName

	var errs string
	var wg sync.WaitGroup
	wg.Add(len(targets))

	for _, target := range targets {
		go func(target string) {
			err = sendFile(bytes, params, handler.Filename, target)
			if err != nil {
				errs = fmt.Sprintf("%s\n%s", errs, err.Error())
			}
			wg.Done()
		}(target)
	}

	if errs != "" {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("Error broadcasting file: %s\n", err.Error())
		fmt.Fprintf(w, "Error: %s", err.Error())
	} else {
		fmt.Printf("Challenge %s successfully broadcasted\n", challName)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, "Challenge file broadcasted successfully")
	}
}

func setupRoutes() error {
	http.HandleFunc("/upload", upload)
	return http.ListenAndServe(":3003", nil)
}

func main() {
	if err := setupRoutes(); err != nil {
		log.Fatal(err)
	}
}
