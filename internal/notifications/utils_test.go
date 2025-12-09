package notifications

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/jbcom/secretsync/api/v1alpha1"
	aws "github.com/jbcom/secretsync/pkg/client/aws"
	vault "github.com/jbcom/secretsync/pkg/client/vault"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Tests for renderTemplate function
func TestRenderTemplate_Success(t *testing.T) {
	vaultClient := &vault.VaultClient{
		Address: "http://vault.example.com",
		Path:    "secret/data",
	}

	syncConfig := v1alpha1.SecretSync{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretSync",
			APIVersion: "secretsync.jbcom.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-secretsync",
			Namespace: "default",
		},
		Spec: v1alpha1.SecretSyncSpec{
			Source: vaultClient,
			Dest: []*v1alpha1.StoreConfig{
				{
					AWS: &aws.AwsClient{
						Region: "us-east-1",
						Name:   "secret/data",
					},
				},
			},
			SyncDelete: new(bool),
			DryRun:     new(bool),
		},
		Status: v1alpha1.SecretSyncStatus{
			Status: "success",
		},
	}

	notificationMessage := v1alpha1.NotificationMessage{
		Event:           v1alpha1.NotificationEventSyncSuccess,
		Message:         "Sync completed successfully",
		SecretSync: syncConfig,
	}

	templateString := `
Event: {{.Event}}
Message: {{.Message}}
SecretSync:
  Name: {{.SecretSync.ObjectMeta.Name}}
  Namespace: {{.SecretSync.ObjectMeta.Namespace}}
  Source Address: {{.SecretSync.Spec.Source.Address}}
  Destination: {{range .SecretSync.Spec.Dest}}{{.AWS.Name}} ({{.AWS.Region}}){{end}}
  Status: {{.SecretSync.Status.Status}}
`

	expectedOutput := `
Event: success
Message: Sync completed successfully
SecretSync:
  Name: example-secretsync
  Namespace: default
  Source Address: http://vault.example.com
  Destination: secret/data (us-east-1)
  Status: success
`

	output, err := renderTemplate(templateString, notificationMessage)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestRenderTemplate_WithJSON(t *testing.T) {
	vaultClient := &vault.VaultClient{
		Address: "http://vault.example.com",
		Path:    "secret/data",
	}

	syncConfig := v1alpha1.SecretSync{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretSync",
			APIVersion: "secretsync.jbcom.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-secretsync",
			Namespace: "default",
		},
		Spec: v1alpha1.SecretSyncSpec{
			Source: vaultClient,
			Dest: []*v1alpha1.StoreConfig{
				{
					AWS: &aws.AwsClient{
						Region: "us-east-1",
						Name:   "secret/data",
					},
				},
			},
			SyncDelete: new(bool),
			DryRun:     new(bool),
		},
		Status: v1alpha1.SecretSyncStatus{
			Status: "success",
		},
	}

	notificationMessage := v1alpha1.NotificationMessage{
		Event:           v1alpha1.NotificationEventSyncSuccess,
		Message:         "Sync completed successfully",
		SecretSync: syncConfig,
	}

	templateString := `
Event: {{.Event}}
SecretSync JSON: {{json .SecretSync}}
`

	syncConfigJSON, _ := json.Marshal(notificationMessage.SecretSync)
	expectedOutput := fmt.Sprintf(`
Event: success
SecretSync JSON: %s
`, syncConfigJSON)

	output, err := renderTemplate(templateString, notificationMessage)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}

func TestRenderTemplate_EmptyValues(t *testing.T) {
	syncConfig := v1alpha1.SecretSync{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SecretSync",
			APIVersion: "secretsync.jbcom.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "",
			Namespace: "",
		},
		Spec: v1alpha1.SecretSyncSpec{
			Source: nil,
			Dest:   nil,
		},
		Status: v1alpha1.SecretSyncStatus{
			Status: "",
		},
	}

	notificationMessage := v1alpha1.NotificationMessage{
		Event:           v1alpha1.NotificationEventSyncFailure,
		Message:         "Sync failed",
		SecretSync: syncConfig,
	}

	templateString := `
Event: {{.Event}}
Message: {{.Message}}
SecretSync:
  Name: {{.SecretSync.ObjectMeta.Name}}
  Namespace: {{.SecretSync.ObjectMeta.Namespace}}
  Source Address: {{if .SecretSync.Spec.Source}}{{.SecretSync.Spec.Source.Address}}{{else}}<no value>{{end}}
  Destination: {{if .SecretSync.Spec.Dest}}{{range .SecretSync.Spec.Dest}}{{.AWS.Name}} ({{.AWS.Region}}){{end}}{{else}}<no value>{{end}}
  Status: {{.SecretSync.Status.Status}}
`

	expectedOutput := `
Event: failure
Message: Sync failed
SecretSync:
  Name: 
  Namespace: 
  Source Address: <no value>
  Destination: <no value>
  Status: 
`

	output, err := renderTemplate(templateString, notificationMessage)
	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}
