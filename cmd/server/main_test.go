package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFatalOnErr_NilError(t *testing.T) {
	// Should not panic when error is nil
	assert.NotPanics(t, func() {
		fatalOnErr(nil, "no error")
	})
}

func TestFatalOnErr_NonNilError(t *testing.T) {
	// Should panic when error is not nil
	assert.Panics(t, func() {
		fatalOnErr(errForTest("test error"), "test message")
	})
}

type errForTest string

func (e errForTest) Error() string { return string(e) }
