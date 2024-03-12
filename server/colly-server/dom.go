package collyserver

import "regexp"

type DomSelector interface {
	ByRegexp(resource string) string
}

func ByRegexp(resource string) string {
	reg := regexp.MustCompile(`<div class="text-base md:text-xl mb-1">(.*?)</div>`)
	params := reg.FindStringSubmatch(resource)
	if len(params) != 2 {
		return ""
	}
	return params[1]
}
