package main

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/dprotaso/go-yit"
	"github.com/skirsten/s3-presign-yaml/internal/presign"
	"gopkg.in/yaml.v3"
)

func Run() error {
	if len(os.Args) != 2 {
		return errors.New("usage: s3-presign-yaml ( FILENAME | - )")
	}

	mc, err := presign.NewMinioClient()
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

		it = it.Filter(yit.WithPrefix(fmt.Sprintf("%s://", presign.Scheme)))

		for node, ok := it(); ok; node, ok = it() {
			ref := strings.TrimSpace(node.Value)

			u, err := url.Parse(ref)
			if err != nil {
				return err
			}

			signed, err := presign.Presign(mc, u)
			if err != nil {
				return err
			}

			node.Value = signed
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
