### Entro

Entro correlates secrets metadata stored in a Secrets Manager to its audit trail events. It generates a report, that can be downloaded.


#### Installation

- Clone the repository.
- At the root of the repository, and run `go build`.
- Run the binary to start the server: `./entro`.

#### Requirements

The binary needs to be able to write into `/tmp`.