package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dunglas/calavera/extractor"
	"github.com/dunglas/calavera/schema"
)

const filePerms = 0644
const dirPerms = 0755

func main() {
	flag.Usage = func() {
		fmt.Println("calavera input_directory output_directory")
	}

	prettifyBool := flag.Bool("prettify", false, "Prettify json output")

	flag.Parse()

	if len(flag.Args()) != 2 {
		log.Fatalln("Input and output directories are mandatory arguments.")
	}

	var files []string
	var extractors = []extractor.Extractor{extractor.Markdown{}, extractor.Git{}}

	inputPath, err := filepath.Abs(flag.Arg(0))
	check(err)

	outputPath, err := filepath.Abs(flag.Arg(1))
	check(err)

	wd, err := os.Getwd()
	if nil != err {
		check(err)
	}

	if err := os.Chdir(inputPath); err != nil {
		check(err)
	}

	walkFunc := func(path string, _ os.FileInfo, err error) error {
		if nil != err || !strings.HasSuffix(path, ".md") {
			return nil
		}

		abs, err := filepath.Abs(path)
		check(err)
		rel, err := filepath.Rel(inputPath, abs)
		check(err)
		files = append(files, rel)

		return nil
	}

	if err := filepath.Walk(inputPath, walkFunc); nil != err {
		check(err)
	}

	entrypoint := schema.NewItemList()
	var wg sync.WaitGroup
	for _, file := range files {
		wg.Add(1)
		go func(file string) {
			convert(file, outputPath, extractors, *prettifyBool)
			defer wg.Done()
		}(file)

		entrypoint.Element = append(entrypoint.Element, strings.Replace(file, ".md", ".jsonld", 1))
	}

	wg.Wait()

	if err := os.Chdir(wd); err != nil {
		check(err)
	}

	check(ioutil.WriteFile(outputPath+"/_index.jsonld", marshal(entrypoint, *prettifyBool), filePerms))
}

func marshal(v interface{}, prettify bool) []byte {
	var jsonContent []byte
	var err error
	if prettify {
		jsonContent, err = json.MarshalIndent(v, "", "\t")
	} else {
		jsonContent, err = json.Marshal(v)
	}
	check(err)

	return jsonContent
}

func convert(path string, outputDirectory string, extractors []extractor.Extractor, prettify bool) {
	creativeWork := schema.NewCreativeWork()

	for _, extractor := range extractors {
		err := extractor.Extract(creativeWork, path)
		check(err)
	}

	jsonContent := marshal(creativeWork, prettify)

	outputPath := fmt.Sprint(outputDirectory, "/", path[:len(path)-3], ".jsonld")
	outputSubdirectory := filepath.Dir(outputPath)

	err := os.MkdirAll(outputSubdirectory, dirPerms)
	check(err)

	err = ioutil.WriteFile(outputPath, jsonContent, filePerms)
	check(err)
}

func check(err error) {
	if err != nil {
		log.Fatalln(err)
		panic(err)
	}
}
