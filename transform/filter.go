package transform

import (
	"github.com/davidebianchi/helmtemplate-generator/config"
)

// FilterDocuments applies include/exclude filters to a list of documents,
// returning only the documents that pass the filter criteria.
// If filter is nil, all documents are returned unchanged.
func FilterDocuments(docs []*Document, filter *config.Filter) []*Document {
	if filter == nil {
		return docs
	}

	result := make([]*Document, 0, len(docs))
	for _, doc := range docs {
		if shouldIncludeDocument(doc, filter) {
			result = append(result, doc)
		}
	}
	return result
}

// shouldIncludeDocument determines whether a single document passes the filter.
//
// Include is scoped by kind: it only restricts resources whose kind is
// mentioned in at least one include entry. Resources of other kinds pass through.
//
// Exclude always applies: if a document matches any exclude entry, it is excluded.
func shouldIncludeDocument(doc *Document, filter *config.Filter) bool {
	// Check include rules (scoped by kind)
	if len(filter.Include) > 0 {
		docKind := doc.GetKind()
		kindInFilter := false
		for i := range filter.Include {
			for _, k := range filter.Include[i].Kinds {
				if k == docKind {
					kindInFilter = true
					break
				}
			}
			if kindInFilter {
				break
			}
		}

		// If the doc's kind is mentioned in any include entry,
		// it must match at least one include entry to be kept
		if kindInFilter {
			included := false
			for i := range filter.Include {
				if MatchesDocument(doc, &filter.Include[i]) {
					included = true
					break
				}
			}
			if !included {
				return false
			}
		}
		// If the doc's kind is NOT mentioned, it passes through
	}

	// Check exclude rules (any match = excluded)
	for i := range filter.Exclude {
		if MatchesDocument(doc, &filter.Exclude[i]) {
			return false
		}
	}

	return true
}
