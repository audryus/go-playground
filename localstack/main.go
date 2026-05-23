package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Custom S3 EndpointResolverV2 for LocalStack
type localstackS3Resolver struct {
	url string
}

func (r localstackS3Resolver) ResolveEndpoint(ctx context.Context, params s3.EndpointParameters) (aws.Endpoint, error) {
	return aws.Endpoint{
		URL:               r.url,
		HostnameImmutable: true,
	}, nil
}

func main() {
	endpoint := "http://localstack.home.arpa:4566"
	bucket := "test-bucket"
	key := "test.txt"
	content := []byte("Hello LocalStack S3!")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		)),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	})

	bucketParams := &s3.CreateBucketInput{
		Bucket: &bucket,
	}

	_, err = client.CreateBucket(context.Background(), bucketParams)
	if err != nil {
		log.Fatal("Creating bucket:", err)
	}

	// Put object
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(content),
	})
	if err != nil {
		log.Fatal("PutObject error:", err)
	}

	fmt.Println("File uploaded.")

	getObj, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Fatal("GetObject error:", err)
	}
	defer getObj.Body.Close()

	body, _ := io.ReadAll(getObj.Body)
	fmt.Println("File retrieved:", string(body))

	// Delete object
	_, err = client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Fatal("DeleteObject error:", err)
	}

	deleteParams := &s3.DeleteBucketInput{
		Bucket: &bucket,
	}

	_, err = client.DeleteBucket(context.Background(), deleteParams)
	if err != nil {
		log.Fatal("Delete Bucket error:", err)
	}

	fmt.Println("Bucket deleted.")
}
