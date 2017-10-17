package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecureInput(t *testing.T) {
	t.Log("secure empty")
	{
		expected := ""
		got := secureInput("")
		require.Equal(t, expected, got)
	}

	t.Log("secure SHORT (<6) password")
	{
		expected := "***"
		got := secureInput("test")
		require.Equal(t, expected, got)
	}

	t.Log("secure (>6) password")
	{
		expected := "***"
		got := secureInput("asdfghjk")
		require.Equal(t, expected, got)
	}

	t.Log("secure SHORT (<6) url")
	{
		expected := "http://***"
		got := secureInput("http://te.hu")
		require.Equal(t, expected, got)
	}

	t.Log("secure url")
	{
		expected := "http://t***u"
		got := secureInput("http://test.hu")
		require.Equal(t, expected, got)
	}

	t.Log("secure LONG url")
	{
		expected := "http://tes***.hu"
		got := secureInput("http://test/alpha/beta.hu")
		require.Equal(t, expected, got)
	}

	t.Log("secure SHORT (<6) path")
	{
		expected := "file://***"
		got := secureInput("file://test")
		require.Equal(t, expected, got)
	}

	t.Log("secure path")
	{
		expected := "file://t***a"
		got := secureInput("file://test/beta")
		require.Equal(t, expected, got)
	}

	t.Log("secure LONG path")
	{
		expected := "file://tes***eta"
		got := secureInput("file://test/apha/beta")
		require.Equal(t, expected, got)
	}
}

func TestStip(t *testing.T) {
	t.Log(`Nothing to strip`)
	{
		line := `/Library/Keychains/System.keychain`

		got := strip(line)
		expected := `/Library/Keychains/System.keychain`
		require.Equal(t, expected, got)
	}

	t.Log(`Strip removes: (")`)
	{
		line := `"/Library/Keychains/System.keychain"`

		got := strip(line)
		expected := `/Library/Keychains/System.keychain`
		require.Equal(t, expected, got)
	}

	t.Log(`Strip removes: (\t)`)
	{
		line := `    /Library/Keychains/System.keychain       `

		got := strip(line)
		expected := `/Library/Keychains/System.keychain`
		require.Equal(t, expected, got)
	}

	t.Log(`Strip removes: (\n)`)
	{
		line := `

    /Library/Keychains/System.keychain

    `

		got := strip(line)
		expected := `/Library/Keychains/System.keychain`
		require.Equal(t, expected, got)
	}

	t.Log(`Strip`)
	{
		line := `

                      "/Library/Keychains/System.keychain"

    `

		got := strip(line)
		expected := `/Library/Keychains/System.keychain`
		require.Equal(t, expected, got)
	}
}

func TestSplitAndStrip(t *testing.T) {
	{
		str := `    "/Users/bitrise/Library/Keychains/login.keychain-db"
		"/Users/bitrise/Library/Keychains/login.keychain-db"
		"/Library/Keychains/System.keychain"`

		sep := "\n"
		expected := []string{"/Users/bitrise/Library/Keychains/login.keychain-db", "/Users/bitrise/Library/Keychains/login.keychain-db", "/Library/Keychains/System.keychain"}
		got := splitAndStrip(str, sep)
		require.Equal(t, expected, got)
	}
}

func TestSplitAndTrimSpace(t *testing.T) {
	{
		str := "pth1 | pth2 |   pth3"
		sep := "|"
		expected := []string{"pth1", "pth2", "pth3"}
		got := splitAndTrimSpace(str, sep)
		require.Equal(t, expected, got)
	}

	{
		str := "|"
		sep := "|"
		expected := []string{}
		got := splitAndTrimSpace(str, sep)
		require.Equal(t, expected, got)
	}

	{
		str := ""
		sep := "|"
		expected := []string{}
		got := splitAndTrimSpace(str, sep)
		require.Equal(t, expected, got)
	}

	{
		str := " | pth2 |   pth3"
		sep := "|"
		expected := []string{"pth2", "pth3"}
		got := splitAndTrimSpace(str, sep)
		require.Equal(t, expected, got)
	}

	{
		str := "pth1 |  |   pth3"
		sep := "|"
		expected := []string{"pth1", "pth3"}
		got := splitAndTrimSpace(str, sep)
		require.Equal(t, expected, got)
	}
}

func TestAppendWithoutDuplicatesAndKeepOrder(t *testing.T) {
	t.Log()
	{
		list := []string{"a", "b", "c"}
		item := "d"

		expected := []string{"a", "b", "c", "d"}
		got := appendWithoutDuplicatesAndKeepOrder(list, item)
		require.Equal(t, expected, got)
	}

	t.Log()
	{
		list := []string{"a", "b", "c"}
		item := "a"

		expected := []string{"a", "b", "c"}
		got := appendWithoutDuplicatesAndKeepOrder(list, item)
		require.Equal(t, expected, got)
	}

	t.Log()
	{
		list := []string{"a", "a", "b"}
		item := "a"

		expected := []string{"a", "b"}
		got := appendWithoutDuplicatesAndKeepOrder(list, item)
		require.Equal(t, expected, got)
	}
}
