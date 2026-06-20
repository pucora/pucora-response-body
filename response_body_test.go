package responsebody_test

import (
	"context"
	"testing"

	"github.com/pucora/lura/v2/config"
	"github.com/pucora/lura/v2/proxy"
	responsebody "github.com/pucora/pucora-response-body"
)

func TestProxyFactory(t *testing.T) {
	expectedData := map[string]interface{}{"msg": "hello world"}
	next := proxy.FactoryFunc(func(_ *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return &proxy.Response{
				Data:       expectedData,
				IsComplete: true,
			}, nil
		}, nil
	})

	cfg := &config.EndpointConfig{
		ExtraConfig: map[string]interface{}{
			responsebody.ReplacerNamespace: map[string]interface{}{
				"msg": map[string]interface{}{
					"find":    "world",
					"replace": "moon",
				},
			},
		},
	}

	pf := responsebody.ProxyFactory(next)
	p, err := pf.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := p(context.Background(), &proxy.Request{})
	if err != nil {
		t.Fatal(err)
	}

	v, ok := resp.Data["msg"]
	if !ok || v != "hello moon" {
		t.Errorf("unexpected output: %v", resp.Data)
	}
}

func TestGeneratorProxyFactory(t *testing.T) {
	next := proxy.FactoryFunc(func(_ *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return &proxy.Response{
				Data:       map[string]interface{}{"old": "data"},
				IsComplete: true,
			}, nil
		}, nil
	})

	cfg := &config.EndpointConfig{
		ExtraConfig: map[string]interface{}{
			responsebody.GeneratorNamespace: map[string]interface{}{
				"static": map[string]interface{}{"new": "generated"},
			},
		},
	}

	pf := responsebody.ProxyFactory(next)
	p, err := pf.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := p(context.Background(), &proxy.Request{})
	if err != nil {
		t.Fatal(err)
	}

	v, ok := resp.Data["new"]
	if !ok || v != "generated" {
		t.Errorf("unexpected output: %v", resp.Data)
	}
}
