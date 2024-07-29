package storage

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func UploadFileToS3(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получение дополнительных данных
	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Поле username отсутствует", http.StatusBadRequest)
		return
	}

	// Извлечение токена из заголовка Authorization
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization header is required", http.StatusUnauthorized)
		return
	}

	// Проверка формата заголовка и извлечение токена
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
		return
	}
	token := parts[1]

	authOk := CheckUser(username, token)
	if !authOk {
		http.Error(w, "Failed to authentification", http.StatusInternalServerError)
		return
	}

	// Чтение файла из формы данных
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	accessKey, secretKey := GetKeys(username)
	if accessKey == "" || secretKey == "" {
		http.Error(w, "Error getting keys", http.StatusBadRequest)
		return
	}

	// Initialize a session using Amazon S3
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-west-2"),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:         aws.String("https://storage.clo.ru"),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusBadRequest)
		return
	}

	// Create S3 service client
	svc := s3.New(sess)

	// Upload the file to S3
	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(username + "-default-bucket"),
		Key:    aws.String(handler.Filename),
		Body:   file,
		ACL:    aws.String("public-read"), // Adjust the ACL as per your requirement
	})
	if err != nil {
		http.Error(w, "Failed to upload file", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully!\n")
}
