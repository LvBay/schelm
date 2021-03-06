package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
)

var force bool = false

func init() {
	flag.BoolVar(&force, "f", false, "Delete existing output directory")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: schelm [options] OUTPUT_DIR\n")
		flag.PrintDefaults()
	}
}

var yamlSeperator = []byte("---\n# Source: ")

func scanYamlSpecs(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, yamlSeperator); i >= 0 {
		// We have a full newline-terminated line.
		return i + len(yamlSeperator), data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func splitSpec(token string) (string, string) {
	if i := strings.Index(token, "\n"); i >= 0 {
		return token[0:i], token[i+1:]
	}
	return "", ""
}

func main() {

	flag.Parse()

	if flag.Arg(0) == "" {
		flag.Usage()
		os.Exit(1)
	}

	output_directory := flag.Arg(0)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(scanYamlSpecs)
	//Allow for tokens (specs) up to 1M in size
	scanner.Buffer(make([]byte, bufio.MaxScanTokenSize), 1048576)
	//Discard the first result, we only care about everything after the first seperator
	scanner.Scan()
	flag := true

	for scanner.Scan() {
		source, content := splitSpec(scanner.Text())
		destinationFile := path.Join(output_directory, source)
		destinationFile = strings.Replace(destinationFile, "/templates", "", -1)
		destinationFile = strings.Replace(destinationFile, "/charts", "", -1)
		dir := path.Dir(destinationFile)
		if force && flag {
			if len(strings.Split(source, "/")) == 0 {
				log.Fatalf(`invalid output source dir`, source)
				return
			}
			firstPath := strings.Split(source, "/")[0]
			oldRootPath := path.Join(output_directory, firstPath)
			log.Printf("Deleting %s (force)\n", oldRootPath)
			os.RemoveAll(oldRootPath)
			flag = false
		}
		if err := os.MkdirAll(dir, 0750); err != nil {
			log.Fatalf("Error creating %s: %s ", dir, err)
		}
		if _, err := os.Stat(destinationFile); os.IsNotExist(err) {
			log.Printf("Creating %s", destinationFile)
			if err := ioutil.WriteFile(destinationFile, []byte(content), 0640); err != nil {
				log.Fatalf("Error: %s", err)
			}
		} else {
			f, err := os.OpenFile(destinationFile, os.O_APPEND|os.O_WRONLY, 0640)
			if err != nil {
				log.Fatalf("Error re-opening file %s: %s", destinationFile, err)
			}
			f.WriteString("\n---\n")
			f.WriteString(content)
			f.Close()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error while processing manifest: %s", err)
	}

}
