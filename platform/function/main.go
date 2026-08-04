package main

import (
	"log"
	"os"

	function "github.com/crossplane/function-sdk-go"
)

func main() {
	logger, err := function.NewLogger(false)
	if err != nil {
		log.Fatal(err)
	}
	if err := function.Serve(
		&Function{log: logger},
		function.MTLSCertificates(os.Getenv("TLS_SERVER_CERTS_DIR")),
		function.MaxRecvMessageSize(4*1024*1024),
	); err != nil {
		log.Fatal(err)
	}
}
