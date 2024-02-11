package external_traefik_config_plugin_test

import (
	"context"
	"encoding/json"
	"github.com/iappx/external_traefik_config_plugin"
	"testing"
)

func TestNew(t *testing.T) {
	config := external_traefik_config_plugin.CreateConfig()
	config.PollInterval = "1s"

	provider, err := external_traefik_config_plugin.New(context.Background(), config, "test")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		err = provider.Stop()
		if err != nil {
			t.Fatal(err)
		}
	})

	err = provider.Init()
	if err != nil {
		t.Fatal(err)
	}

	cfgChan := make(chan json.Marshaler)

	err = provider.Provide(cfgChan)
	if err != nil {
		t.Fatal(err)
	}
}
