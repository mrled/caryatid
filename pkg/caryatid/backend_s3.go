/*
The localfile backend, for dealing with a Vagrant catalog on a local filesystem
*/

package caryatid

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type CaryatidS3Backend struct {
	AwsSession   *session.Session
	S3Service    *s3.S3
	S3Downloader *s3manager.Downloader
	S3Uploader   *s3manager.Uploader
	Manager      *BackendManager

	CatalogLocation *caryatidS3Location
}

type caryatidS3Location struct {
	Bucket   string
	Resource string
}

func uri2s3location(uri string) (loc *caryatidS3Location, err error) {
	s3Regex := regexp.MustCompile("^s3://([a-zA-Z0-9\\-_]+)/(.*)")
	result := s3Regex.FindAllStringSubmatch(uri, -1)

	if result == nil {
		err = fmt.Errorf("Invalid S3 URI '%v'", uri)
		return
	} else if len(result) != 1 || len(result[0]) != 3 {
		err = fmt.Errorf("Apparently the regexp is wrong and I don't know what I'm doing, sorry")
		return
	}

	log.Printf("result:\n%v\n", result)

	loc = new(caryatidS3Location)
	loc.Bucket = result[0][1]
	loc.Resource = result[0][2]
	return
}

func (backend *CaryatidS3Backend) verifyCredential() (err error) {
	_, err = backend.S3Service.ListObjects(&s3.ListObjectsInput{
		Bucket: aws.String(backend.CatalogLocation.Bucket),
	})
	return
}

func (backend *CaryatidS3Backend) SetManager(manager *BackendManager) (err error) {
	backend.Manager = manager

	backend.AwsSession = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	backend.S3Service = s3.New(backend.AwsSession)
	backend.S3Downloader = s3manager.NewDownloader(backend.AwsSession)
	backend.S3Uploader = s3manager.NewUploader(backend.AwsSession)

	backend.CatalogLocation, err = uri2s3location(backend.Manager.CatalogUri)
	if err != nil {
		return
	}

	return
}

func (backend *CaryatidS3Backend) GetManager() (manager *BackendManager, err error) {
	manager = backend.Manager
	if manager == nil {
		err = fmt.Errorf("The Manager property was not set")
	}
	return
}

func (backend *CaryatidS3Backend) SetCredential(backendCredential string) (err error) {
	if backendCredential == "" {
		err = fmt.Errorf("Backend credential is empty")
		return
	}
	err = backend.verifyCredential()
	return
}

func (backend *CaryatidS3Backend) GetCatalogBytes() (catalogBytes []byte, err error) {
	var (
		dlerr         error
		catalogExists bool
	)

	catalogExists = true
	dlBuffer := &aws.WriteAtBuffer{}
	_, dlerr = backend.S3Downloader.Download(
		dlBuffer,
		&s3.GetObjectInput{
			Bucket: aws.String(backend.CatalogLocation.Bucket),
			Key:    aws.String(backend.CatalogLocation.Resource),
		},
	)

	if aerr, ok := dlerr.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchKey:
			log.Printf("No file at '%v'; starting with empty catalog\n", backend.Manager.CatalogUri)
			catalogExists = false
		case s3.ErrCodeNoSuchBucket:
			err = fmt.Errorf("Bucket '%v' does not exist\n", backend.CatalogLocation.Bucket)
		default:
			err = dlerr
		}
	}

	if err != nil {
		log.Printf("CaryatidS3Backend.GetCatalogBytes(): Could not download from S3: %v", err)
		return
	}

	if catalogExists {
		catalogBytes = dlBuffer.Bytes()
	} else {
		catalogBytes = []byte("{}")
	}

	return
}

func (backend *CaryatidS3Backend) SetCatalogBytes(serializedCatalog []byte) (err error) {
	upParams := &s3manager.UploadInput{
		Bucket: aws.String(backend.CatalogLocation.Bucket),
		Key:    aws.String(backend.CatalogLocation.Resource),
		Body:   bytes.NewReader(serializedCatalog),
	}

	_, err = backend.S3Uploader.Upload(upParams)
	if err != nil {
		log.Println("CaryatidS3Backend.SetCatalogBytes(): Error trying to upload catalog: ", err)
		return
	}
	return
}

func (backend *CaryatidS3Backend) CopyBoxFile(path string, boxName string, boxVersion string, boxProvider string) (err error) {
	var (
		boxFileLoc  caryatidS3Location
		fileHandler *os.File
	)

	boxFileLoc.Bucket = backend.CatalogLocation.Bucket

	// TODO: Do we need the if statement? Can we just use the second version and be OK if LastIndex() returns -1 ?
	lastSlashIdx := strings.LastIndex(backend.CatalogLocation.Resource, "/")
	if lastSlashIdx < 0 {
		boxFileLoc.Resource = fmt.Sprintf(
			"%v/%v_%v_%v.box",
			boxName, boxName, boxVersion, boxProvider)
	} else {
		boxFileLoc.Resource = fmt.Sprintf(
			"%v/%v/%v_%v_%v.box",
			backend.CatalogLocation.Resource[0:lastSlashIdx],
			boxName, boxName, boxVersion, boxProvider)
	}

	if fileHandler, err = os.Open(path); err != nil {
		return
	}
	defer fileHandler.Close()

	_, err = backend.S3Uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(boxFileLoc.Bucket),
		Key:    aws.String(boxFileLoc.Resource),
		Body:   fileHandler,
	})
	if err != nil {
		return
	}

	return
}

func (backend *CaryatidS3Backend) DeleteFile(uri string) (err error) {
	var (
		fileLoc *caryatidS3Location
	)

	if fileLoc, err = uri2s3location(uri); err != nil {
		return
	}

	_, err = backend.S3Service.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(fileLoc.Bucket),
		Key:    aws.String(fileLoc.Resource),
	})
	if err != nil {
		return
	}

	err = backend.S3Service.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(fileLoc.Bucket),
		Key:    aws.String(fileLoc.Resource),
	})
	if err != nil {
		return
	}

	return
}

func (backend *CaryatidS3Backend) Scheme() string {
	return "s3"
}
