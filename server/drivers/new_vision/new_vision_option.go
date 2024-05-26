package new_vision

import (
	"github.com/PuerkitoBio/goquery"
	"strings"
	"videocapture/server/spider"
)

// parseData 解析多信息
func parseData(r *goquery.Document, domKey, selector string, attrs map[int]string) (documents []*Dom) {
	document := &Dom{DomKey: domKey, DomVal: "", Sort: 1, Attrs: map[string]string{}}
	for t, attr := range attrs {
		if t == AttrTypeAttr {
			if s, exists := r.Find(selector).First().Attr(attr); exists {
				document.Attrs[attr] = strings.TrimSpace(s)
			}
		} else if t == AttrTypeText {
			document.DomVal = strings.TrimSpace(r.Find(selector).First().Text())
		}
	}

	documents = append(documents, document)
	return
}

// parseListData 解析多信息
func parseListData(r *goquery.Document, domKey, selector string, attrs map[int]string) (documents []*Dom) {
	documents = make([]*Dom, 0)
	r.Find(selector).Each(func(i int, rs *goquery.Selection) {
		dom := &Dom{DomKey: domKey, DomVal: strings.TrimSpace(rs.Text()), Sort: i + 1, Attrs: map[string]string{}}
		for t, attr := range attrs {
			if t == AttrTypeAttr {
				if s, exists := rs.Attr(attr); exists {
					dom.Attrs[attr] = strings.TrimSpace(s)
				}
			} else if t == AttrTypeText {
				dom.DomVal = strings.TrimSpace(rs.Text())
			}
		}

		documents = append(documents, dom)
	})
	return
}

func GenerateOpts(pt spider.SpiderType) []DomOpt {
	opts := make([]DomOpt, 0)
	columns := PageTypeColumns[pt]
	for _, column := range columns {
		if conf, ok := VideoColumnConfigs[column]; ok {
			opts = append(opts, createOptFunc(column, conf))
		}
	}
	return opts
}

func createOptFunc(key string, conf *ColumnSpiderConfig) DomOpt {
	if conf.ResultType == "list" {
		return func(doc *goquery.Document) []*Dom {
			return parseListData(doc, key, conf.Selector, conf.DataMap)
		}
	} else {
		return func(doc *goquery.Document) []*Dom {
			return parseData(doc, key, conf.Selector, conf.DataMap)
		}
	}
}
