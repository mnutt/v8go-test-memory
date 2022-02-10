package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	v8 "rogchap.com/v8go"
)

type HtmlParser struct {
	ctx      *v8.Context
	jsObject *v8.ObjectTemplate
	doc      *goquery.Document
}

type HtmlParserSelection struct {
	ctx       *v8.Context
	jsObject  *v8.ObjectTemplate
	selection *goquery.Selection
}

func InjectParser(isolate *v8.Isolate, variableName string, global *v8.ObjectTemplate) error {

	htmlParser := v8.NewFunctionTemplate(isolate, getHtmlParserCallback())

	err := global.Set(variableName, htmlParser)
	return err
}

func getHtmlParserCallback() v8.FunctionCallback {
	return func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		html := args[0].String()
		ctx := info.Context()

		parser, err := NewHtmlParser(ctx, html)
		if err != nil {
			return nil
		}

		jsParser, err := parser.jsObject.NewInstance(ctx)
		if err != nil {
			log.Panic("Error!")
		}

		return jsParser.Value
	}
}

func NewHtmlParser(ctx *v8.Context, html string) (*HtmlParser, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {
		return nil, err
	}

	isolate := ctx.Isolate()

	jsObject := v8.NewObjectTemplate(isolate)

	parser := &HtmlParser{
		ctx:      ctx,
		jsObject: jsObject,
		doc:      doc,
	}

	_ = parser.jsObject.Set("find", v8.NewFunctionTemplate(isolate, parser.getFindCallback()), v8.ReadOnly)

	return parser, nil
}

func NewHtmlParserSelection(ctx *v8.Context, selection *goquery.Selection) (*HtmlParserSelection, error) {
	isolate := ctx.Isolate()
	jsObject := v8.NewObjectTemplate(isolate)

	parserSelection := &HtmlParserSelection{
		ctx:       ctx,
		jsObject:  jsObject,
		selection: selection,
	}

	_ = parserSelection.jsObject.Set("each", v8.NewFunctionTemplate(isolate, parserSelection.eachFn), v8.ReadOnly)
	_ = parserSelection.jsObject.Set("map", v8.NewFunctionTemplate(isolate, parserSelection.mapFn), v8.ReadOnly)
	_ = parserSelection.jsObject.Set("attr", v8.NewFunctionTemplate(isolate, parserSelection.attrFn), v8.ReadOnly)

	return parserSelection, nil
}

func (s *HtmlParserSelection) eachFn(info *v8.FunctionCallbackInfo) *v8.Value {
	args := info.Args()
	cb, err := args[0].AsFunction()
	ctx := info.Context()
	iso := ctx.Isolate()

	if err != nil {
		strErr, _ := v8.NewValue(iso, "First argument of each() must be a function")
		iso.ThrowException(strErr)
	}

	for i := range s.selection.Nodes {
		subselection, err := NewHtmlParserSelection(ctx, s.selection.Eq(i))

		if err != nil {
			throwError(iso, "Error instantiating subselection")
			return nil
		}

		subselectionObject, err := subselection.jsObject.NewInstance(info.Context())
		if err != nil {
			throwError(iso, "Error instantiating subselection")
			return nil
		}

		_, err = cb.Call(v8.Null(iso), subselectionObject)

		if err != nil {
			throwError(iso, "Error calling callback")
			return nil
		}
	}

	return nil
}

func (s *HtmlParserSelection) mapFn(info *v8.FunctionCallbackInfo) *v8.Value {
	args := info.Args()
	ctx := info.Context()
	iso := ctx.Isolate()

	cb, err := args[0].AsFunction()
	if err != nil {
		throwError(iso, "First argument of map() must be a function")
		return nil
	}

	arrayValue, err := ctx.RunScript("new Array()", "")
	if err != nil {
		throwError(iso, "Unable to create allocate array")
		return nil
	}

	arrayObj := arrayValue.Object()

	err = arrayObj.Set("length", uint32(s.selection.Size()))
	if err != nil {
		throwError(iso, fmt.Sprintf("Could not set map array length: %s", err))
		return nil
	}

	for i := range s.selection.Nodes {
		subselection, err := NewHtmlParserSelection(ctx, s.selection.Eq(i))

		if err != nil {
			throwError(iso, fmt.Sprintf("Error instantiating subselection: %s", err))
			return nil
		}

		subselectionObject, err := subselection.jsObject.NewInstance(info.Context())
		if err != nil {
			throwError(iso, fmt.Sprintf("Error instantiating subselection: %s", err))
			return nil
		}

		result, err := cb.Call(v8.Null(iso), subselectionObject)

		if err != nil {
			throwError(iso, fmt.Sprintf("Error running callback: %s", err))
			return nil
		}

		err = arrayObj.SetIdx(uint32(i), result)

		if err != nil {
			throwError(iso, fmt.Sprintf("Error adding element to array: %s", err))
			return nil
		}
	}

	return arrayObj.Value
}

func throwError(iso *v8.Isolate, err string) {
	strErr, _ := v8.NewValue(iso, err)
	iso.ThrowException(strErr)
}

func (s *HtmlParserSelection) attrFn(info *v8.FunctionCallbackInfo) *v8.Value {
	args := info.Args()
	iso := info.Context().Isolate()

	attr, exists := s.selection.Attr(args[0].String())

	if exists {
		val, _ := v8.NewValue(iso, attr)
		return val
	} else {
		return v8.Null(iso)
	}
}

func (p *HtmlParser) getFindCallback() v8.FunctionCallback {
	return func(info *v8.FunctionCallbackInfo) *v8.Value {
		args := info.Args()
		query := args[0].String()
		ctx := info.Context()
		iso := ctx.Isolate()

		selection, err := NewHtmlParserSelection(ctx, p.doc.Find(query))

		if err != nil {
			throwError(iso, fmt.Sprintf("Error instantiating html parser: %s", err))
			return nil
		}

		jsSelection, err := selection.jsObject.NewInstance(ctx)

		if err != nil {
			throwError(iso, fmt.Sprintf("Error instantiating html parser: %s", err))
			return nil
		}

		return jsSelection.Value
	}
}
