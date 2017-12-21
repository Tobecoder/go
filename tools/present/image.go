package present

func init() {
	Register("image", parseImage)
}

type Image struct {
	URL    string
	Width  int
	Height int
}

func (i Image) TemplateName() string { return "image" }

func parseImage(ctx *Context, fileName string, lineno int, text string) (Elem, error) {
	var e Elem
	return e, nil
}