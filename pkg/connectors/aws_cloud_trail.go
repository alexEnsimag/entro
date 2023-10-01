package connectors

import (
	"alex/entro/pkg/report"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
)

// AuditTrailManager is a set of operations on an audit trail manager
type AuditTrailManager interface {
	ListAuditTrails(secretName string) ([]report.AuditTrail, error)
}

// AWSCloudTrail is an implementation of AuditTrailManager for AWS CloudTrail
type AWSCloudTrail struct {
	AWSSession session.Session
	Region     string
}

func (impl AWSCloudTrail) ListAuditTrails(secretName string) ([]report.AuditTrail, error) {
	svc := cloudtrail.New(&impl.AWSSession)

	maxResult := int64(100)
	resourceName := "ResourceName"
	eventSource := "EventSource"
	eventSourceSecretsManager := "secretsmanager.amazonaws.com"
	events, err := svc.LookupEvents(&cloudtrail.LookupEventsInput{
		LookupAttributes: []*cloudtrail.LookupAttribute{
			{
				AttributeKey:   &eventSource,
				AttributeValue: &eventSourceSecretsManager,
			},
			{
				AttributeKey:   &resourceName,
				AttributeValue: &secretName,
			},
		},
		MaxResults: &maxResult,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list trails for secret %s: %w", secretName, err)
	}

	var res []report.AuditTrail
	for _, e := range events.Events {
		res = append(res, report.AuditTrail{
			UserName: *e.Username,
			Action:   *e.EventName,
			Time:     *e.EventTime,
		})
	}
	return res, nil
}
