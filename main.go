package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dprotaso/go-yit"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"
	"gopkg.in/yaml.v3"
)

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

const Scheme = "s3-presign"
const Example = "s3-presign://get@my-bucket/path/to/object"

func Run() error {
	ctx := context.Background()

	if len(os.Args) != 2 {
		return errors.New("usage: s3-presign-yaml ( FILENAME | - )")
	}

	mc, err := NewMinioClient()
	if err != nil {
		return err
	}

	fname := os.Args[1]

	var reader io.Reader
	if fname == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(fname)
		if err != nil {
			return err
		}
		defer file.Close()
		reader = file
	}

	decoder := yaml.NewDecoder(reader)
	encoder := yaml.NewEncoder(os.Stdout)

	for {
		var node yaml.Node
		err = decoder.Decode(&node)
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		it := yit.FromNode(&node).
			RecurseNodes().
			Filter(yit.StringValue)

		it = it.Filter(yit.WithPrefix(fmt.Sprintf("%s://", Scheme)))

		for node, ok := it(); ok; node, ok = it() {
			ref := strings.TrimSpace(node.Value)

			u, err := url.Parse(ref)
			if err != nil {
				return err
			}

			// [scheme:][//[userinfo@]host][/]path[?query][#fragment]

			if u.Scheme != Scheme {
				return fmt.Errorf("unsupported scheme: %s", u.Scheme)
			}

			method := strings.ToUpper(u.User.Username())
			bucketName := u.Host
			objectName := u.Path[1:]
			duration := 1 * time.Hour

			if method == "" {
				method = "GET"
			}

			if u.Fragment != "" {
				duration, err = time.ParseDuration(u.Fragment)
				if err != nil {
					return err
				}
			}

			if bucketName == "" {
				return fmt.Errorf("bucket name needs to be defined e.g. %s", Example)
			}

			err = s3utils.CheckValidObjectName(objectName)
			if err != nil {
				return err
			}

			u, err = mc.Presign(ctx, method, bucketName, objectName, duration, u.Query())
			if err != nil {
				return err
			}

			node.Value = u.String()
		}

		err = encoder.Encode(&node)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	err := Run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
