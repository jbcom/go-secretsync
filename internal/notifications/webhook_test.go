package notifications

import (
	"reflect"
	"testing"

	"github.com/jbcom/secretsync/api/v1alpha1"
	"github.com/jbcom/secretsync/pkg/client/aws"
	"github.com/jbcom/secretsync/pkg/client/vault"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMessagePayload(t *testing.T) {
	tests := []struct {
		name    string
		message v1alpha1.NotificationMessage
		body    string
		want    string
	}{
		{
			name: "plain text body",
			message: v1alpha1.NotificationMessage{
				Event:   v1alpha1.NotificationEventSyncSuccess,
				Message: "Sync completed successfully",
				SecretSync: v1alpha1.SecretSync{
					TypeMeta: metav1.TypeMeta{
						Kind:       "SecretSync",
						APIVersion: "secretsync.jbcom.dev/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-secretsync",
						Namespace: "default",
					},
					Spec: v1alpha1.SecretSyncSpec{
						Source: &vault.VaultClient{
							Address: "http://vault.example.com",
							Path:    "secret/data",
						},
						Dest: []*v1alpha1.StoreConfig{
							{
								AWS: &aws.AwsClient{
									Region: "us-east-1",
									Name:   "secret/data",
								},
							},
						},
					},
				},
			},
			body: "hello",
			want: "hello",
		},
		{
			name: "templated message",
			message: v1alpha1.NotificationMessage{
				Event:   v1alpha1.NotificationEventSyncSuccess,
				Message: "Sync completed successfully",
				SecretSync: v1alpha1.SecretSync{
					TypeMeta: metav1.TypeMeta{
						Kind:       "SecretSync",
						APIVersion: "secretsync.jbcom.dev/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-secretsync",
						Namespace: "default",
					},
					Spec: v1alpha1.SecretSyncSpec{
						Source: &vault.VaultClient{
							Address: "http://vault.example.com",
							Path:    "secret/data",
						},
						Dest: []*v1alpha1.StoreConfig{
							{
								AWS: &aws.AwsClient{
									Region: "us-east-1",
									Name:   "secret/data",
								},
							},
						},
					},
				},
			},
			body: "status: {{.Event}}",
			want: "status: success",
		},
		{
			name: "deeply nested templated message",
			message: v1alpha1.NotificationMessage{
				Event:   v1alpha1.NotificationEventSyncSuccess,
				Message: "Sync completed successfully",
				SecretSync: v1alpha1.SecretSync{
					TypeMeta: metav1.TypeMeta{
						Kind:       "SecretSync",
						APIVersion: "secretsync.jbcom.dev/v1alpha1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "example-secretsync",
						Namespace: "default",
					},
					Spec: v1alpha1.SecretSyncSpec{
						Source: &vault.VaultClient{
							Address: "http://vault.example.com",
							Path:    "secret/data",
						},
						Dest: []*v1alpha1.StoreConfig{
							{
								AWS: &aws.AwsClient{
									Region: "us-east-1",
									Name:   "secret/data",
								},
							},
						},
					},
					Status: v1alpha1.SecretSyncStatus{
						Status: "success",
					},
				},
			},
			body: "status: {{.Event}}, message: {{.Message}}, sync status: {{.SecretSync.Status.Status}}",
			want: "status: success, message: Sync completed successfully, sync status: success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messagePayload(tt.message, tt.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("messagePayload() = %v, want %v", got, tt.want)
			}
		})
	}
}
