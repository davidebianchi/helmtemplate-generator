package transform

import (
	"testing"

	"github.com/davidebianchi/helmtemplate-generator/config"
)

func docFromYAML(t *testing.T, input string) *Document {
	t.Helper()
	docs, err := ParseDocuments([]byte(input))
	if err != nil {
		t.Fatalf("failed to parse document: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("expected at least 1 document")
	}
	return docs[0]
}

func TestMatchesDocument_NilMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: ConfigMap\nmetadata:\n  name: test")
	if !MatchesDocument(doc, nil) {
		t.Error("nil match should match everything")
	}
}

func TestMatchesDocument_KindMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test")
	match := &config.Match{Kinds: []string{"Deployment"}}
	if !MatchesDocument(doc, match) {
		t.Error("expected Deployment to match")
	}
}

func TestMatchesDocument_KindNoMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Service\nmetadata:\n  name: test")
	match := &config.Match{Kinds: []string{"Deployment"}}
	if MatchesDocument(doc, match) {
		t.Error("expected Service not to match Deployment")
	}
}

func TestMatchesDocument_MultipleKinds(t *testing.T) {
	doc := docFromYAML(t, "kind: Service\nmetadata:\n  name: test")
	match := &config.Match{Kinds: []string{"Deployment", "Service"}}
	if !MatchesDocument(doc, match) {
		t.Error("expected Service to match [Deployment, Service]")
	}
}

func TestMatchesDocument_NameWildcard(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: my-deployment")
	match := &config.Match{Name: "my-*"}
	if !MatchesDocument(doc, match) {
		t.Error("expected my-deployment to match my-*")
	}
}

func TestMatchesDocument_NameWildcardNoMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: other-deploy")
	match := &config.Match{Name: "my-*"}
	if MatchesDocument(doc, match) {
		t.Error("expected other-deploy not to match my-*")
	}
}

func TestMatchesDocument_LabelMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test\n  labels:\n    app: myapp")
	match := &config.Match{Labels: map[string]string{"app": "myapp"}}
	if !MatchesDocument(doc, match) {
		t.Error("expected labels to match")
	}
}

func TestMatchesDocument_LabelNoMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test\n  labels:\n    app: other")
	match := &config.Match{Labels: map[string]string{"app": "myapp"}}
	if MatchesDocument(doc, match) {
		t.Error("expected labels not to match")
	}
}

func TestMatchesDocument_LabelMissing(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test")
	match := &config.Match{Labels: map[string]string{"app": "myapp"}}
	if MatchesDocument(doc, match) {
		t.Error("expected no match when document has no labels")
	}
}
