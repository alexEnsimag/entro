### Entro

Entro correlates secrets metadata stored in a Secrets Manager to its audit trail events. It generates a report, that can be downloaded. It currently supports only AWS.


#### Installation

- Clone the repository.
- At the root of the repository, and run `go build`.
- Run the binary to start the server: `./entro`.

#### Requirements

- The binary needs to be able to open the port `8090`.
- The binary needs to be able to write into `/tmp`.

#### Limitations

- Runs only for a single region in AWS
- Cannot accept more than a 1000 concurrent requests
