package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatches_SingleKey(t *testing.T) {
	result := Matches("q", []string{"q"})
	assert.True(t, result)
}

func TestMatches_MultipleKeys(t *testing.T) {
	result := Matches("ctrl+c", []string{"q", "ctrl+c"})
	assert.True(t, result)
}

func TestMatches_NoMatch(t *testing.T) {
	result := Matches("x", []string{"q", "ctrl+c"})
	assert.False(t, result)
}

func TestMatches_EmptyBindings(t *testing.T) {
	result := Matches("q", []string{})
	assert.False(t, result)
}

func TestMatches_SpaceAlias(t *testing.T) {
	result := Matches(" ", []string{"space"})
	assert.True(t, result)
}

func TestMatches_SpaceAliasReverse(t *testing.T) {
	result := Matches(" ", []string{" "})
	assert.True(t, result)
}

func TestNormalizeKey(t *testing.T) {
	assert.Equal(t, " ", NormalizeKey("space"))
	assert.Equal(t, "q", NormalizeKey("q"))
}

func TestDefaultKeyMap_MatchesHardcoded(t *testing.T) {
	km := DefaultKeyMap()

	assert.Equal(t, []string{"k", "up"}, km.Up)
	assert.Equal(t, []string{"j", "down"}, km.Down)
	assert.Equal(t, []string{"h"}, km.PaneLeft)
	assert.Equal(t, []string{"l", "right"}, km.PaneRight)
	assert.Equal(t, []string{"left"}, km.TreeNav)
	assert.Equal(t, []string{" "}, km.Collapse)
	assert.Equal(t, []string{"tab"}, km.CycleFilter)
	assert.Equal(t, []string{"t"}, km.CycleTime)
	assert.Equal(t, []string{"/"}, km.Search)
	assert.Equal(t, []string{"r"}, km.Refresh)
	assert.Equal(t, []string{"enter"}, km.Launch)
	assert.Equal(t, []string{"q", "ctrl+c"}, km.Quit)
}
