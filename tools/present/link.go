package present

import "net/url"

type Link struct {
	URL   *url.URL
	Label string
}

func (l Link) TemplateName() string { return "link" }