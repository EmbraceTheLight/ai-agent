package utils

import (
	"time"

	"github.com/yitter/idgenerator-go/idgen"
)

func init() {
	tmp := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	baseTime := tmp / 1e6
	var options = idgen.NewIdGeneratorOptions(1)
	options.BaseTime = baseTime
	idgen.SetIdGenerator(options)
}

// GetID generates ID by snowflake algorithm
func GetID() int64 {
	return idgen.NextId()
}
