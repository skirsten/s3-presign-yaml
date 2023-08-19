package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	"github.com/skirsten/s3-presign-yaml/internal/presign"
)

func Run() error {
	if len(os.Args) != 2 {
		return errors.New("usage: s3-presign-text ( FILENAME | - )")
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

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		text := scanner.Text()

		for len(text) > 0 {
			match := presign.Regex.FindStringIndex(text)
			if match == nil {
				_, err := os.Stdout.WriteString(text) // write remaining text
				if err != nil {
					return err
				}
				break
			}

			_, err := os.Stdout.WriteString(text[:match[0]]) // write until match starts
			if err != nil {
				return err
			}

			u, err := url.Parse(text[match[0]:match[1]])
			if err != nil {
				return err
			}

			signed, err := presign.Presign(mc, u)
			if err != nil {
				return err
			}

			_, err = os.Stdout.WriteString(signed)
			if err != nil {
				return err
			}

			text = text[match[1]:]
		}

		os.Stdout.WriteString("\n")
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
