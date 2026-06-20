package responsebody

import (
	"context"
	"regexp"

	"github.com/pucora/lura/v2/config"
	"github.com/pucora/lura/v2/proxy"
)

const ReplacerNamespace = "modifier/response-body"
const GeneratorNamespace = "modifier/response-body-generator"

type replacementRule struct {
	pathStr string
	find    *regexp.Regexp
	replace string
}

func ProxyFactory(next proxy.Factory) proxy.Factory {
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		nextProxy, err := next.New(cfg)
		if err != nil {
			return proxy.NoopProxy, err
		}

		rules := getReplacerConfig(cfg.ExtraConfig)
		generatorMode, generatorData := getGeneratorConfig(cfg.ExtraConfig)

		if len(rules) == 0 && generatorMode == "" {
			return nextProxy, nil
		}

		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			resp, err := nextProxy(ctx, req)
			if err != nil || resp == nil {
				return resp, err
			}

			if generatorMode != "" {
				resp.Data = applyGenerator(generatorMode, generatorData, resp.Data)
			}

			if len(rules) > 0 {
				applyReplacements(resp.Data, rules)
			}

			return resp, nil
		}, nil
	})
}

func BackendFactory(next proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		nextProxy := next(cfg)

		rules := getReplacerConfig(cfg.ExtraConfig)
		generatorMode, generatorData := getGeneratorConfig(cfg.ExtraConfig)

		if len(rules) == 0 && generatorMode == "" {
			return nextProxy
		}

		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			resp, err := nextProxy(ctx, req)
			if err != nil || resp == nil {
				return resp, err
			}

			if generatorMode != "" {
				resp.Data = applyGenerator(generatorMode, generatorData, resp.Data)
			}

			if len(rules) > 0 {
				applyReplacements(resp.Data, rules)
			}

			return resp, nil
		}
	}
}

func getReplacerConfig(cfg config.ExtraConfig) []replacementRule {
	v, ok := cfg[ReplacerNamespace]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}

	var rules []replacementRule
	for path, op := range m {
		// we expect a list of {find, replace} objects, but could be a single object
		switch obj := op.(type) {
		case map[string]interface{}:
			findStr, _ := obj["find"].(string)
			replaceStr, _ := obj["replace"].(string)
			if findStr != "" {
				re, err := regexp.Compile(findStr)
				if err == nil {
					rules = append(rules, replacementRule{path, re, replaceStr})
				}
			}
		case []interface{}:
			for _, item := range obj {
				if itemMap, ok := item.(map[string]interface{}); ok {
					findStr, _ := itemMap["find"].(string)
					replaceStr, _ := itemMap["replace"].(string)
					if findStr != "" {
						re, err := regexp.Compile(findStr)
						if err == nil {
							rules = append(rules, replacementRule{path, re, replaceStr})
						}
					}
				}
			}
		}
	}
	return rules
}

func getGeneratorConfig(cfg config.ExtraConfig) (string, interface{}) {
	v, ok := cfg[GeneratorNamespace]
	if !ok {
		return "", nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return "", nil
	}

	template, ok := m["template"].(string)
	if ok {
		return "template", template
	}

	static, ok := m["static"]
	if ok {
		return "static", static
	}

	return "", nil
}

func applyGenerator(mode string, data interface{}, existing map[string]interface{}) map[string]interface{} {
	if mode == "static" {
		if staticMap, ok := data.(map[string]interface{}); ok {
			return staticMap
		}
		// if static isn't a map, wrap it
		return map[string]interface{}{"content": data}
	}
	// mode == "template" (stub implementation for simplicity)
	// would parse the go template with the existing data
	return existing
}

func applyReplacements(data map[string]interface{}, rules []replacementRule) {
	for _, rule := range rules {
		// simplify: only 1 level
		if val, ok := data[rule.pathStr]; ok {
			if strVal, ok := val.(string); ok {
				data[rule.pathStr] = rule.find.ReplaceAllString(strVal, rule.replace)
			}
		}
	}
}
