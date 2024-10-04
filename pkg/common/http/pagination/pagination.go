package pagination

import (
	"errors"
	"strconv"
	"unicode"
)

type Meta struct {
	Page   int `json:"page"`
	Limit  int `json:"limit"`
	Offset int `json:"-"`
	Count  int `json:"count"`
}

func Parse(queryLimit, queryPage string) (meta Meta, err error) {
	meta.Count = 0

	if !IsNumber(queryLimit) {
		return meta, errors.New("Limit must be a valid numeric value")
	}

	if !IsNumber(queryPage) {
		return meta, errors.New("Page must be a valid numeric value")
	}

	limit, err := strconv.Atoi(queryLimit)
	if err != nil {
		limit = 25
	}
	meta.Limit = limit

	page, err := strconv.Atoi(queryPage)
	if err != nil {
		page = 1
	}
	meta.Offset = (page - 1) * limit
	meta.Page = page
	return meta, nil
}

func IsNumber(s string) bool {
	for _, r := range s {
		if !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}
