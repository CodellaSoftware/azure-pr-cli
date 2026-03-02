package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDateRange_BothProvided(t *testing.T) {
	from, to, err := parseDateRange("2024-01-15", "2024-01-31")

	require.NoError(t, err)
	assert.Equal(t, 2024, from.Year())
	assert.Equal(t, time.January, from.Month())
	assert.Equal(t, 15, from.Day())
	assert.Equal(t, 0, from.Hour())
	assert.Equal(t, 0, from.Minute())
	assert.Equal(t, 0, from.Second())

	assert.Equal(t, 2024, to.Year())
	assert.Equal(t, time.January, to.Month())
	assert.Equal(t, 31, to.Day())
	assert.Equal(t, 23, to.Hour())
	assert.Equal(t, 59, to.Minute())
	assert.Equal(t, 59, to.Second())
}

func TestParseDateRange_ToIsEndOfDay(t *testing.T) {
	_, to, err := parseDateRange("2024-03-01", "2024-03-15")

	require.NoError(t, err)
	assert.Equal(t, 23, to.Hour())
	assert.Equal(t, 59, to.Minute())
	assert.Equal(t, 59, to.Second())
}

func TestParseDateRange_SameDayFromAndTo(t *testing.T) {
	from, to, err := parseDateRange("2024-06-15", "2024-06-15")

	require.NoError(t, err)
	assert.Equal(t, from.Day(), to.Day())
	assert.True(t, to.After(from))
}

func TestParseDateRange_EmptyFrom_DefaultsToStartOfMonth(t *testing.T) {
	now := time.Now().UTC()
	from, _, err := parseDateRange("", "2099-12-31")

	require.NoError(t, err)
	assert.Equal(t, now.Year(), from.Year())
	assert.Equal(t, now.Month(), from.Month())
	assert.Equal(t, 1, from.Day())
	assert.Equal(t, 0, from.Hour())
	assert.Equal(t, 0, from.Minute())
	assert.Equal(t, 0, from.Second())
}

func TestParseDateRange_EmptyTo_DefaultsToEndOfMonth(t *testing.T) {
	now := time.Now().UTC()
	_, to, err := parseDateRange("2000-01-01", "")

	require.NoError(t, err)
	assert.Equal(t, now.Year(), to.Year())
	assert.Equal(t, now.Month(), to.Month())
	assert.Equal(t, 23, to.Hour())
	assert.Equal(t, 59, to.Minute())
	assert.Equal(t, 59, to.Second())

	expectedLastDay := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1).Day()
	assert.Equal(t, expectedLastDay, to.Day())
}

func TestParseDateRange_BothEmpty_CurrentMonth(t *testing.T) {
	now := time.Now().UTC()
	from, to, err := parseDateRange("", "")

	require.NoError(t, err)
	assert.Equal(t, now.Year(), from.Year())
	assert.Equal(t, now.Month(), from.Month())
	assert.Equal(t, 1, from.Day())
	assert.Equal(t, now.Year(), to.Year())
	assert.Equal(t, now.Month(), to.Month())
	assert.True(t, to.After(from))
}

func TestParseDateRange_InvalidFromDate(t *testing.T) {
	_, _, err := parseDateRange("not-a-date", "2024-01-31")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from date format")
}

func TestParseDateRange_InvalidToDate(t *testing.T) {
	_, _, err := parseDateRange("2024-01-01", "31/01/2024")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to date format")
}

func TestParseDateRange_FromAfterTo(t *testing.T) {
	_, _, err := parseDateRange("2024-02-01", "2024-01-01")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "from date must be before to date")
}

func TestParseDateRange_WrongFormatSlashes(t *testing.T) {
	_, _, err := parseDateRange("01/15/2024", "2024-01-31")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from date format")
}

func TestParseDateRange_MonthBoundary(t *testing.T) {
	from, to, err := parseDateRange("2024-02-28", "2024-03-01")

	require.NoError(t, err)
	assert.Equal(t, time.February, from.Month())
	assert.Equal(t, time.March, to.Month())
	assert.True(t, to.After(from))
}
