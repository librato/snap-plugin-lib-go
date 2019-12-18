package metrictree

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	NsSeparator                  = "/"
	regexBeginIndicator          = "{"
	regexEndIndicator            = "}"
	staticAnyMatcher             = "*"
	staticRecursiveAnyMatcher    = "**"
	dynamicElementBeginIndicator = "["
	dynamicElementEndIndicator   = "]"
	dynamicElementEqualIndicator = "="
)

const minNamespaceElements = 2

const initCacheSize = 100

var filteredNsBuffer = make(map[string]namespaceElement, initCacheSize)
var noFilteredNsBuffer = make(map[string]namespaceElement, initCacheSize)

// Parsing whole selector (ie. "/plugin/[group={reg}]/group2/metric1) into smaller elements
func ParseNamespace(s string, isFilter bool) (*Namespace, error) {
	ns := &Namespace{}
	splitNs := strings.Split(s, NsSeparator)
	if len(splitNs)-1 < minNamespaceElements {
		return nil, fmt.Errorf("namespace doesn't contain valid numbers of elements (min. %d), %s", minNamespaceElements, s)
	}
	if splitNs[0] != "" {
		return nil, fmt.Errorf("namespace should start with '%s'", NsSeparator)
	}

	for i, nsElem := range splitNs[1:] {
		var parsedEl namespaceElement
		var ok bool
		var err error

		if isFilter {
			parsedEl, ok = filteredNsBuffer[nsElem]
		} else {
			parsedEl, ok = noFilteredNsBuffer[nsElem]
		}

		if !ok {
			parsedEl, err = parseNamespaceElement(nsElem, isFilter)
			if err != nil {
				return nil, fmt.Errorf("can't parse namespace (%s), error at index %d: %s", s, i, err)
			}
		}

		if _, ok := parsedEl.(*staticRecursiveAnyElement); ok && i != len(splitNs[1:])-1 {
			return nil, fmt.Errorf("recursive any-matcher (**) can be placed only as the last element")
		}

		if isFilter {
			filteredNsBuffer[nsElem] = parsedEl
		} else {
			noFilteredNsBuffer[nsElem] = parsedEl
		}

		ns.elements = append(ns.elements, parsedEl)
	}

	return ns, nil
}

// Parsing single selector (ie. [group={reg}])
func parseNamespaceElement(s string, isFilter bool) (namespaceElement, error) {
	if containsGroup(s) { // is it group []?
		dynElem := s[1 : len(s)-1]
		eqIndex := strings.Index(dynElem, dynamicElementEqualIndicator)

		if eqIndex != -1 { // is it group with value [group=id]
			groupName := dynElem[0:eqIndex]
			groupValue := dynElem[eqIndex+1:]

			if !isValidIdentifier(groupName) {
				return nil, fmt.Errorf("invalid character(s) used for group name [%s]", groupName)
			}

			if containsRegexp(groupValue) {
				regexStr := groupValue[1 : len(groupValue)-1]
				r, err := regexp.Compile(regexStr)
				if err != nil {
					return nil, fmt.Errorf("invalid regular expression (%s): %s", regexStr, err)
				}
				return newDynamicRegexpElement(groupName, r), nil
			}

			if isValidIdentifier(groupValue) {
				return newDynamicSpecificElement(groupName, groupValue), nil
			}

			return nil, fmt.Errorf("invalid character(s) used for group value [%s]", groupValue)
		}

		if isValidIdentifier(dynElem) {
			return newDynamicAnyElement(dynElem), nil
		}

		return nil, fmt.Errorf("invalid character(s) used for group value [%s]", dynElem)
	}

	if containsRegexp(s) { // is it {regex}
		regexStr := s[1 : len(s)-1]
		r, err := regexp.Compile(regexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid regular expression (%s): %s", regexStr, err)
		}

		if isFilter {
			return newStaticRegexpAcceptingGroupElement(r), nil
		} else {
			return newStaticRegexpElement(r), nil
		}
	}

	if s == staticRecursiveAnyMatcher { // is it **
		return newStaticRecursiveAnyElement(), nil
	}

	if s == staticAnyMatcher { // is it *
		return newStaticAnyElement(), nil
	}

	if isValidIdentifier(s) { // is it static element ie. metric
		if isFilter {
			return newStaticSpecificAcceptingGroupElement(s), nil
		} else {
			return newStaticSpecificElement(s), nil
		}
	}

	return nil, fmt.Errorf("invalid character(s) used for element [%s]", s)
}

/*****************************************************************************/

func isSurroundedWith(s string, prefix, suffix string) bool {
	if !strings.HasPrefix(s, prefix) || !strings.HasSuffix(s, suffix) {
		return false
	}

	return true
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	for _, el := range s {
		switch {
		case el >= 'A' && el <= 'Z':
		case el >= 'a' && el <= 'z':
		case el >= '0' && el <= '9':
		case el == '-' || el == '_':
		default:
			return false
		}
	}

	return true
}

func containsRegexp(s string) bool {
	return isSurroundedWith(s, regexBeginIndicator, regexEndIndicator)
}

func containsGroup(s string) bool {
	return isSurroundedWith(s, dynamicElementBeginIndicator, dynamicElementEndIndicator)
}
