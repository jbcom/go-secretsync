package sync

import (
	"context"
	"errors"

	"github.com/jbcom/secretsync/api/v1alpha1"
	"github.com/jbcom/secretsync/pkg/client/aws"
	"github.com/jbcom/secretsync/pkg/client/vault"
	"github.com/jbcom/secretsync/pkg/discovery/identitycenter"
	"github.com/jbcom/secretsync/pkg/driver"
	log "github.com/sirupsen/logrus"
)

// clientGenerator initializes clients for the sync operation
func clientGenerator(ctx context.Context, j SyncJob) (*SyncClients, error) {
	l := log.WithFields(log.Fields{"action": "clientGenerator"})
	l.Trace("start")
	defer l.Trace("end")

	scs, err := InitSyncConfigClients(j.SyncConfig)
	if err != nil {
		l.Error(err)
		j.Error = err
		return nil, err
	}

	cerr := scs.CreateClients(ctx)
	if cerr != nil {
		l.Error(cerr)
		j.Error = cerr
		return nil, cerr
	}
	return scs, nil
}

func setStoreGlobalDefaults(s *v1alpha1.VaultSecretSync) error {
	l := log.WithFields(log.Fields{
		"action": "setStoreGlobalDefaults",
	})
	l.Trace("start")
	defer l.Trace("end")
	if s.Spec.Source == nil || s.Spec.Dest == nil {
		l.Error("source or dest is nil")
		return errors.New("source or dest is nil")
	}
	if DefaultConfigs[driver.DriverNameVault] != nil {
		l.Trace("set source defaults")
		err := s.Spec.Source.SetDefaults(DefaultConfigs[driver.DriverNameVault].Vault)
		if err != nil {
			l.Error(err)
			return err
		}
		l.Tracef("source: %v", s.Spec.Source)
	}
	l.Trace("set dest defaults")
	for _, d := range s.Spec.Dest {
		var err error
		if d.AWS != nil && DefaultConfigs[driver.DriverNameAws] != nil {
			err = d.AWS.SetDefaults(DefaultConfigs[driver.DriverNameAws].AWS)
		}
		if d.IdentityCenter != nil && DefaultConfigs[driver.DriverNameIdentityCenter] != nil {
			err = d.IdentityCenter.SetDefaults(DefaultConfigs[driver.DriverNameIdentityCenter].IdentityCenter)
		}
		if d.Vault != nil && DefaultConfigs[driver.DriverNameVault] != nil {
			err = d.Vault.SetDefaults(DefaultConfigs[driver.DriverNameVault].Vault)
		}
		if err != nil {
			l.Error(err)
			return err
		}
	}
	return nil
}

func InitSyncConfigClients(sc v1alpha1.VaultSecretSync) (*SyncClients, error) {
	l := log.WithFields(log.Fields{
		"action": "sc.InitSyncConfigClients",
	})
	l.Trace("start")
	if sc.Spec.Source == nil {
		l.Error("source is nil")
		return nil, errors.New("source is nil")
	}
	if sc.Spec.Dest == nil {
		l.Error("dest is nil")
		return nil, errors.New("dest is nil")
	}
	scs := &SyncClients{}
	var err error
	if err := setStoreGlobalDefaults(&sc); err != nil {
		l.Error(err)
		return nil, err
	}
	scs.Source, err = vault.NewClient(sc.Spec.Source)
	if err != nil {
		l.Error(err)
		return nil, err
	}
	for _, d := range sc.Spec.Dest {
		if d.AWS != nil {
			client, err := aws.NewClient(d.AWS)
			if err != nil {
				l.Error(err)
				return nil, err
			}
			scs.Dest = append(scs.Dest, client)
		} else if d.IdentityCenter != nil {
			client, err := identitycenter.NewClient(d.IdentityCenter)
			if err != nil {
				l.Error(err)
				return nil, err
			}
			scs.Dest = append(scs.Dest, client)
		} else if d.Vault != nil {
			client, err := vault.NewClient(d.Vault)
			if err != nil {
				l.Error(err)
				return nil, err
			}
			scs.Dest = append(scs.Dest, client)
		}
		l.WithField("dest", scs.Dest).Trace("added dest")
	}
	l.Trace("end")
	return scs, nil
}
