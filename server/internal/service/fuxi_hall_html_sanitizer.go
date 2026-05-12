package service

import (
	stdhtml "html"
	"strings"

	xhtml "golang.org/x/net/html"
)

var (
	fuxiHallAllowedHTMLTags = map[string]struct{}{
		"p":          {},
		"br":         {},
		"strong":     {},
		"b":          {},
		"em":         {},
		"i":          {},
		"u":          {},
		"s":          {},
		"ul":         {},
		"ol":         {},
		"li":         {},
		"blockquote": {},
		"h1":         {},
		"h2":         {},
		"h3":         {},
		"h4":         {},
		"a":          {},
		"code":       {},
		"pre":        {},
		"img":        {},
	}
	fuxiHallDroppedHTMLTags = map[string]struct{}{
		"script": {},
		"style":  {},
		"iframe": {},
		"object": {},
	}
	fuxiHallAllowedHTMLAttrs = map[string]map[string]struct{}{
		"a": {
			"href":   {},
			"target": {},
			"rel":    {},
		},
		"img": {
			"src": {},
			"alt": {},
		},
	}
	fuxiHallVoidHTMLTags = map[string]struct{}{
		"br":  {},
		"img": {},
	}
)

func sanitizeRichTextHTML(input string) string {
	source := strings.TrimSpace(input)
	if source == "" {
		return ""
	}

	doc, err := xhtml.Parse(strings.NewReader("<div>" + source + "</div>"))
	if err != nil {
		return ""
	}
	container := findFirstHTMLElement(doc, "div")
	if container == nil {
		return ""
	}

	var builder strings.Builder
	for node := container.FirstChild; node != nil; node = node.NextSibling {
		renderSanitizedHTMLNode(&builder, node)
	}
	return strings.TrimSpace(builder.String())
}

func findFirstHTMLElement(node *xhtml.Node, tag string) *xhtml.Node {
	if node == nil {
		return nil
	}
	if node.Type == xhtml.ElementNode && strings.EqualFold(node.Data, tag) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		found := findFirstHTMLElement(child, tag)
		if found != nil {
			return found
		}
	}
	return nil
}

func renderSanitizedHTMLNode(builder *strings.Builder, node *xhtml.Node) {
	switch node.Type {
	case xhtml.TextNode:
		builder.WriteString(stdhtml.EscapeString(node.Data))
	case xhtml.ElementNode:
		tag := strings.ToLower(strings.TrimSpace(node.Data))
		if tag == "" {
			return
		}
		if _, dropped := fuxiHallDroppedHTMLTags[tag]; dropped {
			return
		}
		if _, allowed := fuxiHallAllowedHTMLTags[tag]; !allowed {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				renderSanitizedHTMLNode(builder, child)
			}
			return
		}

		builder.WriteString("<")
		builder.WriteString(tag)
		for _, attr := range filterSanitizedHTMLAttrs(tag, node.Attr) {
			builder.WriteString(" ")
			builder.WriteString(attr.Key)
			builder.WriteString(`="`)
			builder.WriteString(stdhtml.EscapeString(attr.Val))
			builder.WriteString(`"`)
		}

		if _, isVoid := fuxiHallVoidHTMLTags[tag]; isVoid {
			builder.WriteString(">")
			return
		}

		builder.WriteString(">")
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderSanitizedHTMLNode(builder, child)
		}
		builder.WriteString("</")
		builder.WriteString(tag)
		builder.WriteString(">")
	}
}

func filterSanitizedHTMLAttrs(tag string, attrs []xhtml.Attribute) []xhtml.Attribute {
	allowedAttrs := fuxiHallAllowedHTMLAttrs[tag]
	if len(allowedAttrs) == 0 {
		return nil
	}

	result := make([]xhtml.Attribute, 0, len(attrs))
	for _, attr := range attrs {
		key := strings.ToLower(strings.TrimSpace(attr.Key))
		if _, ok := allowedAttrs[key]; !ok {
			continue
		}

		value := strings.TrimSpace(attr.Val)
		switch tag {
		case "a":
			if key == "href" {
				if !isSafeAnchorURL(value) {
					continue
				}
			}
			if key == "target" {
				if value != "_blank" && value != "_self" {
					continue
				}
			}
		case "img":
			if key == "src" && !isSafeImageURL(value) {
				continue
			}
		}

		result = append(result, xhtml.Attribute{Key: key, Val: value})
	}
	return result
}

func isSafeAnchorURL(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:")
}

func isSafeImageURL(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "data:image/")
}
