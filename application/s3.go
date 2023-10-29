package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type s3Service struct {
	s3Client *s3.Client
}

var S3 s3Service

func (service *s3Service) NewClient(options s3.Options) {
	service.s3Client = s3.New(options)
}

// ListBuckets lists the buckets in the current account.
func (service *s3Service) ListBuckets() ([]types.Bucket, error) {
	result, err := service.s3Client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	var buckets []types.Bucket
	if err != nil {
		log.Printf("Couldn't list buckets for your account. Here's why: %v\n", err)
	} else {
		buckets = result.Buckets
	}
	return buckets, err
}

// BucketExists checks whether a bucket exists in the current account.
func (service *s3Service) BucketExists(bucketName string) (bool, error) {
	_, err := service.s3Client.HeadBucket(context.TODO(), &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	exists := true
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				log.Printf("Bucket %v is available.\n", bucketName)
				exists = false
				err = nil
			default:
				log.Printf("Either you don't have access to bucket %v or another error occurred. "+
					"Here's what happened: %v\n", bucketName, err)
			}
		}
	} else {
		log.Printf("Bucket %v exists and you already own it.", bucketName)
	}

	return exists, err
}

// CreateBucket creates a bucket with the specified name in the specified Region.
func (service *s3Service) CreateBucket(name string, region string) error {
	_, err := service.s3Client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(name),
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		},
	})
	if err != nil {
		log.Printf("Couldn't create bucket %v in Region %v. Here's why: %v\n",
			name, region, err)
	}
	return err
}

// UploadFile reads from a file and puts the data into an object in a bucket.
func (service *s3Service) UploadFile(bucketName string, objectKey string, fileName string) error {
	file, err := os.Open(fileName)
	if err != nil {
		log.Printf("Couldn't open file %v to upload. Here's why: %v\n", fileName, err)
	} else {
		defer file.Close()
		_, err = service.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(objectKey),
			Body:   file,
		})
		if err != nil {
			log.Printf("Couldn't upload file %v to %v:%v. Here's why: %v\n",
				fileName, bucketName, objectKey, err)
		}
	}
	return err
}

// UploadLargeObject uses an upload manager to upload data to an object in a bucket.
// The upload manager breaks large data into parts and uploads the parts concurrently.
func (service *s3Service) UploadLargeObject(bucketName string, objectKey string, largeObject []byte) error {
	largeBuffer := bytes.NewReader(largeObject)
	var partMiBs int64 = 10
	uploader := manager.NewUploader(service.s3Client, func(u *manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
	})
	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   largeBuffer,
	})
	if err != nil {
		log.Printf("Couldn't upload large object to %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}

	return err
}

// DownloadFile gets an object from a bucket and stores it in a local file.
func (service *s3Service) DownloadFile(bucketName string, objectKey string, fileName string) error {
	result, err := service.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("Couldn't get object %v:%v. Here's why: %v\n", bucketName, objectKey, err)
		return err
	}
	defer result.Body.Close()
	file, err := os.Create(fileName)
	if err != nil {
		log.Printf("Couldn't create file %v. Here's why: %v\n", fileName, err)
		return err
	}
	defer file.Close()
	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Printf("Couldn't read object body from %v. Here's why: %v\n", objectKey, err)
	}
	_, err = file.Write(body)
	return err
}

// DownloadLargeObject uses a download manager to download an object from a bucket.
// The download manager gets the data in parts and writes them to a buffer until all of
// the data has been downloaded.
func (service *s3Service) DownloadLargeObject(bucketName string, objectKey string) ([]byte, error) {
	var partMiBs int64 = 10
	downloader := manager.NewDownloader(service.s3Client, func(d *manager.Downloader) {
		d.PartSize = partMiBs * 1024 * 1024
	})
	buffer := manager.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(context.TODO(), buffer, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		log.Printf("Couldn't download large object from %v:%v. Here's why: %v\n",
			bucketName, objectKey, err)
	}
	return buffer.Bytes(), err
}

// CopyToFolder copies an object in a bucket to a subfolder in the same bucket.
func (service *s3Service) CopyToFolder(bucketName string, objectKey string, folderName string) error {
	_, err := service.s3Client.CopyObject(context.TODO(), &s3.CopyObjectInput{
		Bucket:     aws.String(bucketName),
		CopySource: aws.String(fmt.Sprintf("%v/%v", bucketName, objectKey)),
		Key:        aws.String(fmt.Sprintf("%v/%v", folderName, objectKey)),
	})
	if err != nil {
		log.Printf("Couldn't copy object from %v:%v to %v:%v/%v. Here's why: %v\n",
			bucketName, objectKey, bucketName, folderName, objectKey, err)
	}
	return err
}

// ListObjects lists the objects in a bucket.
func (service *s3Service) ListObjects(bucketName string) ([]types.Object, error) {
	result, err := service.s3Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	var contents []types.Object
	if err != nil {
		log.Printf("Couldn't list objects in bucket %v. Here's why: %v\n", bucketName, err)
	} else {
		contents = result.Contents
	}
	return contents, err
}

// DeleteObjects deletes a list of objects from a bucket.
func (service *s3Service) DeleteObjects(bucketName string, objectKeys []string) error {
	var objectIds []types.ObjectIdentifier
	for _, key := range objectKeys {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(key)})
	}
	_, err := service.s3Client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &types.Delete{Objects: objectIds},
	})
	if err != nil {
		log.Printf("Couldn't delete objects from bucket %v. Here's why: %v\n", bucketName, err)
	}
	return err
}

// DeleteBucket deletes a bucket. The bucket must be empty or an error is returned.
func (service *s3Service) DeleteBucket(bucketName string) error {
	_, err := service.s3Client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName)})
	if err != nil {
		log.Printf("Couldn't delete bucket %v. Here's why: %v\n", bucketName, err)
	}
	return err
}
