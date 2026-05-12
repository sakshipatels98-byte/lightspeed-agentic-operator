package proposal

import (
	"context"
	"strings"
	"testing"

	agenticv1alpha1 "github.com/openshift/lightspeed-agentic-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRCA_Success(t *testing.T) {
	streams, out, _ := fakeStreams()
	fc := fake.NewClientBuilder().WithScheme(testScheme()).Build()

	o := &RCAOptions{
		client:      fc,
		namespace:   "intelliaide-mcp-server",
		agent:       "smart",
		request:     "etcd pods not ready in openshift-etcd",
		skillsImage: defaultIntelliAideSkillsImage,
		IOStreams:    streams,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "proposal/") {
		t.Errorf("expected proposal/ in output, got: %s", output)
	}
	if !strings.Contains(output, "created") {
		t.Errorf("expected 'created' in output, got: %s", output)
	}
	if !strings.Contains(output, "approve") {
		t.Errorf("expected approve hint in output, got: %s", output)
	}
}

func TestRCA_ProposalShape(t *testing.T) {
	streams, _, _ := fakeStreams()
	fc := fake.NewClientBuilder().WithScheme(testScheme()).Build()

	o := &RCAOptions{
		client:           fc,
		namespace:        "intelliaide-mcp-server",
		agent:            "smart",
		request:          "Cluster update stalled",
		skillsImage:      "quay.io/myorg/intelliaide-skills:dev",
		targetNamespaces: []string{"openshift-cluster-version"},
		IOStreams:         streams,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	list := &agenticv1alpha1.ProposalList{}
	if err := fc.List(context.Background(), list); err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(list.Items))
	}
	p := list.Items[0]

	// GenerateName prefix
	if p.GenerateName != "rca-" {
		t.Errorf("expected GenerateName 'rca-', got %q", p.GenerateName)
	}

	// Source label
	if p.Labels["agentic.openshift.io/source"] != "intelliaide" {
		t.Errorf("expected source label 'intelliaide', got %q", p.Labels["agentic.openshift.io/source"])
	}

	// Request field preserved
	if p.Spec.Request != "Cluster update stalled" {
		t.Errorf("expected request %q, got %q", "Cluster update stalled", p.Spec.Request)
	}

	// Analysis step set with correct agent
	if p.Spec.Analysis == nil || p.Spec.Analysis.Agent != "smart" {
		t.Errorf("expected analysis agent 'smart', got %v", p.Spec.Analysis)
	}

	// No execution or verification step (advisory only)
	if p.Spec.Execution != nil {
		t.Errorf("expected no execution step for RCA advisory proposal, got %v", p.Spec.Execution)
	}
	if p.Spec.Verification != nil {
		t.Errorf("expected no verification step for RCA advisory proposal, got %v", p.Spec.Verification)
	}

	// Skills image set
	if len(p.Spec.Tools.Skills) != 1 {
		t.Fatalf("expected 1 skills source, got %d", len(p.Spec.Tools.Skills))
	}
	if p.Spec.Tools.Skills[0].Image != "quay.io/myorg/intelliaide-skills:dev" {
		t.Errorf("expected skills image 'quay.io/myorg/intelliaide-skills:dev', got %q",
			p.Spec.Tools.Skills[0].Image)
	}

	// OutputSchema populated
	if p.Spec.Tools.OutputSchema == nil {
		t.Fatal("expected outputSchema to be set")
	}
	if p.Spec.Tools.OutputSchema.Type != "object" {
		t.Errorf("expected outputSchema.type 'object', got %q", p.Spec.Tools.OutputSchema.Type)
	}

	// rcaSummary and options in schema
	if _, ok := p.Spec.Tools.OutputSchema.Properties["rcaSummary"]; !ok {
		t.Error("expected outputSchema to have 'rcaSummary' property")
	}
	if _, ok := p.Spec.Tools.OutputSchema.Properties["options"]; !ok {
		t.Error("expected outputSchema to have 'options' property")
	}

	// Target namespaces
	if len(p.Spec.TargetNamespaces) != 1 || p.Spec.TargetNamespaces[0] != "openshift-cluster-version" {
		t.Errorf("expected target namespaces [openshift-cluster-version], got %v", p.Spec.TargetNamespaces)
	}
}

