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

func DeleteFileFromS3(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получение параметра username и filename из строки запроса
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

	// Получение ключей доступа
	accessKey, secretKey := GetKeys(username)
	if accessKey == "" || secretKey == "" {
		http.Error(w, "Ошибка получения ключей", http.StatusInternalServerError)
		return
	}

	// Создание новой сессии AWS с заданными ключами доступа
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-west-2"), // Укажите нужный регион
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		Endpoint:         aws.String("https://storage.clo.ru"),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		fmt.Errorf("ошибка при создании сессии AWS: %w", err)
		return
	}

	// Создание клиента S3
	svc := s3.New(sess)

	// Удаление объекта из S3
	_, err = svc.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		fmt.Errorf("ошибка при удалении объекта из S3: %w", err)
		return
	}

	// Ожидание завершения удаления
	err = svc.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
	})
	if err != nil {
		fmt.Errorf("ошибка при ожидании удаления объекта из S3: %w", err)
		return
	}

	// Отправка успешного ответа
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Файл %s успешно удалён из бакета %s", filename, bucketName)
}
