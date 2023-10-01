package sources

import (
	"alex/entro/pkg/report"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// SecretsManager is a set of operations on a secret manager
type SecretsManager interface {
	ListSecrets() ([]report.SecretMetadata, error)
}

// AWSSecretsManager is an implementation of SecretsManager for AWS Secrets Manager
type AWSSecretsManager struct {
	AWSSession session.Session
	Region     string
}

func (impl AWSSecretsManager) ListSecrets() ([]report.SecretMetadata, error) {
	svc := secretsmanager.New(&impl.AWSSession)

	trueVar := true
	maxResult := int64(100)
	secrets, err := svc.ListSecrets(&secretsmanager.ListSecretsInput{
		Filters:                nil,
		IncludePlannedDeletion: &trueVar,
		MaxResults:             &maxResult,
		NextToken:              nil,
		SortOrder:              nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	var res []report.SecretMetadata
	for _, secret := range secrets.SecretList {
		metadata := report.SecretMetadata{
			ID:     *secret.ARN,
			Name:   *secret.Name,
			Region: impl.Region,
		}
		res = append(res, metadata)
	}
	return res, nil
}