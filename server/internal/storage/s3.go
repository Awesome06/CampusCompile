package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var S3Client *s3.Client

const BucketName = "campus-testcases"

func InitS3() {
	// 1. STRICT SECRETS VALIDATION
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")

	if endpoint == "" || accessKey == "" || secretKey == "" {
		log.Fatal("FATAL STARTUP ERROR: S3/MinIO credentials (S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY) are missing")
	}

	// 2. LOAD CONFIGURATION (Without the deprecated global resolver)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Unable to load S3 config: %v", err)
	}

	// 3. APPLY BASE ENDPOINT DIRECTLY TO THE CLIENT
	S3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // Required for MinIO
	})

	// AUTO-CREATE BUCKET IF IT DOESN'T EXIST (With proper error inspection)
	ctx := context.TODO()
	_, err = S3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(BucketName),
	})

	if err != nil {
		// Only attempt to auto-create if we receive a definitive "Not Found" error
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchBucket") || strings.Contains(err.Error(), "404") {
			fmt.Printf("[*] MinIO Bucket '%s' not found. Creating it now...\n", BucketName)
			_, createErr := S3Client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: aws.String(BucketName),
			})
			if createErr != nil {
				log.Fatalf("Failed to auto-create S3 bucket: %v", createErr)
			}
			fmt.Printf("[+] Bucket '%s' created successfully!\n", BucketName)
		} else {
			// If it's an auth failure, network timeout, or bad endpoint, crash immediately
			log.Fatalf("FATAL STARTUP ERROR: Could not connect to MinIO/S3: %v", err)
		}
	} else {
		fmt.Printf("[*] Connected to MinIO (Bucket: %s)\n", BucketName)
	}
}
