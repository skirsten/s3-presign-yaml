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
	endpoint := os.Getenv("S3_ENDPOINT")
	if endpoint == "" {
		return nil, errors.New("required env var: S3_ENDPOINT")
	}
	region := os.Getenv("S3_REGION")
	accessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	if accessKeyId == "" {
		return nil, errors.New("required env var: AWS_ACCESS_KEY_ID")
	}
	secretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	if secretAccessKey == "" {
		return nil, errors.New("required env var: AWS_SECRET_ACCESS_KEY")
	}

	secure := true
	if strings.HasPrefix(endpoint, "https://") {
		secure = true
		endpoint = endpoint[len("https://"):]
	} else if strings.HasPrefix(endpoint, "http://") {
		secure = false
		endpoint = endpoint[len("http://"):]
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyId, secretAccessKey, ""),
		Secure: secure,
		Region: region,
	})
	if err != nil {
		return nil, err
	}

	return minioClient, nil
}

func Presign(mc *minio.Client, u *url.URL) (string, error) {
	// [scheme:][//[userinfo@]host][/]path[?query][#fragment]

	if u.Scheme != Scheme {
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	method := strings.ToUpper(u.User.Username())
	endpoint, _ := u.User.Password()
	bucketName := u.Host
	objectName := u.Path[1:]
	expires := 1 * time.Hour

	if method == "" {
		method = "GET"
	}

	query, err := url.ParseQuery(u.Fragment)
	if err != nil {
		return "", err
	}

	if value := query.Get("expires"); value != "" {
		_expires, err := time.ParseDuration(value)
		if err != nil {
			return "", err
		}
		expires = _expires
	}

	if bucketName == "" {
		return "", fmt.Errorf("bucket name needs to be defined e.g. %s", Example)
	}

	err = s3utils.CheckValidObjectName(objectName)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	u, err = mc.Presign(ctx, method, bucketName, objectName, expires, u.Query())
	if err != nil {
		return "", err
	}

	response := u.String()

	if endpoint != "" {
		response = endpoint + response[len(u.Scheme)+2:]
	}

	return response, nil
}
