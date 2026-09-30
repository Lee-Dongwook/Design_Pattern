// Command server starts the Control Plane HTTP API.
// Concrete adapters are wired here as the platform gains infrastructure.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	transport "github.com/Design_Pattern/services/control-plane/internal/adapters/inbound/http"
	"github.com/Design_Pattern/services/control-plane/internal/adapters/outbound/memory"
	"github.com/Design_Pattern/services/control-plane/internal/adapters/outbound/postgres"
	"github.com/Design_Pattern/services/control-plane/internal/applications"
)

func main() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}

	var api *applications.Service
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		store, err := postgres.NewStore(context.Background(), databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		defer store.Close()
		api = applications.NewService(postgres.Pipelines(store), postgres.Runs(store), postgres.Tasks(store))
		log.Print("using PostgreSQL persistence")
	} else {
		store := memory.NewStore()
		api = applications.NewService(memory.Pipelines(store), memory.Runs(store), memory.Tasks(store))
		log.Print("DATABASE_URL is not set; using in-memory persistence")
	}
	handler := transport.NewHandler(api)
	log.Printf("control-plane listening on %s", address)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatal(err)
	}
}
