module mtls_codelab

replace google.golang.org/api => /usr/local/google/home/shinfan/go/src/google-api-go-client

go 1.16

require (
	cloud.google.com/go/pubsub v1.3.1
	github.com/ThalesIgnite/crypto11 v1.2.4
	google.golang.org/api v0.46.0
)
