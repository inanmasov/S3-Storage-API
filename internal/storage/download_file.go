package storage

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func DownloadFileFromS3(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")
	filename := r.URL.Query().Get("filename")
	if username == "" || filename == "" {
		http.Error(w, "Отсутствуют параметры username или filename", http.StatusBadRequest)
		return
	}
	bucketName := username + "-default-bucket"

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

	accessKey, secretKey := GetKeys(username)
	if accessKey == "" || secretKey == "" {
		fmt.Errorf("error getting keys")
		return
	}
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-west-2"),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:         aws.String("https://storage.clo.ru"),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		fmt.Errorf("failed to create session, %v", err)
		return
	}

	// Create S3 service client
	svc := s3.New(sess)

	// Get the file from S3
	output, err := svc.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		fmt.Errorf("failed to get object, %v", err)
		return
	}
	defer output.Body.Close()

	// Установка заголовков для ответа
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.Header().Set("Content-Type", *output.ContentType)

	// Копирование содержимого файла в http.ResponseWriter
	_, err = io.Copy(w, output.Body)
	if err != nil {
		http.Error(w, "Ошибка при отправке файла: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
