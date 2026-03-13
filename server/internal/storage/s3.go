package storage

import (
	"context"
	"fmt"
	"log"
	"os"

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

	// 4. AUTO-CREATE BUCKET IF IT DOESN'T EXIST
	ctx := context.TODO()
	_, err = S3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(BucketName),
	})

	if err != nil {
		fmt.Printf("[*] MinIO Bucket '%s' not found. Creating it now...\n", BucketName)
		_, err = S3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(BucketName),
		})
		if err != nil {
			log.Fatalf("Failed to auto-create S3 bucket: %v", err)
		}
		fmt.Printf("[+] Bucket '%s' created successfully!\n", BucketName)
	} else {
		fmt.Printf("[*] Connected to MinIO (Bucket: %s)\n", BucketName)
	}
}
