package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	v8 "rogchap.com/v8go"
)

func main() {
	fmt.Println("Testing memory")

	file, err := os.ReadFile("./file.html")

	if err != nil {
		log.Panic("Unable to read file")
	}

	// Quick and dirty sanitize html to inject it as a string
	html := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(string(file), "/script", "/scri pt"), "\n", ""), "\"", "\\\"")

	var source = fmt.Sprintf(`
h = htmlParser("%s");
f = h.find('table.comment-tree')
console.log(f.attr('border'))
`, html)

	for i := 1; i < 1000; i++ {

		if i%100 == 0 {
			fmt.Printf("Ran %d\n", i)
		}
		isolate := v8.NewIsolate()
		global := v8.NewObjectTemplate(isolate)

		if err := InjectParser(isolate, "htmlParser", global); err != nil {
			log.Panic("could not inject htmlParser")
		}

		context := v8.NewContext(isolate, global)
		_, err := context.RunScript(source, "function.js")

		if err != nil {
			log.Panicf("could not run script: %s", err)
		}

		context.Close()
		isolate.Dispose()
	}

	time.Sleep(180 * time.Second)
}
