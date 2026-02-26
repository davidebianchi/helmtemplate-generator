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

	// Check name (supports wildcards)
	if match.Name != "" {
		docName := doc.GetName()
		matched, err := filepath.Match(match.Name, docName)
		if err != nil || !matched {
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
