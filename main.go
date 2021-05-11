// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// [START bigquery_quickstart]

// Sample bigquery-quickstart creates a Google BigQuery dataset.
package main

import (
	"context"
	"log"

	pubsub "google.golang.org/api/pubsub/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/transport/cert"
	"github.com/ThalesIgnite/crypto11"
)

func main() {
	config := &crypto11.Config{
		Path:       "/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so",
		TokenLabel: "token1",
		Pin:        "mynewpin",
	}
	ctx, err := crypto11.Configure(config)
	if err != nil {
		log.Fatal(err)
	}

	defer ctx.Close()
  cert, err := cert.NewSource(&cert.PKCSConfig{
		Context:        ctx,
		PkcsLabel:      []byte("keylabel1"),
	})
	if err != nil {
		log.Fatalf("cert.NewSource: %v", err)
	}
	// Creates a client.
	_, err  = pubsub.NewService(context.Background(), option.WithClientCertSource(cert))
	if err != nil {
		log.Fatalf("pubsub.NewService: %v", err)
	}
}

// [END bigquery_quickstart]
