package utils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func UploadToS3(ctx context.Context, photo []byte, photoType string, keyName string) error {
	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")
	bucketName := os.Getenv("S3_BUCKET_NAME")
	region := os.Getenv("AWS_REGION")
	// Create a custom AWS config with credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create an S3 client
	client := s3.NewFromConfig(cfg)

	// Prepare the upload input
	input := &s3.PutObjectInput{
		Bucket:        aws.String(bucketName),
		Key:           aws.String(keyName), // e.g., "images/photo.jpg"
		Body:          bytes.NewReader(photo),
		ContentLength: aws.Int64(int64(len(photo))),
		ContentType:   aws.String("image/" + photoType), // or "image/png", etc.
	}

	// Perform the upload
	_, err = client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func GetPresignedURL(ctx context.Context, keyName string) (string, error) {
	accessKey := os.Getenv("AWS_ACCESS_KEY")
	secretKey := os.Getenv("AWS_SECRET_KEY")
	bucketName := os.Getenv("S3_BUCKET_NAME")
	region := os.Getenv("AWS_REGION")
	// Create a custom AWS config with credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		),
	)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	// Create an S3 client
	presigner := s3.NewPresignClient(s3.NewFromConfig(cfg))
	// Prepare the presigned URL input
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(keyName),
	}
	// Generate the presigned URL
	presignedURL, err := presigner.PresignGetObject(ctx, input, func(o *s3.PresignOptions) {
		o.Expires = 15 * time.Minute // Set the expiration time for the presigned URL
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}
	return presignedURL.URL, nil
}
