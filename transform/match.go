package transform

import (
	"github.com/davidebianchi/helmtemplate-generator/config"
)

// MatchesDocument checks if a document matches the given criteria
func MatchesDocument(doc *Document, match *config.Match) bool {
	if match == nil {
		return true
	}

	docKind := doc.GetKind()

	// Check excludeKinds (list)
	for _, k := range match.ExcludeKinds {
		if k == docKind {
			return false
		}
	}

	// Check kinds (list)
	if len(match.Kinds) > 0 {
		found := false
		for _, k := range match.Kinds {
			if k == docKind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check names (supports wildcards, matches if ANY pattern matches)
	if len(match.Names) > 0 {
		docName := doc.GetName()
		nameMatched := false
		for _, pattern := range match.Names {
			if matchWildcard(pattern, docName) {
				nameMatched = true
				break
			}
		}
		if !nameMatched {
			return false
		}
	}

	// Check labels
	if len(match.Labels) > 0 {
		docLabels := doc.GetLabels()
		for key, value := range match.Labels {
			if docLabels[key] != value {
				return false
			}
		}
	}

	return true
}
