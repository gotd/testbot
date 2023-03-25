package main

import (
	"bytes"
	"os"
	"path/filepath"

	yaml "github.com/go-faster/yamlx"
	"github.com/gotd/td/telegram/dcs"
)

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type PodSelector struct {
	MatchLabels map[string]string `yaml:"matchLabels"`
}

type EgressPort struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
}

type IPBlock struct {
	CIDR string `yaml:"cidr"`
}

type ToRule struct {
	IPBlock IPBlock `yaml:"ipBlock"`
}

type EgressRule struct {
	To    []ToRule     `yaml:"to"`
	Ports []EgressPort `yaml:"ports"`
}

type Spec struct {
	PodSelector PodSelector  `yaml:"podSelector"`
	Egress      []EgressRule `yaml:"egress"`
}

type Resource struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

func main() {
	var rules []ToRule
	for _, dc := range dcs.Prod().Options {
		rules = append(rules, ToRule{
			IPBlock: IPBlock{
				CIDR: dc.IPAddress + "/32",
			},
		})
	}
	r := Resource{
		APIVersion: "networking.k8s.io/v1",
		Kind:       "NetworkPolicy",
		Spec: Spec{
			Egress: []EgressRule{
				{
					To: rules,
					Ports: []EgressPort{
						{Port: 443, Protocol: "TCP"},
					},
				},
			},
			PodSelector: PodSelector{
				MatchLabels: map[string]string{
					"app": "testbot",
				},
			},
		},
		Metadata: Metadata{
			Name:      "testbot-egress-telegram",
			Namespace: "gotd-sandbox",
		},
	}

	b := new(bytes.Buffer)
	e := yaml.NewEncoder(b)
	e.SetIndent(2)
	if err := e.Encode(r); err != nil {
		panic(err)
	}

	out := filepath.Join(".k8s", "cnp.egress.yaml")
	if err := os.WriteFile(out, b.Bytes(), 0o644); err != nil {
		panic(err)
	}
}
