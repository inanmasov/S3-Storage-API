package main

import (
	storage "S3Storage/internal/storage"
	"fmt"
	"log"
	"net/http"
	"os"
)

func redirectToHttps(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://127.0.0.1:8443"+r.RequestURI, http.StatusMovedPermanently)
}

func main() {
	http.HandleFunc("/create-user", storage.Create)
	http.HandleFunc("/delete-user", storage.Delete)
	http.HandleFunc("/upload-file", storage.UploadFileToS3)
	http.HandleFunc("/download-file", storage.DownloadFileFromS3)
	http.HandleFunc("/delete-file", storage.DeleteFileFromS3)
	http.HandleFunc("/list-files", storage.ListFilesInBucket)

	log.Println("http/https server start listening on port", 8442, 8443)

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Ошибка получения рабочего каталога:", err)
		return
	}

	go http.ListenAndServeTLS(":8443", dir+"/certificate/server.crt", dir+"/certificate/server.key", nil)

	http.ListenAndServe(":8442", http.HandlerFunc(redirectToHttps))
}
