package web

// Looted from https://github.com/gregberge/content-range, all credit for the tests goes to @gregberge

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/rs/zerolog/log"
)

type ContentRange struct {
	Unit  string
	Start int
	End   int
	Size  int
}

var (
	contentRangeRegex = regexp.MustCompile(`(?m)(\w+) ((\d+)-(\d+)|\*)/(\d+|\*)`)
	ErrContentRange   = errors.New("invalid content-range header")
)

func ParseContentRange(value string) (ContentRange, error) {
	result := ContentRange{}

	parts := contentRangeRegex.FindStringSubmatch(value)
	if parts == nil {
		return ContentRange{}, ErrContentRange
	}
	if len(parts) != 6 { // Should never satisfy this but I'm paranoid
		log.Error().Msg("Failed to parse Content-Range header, parts regexed is not 6")
		return ContentRange{}, ErrContentRange
	}

	result.Unit = parts[1]

	if start, err := strconv.Atoi(parts[3]); err != nil {
		result.Start = -1
	} else {
		result.Start = start
	}

	if end, err := strconv.Atoi(parts[4]); err != nil {
		result.End = -1
	} else {
		result.End = end
	}

	if size, err := strconv.Atoi(parts[5]); err != nil {
		result.Size = -1
	} else {
		result.Size = size
	}

	if result.Size == -1 && result.Start == -1 && result.End == -1 {
		return ContentRange{}, ErrContentRange
	}

	return result, nil
}