func TestRCA_DefaultSkillsImage(t *testing.T) {
	streams, _, _ := fakeStreams()
	fc := fake.NewClientBuilder().WithScheme(testScheme()).Build()

	o := &RCAOptions{
		client:      fc,
		namespace:   "default",
		agent:       "smart",
		request:     "OOM pods",
		skillsImage: defaultIntelliAideSkillsImage,
		IOStreams:    streams,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	list := &agenticv1alpha1.ProposalList{}
	if err := fc.List(context.Background(), list); err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.Items[0].Spec.Tools.Skills[0].Image != defaultIntelliAideSkillsImage {
		t.Errorf("expected default skills image %q, got %q",
			defaultIntelliAideSkillsImage,
			list.Items[0].Spec.Tools.Skills[0].Image)
	}
}

func TestRCA_JSONOutput(t *testing.T) {
	streams, out, _ := fakeStreams()
	fc := fake.NewClientBuilder().WithScheme(testScheme()).Build()

	o := &RCAOptions{
		client:      fc,
		namespace:   "default",
		agent:       "smart",
		request:     "etcd degraded",
		skillsImage: defaultIntelliAideSkillsImage,
		output:      "json",
		IOStreams:    streams,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, `"request"`) {
		t.Errorf("expected JSON with 'request' field, got: %s", got)
	}
	if !strings.Contains(got, `"skills"`) {
		t.Errorf("expected JSON with 'skills' field, got: %s", got)
	}
	if !strings.Contains(got, `"outputSchema"`) {
		t.Errorf("expected JSON with 'outputSchema' field, got: %s", got)
	}
}

func TestRCA_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    RCAOptions
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty request",
			opts:    RCAOptions{request: "  ", skillsImage: defaultIntelliAideSkillsImage},
			wantErr: true,
			errMsg:  "--request",
		},
		{
			name:    "empty skills image",
			opts:    RCAOptions{request: "fix", skillsImage: ""},
			wantErr: true,
			errMsg:  "--skills-image",
		},
		{
			name:    "invalid output format",
			opts:    RCAOptions{request: "fix", skillsImage: defaultIntelliAideSkillsImage, output: "xml"},
			wantErr: true,
		},
		{
			name:    "valid minimal",
			opts:    RCAOptions{request: "fix", skillsImage: defaultIntelliAideSkillsImage},
			wantErr: false,
		},
		{
			name:    "valid with json output",
			opts:    RCAOptions{request: "fix", skillsImage: defaultIntelliAideSkillsImage, output: "json"},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr && tc.errMsg != "" && err != nil && !strings.Contains(err.Error(), tc.errMsg) {
				t.Errorf("error should contain %q, got: %v", tc.errMsg, err)
			}
		})
	}
}

func TestRCA_OutputSchemaStructure(t *testing.T) {
	schema, err := parseRCAOutputSchema()
	if err != nil {
		t.Fatalf("parseRCAOutputSchema: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got %q", schema.Type)
	}

	// Must have required fields
	requiredSet := make(map[string]bool, len(schema.Required))
	for _, r := range schema.Required {
		requiredSet[r] = true
	}
	for _, field := range []string{"options", "rcaSummary"} {
		if !requiredSet[field] {
			t.Errorf("expected %q in required, got %v", field, schema.Required)
		}
	}

	// rcaSummary must have IntelliAide rca_structured fields
	rcaSummary, ok := schema.Properties["rcaSummary"]
	if !ok {
		t.Fatal("outputSchema missing 'rcaSummary' property")
	}
	for _, field := range []string{
		"executiveSummary",
		"chronologyOfEvents",
		"primaryRootCauses",
		"secondaryCauses",
		"aggregatedErrorPatterns",
		"recommendations",
		"evidenceFiles",
		"totalCostUsd",
	} {
		if _, ok := rcaSummary.Properties[field]; !ok {
			t.Errorf("rcaSummary missing field %q", field)
		}
	}

	// rcaSummary required fields
	summaryRequired := make(map[string]bool)
	for _, r := range rcaSummary.Required {
		summaryRequired[r] = true
	}
	for _, req := range []string{"executiveSummary", "primaryRootCauses", "recommendations"} {
		if !summaryRequired[req] {
			t.Errorf("rcaSummary.required missing %q", req)
		}
	}
}

func TestRCA_RequestTrimmed(t *testing.T) {
	streams, _, _ := fakeStreams()
	fc := fake.NewClientBuilder().WithScheme(testScheme()).Build()

	o := &RCAOptions{
		client:      fc,
		namespace:   "default",
		agent:       "smart",
		request:     "  etcd degraded  ",
		skillsImage: defaultIntelliAideSkillsImage,
		IOStreams:    streams,
	}

	if err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	list := &agenticv1alpha1.ProposalList{}
	if err := fc.List(context.Background(), list); err != nil {
		t.Fatalf("List: %v", err)
	}
	if list.Items[0].Spec.Request != "etcd degraded" {
		t.Errorf("expected request trimmed to 'etcd degraded', got %q", list.Items[0].Spec.Request)
	}
}
