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
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           os.Getenv("S3_ENDPOINT"),
			SigningRegion: "us-east-1",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("S3_ACCESS_KEY"),
			os.Getenv("S3_SECRET_KEY"),
			"",
		)),
	)
	if err != nil {
		log.Fatalf("Unable to load S3 config: %v", err)
	}

	S3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// 👇 NEW CODE: Auto-create bucket if it doesn't exist
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
