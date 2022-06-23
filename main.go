package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
)

func main() {
	var appendMode bool
	flag.BoolVar(&appendMode, "a", false, "Append the value instead of replacing it")

	var ignorePath bool
	flag.BoolVar(&ignorePath, "ignore-path", false, "Ignore the path when considering what constitutes a duplicate")

	var wordlistFilename string
	flag.StringVar(&wordlistFilename, "w", "", "Wordlist with param values")

	flag.Parse()

	wordlist, err := readWordlist(wordlistFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read wordlist file %s: %v\n", wordlistFilename, err)
		os.Exit(1)
	}
	if len(wordlist) == 0 {
		wordlist = []string{flag.Arg(0)}
	}

	seen := make(map[string]bool)

	// read URLs on stdin, then replace the values in the query string
	// with some user-provided value
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		u, err := url.Parse(sc.Text())
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse url %s [%s]\n", sc.Text(), err)
			continue
		}

		// Go's maps aren't ordered, but we want to use all the param names
		// as part of the key to output only unique requests. To do that, put
		// them into a slice and then sort it.
		pp := make([]string, 0)
		for p, _ := range u.Query() {
			pp = append(pp, p)
		}
		sort.Strings(pp)

		key := fmt.Sprintf("%s%s?%s", u.Hostname(), u.EscapedPath(), strings.Join(pp, "&"))
		if ignorePath {
			key = fmt.Sprintf("%s?%s", u.Hostname(), strings.Join(pp, "&"))
		}

		// Only output each host + path + params combination once
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = true

		resultQueries := replaceQueryStrings(u.Query(), wordlist, appendMode)
		for _, rqs := range resultQueries {
			u.RawQuery = rqs
			fmt.Printf("%s\n", u)
		}

	}

}

func readWordlist(filename string) ([]string, error) {
	if filename == "" {
		return nil, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var wordlist []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		wordlist = append(wordlist, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return wordlist, nil
}

func replaceQueryStrings(input url.Values, wordlist []string, appendMode bool) []string {
	results := make([]string, 0, len(wordlist))
	for _, w := range wordlist {
		qs := url.Values{}
		for param, vv := range input {
			if appendMode {
				qs.Set(param, vv[0]+w)
			} else {
				qs.Set(param, w)
			}
		}
		results = append(results, qs.Encode())
	}
	return results
}
