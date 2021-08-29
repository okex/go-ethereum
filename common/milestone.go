package common

import (
	"strconv"
	"sync"
)

var (
	MILESTONE_VENUS_HEIGHT     string
	milestoneVenusHeight       int64

	once                         sync.Once
)

func string2number(input string) int64 {
	if len(input) == 0 {
		input = "0"
	}
	res, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		panic(err)
	}
	return res
}

func initVersionBlockHeight() {
	once.Do(func() {
		milestoneVenusHeight = string2number(MILESTONE_VENUS_HEIGHT)
	})
}

func init() {
	initVersionBlockHeight()
}

//fix CVE-2021-39137
func HigherThanVenus(height int64) bool {
	if milestoneVenusHeight == 0 {
		// milestoneMercuryHeight not enabled
		return false
	}
	return height > milestoneVenusHeight
}