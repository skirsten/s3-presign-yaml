# s3-presign-yaml

A tiny utility that presigns s3 urls in yaml files:

```yaml
a: s3-presign://my-bucket/path/to/object
b: s3-presign://get@my-bucket/path/to/object#24h
c:
  - s3-presign://put@my-bucket/path/to/object?extra=query
  - s3-presign://PATCH@my-bucket/path/to/object
```

```sh
go install github.com/skirsten/s3-presign-yaml@latest

export AWS_S3_ENDPOINT="..."
export AWS_ACCESS_KEY_ID="..."
export AWS_SECRET_ACCESS_KEY="..."

s3-presign-yaml file.yaml > output.yaml

cat job.yaml | s3-presign-yaml - | kubectl apply -f -
```
