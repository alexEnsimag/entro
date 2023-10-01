### Operations

The server offers three different operations:
- request a report: sends a report to a queue for generation, returns the report ID
- get the status of a report: return either `creating`, `created`, or `failed`
- get a link to download the report

### Components

- an API server 
- a queue for report requests - currently a go channel
- a database, used to store the status of the report - currently an in-memory database
- an external storage, used to store the reports - currently local in `/tmp`
- connectors to external services, uses to retrieve the data needed for reports

### Connectors

A connector connects to an external service to retrieve an abstracted set of data. There are two types of connectors (see in `pkg/connectors`, the interfaces `SecretsManager` and `AuditTrailManager`):
- A connector to secret managers, which is used to list the secrets metadata 
- A connector to audit trail managers, which is used to list the audit logs for a secret

The data abstractions are defined in `pkd/report/report.go`, see the structs `SecretMetadata` and `AuditTrail`. The reasons for these abstraction, is to be able to connect to any type of secrets manager and audit trail manager, and return a unified format for the retrieved information.

### Report request

A request is a set of three fields:
- a report ID
- a connector to a secrets manager
- a connector to an audit trail manager

Therefore, the queue can accept reports independently from a cloud provider.