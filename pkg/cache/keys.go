package cache

import (
	"strconv"

	"github.com/cespare/xxhash/v2"

	"history-api/pkg/jsonx"
)

func Key(prefix, id string) string {
	return prefix + ":" + id
}

func Key2(prefix, first, second string) string {
	return prefix + ":" + first + ":" + second
}

func QueryKey(prefix string, params any) string {
	data, _ := jsonx.Marshal(params)
	return prefix + ":" + strconv.FormatUint(xxhash.Sum64(data), 16)
}
