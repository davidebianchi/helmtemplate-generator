package transform

import (
	"path/filepath"

	"github.com/davidebianchi/helmtemplate-generator/config"
)

// MatchesDocument checks if a document matches the given criteria
func MatchesDocument(doc *Document, match *config.Match) bool {
	if match == nil {
		return true
	}

	// Check kinds (list)
	if len(match.Kinds) > 0 {
		docKind := doc.GetKind()
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
			matched, err := filepath.Match(pattern, docName)
			if err == nil && matched {
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
