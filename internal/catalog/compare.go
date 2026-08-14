package catalog

import (
	"regexp"
	"strconv"
	"strings"
)

var versionSegRE = regexp.MustCompile(`\d+`)

func versionParts(s string) []int {
	s = strings.TrimLeft(s, "vVgo")
	var parts []int
	for _, m := range versionSegRE.FindAllString(s, -1) {
		n, _ := strconv.Atoi(m)
		parts = append(parts, n)
	}
	return parts
}

// CompareVersions compares two version strings ("go1.27.4", "v24.19.0",
// "0.64.1") numerically, ignoring "go"/"v" prefixes. It returns -1, 0 or 1.
func CompareVersions(a, b string) int {
	an, bn := versionParts(a), versionParts(b)
	for i := 0; i < len(an) && i < len(bn); i++ {
		if an[i] < bn[i] {
			return -1
		}
		if an[i] > bn[i] {
			return 1
		}
	}
	switch {
	case len(an) < len(bn):
		return -1
	case len(an) > len(bn):
		return 1
	}
	return 0
}
