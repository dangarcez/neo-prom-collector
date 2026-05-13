package config

import (
	"fmt"
	"strings"
	"time"
)

type RuntimeConfig struct {
	Env  EnvConfig
	File FileConfig
}

type EnvConfig struct {
	ConfigPath          string
	Neo4jURI            string
	Neo4jDatabase       string
	Neo4jUsername       string
	Neo4jPassword       string
	Neo4jTimeout        time.Duration
	VerifyConnectivity  bool
	MaxDatapointWorkers int
	LogLevel            string
	LogFormat           string
}

type FileConfig struct {
	PromTargets []PromTargetConfig `yaml:"prom_targets"`
}

func (c *FileConfig) Normalize() {
	for i := range c.PromTargets {
		c.PromTargets[i].Normalize()
	}
}

func (c FileConfig) AllTargetsDryRun() bool {
	if len(c.PromTargets) == 0 {
		return false
	}

	for _, target := range c.PromTargets {
		if !target.Runtime.DryRun {
			return false
		}
	}

	return true
}

type PromTargetConfig struct {
	Name           string              `yaml:"name"`
	BaseURL        string              `yaml:"base_url"`
	AzureAuth      *AzureAuthConfig    `yaml:"azure_auth"`
	TimeoutSeconds int                 `yaml:"timeout_seconds"`
	VerifyTLS      *bool               `yaml:"verify_tls"`
	Runtime        TargetRuntimeConfig `yaml:"runtime"`
	Jobs           []JobConfig         `yaml:"jobs"`
}

func (t *PromTargetConfig) Normalize() {
	if t.TimeoutSeconds <= 0 {
		t.TimeoutSeconds = 10
	}

	if t.Runtime.DefaultIntervalSeconds <= 0 {
		t.Runtime.DefaultIntervalSeconds = 60
	}

	if t.AzureAuth != nil {
		t.AzureAuth.Normalize()
	}

	for i := range t.Jobs {
		t.Jobs[i].Normalize(t.Runtime.DefaultIntervalSeconds)
	}
}

func (t PromTargetConfig) VerifyTLSEnabled() bool {
	if t.VerifyTLS == nil {
		return true
	}

	return *t.VerifyTLS
}

type AzureAuthConfig struct {
	ManagedIdentityID string `yaml:"managed_identity_id"`
}

func (a *AzureAuthConfig) Normalize() {
	a.ManagedIdentityID = strings.TrimSpace(a.ManagedIdentityID)
}

type TargetRuntimeConfig struct {
	DefaultIntervalSeconds int     `yaml:"default_interval_seconds"`
	SleepSeconds           float64 `yaml:"sleep_seconds"`
	DryRun                 bool    `yaml:"dry_run"`
}

type JobConfig struct {
	Name            string                       `yaml:"name"`
	Query           string                       `yaml:"query"`
	IntervalSeconds int                          `yaml:"interval_seconds"`
	Nodes           []NodeTemplateConfig         `yaml:"nodes"`
	Relationships   []RelationshipTemplateConfig `yaml:"relationships"`
}

func (j *JobConfig) Normalize(defaultIntervalSeconds int) {
	if j.IntervalSeconds <= 0 {
		j.IntervalSeconds = defaultIntervalSeconds
	}

	for i := range j.Nodes {
		j.Nodes[i].Normalize()
	}

	for i := range j.Relationships {
		j.Relationships[i].Normalize()
	}
}

func (j JobConfig) Interval() time.Duration {
	return time.Duration(j.IntervalSeconds) * time.Second
}

type NodeTemplateConfig struct {
	Type                  string                      `yaml:"type"`
	Types                 []string                    `yaml:"types"`
	TemplateHashes        []string                    `yaml:"template_hashes"`
	UpdatePolicy          string                      `yaml:"update_policy"`
	ExpirationTimeMin     *int                        `yaml:"expiration_time_min"`
	StaticProperties      map[string]any              `yaml:"static_properties"`
	LabelProperties       map[string]string           `yaml:"label_properties"`
	ConditionalProperties []ConditionalPropertyConfig `yaml:"conditional_properties"`
	PropertyTransforms    []PropertyTransformConfig   `yaml:"property_transforms"`
	Conditions            []ConditionConfig           `yaml:"conditions"`
}

func (n *NodeTemplateConfig) Normalize() {
	if len(n.Types) == 0 && strings.TrimSpace(n.Type) != "" {
		n.Types = []string{strings.TrimSpace(n.Type)}
	}

	if strings.TrimSpace(n.UpdatePolicy) == "" {
		n.UpdatePolicy = "create"
	}

	if n.StaticProperties == nil {
		n.StaticProperties = map[string]any{}
	}

	if n.LabelProperties == nil {
		n.LabelProperties = map[string]string{}
	}

	for i := range n.PropertyTransforms {
		n.PropertyTransforms[i].Normalize()
	}
}

func (n NodeTemplateConfig) NormalizedTypes() []string {
	out := make([]string, 0, len(n.Types))
	for _, value := range n.Types {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}

	return out
}

func (n NodeTemplateConfig) HasNameProperty() bool {
	if _, ok := n.StaticProperties["name"]; ok {
		return true
	}

	_, ok := n.LabelProperties["name"]
	return ok
}

