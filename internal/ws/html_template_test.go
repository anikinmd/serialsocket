package ws

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTerminalHTML_ContainsBasics(t *testing.T) {
	require.Contains(t, terminalHTML, "<!DOCTYPE html>")
	require.Contains(t, terminalHTML, "<div class=\"header\">")
	require.Contains(t, terminalHTML, "<script src=")
}
