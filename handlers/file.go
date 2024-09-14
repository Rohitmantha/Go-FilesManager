package handlers

import (
    "fmt"
    "auth-app/config" // Ensure you have the correct import for your DB package
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/credentials"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3/s3manager"
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "log"
    "mime/multipart"
    "net/http"
    "os"
    "time"
)

// AWS S3 Configuration
var (
    awsRegion   string
    bucketName  string
    maxFileSize  = int64(10 * 1024 * 1024) // Max file size set to 10 MB
)

func init() {
    // Load environment variables from .env file
    if err := godotenv.Load(); err != nil {
        log.Fatalf("Error loading .env file")
    }

    awsRegion = os.Getenv("AWS_REGION")
    bucketName = os.Getenv("BUCKET_NAME")
}

// File Upload Handler
func UploadFile(c *gin.Context) {
    // Parse the user ID from the request context
    userID, ok := c.Get("user_id")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    // Assert the user ID as float64 and convert to int64 if necessary
    var userIDInt int64
    switch v := userID.(type) {
    case int64:
        userIDInt = v
    case float64:
        userIDInt = int64(v)
    default:
        log.Printf("Unexpected type for user_id: %T, value: %v\n", userID, userID)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user ID"})
        return
    }

    // Get file from form
    file, header, err := c.Request.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
        return
    }
    defer file.Close()

    // Validate file size
    if header.Size > maxFileSize {
        c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File exceeds maximum size of %d MB", maxFileSize/(1024*1024))})
        return
    }

    // Save file metadata to MySQL
    fileName := header.Filename
    fileSize := header.Size
    uploadDate := time.Now()

    // Create a channel to get the result of file upload and metadata saving
    resultChan := make(chan string)
    errorChan := make(chan error)

    // Handle large files with Goroutines
    go func() {
        // Upload file to S3
        fileURL, err := uploadToS3(file, fileName)
        if err != nil {
            errorChan <- fmt.Errorf("error uploading file to S3: %v", err)
            return
        }

        // Save metadata in MySQL
        if err := saveFileMetadata(userIDInt, fileName, fileSize, uploadDate, fileURL); err != nil {
            errorChan <- fmt.Errorf("error saving file metadata: %v", err)
            return
        }

        resultChan <- fileURL
    }()

    // Wait for the goroutine to finish and respond
    select {
    case fileURL := <-resultChan:
        c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully", "file_url": fileURL})
    case err := <-errorChan:
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    }
}

// Save file metadata in MySQL
func saveFileMetadata(userID int64, fileName string, fileSize int64, uploadDate time.Time, fileURL string) error {
    query := "INSERT INTO files (user_id, file_name, file_size, upload_date, s3_url) VALUES (?, ?, ?, ?, ?)"
    _, err := config.DB.Exec(query, userID, fileName, fileSize, uploadDate, fileURL)
    return err
}

// Upload file to S3
func uploadToS3(file multipart.File, fileName string) (string, error) {
    // Create a new session with AWS
    sess, err := session.NewSession(&aws.Config{
        Region:      aws.String(awsRegion),
        Credentials: credentials.NewStaticCredentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), ""),
    })
    if err != nil {
        return "", fmt.Errorf("could not create AWS session: %v", err)
    }

    // Create an S3 uploader
    uploader := s3manager.NewUploader(sess)

    // Upload the file to S3
    result, err := uploader.Upload(&s3manager.UploadInput{
        Bucket: aws.String(bucketName),
        Key:    aws.String(fileName),
        Body:   file,
    })
    if err != nil {
        return "", fmt.Errorf("could not upload file: %v", err)
    }

    // Return the file URL
    return result.Location, nil
}