type RelationshipTemplateConfig struct {
	Type                  string                      `yaml:"type"`
	TemplateHash          string                      `yaml:"template_hash"`
	TemplateHashes        []string                    `yaml:"template_hashes"`
	UpdatePolicy          string                      `yaml:"update_policy"`
	ExpirationTimeMin     *int                        `yaml:"expiration_time_min"`
	StaticProperties      map[string]any              `yaml:"static_properties"`
	LabelProperties       map[string]string           `yaml:"label_properties"`
	ConditionalProperties []ConditionalPropertyConfig `yaml:"conditional_properties"`
	PropertyTransforms    []PropertyTransformConfig   `yaml:"property_transforms"`
	Conditions            []ConditionConfig           `yaml:"conditions"`
	Source                RelationshipEndpointConfig  `yaml:"source"`
	Target                RelationshipEndpointConfig  `yaml:"target"`
}

func (r *RelationshipTemplateConfig) Normalize() {
	if strings.TrimSpace(r.TemplateHash) == "" && len(r.TemplateHashes) == 1 {
		r.TemplateHash = strings.TrimSpace(r.TemplateHashes[0])
	}

	if strings.TrimSpace(r.UpdatePolicy) == "" {
		r.UpdatePolicy = "create"
	}

	if r.StaticProperties == nil {
		r.StaticProperties = map[string]any{}
	}

	if r.LabelProperties == nil {
		r.LabelProperties = map[string]string{}
	}

	for i := range r.PropertyTransforms {
		r.PropertyTransforms[i].Normalize()
	}

	r.Source.Normalize()
	r.Target.Normalize()
}

func (r RelationshipTemplateConfig) NormalizedTemplateHash() string {
	return strings.TrimSpace(r.TemplateHash)
}

type RelationshipEndpointConfig struct {
	Type                   string                    `yaml:"type"`
	MatchAttributes        SelectorAttributes        `yaml:"match_attributes"`
	PriorTransform         []PropertyTransformConfig `yaml:"prior_transform"`
	LegacyMatchLabelAttrs  map[string]any            `yaml:"match_label_attributes"`
	LegacyMatchStaticAttrs map[string]any            `yaml:"match_static_attributes"`
}

func (e *RelationshipEndpointConfig) Normalize() {
	e.MatchAttributes.Normalize()

	for i := range e.PriorTransform {
		e.PriorTransform[i].Normalize()
	}

	for key, value := range e.LegacyMatchStaticAttrs {
		e.MatchAttributes.Static[key] = value
	}

	for key, value := range e.LegacyMatchLabelAttrs {
		if key == "static" {
			nested, ok := value.(map[string]any)
			if ok {
				for nestedKey, nestedValue := range nested {
					e.MatchAttributes.Static[nestedKey] = nestedValue
				}
			}
			continue
		}

		if value != nil {
			e.MatchAttributes.Labels[key] = fmt.Sprint(value)
		}
	}
}

type SelectorAttributes struct {
	Static map[string]any    `yaml:"static"`
	Labels map[string]string `yaml:"labels"`
}

func (s *SelectorAttributes) Normalize() {
	if s.Static == nil {
		s.Static = map[string]any{}
	}

	if s.Labels == nil {
		s.Labels = map[string]string{}
	}
}

type ConditionalPropertyConfig struct {
	Type       string            `yaml:"type"`
	Name       string            `yaml:"name"`
	Value      any               `yaml:"value"`
	FromLabel  string            `yaml:"from_label"`
	Conditions []ConditionConfig `yaml:"conditions"`
}

type PropertyTransformConfig struct {
	Property string                    `yaml:"property"`
	Process  []PropertyProcessorConfig `yaml:"process"`
}

func (c *PropertyTransformConfig) Normalize() {
	c.Property = strings.TrimSpace(c.Property)

	for i := range c.Process {
		c.Process[i].Normalize()
	}
}

type PropertyProcessorConfig struct {
	Type    string `yaml:"type"`
	Pattern string `yaml:"pattern"`
	Output  string `yaml:"output"`
}

func (c *PropertyProcessorConfig) Normalize() {
	c.Type = NormalizePropertyProcessorType(c.Type)
	c.Pattern = strings.TrimSpace(c.Pattern)
}

const (
	PropertyProcessorTypeToUpper = "TO_UPPER"
	PropertyProcessorTypeToLower = "TO_LOWER"
	PropertyProcessorTypeRegex   = "REGEX"
)

func NormalizePropertyProcessorType(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func IsSupportedPropertyProcessorType(value string) bool {
	switch NormalizePropertyProcessorType(value) {
	case PropertyProcessorTypeToUpper, PropertyProcessorTypeToLower, PropertyProcessorTypeRegex:
		return true
	default:
		return false
	}
}

func NormalizeRegexPattern(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= 2 && strings.HasPrefix(trimmed, "/") && strings.HasSuffix(trimmed, "/") {
		return strings.TrimSpace(trimmed[1 : len(trimmed)-1])
	}
	return trimmed
}

type ConditionConfig struct {
	Type        string   `yaml:"type"`
	Label       string   `yaml:"label"`
	Equals      any      `yaml:"equals"`
	NotEquals   any      `yaml:"not_equals"`
	GreaterThan *float64 `yaml:"greater_than"`
	LessThan    *float64 `yaml:"less_than"`
}
