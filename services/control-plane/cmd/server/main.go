// Command server starts the Control Plane HTTP API.
// Concrete adapters are wired here as the platform gains infrastructure.
package main

import (
	"log"
	"net/http"
	"os"

	transport "github.com/Design_Pattern/services/control-plane/internal/adapters/inbound/http"
	"github.com/Design_Pattern/services/control-plane/internal/adapters/outbound/memory"
	"github.com/Design_Pattern/services/control-plane/internal/applications"
)

func main() {
	address := os.Getenv("HTTP_ADDR")
	if address == "" {
		address = ":8080"
	}

	store := memory.NewStore()
	api := applications.NewService(memory.Pipelines(store), memory.Runs(store))
	handler := transport.NewHandler(api)
	log.Printf("control-plane listening on %s", address)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatal(err)
	}
}
