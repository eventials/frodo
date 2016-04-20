package main

import "testing"

func TestDefaultValueEmpty(t *testing.T) {
    if defaultValue("", "test") != "test" {
        t.Fatal("Expected 'test'.")
    }
}

func TestDefaultValueNotEmpty(t *testing.T) {
    if defaultValue("value", "test") != "value" {
        t.Fatal("Expected 'value'.")
    }
}
