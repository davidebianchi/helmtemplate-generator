package transform

import (
	"testing"

	"github.com/davidebianchi/helmtemplate-generator/config"
	"github.com/stretchr/testify/require"
)

func docFromYAML(t *testing.T, input string) *Document {
	t.Helper()
	docs, err := ParseDocuments([]byte(input))
	require.NoError(t, err)
	require.NotEmpty(t, docs, "expected at least 1 document")
	return docs[0]
}

func TestMatchesDocument_NilMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: ConfigMap\nmetadata:\n  name: test")
	require.True(t, MatchesDocument(doc, nil), "nil match should match everything")
}

func TestMatchesDocument_KindMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test")
	match := &config.Match{Kinds: []string{"Deployment"}}
	require.True(t, MatchesDocument(doc, match), "expected Deployment to match")
}

func TestMatchesDocument_KindNoMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Service\nmetadata:\n  name: test")
	match := &config.Match{Kinds: []string{"Deployment"}}
	require.False(t, MatchesDocument(doc, match), "expected Service not to match Deployment")
}

func TestMatchesDocument_MultipleKinds(t *testing.T) {
	doc := docFromYAML(t, "kind: Service\nmetadata:\n  name: test")
	match := &config.Match{Kinds: []string{"Deployment", "Service"}}
	require.True(t, MatchesDocument(doc, match), "expected Service to match [Deployment, Service]")
}

func TestMatchesDocument_ExcludeKinds(t *testing.T) {
	doc := docFromYAML(t, "kind: Job\nmetadata:\n  name: test")
	match := &config.Match{ExcludeKinds: []string{"Job", "CronJob"}}
	require.False(t, MatchesDocument(doc, match), "expected Job to be excluded")
}

func TestMatchesDocument_ExcludeKindsNoMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test")
	match := &config.Match{ExcludeKinds: []string{"Job", "CronJob"}}
	require.True(t, MatchesDocument(doc, match), "expected Deployment not to be excluded")
}

func TestMatchesDocument_ExcludeKindsWithKinds(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test")
	match := &config.Match{Kinds: []string{"Deployment", "Job"}, ExcludeKinds: []string{"Job"}}
	require.True(t, MatchesDocument(doc, match), "expected Deployment to match kinds and not be excluded")

	doc2 := docFromYAML(t, "kind: Job\nmetadata:\n  name: test")
	require.False(t, MatchesDocument(doc2, match), "expected Job to be excluded even though it's in kinds")
}

func TestMatchesDocument_NameWildcard(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: my-deployment")
	match := &config.Match{Names: []string{"my-*"}}
	require.True(t, MatchesDocument(doc, match), "expected my-deployment to match my-*")
}

func TestMatchesDocument_NameWildcardNoMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: other-deploy")
	match := &config.Match{Names: []string{"my-*"}}
	require.False(t, MatchesDocument(doc, match), "expected other-deploy not to match my-*")
}

func TestMatchesDocument_NamesMultiplePatterns(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: other-deploy")
	match := &config.Match{Names: []string{"my-*", "other-*"}}
	require.True(t, MatchesDocument(doc, match), "expected other-deploy to match [my-*, other-*]")
}

func TestMatchesDocument_NamesNoneMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: unrelated")
	match := &config.Match{Names: []string{"my-*", "other-*"}}
	require.False(t, MatchesDocument(doc, match), "expected unrelated not to match [my-*, other-*]")
}

func TestMatchesDocument_NamesEmpty(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: anything")
	match := &config.Match{Names: []string{}}
	require.True(t, MatchesDocument(doc, match), "empty Names should match everything")
}

func TestMatchesDocument_LabelMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test\n  labels:\n    app: myapp")
	match := &config.Match{Labels: map[string]string{"app": "myapp"}}
	require.True(t, MatchesDocument(doc, match), "expected labels to match")
}

func TestMatchesDocument_LabelNoMatch(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test\n  labels:\n    app: other")
	match := &config.Match{Labels: map[string]string{"app": "myapp"}}
	require.False(t, MatchesDocument(doc, match), "expected labels not to match")
}

func TestMatchesDocument_LabelMissing(t *testing.T) {
	doc := docFromYAML(t, "kind: Deployment\nmetadata:\n  name: test")
	match := &config.Match{Labels: map[string]string{"app": "myapp"}}
	require.False(t, MatchesDocument(doc, match), "expected no match when document has no labels")
}
