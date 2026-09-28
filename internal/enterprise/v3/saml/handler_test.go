package saml

import (
	crewsaml "github.com/crewjam/saml"
	"testing"
)

func TestAttributeExtraction(t *testing.T) {
	a := &crewsaml.Assertion{AttributeStatements: []crewsaml.AttributeStatement{{Attributes: []crewsaml.Attribute{{Name: "mail", Values: []crewsaml.AttributeValue{{Value: "User@Example.com"}}}}}}}
	if got := attribute(a, "email", "mail"); got != "User@Example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestAttributeExtractionFriendlyName(t *testing.T) {
	a := &crewsaml.Assertion{AttributeStatements: []crewsaml.AttributeStatement{{Attributes: []crewsaml.Attribute{{FriendlyName: "displayName", Values: []crewsaml.AttributeValue{{Value: "Test User"}}}}}}}
	if got := attribute(a, "name", "displayName"); got != "Test User" {
		t.Fatalf("got %q", got)
	}
}
