package filter

import "apitester/pkg/utils"

func MatchTags(caseTags []string, includeTags []string, excludeTags []string) bool {
	if !matchIncludeTags(caseTags, includeTags) {
		return false
	}

	if !matchExcludeTags(caseTags, excludeTags) {
		return false
	}

	return true
}

func matchIncludeTags(caseTags []string, includeTags []string) bool {
	if len(includeTags) == 0 {
		return true
	}

	for _, tag := range includeTags {
		if !utils.ContainsString(caseTags, tag) {
			return false
		}
	}

	return true
}

func matchExcludeTags(caseTags []string, excludeTags []string) bool {
	if len(excludeTags) == 0 {
		return true
	}

	for _, tag := range excludeTags {
		if utils.ContainsString(caseTags, tag) {
			return false
		}
	}

	return true
}

func HasAnyTag(caseTags []string, tags []string) bool {
	if len(tags) == 0 {
		return true
	}

	for _, tag := range tags {
		if utils.ContainsString(caseTags, tag) {
			return true
		}
	}

	return false
}

func HasAllTags(caseTags []string, tags []string) bool {
	if len(tags) == 0 {
		return true
	}

	for _, tag := range tags {
		if !utils.ContainsString(caseTags, tag) {
			return false
		}
	}

	return true
}
