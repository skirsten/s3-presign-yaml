package presign

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

const Scheme = "s3-presign"
const Example = "s3-presign://get@my-bucket/path/to/object"

// TODO: This regex can be improved significantly...
var Regex = regexp.MustCompile(`s3-presign://[a-zA-Z0-9\@\-\_\/\?\=\#\%\.\~\+]+`)

func NewMinioClient() (*minio.Client, error) {
	endpoint := os.Getenv("AWS_S3_ENDPOINT")
	if endpoint == "" {
		return nil, errors.New("required env var: AWS_S3_ENDPOINT")
	}
	accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKeyId == "" {
		return nil, errors.New("required env var: AWS_ACCESS_KEY_ID")
	}
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		return nil, errors.New("required env var: AWS_SECRET_ACCESS_KEY")
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyId, secretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}

	return minioClient, nil
}

func Presign(mc *minio.Client, url *url.URL) (string, error) {
	// [scheme:][//[userinfo@]host][/]path[?query][#fragment]

	if url.Scheme != Scheme {
		return "", fmt.Errorf("unsupported scheme: %s", url.Scheme)
	}

	method := strings.ToUpper(url.User.Username())
	bucketName := url.Host
	objectName := url.Path[1:]
	duration := 1 * time.Hour

	if method == "" {
		method = "GET"
	}

	if url.Fragment != "" {
		_duration, err := time.ParseDuration(url.Fragment)
		if err != nil {
			return "", err
		}
		duration = _duration
	}

	if bucketName == "" {
		return "", fmt.Errorf("bucket name needs to be defined e.g. %s", Example)
	}

	err := s3utils.CheckValidObjectName(objectName)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	url, err = mc.Presign(ctx, method, bucketName, objectName, duration, url.Query())
	if err != nil {
		return "", err
	}

	return url.String(), nil
}
