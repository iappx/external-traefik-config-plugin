// Package external_traefik_config_plugin contains a demo of the provider's plugin.
package external_traefik_config_plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/traefik/genconf/dynamic"
	"github.com/traefik/genconf/dynamic/tls"
)

type BaseAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type Instance struct {
	ApiHost      string          `json:"apiHost,omitempty"`      // Куда будет отправлен запрос на получение данных
	InstanceHost string          `json:"InstanceHost,omitempty"` // Адрес для проксирования (Туда будут перенаправлены запросы с текущего инстанса)
	EntryPoints  []string        `json:"entryPoints,omitempty"`  // Точки входа для базового инстанса
	Credentials  BaseAuth        `json:"credentials,omitempty"`  // Basic auth, если есть
	CertResolver string          `json:"certResolver,omitempty"` // certResolver бозового инстанса
	Service      dynamic.Service `json:"service,omitempty"`
}

// Config the plugin configuration.
type Config struct {
	PollInterval string               `json:"pollInterval,omitempty"`
	InstanceMap  map[string]*Instance `json:"instances,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		PollInterval: "5s", // 5 * time.Second
		InstanceMap:  make(map[string]*Instance),
	}
}

// Provider a simple provider plugin.
type Provider struct {
	name         string
	pollInterval time.Duration
	Config       *Config

	cancel func()
}

// New creates a new Provider plugin.
func New(ctx context.Context, config *Config, name string) (*Provider, error) {
	pi, err := time.ParseDuration(config.PollInterval)
	if err != nil {
		return nil, err
	}

	return &Provider{
		name:         name,
		pollInterval: pi,
		Config:       config,
	}, nil
}

// Init the provider.
func (p *Provider) Init() error {
	if p.pollInterval <= 0 {
		return fmt.Errorf("poll interval must be greater than 0")
	}

	return nil
}

// Provide creates and send dynamic configuration.
func (p *Provider) Provide(cfgChan chan<- json.Marshaler) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Print(err)
			}
		}()

		p.loadConfiguration(ctx, cfgChan)
	}()

	return nil
}

func (p *Provider) loadConfiguration(ctx context.Context, cfgChan chan<- json.Marshaler) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			configuration := generateConfiguration(t, *p.Config)

			cfgChan <- &dynamic.JSONPayload{Configuration: configuration}

		case <-ctx.Done():
			return
		}
	}
}

// Stop to stop the provider and the related go routines.
func (p *Provider) Stop() error {
	p.cancel()
	return nil
}

func generateConfiguration(date time.Time, config Config) *dynamic.Configuration {
	configuration := &dynamic.Configuration{
		HTTP: &dynamic.HTTPConfiguration{
			Routers:           make(map[string]*dynamic.Router),
			Middlewares:       make(map[string]*dynamic.Middleware),
			Services:          make(map[string]*dynamic.Service),
			ServersTransports: make(map[string]*dynamic.ServersTransport),
		},
		TCP: &dynamic.TCPConfiguration{
			Routers:  make(map[string]*dynamic.TCPRouter),
			Services: make(map[string]*dynamic.TCPService),
		},
		TLS: &dynamic.TLSConfiguration{
			Stores:  make(map[string]tls.Store),
			Options: make(map[string]tls.Options),
		},
		UDP: &dynamic.UDPConfiguration{
			Routers:  make(map[string]*dynamic.UDPRouter),
			Services: make(map[string]*dynamic.UDPService),
		},
	}

	for name, instance := range config.InstanceMap {
		err := fillInstanceConfig(name, *instance, configuration)
		if err != nil {
			os.Stderr.WriteString(fmt.Sprintf("Error when receiving configuration %+v: %+v\n", name, err))
		}
	}

	return configuration
}

func fillInstanceConfig(name string, instance Instance, configuration *dynamic.Configuration) (err error) {
	routers, err := getHttpRouters(instance.ApiHost, instance.Credentials.Username, instance.Credentials.Password)

	if err != nil {
		return err
	}

	configuration.HTTP.Services[name] = &instance.Service

	for i := 0; i < len(routers); i++ {
		router := routers[i]
		routerName := strings.Split(router.Rule, "@")[0]

		configuration.HTTP.Routers[routerName] = &dynamic.Router{
			TLS: &dynamic.RouterTLSConfig{
				CertResolver: instance.CertResolver,
			},
			EntryPoints: instance.EntryPoints,
			Service:     name,
			Rule:        router.Rule,
		}
	}

	return nil
}

func getHttpRouters(host string, username string, password string) (config []*dynamic.Router, err error) {
	// Отправка GET-запроса
	client := &http.Client{}
	req, err := http.NewRequest("GET", host+"/api/http/routers", nil)
	req.SetBasicAuth(username, password)

	// Отправка запроса
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Проверка статуса ответа
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	// Декодирование JSON-ответа в структуру ResponseData
	var responseData []*dynamic.Router
	err = json.NewDecoder(resp.Body).Decode(&responseData)

	if err != nil {
		return nil, err
	}
	return responseData, nil
}
