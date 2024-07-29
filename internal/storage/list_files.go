package storage

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func ListFilesInBucket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
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

	// Создание клиента S3
	svc := s3.New(sess)

	// Параметры для ListObjectsV2
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}

	// Вызов ListObjectsV2 для получения списка объектов
	resp, err := svc.ListObjectsV2(params)
	if err != nil {
		fmt.Errorf("ошибка при получении списка объектов: %w", err)
		return
	}

	type FileInfo struct {
		Name         string `json:"name"`
		Size         int64  `json:"size"`
		LastModified string `json:"last_modified"`
	}

	// Response представляет JSON ответ
	type Response struct {
		Files []FileInfo `json:"files"`
	}

	var files []FileInfo
	for _, item := range resp.Contents {
		files = append(files, FileInfo{
			Name:         *item.Key,
			Size:         *item.Size,
			LastModified: item.LastModified.Format("2006-01-02 15:04:05"),
		})
	}

	response := Response{
		Files: files,
	}

	// Установка заголовков
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Println(response.Files)
	// Сериализация ответа в JSON и отправка
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Ошибка при сериализации ответа: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
