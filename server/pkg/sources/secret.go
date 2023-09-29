package sources

import (
	"alex/entro/server/pkg/report"
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

	// FIXME (alex): handle pagination
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
		// convert tags
		tags := map[string]string{}
		for _, t := range secret.Tags {
			tags[*t.Key] = *t.Value
		}
		if len(tags) == 0 {
			tags = nil
		}

		// save secret metadata
		metadata := report.SecretMetadata{
			ID:     *secret.ARN,
			Region: impl.Region,
			Tags:   tags,
		}
		res = append(res, metadata)
	}
	return res, nil
}
