package metrictree

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const nsSeparator = "/"
const regexBeginIndicator = '{'
const regexEndIndicator = '}'
const staticAnyMatcher = '*'
const dynamicElementBeginIndicator = '['
const dynamicElementEndIndicator = ']'
const dynamicElementEqualIndicator = '='

// Parsing whole selector (ie. "/plugin/[group={reg}]/group2/metric1) into smaller elements
func ParseNamespace(s string) (*Namespace, error) {
	ns := &Namespace{}
	splitedNs := strings.Split(s, nsSeparator)
	if len(splitedNs) < 2 {
		return nil, errors.New("namespace doesn't contain valid numbers of elements")
	}
	if splitedNs[0] != "" {
		return nil, fmt.Errorf("namespace should start with '%s'", nsSeparator)
	}

	for i, nsElem := range splitedNs[1:] {
		parsedEl, err := ParseNamespaceElement(nsElem)
		if err != nil {
			return nil, fmt.Errorf("can't parse namespace (%s), error at index %d: %s", s, i, err)
		}
		ns.elements = append(ns.elements, parsedEl)
	}

	return ns, nil
}

// Parsing single selector (ie. [group={reg}])
func ParseNamespaceElement(s string) (namespaceElement, error) {
	if isSurroundedWith(s, dynamicElementBeginIndicator, dynamicElementEndIndicator) {
		dynElem := s[1 : len(s)-1]
		eqIndex := strings.Index(dynElem, string(dynamicElementEqualIndicator))

		if eqIndex != -1 {
			if len(dynElem) >= 3 && eqIndex > 0 && eqIndex < len(dynElem)-1 {
				groupName := dynElem[0:eqIndex]
				groupValue := dynElem[eqIndex+1:]

				if isSurroundedWith(groupValue, regexBeginIndicator, regexEndIndicator) {
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
			}
		}

		if isValidIdentifier(dynElem) {
			return newDynamicAnyElement(dynElem), nil
		}
	}

	if isSurroundedWith(s, regexBeginIndicator, regexEndIndicator) {
		regexStr := s[1 : len(s)-1]
		r, err := regexp.Compile(regexStr)
		if err != nil {
			return nil, fmt.Errorf("invalid regular expression (%s): %s", regexStr, err)
		}
		return newStaticRegexpElement(r), nil
	}

	if s == string(staticAnyMatcher) {
		return newStaticAnyElement(), nil
	}

	if isValidIdentifier(s) {
		return newStaticSpecificElement(s), nil
	}

	return nil, fmt.Errorf("couldn't recognize selector (%s)", s)
}

func isSurroundedWith(s string, prefix, postfix rune) bool {
	r := []rune(s)
	if len(r) < 2 {
		return false
	}
	if r[0] != prefix || r[len(r)-1] != postfix {
		return false
	}
	return true
}

func isValidIdentifier(s string) bool {
	return len(s) > 0 // todo: check is contains valid characters
}